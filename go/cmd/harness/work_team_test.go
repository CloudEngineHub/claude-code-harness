package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/autoapprove"
	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
	"github.com/Chachamaru127/claude-code-harness/go/internal/floor"
	"github.com/Chachamaru127/claude-code-harness/go/internal/orchestrationledger"
	"github.com/Chachamaru127/claude-code-harness/go/internal/runtimefloor"
)

// passingFloorGate installs a floorGate stub that always passes, so tests that
// exercise the orchestration fan-out are not coupled to the FLOOR contract
// scripts (which would otherwise shell out via the default gate). It restores
// the production gate on cleanup.
func passingFloorGate(t *testing.T) {
	t.Helper()
	orig := floorGate
	floorGate = func(_ string, _ []string, _ floor.ScriptRunner) floor.Report {
		return floor.Report{Passed: true}
	}
	t.Cleanup(func() { floorGate = orig })
}

// recordingFactory returns a teamWorkerFactory replacement whose WorkerFunc:
//   - records every task ID it was invoked with (mutex-guarded) so we can prove
//     each task ran exactly once, and
//   - returns a DISTINCT companion-result.v1 per task derived solely from that
//     task's ID — success for even-indexed IDs, failure for odd-indexed — with
//     NO shelling out.
//
// indexOf maps a task ID to its position so success/failure is deterministic
// and independent of execution order (the Orchestrator runs tasks in parallel).
func recordingFactory(indexOf map[string]int, calls *[]string, mu *sync.Mutex) func(string) breezing.WorkerFunc {
	return func(backend string) breezing.WorkerFunc {
		return func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			mu.Lock()
			*calls = append(*calls, task.ID)
			mu.Unlock()

			r := companionresult.New(backend, task.ID)
			r.Summary = fmt.Sprintf("worked %s", task.ID) // carries this task's own ID
			even := indexOf[task.ID]%2 == 0
			if even {
				r.Success = true
				r.ExitCode = 0
			} else {
				r.Success = false
				r.ExitCode = 7
			}
			return carryResult(r)
		}
	}
}

func TestWorkTeamFansOutNIndependentSubRunsWithSeparation(t *testing.T) {
	passingFloorGate(t) // isolate fan-out separation from the FLOOR backstop
	tasks := []string{"a1", "b2", "c3", "d4", "e5"}
	indexOf := map[string]int{}
	for i, id := range tasks {
		indexOf[id] = i
	}

	var (
		mu    sync.Mutex
		calls []string
	)

	// Inject the fake; restore the production factory afterward.
	orig := teamWorkerFactory
	teamWorkerFactory = recordingFactory(indexOf, &calls, &mu)
	defer func() { teamWorkerFactory = orig }()

	results, err := runTeam(tasks, "codex", 3)
	if err != nil {
		t.Fatalf("runTeam returned error: %v", err)
	}

	// (DoD a) All N tasks ran, each exactly once.
	if len(calls) != len(tasks) {
		t.Fatalf("worker invoked %d times, want %d (calls=%v)", len(calls), len(tasks), calls)
	}
	sortedCalls := append([]string(nil), calls...)
	sort.Strings(sortedCalls)
	for i, id := range tasks {
		if sortedCalls[i] != id {
			t.Errorf("task set mismatch: got %v, want each of %v exactly once", sortedCalls, tasks)
			break
		}
	}

	// Results count matches and preserves input order.
	if len(results) != len(tasks) {
		t.Fatalf("got %d results, want %d", len(results), len(tasks))
	}

	for i, id := range tasks {
		r := results[i]

		// (DoD d) Separation: task X's result carries X's ID, not a sibling's.
		if r.TaskID != id {
			t.Errorf("result[%d].TaskID = %q, want %q (results crossed over)", i, r.TaskID, id)
		}
		if r.Summary != fmt.Sprintf("worked %s", id) {
			t.Errorf("result[%d].Summary = %q, want %q (payload crossed over)", i, r.Summary, fmt.Sprintf("worked %s", id))
		}

		// Per-task success/failure matches the fake: even index → success.
		wantSuccess := i%2 == 0
		if r.Success != wantSuccess {
			t.Errorf("result for %q: Success = %v, want %v", id, r.Success, wantSuccess)
		}
		if wantSuccess && r.ExitCode != 0 {
			t.Errorf("result for %q: ExitCode = %d, want 0", id, r.ExitCode)
		}
		if !wantSuccess && r.ExitCode != 7 {
			t.Errorf("result for %q: ExitCode = %d, want 7", id, r.ExitCode)
		}

		// Every result is a well-formed companion-result.v1.
		if r.Schema != companionresult.SchemaID {
			t.Errorf("result for %q: Schema = %q, want %q", id, r.Schema, companionresult.SchemaID)
		}
		if r.Backend != "codex" {
			t.Errorf("result for %q: Backend = %q, want codex", id, r.Backend)
		}
	}
}

