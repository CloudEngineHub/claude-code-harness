package sublead_test

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

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
	"github.com/Chachamaru127/claude-code-harness/go/internal/sublead"
)

const resultCarrierPrefix = "companion-result.v1:"

type subtaskSummary struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Summary string `json:"summary"`
}

func resultFromTaskResult(backend, taskID string, tr breezing.TaskResult) companionresult.Result {
	if payload, ok := strings.CutPrefix(tr.CommitHash, resultCarrierPrefix); ok {
		if r, err := companionresult.Parse([]byte(payload)); err == nil {
			return r
		}
	}
	r := companionresult.New(backend, taskID)
	if tr.Err != nil {
		r.Success = false
		r.ExitCode = 1
		r.Summary = tr.Err.Error()
	}
	return r
}

func fakePlanner(subtasks []sublead.SubTask) sublead.Planner {
	return func(_ context.Context, laneID, _ string) (sublead.MiniPlan, error) {
		return sublead.MiniPlan{LaneID: laneID, SubTasks: subtasks}, nil
	}
}

func recordingSubWorker(backend string, calls *[]string, mu *sync.Mutex, failID string) breezing.WorkerFunc {
	return func(_ context.Context, task *breezing.Task) breezing.TaskResult {
		mu.Lock()
		*calls = append(*calls, task.ID)
		mu.Unlock()

		r := companionresult.New(backend, task.ID)
		r.Summary = fmt.Sprintf("done %s", task.ID)
		if task.ID == failID {
			r.Success = false
			r.ExitCode = 1
		} else {
			r.Success = true
			r.ExitCode = 0
		}
		return carryResult(r)
	}
}

func carryResult(r companionresult.Result) breezing.TaskResult {
	tr := breezing.TaskResult{
		TaskID:     r.TaskID,
		CommitHash: resultCarrierPrefix + mustMarshal(r),
	}
	if !r.Success {
		tr.Err = fmt.Errorf("subtask %s failed", r.TaskID)
	}
	return tr
}

func mustMarshal(r companionresult.Result) string {
	b, err := r.Marshal()
	if err != nil {
		return ""
	}
	return string(b)
}

func parseSubtaskSummaries(t *testing.T, summary string) []subtaskSummary {
	t.Helper()
	var out []subtaskSummary
	if err := json.Unmarshal([]byte(summary), &out); err != nil {
		t.Fatalf("parse summary JSON: %v (summary=%q)", err, summary)
	}
	return out
}

func TestSubLead_LaneDecomposedParallel(t *testing.T) {
	subtasks := []sublead.SubTask{
		{ID: "s1", Prompt: "p1"},
		{ID: "s2", Prompt: "p2"},
		{ID: "s3", Prompt: "p3"},
	}
	var (
		mu    sync.Mutex
		calls []string
	)
	worker := sublead.NewSubLeadWorker(
		fakePlanner(subtasks),
		recordingSubWorker("cursor", &calls, &mu, ""),
		"cursor",
		3,
	)

	tr := worker(context.Background(), &breezing.Task{ID: "lane-a", Description: "spec-a"})
	r := resultFromTaskResult("cursor", "lane-a", tr)

	if len(calls) != 3 {
		t.Fatalf("subWorker invoked %d times, want 3 (calls=%v)", len(calls), calls)
	}
	sort.Strings(calls)
	want := []string{"s1", "s2", "s3"}
	for i, id := range want {
		if calls[i] != id {
			t.Fatalf("subtask calls = %v, want %v", calls, want)
		}
	}

	summaries := parseSubtaskSummaries(t, r.Summary)
	if len(summaries) != 3 {
		t.Fatalf("lane summary has %d subtask entries, want 3", len(summaries))
	}
	for _, st := range subtasks {
		found := false
		for _, s := range summaries {
			if s.ID == st.ID {
				found = true
				if !s.Success {
					t.Errorf("subtask %s should succeed", st.ID)
				}
			}
		}
		if !found {
			t.Errorf("summary missing subtask %s", st.ID)
		}
	}
	if !r.Success {
		t.Errorf("lane should succeed when all subtasks succeed")
	}
}

