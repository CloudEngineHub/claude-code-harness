package reviewiterate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
)

const testCarrierPrefix = "companion-result.v1:"

func carryTestResult(r companionresult.Result) breezing.TaskResult {
	tr := breezing.TaskResult{
		TaskID:     r.TaskID,
		CommitHash: testCarrierPrefix + string(mustJSON(r)),
	}
	if !r.Success {
		tr.Err = fmt.Errorf("task %s failed", r.TaskID)
	}
	return tr
}

func mustJSON(r companionresult.Result) []byte {
	b, err := r.Marshal()
	if err != nil {
		panic(err)
	}
	return b
}

func initialResult(taskID, summary string) companionresult.Result {
	r := companionresult.New("codex", taskID)
	r.Success = true
	r.ExitCode = 0
	r.Summary = summary
	return r
}

func TestRun_ApproveOnFirstIteration(t *testing.T) {
	var workerCalls int
	cfg := Config{
		Lenses:    []string{"correctness", "security"},
		Reviewers: []Reviewer{noopReviewer("correctness"), noopReviewer("security")},
		Brain: func(_ context.Context, _ string, _ []Review) (Verdict, error) {
			return VerdictApprove, nil
		},
		MaxIters: 3,
		WorkerFunc: func(_ context.Context, _ *breezing.Task) breezing.TaskResult {
			workerCalls++
			return carryTestResult(initialResult("t1", "refined"))
		},
	}

	out, err := Run(context.Background(), cfg, initialResult("t1", "worker output v0"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Verdict != VerdictApprove {
		t.Fatalf("Verdict = %q, want APPROVE", out.Verdict)
	}
	if out.Iterations != 1 {
		t.Fatalf("Iterations = %d, want 1", out.Iterations)
	}
	if workerCalls != 0 {
		t.Fatalf("WorkerFunc calls = %d, want 0 (no refinement)", workerCalls)
	}
	if out.Escalated {
		t.Fatal("Escalated should be false")
	}
}

func TestRun_RequestChangesThenApprove(t *testing.T) {
	var workerCalls int
	round := 0
	cfg := Config{
		Lenses:    []string{"correctness", "security"},
		Reviewers: []Reviewer{findingReviewer("correctness", "missing test"), noopReviewer("security")},
		Brain: func(_ context.Context, _ string, advisories []Review) (Verdict, error) {
			round++
			if round == 1 {
				if len(advisories) == 0 || len(advisories[0].Findings) == 0 {
					t.Fatal("brain round 1 should see advisory findings")
				}
				return VerdictRequestChanges, nil
			}
			return VerdictApprove, nil
		},
		MaxIters: 3,
		WorkerFunc: func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			workerCalls++
			r := initialResult(task.ID, "worker output v1")
			return carryTestResult(r)
		},
	}

	out, err := Run(context.Background(), cfg, initialResult("t1", "worker output v0"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Verdict != VerdictApprove {
		t.Fatalf("Verdict = %q, want APPROVE", out.Verdict)
	}
	if out.Iterations != 2 {
		t.Fatalf("Iterations = %d, want 2", out.Iterations)
	}
	if workerCalls != 1 {
		t.Fatalf("WorkerFunc calls = %d, want 1 (single refinement)", workerCalls)
	}
}

func TestRun_MaxItersUnconverged_Escalates(t *testing.T) {
	cfg := Config{
		Lenses: []string{"correctness", "security"},
		Reviewers: []Reviewer{
			findingReviewer("correctness", "bug A", "bug B"),
			findingReviewer("security", "sec C"),
		},
		Brain: func(_ context.Context, _ string, _ []Review) (Verdict, error) {
			return VerdictRequestChanges, nil
		},
		MaxIters: 2,
		WorkerFunc: func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			return carryTestResult(initialResult(task.ID, "still bad"))
		},
	}

	out, err := Run(context.Background(), cfg, initialResult("lane", "v0"))
	if err != nil {
		t.Fatalf("Run should not error on escalation: %v", err)
	}
	if !out.Escalated {
		t.Fatal("Escalated = false, want true")
	}
	if out.Verdict != VerdictRequestChanges {
		t.Fatalf("Verdict = %q, want REQUEST_CHANGES", out.Verdict)
	}
	if out.Iterations != 2 {
		t.Fatalf("Iterations = %d, want 2", out.Iterations)
	}
	if out.EscalationNote == "" {
		t.Fatal("EscalationNote must not be empty")
	}
	for _, lens := range []string{"correctness", "security"} {
		if !strings.Contains(out.EscalationNote, lens) {
			t.Errorf("EscalationNote %q should mention lens %q", out.EscalationNote, lens)
		}
	}
	if !strings.Contains(out.EscalationNote, "3") && !strings.Contains(out.EscalationNote, "finding") {
		t.Errorf("EscalationNote should include finding count, got %q", out.EscalationNote)
	}
}

func TestRun_FreshContextNotShared(t *testing.T) {
	var (
		mu         sync.Mutex
		sessionIDs []string
	)
	makeReviewer := func(lens string) Reviewer {
		return func(_ context.Context, lensName, _ string) (Review, error) {
			mu.Lock()
			sid := fmt.Sprintf("%s-%d", lensName, len(sessionIDs))
			sessionIDs = append(sessionIDs, sid)
			mu.Unlock()
			return Review{Lens: lens, Findings: []string{"issue"}}, nil
		}
	}

	round := 0
	cfg := Config{
		Lenses:    []string{"correctness", "security", "perf"},
		Reviewers: []Reviewer{makeReviewer("correctness"), makeReviewer("security"), makeReviewer("perf")},
		Brain: func(_ context.Context, _ string, _ []Review) (Verdict, error) {
			round++
			if round < 2 {
				return VerdictRequestChanges, nil
			}
			return VerdictApprove, nil
		},
		MaxIters: 3,
		WorkerFunc: func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			return carryTestResult(initialResult(task.ID, "refined"))
		},
	}

	if _, err := Run(context.Background(), cfg, initialResult("t1", "v0")); err != nil {
		t.Fatalf("Run: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	seen := map[string]struct{}{}
	for _, sid := range sessionIDs {
		if _, dup := seen[sid]; dup {
			t.Fatalf("duplicate session-id %q — fresh-context contract violated (all=%v)", sid, sessionIDs)
		}
		seen[sid] = struct{}{}
	}
	// 2 rounds × 3 lenses = 6 unique sessions
	if len(sessionIDs) != 6 {
		t.Fatalf("got %d reviewer session IDs, want 6 (2 rounds × 3 lenses): %v", len(sessionIDs), sessionIDs)
	}
}

func TestRun_MultiLensesParallel(t *testing.T) {
	const n = 3
	barrier := make(chan struct{})
	var inside int32
	var maxInside int32

	makeBarrierReviewer := func(lens string) Reviewer {
		return func(_ context.Context, lensName, _ string) (Review, error) {
			cur := atomic.AddInt32(&inside, 1)
			for {
				old := atomic.LoadInt32(&maxInside)
				if cur <= old || atomic.CompareAndSwapInt32(&maxInside, old, cur) {
					break
				}
			}
			<-barrier
			atomic.AddInt32(&inside, -1)
			return Review{Lens: lensName}, nil
		}
	}

	reviewers := make([]Reviewer, n)
	lenses := make([]string, n)
	for i := 0; i < n; i++ {
		lenses[i] = fmt.Sprintf("lens-%d", i)
		reviewers[i] = makeBarrierReviewer(lenses[i])
	}

	unblock := make(chan struct{})
	go func() {
		<-unblock
		close(barrier)
	}()

	cfg := Config{
		Lenses:    lenses,
		Reviewers: reviewers,
		Brain: func(_ context.Context, _ string, _ []Review) (Verdict, error) {
			return VerdictApprove, nil
		},
		MaxIters:   1,
		WorkerFunc: func(_ context.Context, _ *breezing.Task) breezing.TaskResult { return breezing.TaskResult{} },
	}

	done := make(chan struct{})
	go func() {
		_, _ = Run(context.Background(), cfg, initialResult("t1", "v0"))
		close(done)
	}()

	// Give reviewers time to enter barrier.
	deadline := make(chan struct{})
	go func() {
		for atomic.LoadInt32(&maxInside) < int32(n) {
			// spin briefly
		}
		close(deadline)
	}()
	select {
	case <-deadline:
	case <-done:
		t.Fatal("reviewers did not reach parallel barrier before Run completed")
	}
	if got := atomic.LoadInt32(&maxInside); got < int32(n) {
		t.Fatalf("max concurrent reviewers = %d, want >= %d (parallel fan-out)", got, n)
	}
	close(unblock)
	<-done
}

func TestRun_BrainOnlyVerdict_NoSelfApprove(t *testing.T) {
	cfg := Config{
		Lenses:    []string{"correctness", "security"},
		Reviewers: []Reviewer{findingReviewer("correctness", "must fix"), noopReviewer("security")},
		Brain: func(_ context.Context, _ string, _ []Review) (Verdict, error) {
			return VerdictRequestChanges, nil
		},
		MaxIters: 1,
		WorkerFunc: func(_ context.Context, task *breezing.Task) breezing.TaskResult {
			return carryTestResult(initialResult(task.ID, "nope"))
		},
	}

	out, err := Run(context.Background(), cfg, initialResult("t1", "v0"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Verdict == VerdictApprove {
		t.Fatal("Run must not override brain REQUEST_CHANGES with APPROVE")
	}
	if out.Verdict != VerdictRequestChanges {
		t.Fatalf("Verdict = %q, want REQUEST_CHANGES", out.Verdict)
	}
}

func TestRun_ConfigValidation(t *testing.T) {
	validBrain := func(_ context.Context, _ string, _ []Review) (Verdict, error) {
		return VerdictApprove, nil
	}
	validWorker := func(_ context.Context, _ *breezing.Task) breezing.TaskResult { return breezing.TaskResult{} }
	base := initialResult("t", "v0")

	cases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "lens reviewer length mismatch",
			cfg: Config{
				Lenses:     []string{"a", "b"},
				Reviewers:  []Reviewer{noopReviewer("a")},
				Brain:      validBrain,
				MaxIters:   1,
				WorkerFunc: validWorker,
			},
		},
		{
			name: "empty lenses",
			cfg: Config{
				Lenses:     nil,
				Reviewers:  nil,
				Brain:      validBrain,
				MaxIters:   1,
				WorkerFunc: validWorker,
			},
		},
		{
			name: "single lens",
			cfg: Config{
				Lenses:     []string{"only"},
				Reviewers:  []Reviewer{noopReviewer("only")},
				Brain:      validBrain,
				MaxIters:   1,
				WorkerFunc: validWorker,
			},
		},
		{
			name: "max iters zero",
			cfg: Config{
				Lenses:     []string{"a", "b"},
				Reviewers:  []Reviewer{noopReviewer("a"), noopReviewer("b")},
				Brain:      validBrain,
				MaxIters:   0,
				WorkerFunc: validWorker,
			},
		},
		{
			name: "max iters negative",
			cfg: Config{
				Lenses:     []string{"a", "b"},
				Reviewers:  []Reviewer{noopReviewer("a"), noopReviewer("b")},
				Brain:      validBrain,
				MaxIters:   -1,
				WorkerFunc: validWorker,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Run(context.Background(), tc.cfg, base)
			if err == nil {
				t.Fatal("expected config validation error, got nil")
			}
		})
	}
}

func noopReviewer(lens string) Reviewer {
	return func(_ context.Context, lensName, _ string) (Review, error) {
		return Review{Lens: lensName}, nil
	}
}

func findingReviewer(lens string, findings ...string) Reviewer {
	return func(_ context.Context, lensName, _ string) (Review, error) {
		return Review{Lens: lensName, Findings: append([]string(nil), findings...)}, nil
	}
}
