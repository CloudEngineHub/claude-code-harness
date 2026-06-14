package nightwatch

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/plans"

	_ "modernc.org/sqlite"
)

const (
	defaultStaleTaskHours     = int64(72)
	defaultOpenDecisionHours  = int64(168)
	defaultUnresolvedLoopHours = int64(1)
)

var (
	nowFn = time.Now

	reOpenStatus = regexp.MustCompile(`(?i)\*\*status\*\*:\s*open`)
	reDecisionH  = regexp.MustCompile(`^##\s+([0-9]{4}-[0-9]{2}-[0-9]{2}):\s*(.+)$`)
)

// StaleConfig holds patrol thresholds loaded from templates/night-watch-config.yaml.
type StaleConfig struct {
	StaleTaskHours     int64
	OpenDecisionHours  int64
}

// LoadStaleConfig reads stale_task_hours / open_decision_hours from config YAML.
func LoadStaleConfig(configPath string) (StaleConfig, error) {
	cfg := StaleConfig{
		StaleTaskHours:    defaultStaleTaskHours,
		OpenDecisionHours: defaultOpenDecisionHours,
	}
	f, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "stale_task_hours:"):
			if v, err := parseYAMLInt(line); err == nil {
				cfg.StaleTaskHours = v
			}
		case strings.HasPrefix(line, "open_decision_hours:"):
			if v, err := parseYAMLInt(line); err == nil {
				cfg.OpenDecisionHours = v
			}
		}
	}
	return cfg, scanner.Err()
}

func parseYAMLInt(line string) (int64, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
}

