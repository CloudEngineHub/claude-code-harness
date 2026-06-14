package failurecodifier

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Chachamaru127/claude-code-harness/go/internal/judgmentledger"
)

const (
	SSOTTargetPatterns  = "patterns.md"
	SSOTTargetDecisions = "decisions.md"
)

type evidenceBucket struct {
	summary string
	target  string
	refs    []string
}

// ExtractOpts configures read-only failure pattern extraction.
type ExtractOpts struct {
	RepoRoot              string
	JudgmentLedgerPath    string
	OrchestrationLedgerPath string
	SchemaPath            string
}

// ExtractFromLedger reads Judgment Ledger + breezing orchestration logs (read-only)
// and returns failure-rule.v1 candidates grouped by recurring signatures.
func ExtractFromLedger(opts ExtractOpts) ([]Rule, error) {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		return nil, fmt.Errorf("repo root required")
	}

	judgmentPath := opts.JudgmentLedgerPath
	if judgmentPath == "" {
		judgmentPath = judgmentledger.DefaultLedgerPath(repoRoot)
	}
	orchPath := opts.OrchestrationLedgerPath
	if orchPath == "" {
		orchPath = defaultOrchestrationLedgerPath(repoRoot)
	}

	type bucket = evidenceBucket

	buckets := make(map[string]*bucket)

	jRecords, err := judgmentledger.LoadAll(judgmentPath)
	if err != nil {
		return nil, fmt.Errorf("load judgment ledger: %w", err)
	}
	for _, rec := range jRecords {
		sig, summary, target := judgmentFailureSignature(rec)
		if sig == "" {
			continue
		}
		ref := fmt.Sprintf("judgment-ledger:%s", rec.ID)
		addEvidence(buckets, sig, summary, target, ref)
	}

	orchFailures, err := loadOrchestrationFailures(orchPath)
	if err != nil {
		return nil, fmt.Errorf("load orchestration ledger: %w", err)
	}
	for _, fail := range orchFailures {
		sig := fail.signature
		ref := fmt.Sprintf("orchestration-ledger:%s:%s", fail.ts, fail.backend)
		addEvidence(buckets, sig, fail.summary, SSOTTargetPatterns, ref)
	}

	if len(buckets) == 0 {
		return nil, nil
	}

	schemaPath := opts.SchemaPath
	if schemaPath == "" {
		schemaPath = DefaultSchemaPath(repoRoot)
	}

	var rules []Rule
	for sig, b := range buckets {
		count := len(b.refs)
		rule := Rule{
			RuleID:             ruleIDFromSignature(sig),
			Confidence:         ConfidenceFromCount(count),
			EvidenceRefs:       append([]string(nil), b.refs...),
			ProposedSSOTTarget: b.target,
			Summary:            b.summary,
			OccurrenceCount:    count,
		}
		if err := ValidateRule(rule, schemaPath); err != nil {
			return nil, fmt.Errorf("validate rule %q: %w", sig, err)
		}
		rules = append(rules, rule)
	}

	sort.Slice(rules, func(i, j int) bool {
		if rules[i].OccurrenceCount != rules[j].OccurrenceCount {
			return rules[i].OccurrenceCount > rules[j].OccurrenceCount
		}
		return rules[i].RuleID < rules[j].RuleID
	})
	return rules, nil
}

func addEvidence(buckets map[string]*evidenceBucket, sig, summary, target, ref string) {
	if sig == "" || ref == "" {
		return
	}
	b, ok := buckets[sig]
	if !ok {
		b = &evidenceBucket{summary: summary, target: target}
		buckets[sig] = b
	}
	b.refs = append(b.refs, ref)
}

func judgmentFailureSignature(rec judgmentledger.Record) (sig, summary, target string) {
	answer := strings.ToLower(strings.TrimSpace(rec.Answer))
	question := strings.ToLower(strings.TrimSpace(rec.Question))
	if answer == "" {
		return "", "", ""
	}
	negative := strings.Contains(answer, "reject") ||
		strings.Contains(answer, "no") ||
		strings.Contains(answer, "stop") ||
		strings.Contains(answer, "wait")
	if !negative {
		for _, tag := range rec.Tags {
			t := strings.ToLower(tag)
			if t == "failure" || t == "regression" || t == "blocked" {
				negative = true
				break
			}
		}
	}
	if !negative {
		return "", "", ""
	}

	target = SSOTTargetDecisions
	if strings.Contains(question, "pattern") || strings.Contains(question, "how") {
		target = SSOTTargetPatterns
	}
	sig = "judgment:" + normalizeSignature(answer) + ":" + normalizeSignature(firstWords(question, 6))
	summary = strings.TrimSpace(rec.Question)
	if summary == "" {
		summary = "judgment failure: " + rec.Answer
	}
	return sig, summary, target
}

type orchFailure struct {
	signature string
	summary   string
	ts        string
	backend   string
}

func loadOrchestrationFailures(path string) ([]orchFailure, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []orchFailure
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		fail, ok := parseOrchestrationFailure(raw)
		if ok {
			out = append(out, fail)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseOrchestrationFailure(raw map[string]json.RawMessage) (orchFailure, bool) {
	var (
		ts         string
		backend    string
		subcommand string
		exitCode   *int
		sessionID  string
		counts     bool
	)
	if v, ok := raw["ts"]; ok {
		_ = json.Unmarshal(v, &ts)
	}
	if v, ok := raw["backend"]; ok {
		_ = json.Unmarshal(v, &backend)
	}
	if v, ok := raw["subcommand"]; ok {
		_ = json.Unmarshal(v, &subcommand)
	}
	if v, ok := raw["exit_code"]; ok && string(v) != "null" {
		var code int
		if json.Unmarshal(v, &code) == nil {
			exitCode = &code
		}
	}
	if v, ok := raw["session_id"]; ok {
		_ = json.Unmarshal(v, &sessionID)
	}
	if v, ok := raw["counts"]; ok {
		switch string(v) {
		case "true":
			counts = true
		case "false":
			counts = false
		default:
			_ = json.Unmarshal(v, &counts)
		}
	}

	failed := false
	reason := sessionID
	if exitCode != nil && *exitCode != 0 {
		failed = true
		reason = fmt.Sprintf("exit_code=%d", *exitCode)
	}
	if subcommand == "companion-result" && !counts {
		failed = true
		if reason == "" {
			reason = "companion failure"
		}
	}
	if !failed {
		return orchFailure{}, false
	}

	sig := fmt.Sprintf("orch:%s:%s:%s", backend, subcommand, normalizeSignature(reason))
	summary := fmt.Sprintf("%s/%s failure: %s", backend, subcommand, reason)
	return orchFailure{
		signature: sig,
		summary:   summary,
		ts:        ts,
		backend:   backend,
	}, true
}

func defaultOrchestrationLedgerPath(repoRoot string) string {
	if v := strings.TrimSpace(os.Getenv("HARNESS_ORCHESTRATION_LEDGER")); v != "" {
		return v
	}
	return filepath.Join(repoRoot, ".claude/state/orchestration-ledger.jsonl")
}

func normalizeSignature(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), "-")
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

func firstWords(s string, n int) string {
	words := strings.Fields(strings.ToLower(s))
	if len(words) > n {
		words = words[:n]
	}
	return strings.Join(words, " ")
}

func ruleIDFromSignature(sig string) string {
	norm := normalizeSignature(sig)
	norm = strings.ReplaceAll(norm, ":", "-")
	if len(norm) > 64 {
		norm = norm[:64]
	}
	return "failure-" + norm
}
