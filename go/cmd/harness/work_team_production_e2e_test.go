package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
	"github.com/Chachamaru127/claude-code-harness/go/internal/orchestrationledger"
	"github.com/Chachamaru127/claude-code-harness/go/internal/runtimefloor"
)

func skipIfWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("companion e2e tests require bash (non-Windows)")
	}
}

// passingRuntimeFloor installs a runtime floor stub that always allows dispatch.
func passingRuntimeFloor(t *testing.T) {
	t.Helper()
	orig := runtimeFloorCheck
	runtimeFloorCheck = func(_ string, _ runtimefloor.Context) runtimefloor.Decision {
		return runtimefloor.Decision{}
	}
	t.Cleanup(func() { runtimeFloorCheck = orig })
}

// fakeCompanionScriptBody returns a bash companion that handles `task --write`
// and prints a backend-specific summary line (first stdout line drives Normalize).
// mode: "ok" exit 0, "fail" exit 3, "rawtext" exit 1 with stderr-only message.
func fakeCompanionScriptBody(backend, mode string) string {
	summaryOK := fmt.Sprintf("e2e-summary-%s", backend)
	failMsg := fmt.Sprintf("%s companion failed deliberately", backend)
	rawMsg := fmt.Sprintf("%s companion raw error text", backend)
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
BACKEND=%q
MODE=%q
SUMMARY_OK=%q
FAIL_MSG=%q
RAW_MSG=%q
if [[ "${1:-}" != "task" ]]; then
  echo "unknown subcommand" >&2
  exit 2
fi
shift
while [[ $# -gt 0 ]]; do
  case "$1" in
    --write) shift; shift ;;
    *) shift ;;
  esac
done
case "$MODE" in
  ok)
    echo "$SUMMARY_OK"
    printf '{"schema":"companion-result.v1","backend":"%%s","success":true}\n' "$BACKEND"
    exit 0
    ;;
  fail)
    echo "$FAIL_MSG" >&2
    exit 3
    ;;
  rawtext)
    echo "$RAW_MSG" >&2
    exit 1
    ;;
  *)
    echo "bad mode" >&2
    exit 2
    ;;