// DetectStaleTasks returns active Plans.md tasks when the file has not been updated within staleHours.
func DetectStaleTasks(plansPath string, staleHours int64, now time.Time) ([]StaleTask, error) {
	fi, err := os.Stat(plansPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	ageHours := now.Sub(fi.ModTime()).Hours()
	if ageHours <= float64(staleHours) {
		return nil, nil
	}

	data, err := os.ReadFile(plansPath)
	if err != nil {
		return nil, err
	}
	var stale []StaleTask
	for _, task := range plans.ParseMarkdown(string(data)) {
		if task.Tags.Done {
			continue
		}
		if !(task.Tags.Wip || task.Tags.Todo || task.Tags.Blocked) {
			continue
		}
		stale = append(stale, StaleTask{
			TaskID:   task.TaskID,
			Status:   task.Status,
			AgeHours: ageHours,
		})
	}
	return stale, nil
}

// DetectOpenDecisions returns decisions marked **Status**: Open older than openHours.
func DetectOpenDecisions(decisionsPath string, openHours int64, now time.Time) ([]OpenDecision, error) {
	data, err := os.ReadFile(decisionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var (
		currentID    string
		currentTitle string
		currentDate  time.Time
		hasDate      bool
		open         bool
		results      []OpenDecision
	)

	flush := func() {
		if !open || currentID == "" {
			return
		}
		ageBase := currentDate
		if !hasDate {
			if fi, err := os.Stat(decisionsPath); err == nil {
				ageBase = fi.ModTime()
			} else {
				ageBase = now
			}
		}
		ageHours := now.Sub(ageBase).Hours()
		if ageHours > float64(openHours) {
			results = append(results, OpenDecision{
				DecisionID: currentID,
				Title:      currentTitle,
				AgeHours:   ageHours,
			})
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := reDecisionH.FindStringSubmatch(trimmed); len(m) == 3 {
			flush()
			currentID = strings.TrimSpace(m[2])
			currentTitle = currentID
			if d, err := time.Parse("2006-01-02", m[1]); err == nil {
				currentDate = d
				hasDate = true
			} else {
				hasDate = false
			}
			open = false
			continue
		}
		if reOpenStatus.MatchString(trimmed) {
			open = true
		}
	}
	flush()
	return results, nil
}

// UnresolvedLoopsFromMailbox scans bridge_events for request events without a matching response.
func UnresolvedLoopsFromMailbox(dbPath string, now time.Time) ([]UnresolvedLoop, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT event_id, source, event_type, payload_json, ts FROM bridge_events ORDER BY ts ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pending struct {
		eventID   string
		eventType string
		source    string
		taskID    string
		ts        int64
	}
	requests := make(map[string]pending)
	responses := make(map[string]bool)

	for rows.Next() {
		var eventID, source, eventType, payloadJSON string
		var ts int64
		if err := rows.Scan(&eventID, &source, &eventType, &payloadJSON, &ts); err != nil {
			return nil, err
		}
		key := loopKey(eventType, payloadJSON)
		if key == "" {
			continue
		}
		if isLoopResponse(eventType) {
			responses[key] = true
			continue
		}
		if isLoopRequest(eventType) {
			taskID := extractTaskID(payloadJSON)
			requests[key] = pending{
				eventID:   eventID,
				eventType: eventType,
				source:    source,
				taskID:    taskID,
				ts:        ts,
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var loops []UnresolvedLoop
	for key, req := range requests {
		if responses[key] {
			continue
		}
		eventTime := time.Unix(0, req.ts)
		ageHours := now.Sub(eventTime).Hours()
		if ageHours < float64(defaultUnresolvedLoopHours) {
			continue
		}
		loops = append(loops, UnresolvedLoop{
			EventID:   req.eventID,
			EventType: req.eventType,
			Source:    req.source,
			TaskID:    req.taskID,
			AgeHours:  ageHours,
		})
	}
	return loops, nil
}

func isLoopRequest(eventType string) bool {
	switch eventType {
	case "advisor-request", "review-request", "worker-report":
		return true
	default:
		return strings.HasSuffix(eventType, "-request")
	}
}

func isLoopResponse(eventType string) bool {
	switch eventType {
	case "advisor-response", "review-result":
		return true
	default:
		return strings.HasSuffix(eventType, "-response") || strings.HasSuffix(eventType, "-result")
	}
}

func loopKey(eventType, payloadJSON string) string {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return ""
	}
	taskID, _ := payload["task_id"].(string)
	trigger, _ := payload["trigger_hash"].(string)
	if taskID == "" && trigger == "" {
		if conv, ok := payload["conversation_id"].(string); ok {
			return eventType + ":" + conv
		}
		return ""
	}
	return taskID + ":" + trigger
}

func extractTaskID(payloadJSON string) string {
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return ""
	}
	if taskID, ok := payload["task_id"].(string); ok {
		return taskID
	}
	return ""
}

// BuildReportOptions configures report generation.
type BuildReportOptions struct {
	RepoRoot       string
	DryRun         bool
	Now            time.Time
	BridgeHome     string
	PlansPath      string
	DecisionsPath  string
	ConfigPath     string
	SchemaPath     string
}

// BuildReport assembles a schema-valid night-watch report.
func BuildReport(opts BuildReportOptions) (Report, error) {
	now := opts.Now
	if now.IsZero() {
		now = nowFn()
	}
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return Report{}, err
		}
		repoRoot = wd
	}

	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = filepath.Join(repoRoot, ConfigRelPath)
	}
	staleCfg, err := LoadStaleConfig(configPath)
	if err != nil {
		return Report{}, err
	}

	health := Check()
	loops, err := unresolvedLoopsForReport(opts, now)
	if err != nil {
		return Report{}, err
	}

	plansPath := opts.PlansPath
	if plansPath == "" {
		plansPath = filepath.Join(repoRoot, "Plans.md")
	}
	staleTasks, err := DetectStaleTasks(plansPath, staleCfg.StaleTaskHours, now)
	if err != nil {
		return Report{}, err
	}

	decisionsPath := opts.DecisionsPath
	if decisionsPath == "" {
		decisionsPath = filepath.Join(repoRoot, ".claude", "memory", "decisions.md")
	}
	openDecisions, err := DetectOpenDecisions(decisionsPath, staleCfg.OpenDecisionHours, now)
	if err != nil {
		return Report{}, err
	}

	report := Report{
		SchemaVersion:   "night-watch-report.v1",
		GeneratedAt:     now.UTC().Format(time.RFC3339),
		DryRun:          opts.DryRun,
		Health:          ReportHealth{Healthy: health.Healthy, Reason: health.Reason},
		UnresolvedLoops: loops,
		StaleTasks:      staleTasks,
		OpenDecisions:   openDecisions,
	}
	if report.UnresolvedLoops == nil {
		report.UnresolvedLoops = []UnresolvedLoop{}
	}
	if report.StaleTasks == nil {
		report.StaleTasks = []StaleTask{}
	}
	if report.OpenDecisions == nil {
		report.OpenDecisions = []OpenDecision{}
	}

	schemaPath := opts.SchemaPath
	if schemaPath == "" {
		schemaPath = DefaultSchemaPath(repoRoot)
	}
	if err := ValidateReport(report, schemaPath); err != nil {
		return Report{}, err
	}
	return report, nil
}

func unresolvedLoopsForReport(opts BuildReportOptions, now time.Time) ([]UnresolvedLoop, error) {
	home := opts.BridgeHome
	if home == "" {
		var err error
		home, err = bridgeHome()
		if err != nil {
			return []UnresolvedLoop{}, nil
		}
	}
	cfgPath := filepath.Join(home, "channels.json")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return []UnresolvedLoop{}, nil
	}
	var bridgeCfg struct {
		MailboxDB string `json:"mailbox_db"`
	}
	if err := json.Unmarshal(data, &bridgeCfg); err != nil || bridgeCfg.MailboxDB == "" {
		return []UnresolvedLoop{}, nil
	}
	loops, err := UnresolvedLoopsFromMailbox(bridgeCfg.MailboxDB, now)
	if err != nil {
		return nil, err
	}
	if loops == nil {
		return []UnresolvedLoop{}, nil
	}
	return loops, nil
}

// WithNow replaces the clock for tests.
func WithNow(fn func() time.Time) func() {
	prev := nowFn
	nowFn = fn
	return func() { nowFn = prev }
}
