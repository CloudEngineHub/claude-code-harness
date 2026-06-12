package integrate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Chachamaru127/claude-code-harness/go/internal/floor"
	"github.com/Chachamaru127/claude-code-harness/go/internal/orchestrationledger"
)

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

type defaultGitRunner struct{}

func (defaultGitRunner) Run(ctx context.Context, repoRoot string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	combined := string(out)
	if err != nil {
		return combined, combined, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, combined)
	}
	return combined, "", nil
}

// Integrate rebases a task branch onto trunk, cherry-picks it with floor.Gate, and commits.
func Integrate(ctx context.Context, opts Options) (Result, error) {
	start := time.Now()
	if err := validateOptions(opts); err != nil {
		return Result{}, err
	}

	git := opts.GitRunner
	if git == nil {
		git = defaultGitRunner{}
	}
	stdout := opts.Stdout
	if stdout == nil {
		stdout = io.Discard
	}

	if _, _, err := git.Run(ctx, opts.RepoRoot, "checkout", opts.TaskBranch); err != nil {
		return Result{}, fmt.Errorf("checkout task branch: %w", err)
	}
	var rereResolved bool
	rrBeforeRebase := rrCacheModTime(opts.RepoRoot)
	rebaseRere, rebaseErr := runRebase(ctx, git, opts.RepoRoot, opts.TrunkBranch)
	if rebaseErr != nil {
		_, _, _ = git.Run(ctx, opts.RepoRoot, "rebase", "--abort")
		return Result{}, fmt.Errorf("rebase task onto trunk: %w", rebaseErr)
	}
	if rebaseRere || rrCacheModTime(opts.RepoRoot).After(rrBeforeRebase) {
		rereResolved = true
	}

	if _, _, err := git.Run(ctx, opts.RepoRoot, "checkout", opts.TrunkBranch); err != nil {
		return Result{}, fmt.Errorf("checkout trunk: %w", err)
	}

	rrBeforePick := rrCacheModTime(opts.RepoRoot)
	cpErr := runCherryPickNoCommit(ctx, git, opts.RepoRoot, opts.TaskBranch)
	if cpErr != nil {
		unmerged, _ := hasUnmerged(ctx, git, opts.RepoRoot)
		if unmerged {
			_, _, _ = git.Run(ctx, opts.RepoRoot, "cherry-pick", "--abort")
			return Result{}, fmt.Errorf("cherry-pick task: %w", cpErr)
		}
	}
	if rrCacheUsed(cpErr) || rrCacheModTime(opts.RepoRoot).After(rrBeforePick) {
		rereResolved = true
	}

	changedFiles, err := stagedFiles(ctx, git, opts.RepoRoot)
	if err != nil {
		abortCherryPick(ctx, git, opts.RepoRoot)
		return Result{}, err
	}

	floorReport := floor.Gate(opts.RepoRoot, changedFiles, opts.ScriptRunner)
	if !floorReport.Passed {
		abortCherryPick(ctx, git, opts.RepoRoot)
		return Result{FloorReport: floorReport}, fmt.Errorf("floor gate failed: %+v", floorReport)
	}

	msg := fmt.Sprintf("integrate(task): %s seq=%d", opts.TaskBranch, opts.Sequence)
	if _, _, err := git.Run(ctx, opts.RepoRoot, "commit", "-m", msg); err != nil {
		abortCherryPick(ctx, git, opts.RepoRoot)
		return Result{FloorReport: floorReport}, fmt.Errorf("commit integration: %w", err)
	}

	shaOut, _, err := git.Run(ctx, opts.RepoRoot, "rev-parse", "HEAD")
	if err != nil {
		return Result{FloorReport: floorReport}, fmt.Errorf("rev-parse HEAD: %w", err)
	}
	commitSHA := strings.TrimSpace(shaOut)

	record := IntegrationRecord{
		Sequence:     opts.Sequence,
		TaskBranch:   opts.TaskBranch,
		TrunkBranch:  opts.TrunkBranch,
		CommitSHA:    commitSHA,
		RereResolved: rereResolved,
		DurationMs:   time.Since(start).Milliseconds(),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	if opts.LedgerWriter != nil {
		opts.LedgerWriter(record)
	} else {
		exit := 0
		orchestrationledger.EmitIntegration(orchestrationledger.IntegrationOpts{
			Backend:      "lead",
			RepoRoot:     opts.RepoRoot,
			Write:        true,
			ExitCode:     orchestrationledger.IntPtr(exit),
			DurationMs:   record.DurationMs,
			Sequence:     record.Sequence,
			TaskBranch:   record.TaskBranch,
			TrunkBranch:  record.TrunkBranch,
			CommitSHA:    record.CommitSHA,
			RereResolved: record.RereResolved,
			FloorPass:    floorReport.Passed,
		})
	}

	fmt.Fprintf(stdout, "integrated %s -> %s (%s)\n", opts.TaskBranch, opts.TrunkBranch, commitSHA)

	return Result{Record: record, FloorReport: floorReport}, nil
}

func validateOptions(opts Options) error {
	if strings.TrimSpace(opts.RepoRoot) == "" {
		return errors.New("integrate: RepoRoot is required")
	}
	if strings.TrimSpace(opts.TrunkBranch) == "" {
		return errors.New("integrate: TrunkBranch is required")
	}
	if strings.TrimSpace(opts.TaskBranch) == "" {
		return errors.New("integrate: TaskBranch is required")
	}
	if opts.Sequence <= 0 {
		return errors.New("integrate: Sequence must be positive")
	}
	return nil
}

func runRebase(ctx context.Context, git GitRunner, repoRoot, trunk string) (bool, error) {
	_, stderr, err := git.Run(ctx, repoRoot, "rebase", trunk)
	rere := strings.Contains(stderr, "using previous resolution")
	if err == nil {
		return rere, nil
	}
	if rere {
		if _, _, contErr := git.Run(ctx, repoRoot, "rebase", "--continue"); contErr == nil {
			return true, nil
		}
	}
	for i := 0; i < 8; i++ {
		inRebase, _ := isRebaseInProgress(ctx, git, repoRoot)
		if !inRebase {
			return rere, err
		}
		unmerged, uerr := hasUnmerged(ctx, git, repoRoot)
		if uerr != nil {
			return rere, uerr
		}
		if !unmerged {
			if _, _, contErr := git.Run(ctx, repoRoot, "rebase", "--continue"); contErr == nil {
				return rere, nil
			}
		}
		if unmerged {
			return rere, err
		}
	}
	return rere, err
}

func runCherryPickNoCommit(ctx context.Context, git GitRunner, repoRoot, taskBranch string) error {
	_, stderr, err := git.Run(ctx, repoRoot, "cherry-pick", "--no-commit", taskBranch)
	if err == nil {
		return nil
	}
	if strings.Contains(stderr, "using previous resolution") {
		if clean, _ := isIndexClean(ctx, git, repoRoot); clean || !hasUnmergedFiles(ctx, git, repoRoot) {
			return nil
		}
	}
	for i := 0; i < 8; i++ {
		inPick, _ := isCherryPickInProgress(ctx, git, repoRoot)
		if !inPick {
			return err
		}
		unmerged, uerr := hasUnmerged(ctx, git, repoRoot)
		if uerr != nil {
			return uerr
		}
		if !unmerged {
			return nil
		}
	}
	return err
}

func stagedFiles(ctx context.Context, git GitRunner, repoRoot string) ([]string, error) {
	out, _, err := git.Run(ctx, repoRoot, "diff", "--cached", "--name-only")
	if err != nil {
		return nil, fmt.Errorf("list staged files: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func hasUnmerged(ctx context.Context, git GitRunner, repoRoot string) (bool, error) {
	out, _, err := git.Run(ctx, repoRoot, "ls-files", "-u")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func hasUnmergedFiles(ctx context.Context, git GitRunner, repoRoot string) bool {
	ok, _ := hasUnmerged(ctx, git, repoRoot)
	return ok
}

func isIndexClean(ctx context.Context, git GitRunner, repoRoot string) (bool, error) {
	out, _, err := git.Run(ctx, repoRoot, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

func isRebaseInProgress(ctx context.Context, git GitRunner, repoRoot string) (bool, error) {
	_, err := os.Stat(filepath.Join(repoRoot, ".git", "rebase-merge"))
	if err == nil {
		return true, nil
	}
	_, err = os.Stat(filepath.Join(repoRoot, ".git", "rebase-apply"))
	return err == nil, nil
}

func isCherryPickInProgress(ctx context.Context, git GitRunner, repoRoot string) (bool, error) {
	_, err := os.Stat(filepath.Join(repoRoot, ".git", "CHERRY_PICK_HEAD"))
	return err == nil, nil
}

func abortCherryPick(ctx context.Context, git GitRunner, repoRoot string) {
	inPick, _ := isCherryPickInProgress(ctx, git, repoRoot)
	if inPick {
		_, _, _ = git.Run(ctx, repoRoot, "cherry-pick", "--abort")
	}
	_, _, _ = git.Run(ctx, repoRoot, "reset", "--hard", "HEAD")
}

func rrCacheModTime(repoRoot string) time.Time {
	info, err := os.Stat(filepath.Join(repoRoot, ".git", "rr-cache"))
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func rrCacheUsed(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "previous resolution") ||
		strings.Contains(msg, "Resolved") ||
		strings.Contains(msg, "rerere")
}