func TestRunTeamSingleTask(t *testing.T) {
	passingFloorGate(t)
	var (
		mu    sync.Mutex
		calls []string
	)
	indexOf := map[string]int{"solo": 0}

	orig := teamWorkerFactory
	teamWorkerFactory = recordingFactory(indexOf, &calls, &mu)
	defer func() { teamWorkerFactory = orig }()

	results, err := runTeam([]string{"solo"}, "cursor", 0)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].TaskID != "solo" || !results[0].Success {
		t.Errorf("solo result = %+v, want success with TaskID=solo", results[0])
	}
	if results[0].Backend != "cursor" {
		t.Errorf("Backend = %q, want cursor", results[0].Backend)
	}
}

func TestParseTeamArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantBackend string
		wantTasks   []string
		wantErr     bool
	}{
		{"team marker + default backend", []string{"--team", "t1", "t2"}, "codex", []string{"t1", "t2"}, false},
		{"explicit cursor backend", []string{"--team", "--backend", "cursor", "t1"}, "cursor", []string{"t1"}, false},
		{"backend equals form", []string{"--team", "--backend=cursor", "x"}, "cursor", []string{"x"}, false},
		{"unsupported backend", []string{"--team", "--backend", "bogus", "t1"}, "", nil, true},
		{"unknown flag rejected", []string{"--team", "--nope", "t1"}, "", nil, true},
		{"backend without value", []string{"--team", "--backend"}, "", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backend, tasks, err := parseTeamArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got backend=%q tasks=%v", backend, tasks)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if backend != tc.wantBackend {
				t.Errorf("backend = %q, want %q", backend, tc.wantBackend)
			}
			if len(tasks) != len(tc.wantTasks) {
				t.Fatalf("tasks = %v, want %v", tasks, tc.wantTasks)
			}
			for i := range tc.wantTasks {
				if tasks[i] != tc.wantTasks[i] {
					t.Errorf("tasks[%d] = %q, want %q", i, tasks[i], tc.wantTasks[i])
				}
			}
		})
	}
}

// TestProductionCompanionWorkerMissingScript proves the production worker
// degrades to a FAILED companion-result.v1 (exit 127) instead of crashing when
// the companion script cannot be resolved.
func TestProductionCompanionWorkerMissingScript(t *testing.T) {
	// Point resolution at a temp dir with no scripts/ so resolveCompanionScript
	// returns "". CLAUDE_PLUGIN_ROOT is one of the resolution candidates.
	tmp := t.TempDir()
	t.Setenv("HARNESS_PROJECT_ROOT", tmp)
	t.Setenv("CLAUDE_PLUGIN_ROOT", tmp)

	worker := productionCompanionWorker("codex")
	tr := worker(context.Background(), &breezing.Task{ID: "z9"})

	r := resultFromTaskResult("codex", "z9", tr)
	if r.Success {
		t.Error("missing script should yield Success=false")
	}
	if r.ExitCode != 127 {
		t.Errorf("ExitCode = %d, want 127", r.ExitCode)
	}
	if r.Schema != companionresult.SchemaID {
		t.Errorf("Schema = %q, want %q", r.Schema, companionresult.SchemaID)
	}
	if r.TaskID != "z9" {
		t.Errorf("TaskID = %q, want z9", r.TaskID)
	}
}

// succeedingFactory injects a worker that reports a SUCCESSFUL companion-result
// for every task, carrying the given changed files. It is used to prove the
// FLOOR take-in downgrades a companion-level success when the gate rejects it.
func succeedingFactory(changed []string) func(string) breezing.WorkerFunc {
	return func(backend string) breezing.WorkerFunc {
		return func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			r := companionresult.New(backend, task.ID)
			r.Success = true
			r.ExitCode = 0
			r.Summary = "companion ok"
			r.FilesChanged = append([]string(nil), changed...)
			return carryResult(r)
		}
	}
}

