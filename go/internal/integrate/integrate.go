package integrate

import (
	"context"
	"errors"
	"io"

	"github.com/Chachamaru127/claude-code-harness/go/internal/floor"
)

// ErrNotImplemented is returned by the RED-phase stub.
var ErrNotImplemented = errors.New("integrate: not implemented")

type GitRunner interface {
	Run(ctx context.Context, repoRoot string, args ...string) (stdout string, stderr string, err error)
}

type IntegrationRecord struct {
	Sequence     int
	TaskBranch   string
	TrunkBranch  string
	CommitSHA    string
	RereResolved bool
	DurationMs   int64
	Timestamp    string
}

type Options struct {
	RepoRoot     string
	TrunkBranch  string
	TaskBranch   string
	Sequence     int
	ScriptRunner floor.ScriptRunner
	GitRunner    GitRunner
	LedgerWriter func(IntegrationRecord)
	Stdout       io.Writer
}

type Result struct {
	Record      IntegrationRecord
	FloorReport floor.Report
}

// Integrate rebases a task branch onto trunk, cherry-picks it with floor.Gate, and commits.
func Integrate(ctx context.Context, opts Options) (Result, error) {
	return Result{}, ErrNotImplemented
}
