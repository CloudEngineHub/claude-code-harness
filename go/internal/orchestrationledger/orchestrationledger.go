// Package orchestrationledger appends orchestration-ledger.v1 JSONL records from Go.
// Schema matches scripts/lib/orchestration-ledger.sh (8 scalar fields, fail-open).
package orchestrationledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/gitport"
)

const subcommandTeamDispatch = "team-dispatch"
const subcommandCompanionResult = "companion-result"
const subcommandIntegration = "integration"
const subcommandDialogTurn = "dialog-turn"

// IntegrationOpts records Lead integration (task branch take-in) outcome.
type IntegrationOpts struct {
	Backend      string
	RepoRoot     string
	Write        bool
	ExitCode     *int
	DurationMs   int64
	Sequence     int
	TaskBranch   string
	TrunkBranch  string
	CommitSHA    string
	RereResolved bool
	FloorPass    bool
}

// EmitIntegration appends one integration ledger line. Fail-open: write errors
// are ignored and never propagate to callers.
func EmitIntegration(opts IntegrationOpts) {
	backend := strings.TrimSpace(opts.Backend)
	if backend == "" {
		backend = "lead"
	}
	dur := opts.DurationMs
	exit := opts.ExitCode
	entry := integrationEntry{
		TS:         nowUTC(),
		Backend:    backend,
		Subcommand: subcommandIntegration,
		Write:      opts.Write,
		ExitCode:   exit,
		DurationMs: &dur,
		SessionID:  opts.TaskBranch,
		Counts: integrationCounts{
			Sequence:     opts.Sequence,
			TaskBranch:   opts.TaskBranch,
			TrunkBranch:  opts.TrunkBranch,
			CommitSHA:    opts.CommitSHA,
			RereResolved: opts.RereResolved,
			FloorPass:    opts.FloorPass,
		},
	}
	_ = emitIntegration(entry, opts.RepoRoot)
}

type integrationCounts struct {
	Sequence     int    `json:"sequence"`
	TaskBranch   string `json:"task_branch"`
	TrunkBranch  string `json:"trunk_branch"`
	CommitSHA    string `json:"commit_sha"`
	RereResolved bool   `json:"rere_resolved"`
	FloorPass    bool   `json:"floor_pass"`
}

type integrationEntry struct {
	TS         string            `json:"ts"`
	Backend    string            `json:"backend"`
	Subcommand string            `json:"subcommand"`
	Write      bool              `json:"write"`
	ExitCode   *int              `json:"exit_code"`
	DurationMs *int64            `json:"duration_ms"`
	SessionID  string            `json:"session_id"`
	Counts     integrationCounts `json:"counts"`
}

