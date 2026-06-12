package orchestrationledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitTeamDispatchLedger_WritesJsonl(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "ledger.jsonl")
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", ledger)

	exit := 0
	EmitTeamDispatch(TeamDispatchOpts{
		Backend:    "codex",
		Write:      true,
		ExitCode:   &exit,
		DurationMs: 12,
		Reason:     "HARNESS_AUTO_APPROVE not set",
		Enabled:    false,
		RepoRoot:   dir,
	})

	data, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}

	var entry Entry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.Subcommand != subcommandTeamDispatch {
		t.Fatalf("subcommand = %q, want %q", entry.Subcommand, subcommandTeamDispatch)
	}
	if entry.Backend != "codex" {
		t.Fatalf("backend = %q, want codex", entry.Backend)
	}
	if !entry.Write {
		t.Fatal("write should be true")
	}
	if entry.ExitCode == nil || *entry.ExitCode != 0 {
		t.Fatalf("exit_code = %v, want 0", entry.ExitCode)
	}
	if entry.DurationMs == nil || *entry.DurationMs != 12 {
		t.Fatalf("duration_ms = %v, want 12", entry.DurationMs)
	}
	if entry.SessionID != "HARNESS_AUTO_APPROVE not set" {
		t.Fatalf("session_id(reason) = %q", entry.SessionID)
	}
	if entry.Counts {
		t.Fatal("counts should be false when auto-approve disabled")
	}
}

func TestEmitTeamDispatchLedger_FailOpen(t *testing.T) {
	// Point at a path whose parent is a file so mkdir/append fails.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	badLedger := filepath.Join(blocker, "ledger.jsonl")
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", badLedger)

	// Must not panic; caller exit semantics stay unchanged (no error return).
	EmitTeamDispatch(TeamDispatchOpts{
		Backend:    "cursor",
		Write:      true,
		ExitCode:   IntPtr(0),
		DurationMs: 1,
		Reason:     "disabled",
		Enabled:    false,
	})
}

func TestEmitIntegration_WritesJsonl(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "ledger.jsonl")
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", ledger)

	exit := 0
	EmitIntegration(IntegrationOpts{
		Backend:      "lead",
		RepoRoot:     dir,
		Write:        true,
		ExitCode:     &exit,
		DurationMs:   42,
		Sequence:     2,
		TaskBranch:   "task/92.3.1",
		TrunkBranch:  "trunk",
		CommitSHA:    "abc123",
		RereResolved: true,
		FloorPass:    true,
	})

	data, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}

	var entry map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var subcommand string
	if err := json.Unmarshal(entry["subcommand"], &subcommand); err != nil {
		t.Fatalf("subcommand: %v", err)
	}
	if subcommand != subcommandIntegration {
		t.Fatalf("subcommand = %q, want %q", subcommand, subcommandIntegration)
	}
}

func TestEmitTeamDispatchLedger_SchemaCompatibility(t *testing.T) {
	// Required fields and nullability must match scripts/lib/orchestration-ledger.sh.
	dir := t.TempDir()
	ledger := filepath.Join(dir, "ledger.jsonl")
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", ledger)

	EmitTeamDispatch(TeamDispatchOpts{
		Backend:    "codex",
		Write:      true,
		ExitCode:   nil,
		DurationMs: 0,
		Reason:     "enabled",
		Enabled:    true,
	})

	raw, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	wantKeys := []string{
		"ts", "backend", "subcommand", "write",
		"exit_code", "duration_ms", "session_id", "counts",
	}
	for _, k := range wantKeys {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing required field %q", k)
		}
	}
	if len(m) != len(wantKeys) {
		t.Fatalf("field count = %d, want exactly %d: %v", len(m), len(wantKeys), m)
	}
}