// TestRunTeamDowngradesCompanionSuccessWhenFloorGateFails is the STEP 4
// invariant: a sub-run that succeeds at the companion level is reported as
// FAILED when the injected FLOOR gate rejects its changes. This closes the
// take-in bypass — untrusted changes that don't clear the FLOOR must not be
// reported as success.
func TestRunTeamDowngradesCompanionSuccessWhenFloorGateFails(t *testing.T) {
	orig := teamWorkerFactory
	teamWorkerFactory = succeedingFactory([]string{"go/internal/x.go"})
	defer func() { teamWorkerFactory = orig }()

	var gotFiles []string
	origGate := floorGate
	floorGate = func(_ string, files []string, _ floor.ScriptRunner) floor.Report {
		gotFiles = append([]string(nil), files...)
		return floor.Report{
			Passed: false,
			Steps: []floor.StepResult{
				{Name: floor.StepValidatePlug, Passed: false, Detail: "validate-plugin.sh exit 1"},
			},
		}
	}
	defer func() { floorGate = origGate }()

	results, err := runTeam([]string{"t1"}, "codex", 1)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]

	if r.Success {
		t.Fatalf("companion success must be downgraded to failed when the FLOOR gate fails; result=%+v", r)
	}
	if r.ExitCode == 0 {
		t.Errorf("downgraded result should have a non-zero ExitCode, got 0")
	}
	if !strings.Contains(r.Summary, "FLOOR gate failed") {
		t.Errorf("Summary should explain the FLOOR downgrade, got %q", r.Summary)
	}
	if !strings.Contains(r.Summary, floor.StepValidatePlug) {
		t.Errorf("Summary should name the failing FLOOR step, got %q", r.Summary)
	}
	// The gate was invoked over exactly the files the sub-run reported.
	if len(gotFiles) != 1 || gotFiles[0] != "go/internal/x.go" {
		t.Errorf("gate received files %v, want [go/internal/x.go]", gotFiles)
	}
}

// TestRunTeamKeepsCompanionSuccessWhenFloorGatePasses is the positive control:
// a companion success that clears the FLOOR stays a success.
func TestRunTeamKeepsCompanionSuccessWhenFloorGatePasses(t *testing.T) {
	orig := teamWorkerFactory
	teamWorkerFactory = succeedingFactory([]string{"docs/readme.md"})
	defer func() { teamWorkerFactory = orig }()

	passingFloorGate(t)

	results, err := runTeam([]string{"t1"}, "codex", 1)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if !results[0].Success {
		t.Errorf("companion success that clears the FLOOR must stay success, got %+v", results[0])
	}
	if results[0].Summary != "companion ok" {
		t.Errorf("passing gate must not rewrite Summary, got %q", results[0].Summary)
	}
}

// TestRunTeamFloorGateNotRunForFailedSubRun proves the FLOOR only ever turns a
// success into a failure: a sub-run that already failed at the companion level
// is left untouched and the gate is never consulted for it (it cannot rescue a
// failed run). This also keeps the `--team` "companion script absent" path sane
// (that path yields a failed result, so the gate is skipped).
func TestRunTeamFloorGateNotRunForFailedSubRun(t *testing.T) {
	orig := teamWorkerFactory
	teamWorkerFactory = func(backend string) breezing.WorkerFunc {
		return func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			r := companionresult.New(backend, task.ID)
			r.Success = false
			r.ExitCode = 127
			r.Summary = "companion script not found"
			return carryResult(r)
		}
	}
	defer func() { teamWorkerFactory = orig }()

	gateCalled := false
	origGate := floorGate
	floorGate = func(_ string, _ []string, _ floor.ScriptRunner) floor.Report {
		gateCalled = true
		return floor.Report{Passed: true}
	}
	defer func() { floorGate = origGate }()

	results, err := runTeam([]string{"t1"}, "codex", 1)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if gateCalled {
		t.Error("FLOOR gate must not run for an already-failed sub-run")
	}
	if results[0].Success {
		t.Error("failed sub-run must stay failed")
	}
	if results[0].ExitCode != 127 {
		t.Errorf("failed sub-run ExitCode = %d, want 127 (unchanged)", results[0].ExitCode)
	}
}