func emitIntegration(entry integrationEntry, repoRoot string) error {
	path := ledgerPath(repoRoot)
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// Entry is one orchestration-ledger.v1 line. Nullable fields use pointers so
// JSON encodes them as null when unset, matching the shell helper.
type Entry struct {
	TS         string `json:"ts"`
	Backend    string `json:"backend"`
	Subcommand string `json:"subcommand"`
	Write      bool   `json:"write"`
	ExitCode   *int   `json:"exit_code"`
	DurationMs *int64 `json:"duration_ms"`
	SessionID  string `json:"session_id"`
	Counts     bool   `json:"counts"`
}

// TeamDispatchOpts records team-side auto-approve gating or dispatch floor outcome.
type TeamDispatchOpts struct {
	Backend    string
	Write      bool
	ExitCode   *int
	DurationMs int64
	Reason     string
	Enabled    bool
	RepoRoot   string
}

// CompanionResultOpts records one backend companion sub-run outcome.
type CompanionResultOpts struct {
	Backend    string
	TaskID     string
	Write      bool
	ExitCode   *int
	DurationMs int64
	Success    bool
	RepoRoot   string
}

// EmitCompanionResult appends one companion-result ledger line. Fail-open: write
// errors are ignored and never propagate to callers.
func EmitCompanionResult(opts CompanionResultOpts) {
	if strings.TrimSpace(opts.Backend) == "" {
		return
	}
	dur := opts.DurationMs
	entry := Entry{
		TS:         nowUTC(),
		Backend:    opts.Backend,
		Subcommand: subcommandCompanionResult,
		Write:      opts.Write,
		ExitCode:   opts.ExitCode,
		DurationMs: &dur,
		SessionID:  opts.TaskID,
		Counts:     opts.Success,
	}
	_ = emit(entry, opts.RepoRoot)
}

// EmitTeamDispatch appends one team-dispatch ledger line. Fail-open: write errors
// are ignored and never propagate to callers.
func EmitTeamDispatch(opts TeamDispatchOpts) {
	if strings.TrimSpace(opts.Backend) == "" {
		return
	}
	dur := opts.DurationMs
	entry := Entry{
		TS:         nowUTC(),
		Backend:    opts.Backend,
		Subcommand: subcommandTeamDispatch,
		Write:      opts.Write,
		ExitCode:   opts.ExitCode,
		DurationMs: &dur,
		SessionID:  opts.Reason,
		Counts:     opts.Enabled,
	}
	_ = emit(entry, opts.RepoRoot)
}

// DialogTurnOpts records one dialogloop turn (A↔B round-trip).
type DialogTurnOpts struct {
	Team      string
	FromAgent string
	ToAgent   string
	Round     int
	Reason    string
	RepoRoot  string
}

// EmitDialogTurn appends one dialog-turn ledger line (fail-open).
// SessionID is Team; counts.round holds the dialog round number.
func EmitDialogTurn(opts DialogTurnOpts) {
	team := strings.TrimSpace(opts.Team)
	if team == "" {
		return
	}
	entry := dialogTurnEntry{
		TS:         nowUTC(),
		Backend:    "dialog",
		Subcommand: subcommandDialogTurn,
		Write:      true,
		SessionID:  team,
		Counts: dialogTurnCounts{
			Round:     opts.Round,
			FromAgent: opts.FromAgent,
			ToAgent:   opts.ToAgent,
			Reason:    opts.Reason,
		},
	}
	_ = emitDialogTurn(entry, opts.RepoRoot)
}

type dialogTurnCounts struct {
	Round     int    `json:"round"`
	FromAgent string `json:"from_agent,omitempty"`
	ToAgent   string `json:"to_agent,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type dialogTurnEntry struct {
	TS         string           `json:"ts"`
	Backend    string           `json:"backend"`
	Subcommand string           `json:"subcommand"`
	Write      bool             `json:"write"`
	ExitCode   *int             `json:"exit_code"`
	DurationMs *int64           `json:"duration_ms"`
	SessionID  string           `json:"session_id"`
	Counts     dialogTurnCounts `json:"counts"`
}

func emitDialogTurn(entry dialogTurnEntry, repoRoot string) error {
	path := ledgerPath(repoRoot)
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func emit(entry Entry, repoRoot string) error {
	path := ledgerPath(repoRoot)
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

func ledgerPath(repoRoot string) string {
	if v := strings.TrimSpace(os.Getenv("HARNESS_ORCHESTRATION_LEDGER")); v != "" {
		return v
	}
	root := repoRoot
	if root == "" {
		root = resolveRepoRoot()
	}
	return filepath.Join(root, ".claude/state/orchestration-ledger.jsonl")
}

func resolveRepoRoot() string {
	if v := os.Getenv("HARNESS_PROJECT_ROOT"); v != "" {
		return v
	}
	if v := os.Getenv("PROJECT_ROOT"); v != "" {
		return v
	}
	if out, err := gitport.Output("", "rev-parse", "--show-toplevel"); err == nil {
		if root := strings.TrimSpace(out); root != "" {
			return root
		}
	}
	cwd, _ := os.Getwd()
	return cwd
}

func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// IntPtr returns a pointer to v (ledger nullable exit_code helper).
func IntPtr(v int) *int { return &v }
