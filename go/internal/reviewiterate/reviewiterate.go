package reviewiterate

import (
	"context"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
)

type Verdict string

const (
	VerdictApprove        Verdict = "APPROVE"
	VerdictRequestChanges Verdict = "REQUEST_CHANGES"
)

// Review is one advisory review lens result. Primary verdict is never set here.
type Review struct {
	Lens     string
	Findings []string
	Refined  string
}

// Reviewer runs one fresh-context advisory review for a single lens.
type Reviewer func(ctx context.Context, lensName, workerOutput string) (Review, error)

// BrainVerdict is the brain (claude host) primary verdict over worker output
// and advisory reviews.
type BrainVerdict func(ctx context.Context, workerOutput string, advisories []Review) (Verdict, error)

type Config struct {
	Lenses     []string
	Reviewers  []Reviewer
	Brain      BrainVerdict
	MaxIters   int
	WorkerFunc breezing.WorkerFunc
}

type Outcome struct {
	Verdict        Verdict
	Iterations     int
	Advisories     []Review
	Escalated      bool
	EscalationNote string
	FinalResult    companionresult.Result
}

// Run executes the review→iterate loop. See run.go.
func Run(ctx context.Context, cfg Config, initialResult companionresult.Result) (Outcome, error) {
	return runLoop(ctx, cfg, initialResult)
}
