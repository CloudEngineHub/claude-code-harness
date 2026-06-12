// Package autoapprove gates HARNESS_AUTO_APPROVE behind phase prereqs and worktree scope.
// fail-safe: default OFF; missing any prereq → auto-approve OFF (not overridable by env alone).
package autoapprove

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Chachamaru127/claude-code-harness/go/internal/plans"
)

const (
	PrereqPhase92_1_1 = "92.1.1"
	PrereqPhase92_2_3 = "92.2.3"
	PrereqPhase96_1_2 = "96.1.2"
)

var requiredPrereqs = []string{
	PrereqPhase92_1_1,
	PrereqPhase92_2_3,
	PrereqPhase96_1_2,
}

// PrereqChecker reports whether a named phase prereq is complete.
// Production reads Plans.md / gate files via defaultPrereqChecker; tests inject fakes.
type PrereqChecker func(name string) bool

var prereqChecker PrereqChecker = defaultPrereqChecker

// prereqRepoRoot is set for the duration of AutoApproveEnabled so defaultPrereqChecker
// can locate Plans.md without widening the PrereqChecker signature.
var prereqRepoRoot string

// SetPrereqChecker replaces the prereq checker (tests only). The returned func restores it.
func SetPrereqChecker(c PrereqChecker) func() {
	prev := prereqChecker
	prereqChecker = c
	return func() { prereqChecker = prev }
}

// AutoApproveEnabled returns true only when all four conditions hold:
//
//  1. env HARNESS_AUTO_APPROVE=on (strict: only lowercase "on")
//  2. PrereqPhase92_1_1 done
//  3. PrereqPhase92_2_3 done
//  4. PrereqPhase96_1_2 done
//
// Any missing condition yields fail-safe false; reason is forwarded to orchestration ledger.
func AutoApproveEnabled(repoRoot string) (enabled bool, reason string) {
	prevRoot := prereqRepoRoot
	prereqRepoRoot = repoRoot
	defer func() { prereqRepoRoot = prevRoot }()

	if os.Getenv("HARNESS_AUTO_APPROVE") != "on" {
		return false, "auto-approve:disabled (env=off)"
	}

	var missing []string
	for _, name := range requiredPrereqs {
		if !prereqChecker(name) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return false, "auto-approve:disabled (prereq-missing:" + strings.Join(missing, ",") + ")"
	}
	return true, "auto-approve:enabled"
}

// AppliesTo reports whether path is inside worktreeRoot. Callers use false to escalate
// to human approval; worktree-escape hard-stop remains runtimefloor / wtfingerprint.
func AppliesTo(path string, worktreeRoot string) bool {
	if path == "" || worktreeRoot == "" {
		return false
	}
	wtRoot, err := filepath.Abs(worktreeRoot)
	if err != nil {
		wtRoot = filepath.Clean(worktreeRoot)
	}
	target := path
	if !filepath.IsAbs(path) {
		target = filepath.Join(wtRoot, path)
	}
	absPath, err := filepath.Abs(target)
	if err != nil {
		absPath = filepath.Clean(target)
	}
	return pathUnderWorktree(absPath, wtRoot)
}

func defaultPrereqChecker(name string) bool {
	root := prereqRepoRoot
	if root == "" {
		return false
	}
	if plansDone(root, name) {
		return true
	}
	gate := filepath.Join(root, ".claude", "state", "phase-gates", name+".done")
	if info, err := os.Stat(gate); err == nil && !info.IsDir() {
		return true
	}
	return false
}

func plansDone(repoRoot, taskID string) bool {
	plansPath := filepath.Join(repoRoot, "Plans.md")
	tasks, err := plans.ParseFile(plansPath)
	if err != nil {
		return false
	}
	task := plans.Find(tasks, taskID)
	if task == nil {
		return false
	}
	return task.Tags.Done
}

func pathUnderWorktree(path, worktreeRoot string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(worktreeRoot)
	if cleanPath == cleanRoot {
		return true
	}
	return strings.HasPrefix(cleanPath, cleanRoot+string(filepath.Separator))
}
