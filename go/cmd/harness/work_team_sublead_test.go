package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
	"github.com/Chachamaru127/claude-code-harness/go/internal/sublead"
)

func TestTeamWorkerFactory_SubLeadHierarchyWhenEnvSet(t *testing.T) {
	passingFloorGate(t)
	t.Setenv("HARNESS_TEAM_HIERARCHY", "sublead")

	origPlanner := teamSubLeadPlanner
	teamSubLeadPlanner = func() sublead.Planner {
		return func(_ context.Context, laneID, _ string) (sublead.MiniPlan, error) {
			return sublead.MiniPlan{
				LaneID: laneID,
				SubTasks: []sublead.SubTask{
					{ID: laneID + "-sub", Prompt: "work"},
				},
			}, nil
		}
	}
	defer func() { teamSubLeadPlanner = origPlanner }()

	var (
		mu       sync.Mutex
		subCalls []string
	)
	origSubWorker := teamSubWorkerFactory
	teamSubWorkerFactory = func(backend string) breezing.WorkerFunc {
		return func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			mu.Lock()
			subCalls = append(subCalls, task.ID)
			mu.Unlock()
			r := companionresult.New(backend, task.ID)
			r.Success = true
			r.Summary = "sub ok"
			return carryResult(r)
		}
	}
	defer func() { teamSubWorkerFactory = origSubWorker }()

	results, err := runTeam([]string{"lane-h"}, "cursor", 1)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if len(subCalls) != 1 || subCalls[0] != "lane-h-sub" {
		t.Fatalf("sublead path not used: subCalls=%v", subCalls)
	}
	if results[0].TaskID != "lane-h" {
		t.Errorf("TaskID = %q, want lane-h", results[0].TaskID)
	}

	var summaries []struct {
		ID      string `json:"id"`
		Success bool   `json:"success"`
	}
	if err := json.Unmarshal([]byte(results[0].Summary), &summaries); err != nil {
		t.Fatalf("lane summary should be subtask JSON array: %v", err)
	}
	if len(summaries) != 1 || summaries[0].ID != "lane-h-sub" {
		t.Errorf("summaries = %+v, want lane-h-sub entry", summaries)
	}
}

func TestTeamWorkerFactory_DefaultFlatWithoutEnv(t *testing.T) {
	passingFloorGate(t)
	t.Setenv("HARNESS_TEAM_HIERARCHY", "")

	var (
		mu    sync.Mutex
		calls []string
	)
	indexOf := map[string]int{"flat-1": 0}

	orig := teamWorkerFactory
	teamWorkerFactory = recordingFactory(indexOf, &calls, &mu)
	defer func() { teamWorkerFactory = orig }()

	results, err := runTeam([]string{"flat-1"}, "codex", 1)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if len(calls) != 1 || calls[0] != "flat-1" {
		t.Fatalf("flat factory should invoke worker once per lane task; calls=%v", calls)
	}
	if len(results) != 1 || results[0].TaskID != "flat-1" {
		t.Errorf("results = %+v, want single flat-1 result", results)
	}
}