esac
`, backend, mode, summaryOK, failMsg, rawMsg)
}

func writeFakeCompanionScript(t *testing.T, root, backend, mode string) {
	t.Helper()
	scriptDir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(scriptDir, backend+"-companion.sh")
	if err := os.WriteFile(script, []byte(fakeCompanionScriptBody(backend, mode)), 0o755); err != nil {
		t.Fatal(err)
	}
}

func setupProductionE2ERoot(t *testing.T, backends ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, b := range backends {
		writeFakeCompanionScript(t, root, b, "ok")
	}
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)
	return root
}

func invokeProductionWorker(t *testing.T, backend, taskID string) companionresult.Result {
	t.Helper()
	worker := productionCompanionWorker(backend)
	tr := worker(context.Background(), &breezing.Task{ID: taskID})
	return resultFromTaskResult(backend, taskID, tr)
}

func TestProductionCompanionWorker_E2E_DirectCall(t *testing.T) {
	skipIfWindows(t)
	passingRuntimeFloor(t)
	setupProductionE2ERoot(t, "codex")

	r := invokeProductionWorker(t, "codex", "92.3.2-direct")
	if r.Schema != companionresult.SchemaID {
		t.Fatalf("Schema = %q, want %q", r.Schema, companionresult.SchemaID)
	}
	if r.Backend != "codex" {
		t.Fatalf("Backend = %q, want codex", r.Backend)
	}
	if !r.Success {
		t.Fatalf("expected success, got %+v", r)
	}
	if r.Summary != "e2e-summary-codex" {
		t.Fatalf("Summary = %q, want e2e-summary-codex", r.Summary)
	}
}

func TestProductionCompanionWorker_E2E_ThreeBackendsParallel(t *testing.T) {
	skipIfWindows(t)
	passingRuntimeFloor(t)

	root := t.TempDir()
	// Shared repo root holds all three backend scripts; wt-* dirs mark worktree scope.
	for _, wt := range []string{"wt-a", "wt-b", "wt-c"} {
		if err := os.MkdirAll(filepath.Join(root, wt), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, b := range []string{"claude", "codex", "cursor"} {
		writeFakeCompanionScript(t, root, b, "ok")
	}
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)

	type job struct {
		backend string
		taskID  string
	}
	jobs := []job{
		{"claude", "wt-a"},
		{"codex", "wt-b"},
		{"cursor", "wt-c"},
	}

	var wg sync.WaitGroup
	results := make([]companionresult.Result, len(jobs))
	errs := make([]error, len(jobs))

	for i, j := range jobs {
		wg.Add(1)
		go func(idx int, jb job) {
			defer wg.Done()
			worker := productionCompanionWorker(jb.backend)
			tr := worker(context.Background(), &breezing.Task{ID: jb.taskID})
			r := resultFromTaskResult(jb.backend, jb.taskID, tr)
			results[idx] = r
			if r.Schema != companionresult.SchemaID {
				errs[idx] = fmt.Errorf("%s: schema %q", jb.backend, r.Schema)
			}
			if r.Backend != jb.backend {
				errs[idx] = fmt.Errorf("%s: backend %q", jb.backend, r.Backend)
			}
			wantSummary := "e2e-summary-" + jb.backend
			if r.Summary != wantSummary {
				errs[idx] = fmt.Errorf("%s: summary %q want %q", jb.backend, r.Summary, wantSummary)
			}
			if !r.Success {
				errs[idx] = fmt.Errorf("%s: not successful: %+v", jb.backend, r)
			}
		}(i, j)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("job %s: %v (result=%+v)", jobs[i].backend, err, results[i])
		}
	}
}

func TestProductionCompanionWorker_E2E_FailurePaths(t *testing.T) {
	skipIfWindows(t)

	backends := []string{"claude", "codex", "cursor"}
	modes := []struct {
		name       string
		setup      func(t *testing.T, root, backend string)
		wantOK     bool
		wantExit   int
		summaryHas []string
	}{
		{
			name: "missing scripts directory",
			setup: func(t *testing.T, root, backend string) {
				t.Setenv("HARNESS_PROJECT_ROOT", root)
				t.Setenv("CLAUDE_PLUGIN_ROOT", root)
			},
			wantOK:     false,
			wantExit:   127,
			summaryHas: []string{"companion script not found"},
		},
		{
			name: "script exits non-zero",
			setup: func(t *testing.T, root, backend string) {
				writeFakeCompanionScript(t, root, backend, "fail")
				t.Setenv("HARNESS_PROJECT_ROOT", root)
				t.Setenv("CLAUDE_PLUGIN_ROOT", root)
			},
			wantOK:     false,
			wantExit:   3,
			summaryHas: []string{"companion failed deliberately"},
		},
		{
			name: "script raw text stderr only",
			setup: func(t *testing.T, root, backend string) {
				writeFakeCompanionScript(t, root, backend, "rawtext")
				t.Setenv("HARNESS_PROJECT_ROOT", root)
				t.Setenv("CLAUDE_PLUGIN_ROOT", root)
			},
			wantOK:     false,
			wantExit:   1,
			summaryHas: []string{"raw error text"},
		},
	}

	for _, backend := range backends {
		for _, mode := range modes {
			t.Run(backend+"/"+mode.name, func(t *testing.T) {
				passingRuntimeFloor(t)
				root := t.TempDir()
				mode.setup(t, root, backend)

				r := invokeProductionWorker(t, backend, "fail-"+backend)
				if r.Success != mode.wantOK {
					t.Fatalf("Success = %v, want %v (result=%+v)", r.Success, mode.wantOK, r)
				}
				if r.ExitCode != mode.wantExit {
					t.Fatalf("ExitCode = %d, want %d", r.ExitCode, mode.wantExit)
				}
				if r.Backend != backend {
					t.Fatalf("Backend = %q, want %q", r.Backend, backend)
				}
				for _, frag := range mode.summaryHas {
					if !strings.Contains(r.Summary, frag) {
						t.Fatalf("Summary %q should contain %q", r.Summary, frag)
					}
				}
				if !strings.Contains(r.Summary, backend) && mode.name != "missing scripts directory" {
					t.Fatalf("Summary %q should mention backend %q", r.Summary, backend)
				}
			})
		}
	}
}

func TestProductionCompanionWorker_LedgerBackendPerEntry(t *testing.T) {
	skipIfWindows(t)
	passingRuntimeFloor(t)

	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.jsonl")
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", ledger)
	for _, b := range []string{"claude", "codex", "cursor"} {
		writeFakeCompanionScript(t, root, b, "ok")
	}

	backends := []string{"claude", "codex", "cursor"}
	for i, b := range backends {
		_ = invokeProductionWorker(t, b, fmt.Sprintf("ledger-%d", i))
	}

	entries := readLedgerEntries(t, ledger)
	var resultEntries []orchestrationledger.Entry
	for _, e := range entries {
		if e.Subcommand == "companion-result" {
			resultEntries = append(resultEntries, e)
		}
	}

	if len(resultEntries) != 3 {
		t.Fatalf("got %d companion-result ledger entries, want 3 (all=%v)", len(resultEntries), entries)
	}
	seen := map[string]struct{}{}
	for _, e := range resultEntries {
		seen[e.Backend] = struct{}{}
		if e.Subcommand != "companion-result" {
			t.Fatalf("subcommand = %q, want companion-result", e.Subcommand)
		}
		if e.ExitCode == nil {
			t.Fatalf("backend %s: exit_code should be set", e.Backend)
		}
		if *e.ExitCode != 0 {
			t.Fatalf("backend %s: exit_code = %d, want 0", e.Backend, *e.ExitCode)
		}
	}
	for _, b := range backends {
		if _, ok := seen[b]; !ok {
			t.Fatalf("missing companion-result ledger entry for backend %q", b)
		}
	}
}
