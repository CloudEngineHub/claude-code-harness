package hookhandler

import (
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
