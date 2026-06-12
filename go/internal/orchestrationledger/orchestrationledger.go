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

// Entry is one orchestration-ledger.v1 line. Nullable fields use pointers so
// JSON encodes them as null when unset, matching the shell helper.
type Entry struct {
	TS          string `json:"ts"`
	Backend     string `json:"backend"`
	Subcommand  string `json:"subcommand"`
	Write       bool   `json:"write"`
	ExitCode    *int   `json:"exit_code"`
	DurationMs  *int64 `json:"duration_ms"`
	SessionID   string `json:"session_id"`
	Counts      bool   `json:"counts"`
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