func TestSubLead_ReportsCompanionResultV1(t *testing.T) {
	subtasks := []sublead.SubTask{{ID: "x1", Prompt: "do x1"}}
	worker := sublead.NewSubLeadWorker(
		fakePlanner(subtasks),
		recordingSubWorker("codex", &[]string{}, &sync.Mutex{}, ""),
		"codex",
		1,
	)

	tr := worker(context.Background(), &breezing.Task{ID: "lane-42", Description: "lane spec"})
	r := resultFromTaskResult("codex", "lane-42", tr)

	if r.Schema != companionresult.SchemaID {
		t.Errorf("Schema = %q, want %q", r.Schema, companionresult.SchemaID)
	}
	if r.Backend != "codex" {
		t.Errorf("Backend = %q, want codex", r.Backend)
	}
	if r.TaskID != "lane-42" {
		t.Errorf("TaskID = %q, want lane-42", r.TaskID)
	}
	if !r.Success {
		t.Errorf("expected lane success, got %+v", r)
	}

	summaries := parseSubtaskSummaries(t, r.Summary)
	if len(summaries) != 1 || summaries[0].ID != "x1" {
		t.Errorf("summary subtasks = %+v, want [{id:x1 ...}]", summaries)
	}
}

func TestSubLead_Hierarchy_LeadSubLeadWorker(t *testing.T) {
	planner := func(_ context.Context, laneID, _ string) (sublead.MiniPlan, error) {
		return sublead.MiniPlan{
			LaneID: laneID,
			SubTasks: []sublead.SubTask{
				{ID: laneID + "-a", Prompt: "a"},
				{ID: laneID + "-b", Prompt: "b"},
			},
		}, nil
	}

	var (
		mu         sync.Mutex
		subCalls   []string
		subWorker  = recordingSubWorker("cursor", &subCalls, &mu, "")
		subLead    = sublead.NewSubLeadWorker(planner, subWorker, "cursor", 2)
		leadWorker = subLead // Lead layer uses Sub-Lead as its WorkerFunc
	)

	leadOrch := breezing.NewOrchestrator(leadWorker, breezing.WithMaxParallel(2))
	leadOrch.AddTask(&breezing.Task{ID: "lane-1"})
	leadOrch.AddTask(&breezing.Task{ID: "lane-2"})

	if _, err := leadOrch.Run(context.Background()); err != nil {
		t.Fatalf("lead orchestrator: %v", err)
	}

	leadResults := leadOrch.Results()
	if len(leadResults) != 2 {
		t.Fatalf("lead layer results = %d, want 2 (one per lane)", len(leadResults))
	}
	for _, laneID := range []string{"lane-1", "lane-2"} {
		tr, ok := leadResults[laneID]
		if !ok {
			t.Fatalf("lead missing result for %s", laneID)
		}
		r := resultFromTaskResult("cursor", laneID, tr)
		if r.TaskID != laneID {
			t.Errorf("lead result TaskID = %q, want %q", r.TaskID, laneID)
		}
		summaries := parseSubtaskSummaries(t, r.Summary)
		if len(summaries) != 2 {
			t.Errorf("lane %s summary has %d subtasks, want 2", laneID, len(summaries))
		}
	}

	if len(subCalls) != 4 {
		t.Fatalf("subWorker invoked %d times, want 4 (2 lanes × 2 subtasks); calls=%v", len(subCalls), subCalls)
	}
}

func TestSubLead_SubtaskFailurePropagates(t *testing.T) {
	subtasks := []sublead.SubTask{
		{ID: "ok", Prompt: "p"},
		{ID: "bad", Prompt: "p"},
	}
	worker := sublead.NewSubLeadWorker(
		fakePlanner(subtasks),
		recordingSubWorker("cursor", &[]string{}, &sync.Mutex{}, "bad"),
		"cursor",
		2,
	)

	tr := worker(context.Background(), &breezing.Task{ID: "lane-fail"})
	r := resultFromTaskResult("cursor", "lane-fail", tr)

	if r.Success {
		t.Fatal("lane must fail when a subtask fails")
	}
	summaries := parseSubtaskSummaries(t, r.Summary)
	var foundBad bool
	for _, s := range summaries {
		if s.ID == "bad" && !s.Success {
			foundBad = true
		}
	}
	if !foundBad {
		t.Errorf("summary must include failed subtask bad; got %+v", summaries)
	}
}

