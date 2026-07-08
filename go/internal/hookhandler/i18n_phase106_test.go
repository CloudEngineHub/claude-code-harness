package hookhandler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase106TargetMessagesUseLocalizedHarnessMessage(t *testing.T) {
	state := plansState{
		PmPending:   1,
		CcTodo:      2,
		CcWip:       3,
		CcDone:      4,
		PmConfirmed: 5,
	}

	enSummary := buildSummaryMessage(state, true, true, "en")
	if !strings.Contains(enSummary, "Plans.md update detected") {
		t.Fatalf("English summary should be localized, got:\n%s", enSummary)
	}
	if strings.Contains(enSummary, "更新検知") || strings.Contains(enSummary, "新規タスク") {
		t.Fatalf("English summary should not contain bare Japanese UX text, got:\n%s", enSummary)
	}

	jaSummary := buildSummaryMessage(state, true, true, "ja")
	if !strings.Contains(jaSummary, "Plans.md 更新検知") {
		t.Fatalf("Japanese summary should be preserved behind localizer, got:\n%s", jaSummary)
	}

	enNotification := buildPMNotificationContent(state, true, true, "2026-07-07 12:00:00", "en")
	if !strings.Contains(enNotification, "# Notification for PM") {
		t.Fatalf("English notification should be localized, got:\n%s", enNotification)
	}
	if strings.Contains(enNotification, "PM への通知") || strings.Contains(enNotification, "新規タスク") {
		t.Fatalf("English notification should not contain bare Japanese UX text, got:\n%s", enNotification)
	}

	if got := setupUnknownModeMessage("mystery", "en"); !strings.Contains(got, "unknown mode") {
		t.Fatalf("setup unknown mode should be English by default, got %q", got)
	}
	if got := setupUnknownModeMessage("mystery", "ja"); !strings.Contains(got, "不明なモード") {
		t.Fatalf("setup unknown mode should preserve Japanese opt-in, got %q", got)
	}

	if got := runPrettierCheck("src/app.ts", "warn", "en"); !strings.Contains(got, "recommended") {
		t.Fatalf("Prettier warning should be English by default, got %q", got)
	}
	if got := runPrettierCheck("src/app.ts", "warn", "ja"); !strings.Contains(got, "推奨") {
		t.Fatalf("Prettier warning should preserve Japanese opt-in, got %q", got)
	}

	h := &UserPromptInjectPolicyHandler{}
	workWarning := h.buildWorkModeWarningMessage("pending", "en")
	if !strings.Contains(workWarning, "work mode is still active") {
		t.Fatalf("work mode warning should be English by default, got:\n%s", workWarning)
	}
	if strings.Contains(workWarning, "重要") || strings.Contains(workWarning, "完了処理") {
		t.Fatalf("English work mode warning should not contain bare Japanese UX text, got:\n%s", workWarning)
	}
}

func TestPhase106HookhandlerI18nRatchetRequiresZeroBareJapanese(t *testing.T) {
	root := repositoryRoot(t)
	cmd := exec.Command("bash", filepath.Join(root, "scripts/ci/check-i18n-hookhandler-ratchet.sh"), root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hookhandler i18n ratchet should require zero bare Japanese literals, got error: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "bare Japanese literals at baseline (0)") {
		t.Fatalf("hookhandler i18n ratchet should report zero baseline, got:\n%s", out)
	}
}

func TestPhase106HookhandlerI18nRatchetFailsForNewBareJapanese(t *testing.T) {
	root := repositoryRoot(t)
	hookhandlerDir := filepath.Join(root, "go", "internal", "hookhandler")
	probe := filepath.Join(hookhandlerDir, "zz_i18n_ratchet_probe.go")
	if err := os.WriteFile(probe, []byte("package hookhandler\n\nfunc phase106BareJapaneseProbe() string { return \"新規裸日本語\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(probe) })

	cmd := exec.Command("bash", filepath.Join(root, "scripts/ci/check-i18n-hookhandler-ratchet.sh"), root)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("hookhandler i18n ratchet should fail for a new bare Japanese literal, got success:\n%s", out)
	}
	if !strings.Contains(string(out), "bare Japanese literals rose to") {
		t.Fatalf("hookhandler i18n ratchet should explain the bare Japanese failure, got:\n%s", out)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if fileExists(filepath.Join(dir, "go.mod")) && fileExists(filepath.Join(dir, "..", "scripts", "ci", "check-i18n-hookhandler-ratchet.sh")) {
			return filepath.Clean(filepath.Join(dir, ".."))
		}
		if fileExists(filepath.Join(dir, "scripts", "ci", "check-i18n-hookhandler-ratchet.sh")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}
