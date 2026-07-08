package autoapprove

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/orchestrationledger"
)

func allPrereqsDone() PrereqChecker {
	return func(name string) bool { return true }
}

func withPrereqChecker(t *testing.T, c PrereqChecker) {
	t.Helper()
	restore := SetPrereqChecker(c)
	t.Cleanup(restore)
}

func TestAutoApproveEnabled_DefaultOff(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "")
	withPrereqChecker(t, allPrereqsDone())

	enabled, reason := AutoApproveEnabled(t.TempDir())
	if enabled {
		t.Fatal("expected disabled when env unset")
	}
	if reason != "auto-approve:disabled (env=off)" {
		t.Fatalf("reason = %q, want auto-approve:disabled (env=off)", reason)
	}
	if !strings.Contains(reason, "env=off") {
		t.Fatalf("reason %q must contain env=off", reason)
	}
}

func TestAutoApproveEnabled_OnButPrereqMissing(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "on")
	withPrereqChecker(t, func(name string) bool {
		return name != PrereqPhase96_1_2
	})

	enabled, reason := AutoApproveEnabled(t.TempDir())
	if enabled {
		t.Fatal("expected disabled when a prereq is missing")
	}
	if !strings.Contains(reason, "prereq-missing:") {
		t.Fatalf("reason = %q, want prereq-missing segment", reason)
	}
	if !strings.Contains(reason, PrereqPhase96_1_2) {
		t.Fatalf("reason = %q, want missing prereq %q listed", reason, PrereqPhase96_1_2)
	}
}

func TestAutoApproveEnabled_OnAndAllPrereqsDone(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "on")
	withPrereqChecker(t, allPrereqsDone())

	enabled, reason := AutoApproveEnabled(t.TempDir())
	if !enabled {
		t.Fatalf("expected enabled, reason=%q", reason)
	}
	if reason != "auto-approve:enabled" {
		t.Fatalf("reason = %q, want auto-approve:enabled", reason)
	}
}

func TestAutoApproveEnabled_EnvValueStrictness(t *testing.T) {
	withPrereqChecker(t, allPrereqsDone())
	cases := []string{"true", "1", "yes", "ON", "True", "on "}
	for _, envVal := range cases {
		t.Run(envVal, func(t *testing.T) {
			t.Setenv("HARNESS_AUTO_APPROVE", envVal)
			enabled, reason := AutoApproveEnabled(t.TempDir())
			if enabled {
				t.Fatalf("env %q must not enable auto-approve", envVal)
			}
			if !strings.Contains(reason, "env=off") {
				t.Fatalf("reason = %q, want env=off segment", reason)
			}
		})
	}
}

func TestAutoApproveEnabled_NoEnvOverridesFloor(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "on")
	withPrereqChecker(t, allPrereqsDone())

	worktree := t.TempDir()
	enabled, _ := AutoApproveEnabled(worktree)
	if !enabled {
		t.Fatal("expected enabled with env on and all prereqs done")
	}

	outside := filepath.Join(t.TempDir(), "escape.txt")
	if AppliesTo(outside, worktree) {
		t.Fatal("AppliesTo must be false for paths outside the worktree")
	}
}

func TestAppliesTo_PathInsideWorktree(t *testing.T) {
	worktree := t.TempDir()
	inside := filepath.Join(worktree, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatal(err)
	}
	if !AppliesTo(inside, worktree) {
		t.Fatalf("AppliesTo(%q, %q) = false, want true", inside, worktree)
	}
}

func TestAppliesTo_PathOutsideWorktree(t *testing.T) {
	worktree := t.TempDir()
	outsideRoot := t.TempDir()
	outside := filepath.Join(outsideRoot, "main.go")
	if AppliesTo(outside, worktree) {
		t.Fatalf("AppliesTo(%q, %q) = true, want false", outside, worktree)
	}
}

func TestAppliesTo_RelativePath_Resolved(t *testing.T) {
	worktree := t.TempDir()
	rel := filepath.Join("src", "rel.go")
	abs := filepath.Join(worktree, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if !AppliesTo(rel, worktree) {
		t.Fatalf("AppliesTo(%q, %q) = false, want true for relative path under worktree", rel, worktree)
	}
}

func TestAutoApproveEnabled_ReasonStable(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "on")
	withPrereqChecker(t, func(name string) bool {
		return name != PrereqPhase92_2_3
	})

	root := t.TempDir()
	_, reason1 := AutoApproveEnabled(root)
	_, reason2 := AutoApproveEnabled(root)
	if reason1 != reason2 {
		t.Fatalf("reasons differ: %q vs %q", reason1, reason2)
	}
}

func TestEmitTeamDispatch_RecordsAutoApproveReason(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "ledger.jsonl")
	t.Setenv("HARNESS_ORCHESTRATION_LEDGER", ledger)
	t.Setenv("HARNESS_AUTO_APPROVE", "on")
	withPrereqChecker(t, allPrereqsDone())

	enabled, reason := AutoApproveEnabled(dir)
	if !enabled {
		t.Fatalf("expected enabled for ledger integration, reason=%q", reason)
	}
	if reason != "auto-approve:enabled" {
		t.Fatalf("reason = %q, want auto-approve:enabled", reason)
	}

	exit := 0
	orchestrationledger.EmitTeamDispatch(orchestrationledger.TeamDispatchOpts{
		Backend:    "codex",
		Write:      true,
		ExitCode:   &exit,
		DurationMs: 1,
		Reason:     reason,
		Enabled:    enabled,
		RepoRoot:   dir,
	})

	data, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
	var entry orchestrationledger.Entry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if entry.SessionID != reason {
		t.Fatalf("session_id(reason) = %q, want bytes-identical %q", entry.SessionID, reason)
	}
	if entry.Counts != enabled {
		t.Fatalf("counts = %v, want %v", entry.Counts, enabled)
	}
}

func TestDefaultPrereqChecker_PlansDoneMarker(t *testing.T) {
	root := t.TempDir()
	plans := filepath.Join(root, "Plans.md")
	content := "| Task | 内容 | DoD | Depends | Status |\n" +
		"| --- | --- | --- | --- | --- |\n" +
		"| 92.1.1 | x | y | - | cc:done [abc12345] |\n" +
		"| 92.2.3 | x | y | - | cc:done [def67890] |\n" +
		"| 96.1.2 | x | y | - | cc:TODO |\n"
	if err := os.WriteFile(plans, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	restore := SetPrereqChecker(defaultPrereqChecker)
	t.Cleanup(restore)

	prev := prereqRepoRoot
	prereqRepoRoot = root
	t.Cleanup(func() { prereqRepoRoot = prev })

	if !prereqChecker(PrereqPhase92_1_1) {
		t.Fatal("expected 92.1.1 done from Plans.md marker")
	}
	if prereqChecker(PrereqPhase96_1_2) {
		t.Fatal("expected 96.1.2 not done when status is cc:TODO")
	}
}
