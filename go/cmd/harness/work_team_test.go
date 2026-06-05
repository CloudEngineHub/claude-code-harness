package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
)

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