func TestSubLead_EmptyPlanFails(t *testing.T) {
	worker := sublead.NewSubLeadWorker(
		fakePlanner(nil),
		recordingSubWorker("codex", &[]string{}, &sync.Mutex{}, ""),
		"codex",
		1,
	)

	tr := worker(context.Background(), &breezing.Task{ID: "lane-empty"})
	r := resultFromTaskResult("codex", "lane-empty", tr)

	if r.Success {
		t.Fatal("empty mini-plan must fail the lane")
	}
	if r.ExitCode == 0 {
		t.Errorf("ExitCode = 0, want non-zero for empty plan")
	}
}

func TestSubLead_PlannerErrorFails(t *testing.T) {
	planner := func(_ context.Context, _, _ string) (sublead.MiniPlan, error) {
		return sublead.MiniPlan{}, fmt.Errorf("planner boom")
	}
	worker := sublead.NewSubLeadWorker(
		planner,
		recordingSubWorker("codex", &[]string{}, &sync.Mutex{}, ""),
		"codex",
		1,
	)

	tr := worker(context.Background(), &breezing.Task{ID: "lane-err"})
	r := resultFromTaskResult("codex", "lane-err", tr)

	if r.Success {
		t.Fatal("planner error must fail the lane (fail-loud)")
	}
	if !strings.Contains(r.Summary, "planner boom") {
		t.Errorf("Summary = %q, want planner error detail", r.Summary)
	}
}

func TestSubLead_NoPeerMessaging_GrepAudit(t *testing.T) {
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	forbidden := []string{"SendMessage", "send_input", "chan "}
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".go") || strings.HasSuffix(ent.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, ent.Name()))
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		for _, token := range forbidden {
			if strings.Contains(content, token) {
				t.Errorf("%s contains forbidden peer-messaging token %q", ent.Name(), token)
			}
		}
	}
}

func TestHeadlessCLIPlanner_ParsesMiniPlan(t *testing.T) {
	runner := func(_ context.Context, _ string, args []string, stdin string) (string, int, error) {
		if len(args) < 3 || args[0] != "plan" || args[1] != "decompose" {
			t.Errorf("args = %v, want [plan decompose --lane ...]", args)
		}
		if stdin != "lane spec body" {
			t.Errorf("stdin = %q, want lane spec body", stdin)
		}
		out := `{"lane_id":"lane-9","sub_tasks":[{"id":"t1","prompt":"do t1"},{"id":"t2","prompt":"do t2"}]}`
		return out, 0, nil
	}

	planner := sublead.NewHeadlessCLIPlanner(runner)
	plan, err := planner(context.Background(), "lane-9", "lane spec body")
	if err != nil {
		t.Fatalf("planner: %v", err)
	}
	if plan.LaneID != "lane-9" {
		t.Errorf("LaneID = %q, want lane-9", plan.LaneID)
	}
	if len(plan.SubTasks) != 2 {
		t.Fatalf("SubTasks = %d, want 2", len(plan.SubTasks))
	}
	if plan.SubTasks[0].ID != "t1" || plan.SubTasks[1].ID != "t2" {
		t.Errorf("SubTasks = %+v", plan.SubTasks)
	}
}

func TestHeadlessCLIPlanner_FailsLoudOnBadJSON(t *testing.T) {
	runner := func(_ context.Context, _ string, _ []string, _ string) (string, int, error) {
		return "not-json", 0, nil
	}
	planner := sublead.NewHeadlessCLIPlanner(runner)
	_, err := planner(context.Background(), "lane-x", "spec")
	if err == nil {
		t.Fatal("bad JSON must return error (fail-loud)")
	}
}

func TestHeadlessCLIPlanner_FailsLoudOnNonZeroExit(t *testing.T) {
	runner := func(_ context.Context, _ string, _ []string, _ string) (string, int, error) {
		return "", 1, nil
	}
	planner := sublead.NewHeadlessCLIPlanner(runner)
	_, err := planner(context.Background(), "lane-x", "spec")
	if err == nil {
		t.Fatal("non-zero exit must return error (fail-loud)")
	}
}
