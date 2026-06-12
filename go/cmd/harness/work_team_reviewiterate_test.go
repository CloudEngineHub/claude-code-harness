package main

import (
	"context"
	"sync"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
	"github.com/Chachamaru127/claude-code-harness/go/internal/reviewiterate"
)

func TestReviewIterateEnabled_DefaultOff(t *testing.T) {
	t.Setenv("HARNESS_REVIEW_ITERATE", "")
	if reviewIterateEnabled() {
		t.Fatal("review iterate should default OFF")
	}
}

func TestReviewIterateEnabled_On(t *testing.T) {
	t.Setenv("HARNESS_REVIEW_ITERATE", "on")
	if !reviewIterateEnabled() {
		t.Fatal("HARNESS_REVIEW_ITERATE=on should enable review iterate")
	}
}

func TestRunTeam_FlatPathUnchangedWhenReviewIterateOff(t *testing.T) {
	passingFloorGate(t)
	t.Setenv("HARNESS_REVIEW_ITERATE", "")

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
		t.Fatalf("flat path should invoke worker once; calls=%v", calls)
	}
	if len(results) != 1 || results[0].TaskID != "flat-1" {
		t.Errorf("results = %+v, want flat-1", results)
	}
}

func TestRunTeam_ReviewIterateOn_InvokesLoop(t *testing.T) {
	passingFloorGate(t)
	t.Setenv("HARNESS_REVIEW_ITERATE", "on")

	var loopCalls int
	origRunner := reviewIterateRunner
	reviewIterateRunner = func(_ context.Context, cfg reviewiterate.Config, initial companionresult.Result) (reviewiterate.Outcome, error) {
		loopCalls++
		if initial.TaskID != "lane-x" {
			t.Fatalf("initial result TaskID = %q, want lane-x", initial.TaskID)
		}
		if cfg.WorkerFunc == nil {
			t.Fatal("review iterate config must carry WorkerFunc for refinement")
		}
		return reviewiterate.Outcome{
			Verdict:    reviewiterate.VerdictApprove,
			Iterations: 1,
		}, nil
	}
	defer func() { reviewIterateRunner = origRunner }()

	origFactory := teamWorkerFactory
	teamWorkerFactory = func(backend string) breezing.WorkerFunc {
		return wrapWorkerWithReviewIterate(func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			r := companionresult.New(backend, task.ID)
			r.Success = true
			r.Summary = "worker done"
			return carryResult(r)
		}, backend)
	}
	defer func() { teamWorkerFactory = origFactory }()

	results, err := runTeam([]string{"lane-x"}, "codex", 1)
	if err != nil {
		t.Fatalf("runTeam: %v", err)
	}
	if loopCalls != 1 {
		t.Fatalf("review iterate loop calls = %d, want 1", loopCalls)
	}
	if len(results) != 1 || results[0].TaskID != "lane-x" {
		t.Errorf("results = %+v", results)
	}
}