func installFakeCompanionScript(t *testing.T, root string) {
	t.Helper()
	scriptDir := filepath.Join(root, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "companion-ran")
	script := filepath.Join(scriptDir, "codex-companion.sh")
	body := "#!/bin/bash\ntouch " + marker + "\nexit 99\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)
}

func readLedgerEntries(t *testing.T, path string) []orchestrationledger.Entry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ledger %s: %v", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	out := make([]orchestrationledger.Entry, 0, len(lines))
	for _, line := range lines {
		var e orchestrationledger.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("unmarshal ledger line: %v", err)
		}
		out = append(out, e)
	}
	return out
}

func TestRunTeam_DispatchFloorStopsCompanion(t *testing.T) {
	passingFloorGate(t)
	root := t.TempDir()
	installFakeCompanionScript(t, root)
	marker := filepath.Join(root, "companion-ran")

	origFloor := runtimeFloorCheck
	runtimeFloorCheck = func(_ string, _ runtimefloor.Context) runtimefloor.Decision {
		return runtimefloor.Decision{
			Stopped:  true,
			Category: runtimefloor.CategoryEgress,
			Reason:   "runtime action hard floor: test stop",
		}
	}
	defer func() { runtimeFloorCheck = origFloor }()

	orig := teamWorkerFactory
	teamWorkerFactory = productionCompanionWorker
	defer func() { teamWorkerFactory = orig }()

	results, err := runTeam([]string{"t-floor"}, "codex", 1)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.Success {
		t.Fatal("floor stop must yield Success=false")
	}
	if r.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", r.ExitCode)
	}
	if !strings.HasPrefix(r.Summary, "RUNTIME_FLOOR:") {
		t.Fatalf("Summary = %q, want RUNTIME_FLOOR prefix", r.Summary)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("companion script must not run when runtime floor stops dispatch")
	}
}

func TestRunTeam_AutoApproveLedger_Disabled(t *testing.T) {
	passingFloorGate(t)
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.jsonl")
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", ledger)
	t.Setenv("HARNESS_AUTO_APPROVE", "")

	orig := teamWorkerFactory
	teamWorkerFactory = recordingFactory(map[string]int{"t1": 0}, &[]string{}, &sync.Mutex{})
	defer func() { teamWorkerFactory = orig }()

	if _, err := runTeam([]string{"t1"}, "codex", 1); err != nil {
		t.Fatalf("runTeam: %v", err)
	}

	entries := readLedgerEntries(t, ledger)
	if len(entries) != 1 {
		t.Fatalf("got %d ledger entries, want 1", len(entries))
	}
	e := entries[0]
	if e.Subcommand != "team-dispatch" {
		t.Fatalf("subcommand = %q, want team-dispatch", e.Subcommand)
	}
	if e.SessionID != "auto-approve:disabled (env=off)" {
		t.Fatalf("reason(session_id) = %q, want auto-approve:disabled (env=off)", e.SessionID)
	}
	if e.Counts {
		t.Fatal("counts should be false when auto-approve disabled")
	}
}

func TestRunTeam_AutoApproveLedger_Enabled(t *testing.T) {
	passingFloorGate(t)
	root := t.TempDir()
	ledger := filepath.Join(root, "ledger.jsonl")
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", ledger)
	t.Setenv("HARNESS_AUTO_APPROVE", "on")
	restorePrereq := autoapprove.SetPrereqChecker(func(name string) bool { return true })
	defer restorePrereq()

	orig := teamWorkerFactory
	teamWorkerFactory = recordingFactory(map[string]int{"t1": 0}, &[]string{}, &sync.Mutex{})
	defer func() { teamWorkerFactory = orig }()

	if _, err := runTeam([]string{"t1"}, "cursor", 1); err != nil {
		t.Fatalf("runTeam: %v", err)
	}

	entries := readLedgerEntries(t, ledger)
	if len(entries) != 1 {
		t.Fatalf("got %d ledger entries, want 1", len(entries))
	}
	e := entries[0]
	if e.SessionID != "auto-approve:enabled" {
		t.Fatalf("reason(session_id) = %q, want auto-approve:enabled", e.SessionID)
	}
	if !e.Counts {
		t.Fatal("counts should be true when auto-approve enabled")
	}
	if e.Backend != "cursor" {
		t.Fatalf("backend = %q, want cursor", e.Backend)
	}
}
