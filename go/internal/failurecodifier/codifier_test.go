package failurecodifier

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCodifier_ExtractFromLedger_EmptyCorpusReturnsNoRules(t *testing.T) {
	dir := t.TempDir()
	rules, err := ExtractFromLedger(ExtractOpts{
		RepoRoot:                dir,
		JudgmentLedgerPath:      filepath.Join(dir, "missing-judgment.jsonl"),
		OrchestrationLedgerPath: filepath.Join(dir, "missing-orch.jsonl"),
		SchemaPath:              schemaPath(t),
	})
	if err != nil {
		t.Fatalf("ExtractFromLedger: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("got %d rules, want 0 for empty corpus", len(rules))
	}
}

func TestCodifier_ExtractFromOrchestrationFailures(t *testing.T) {
	dir := t.TempDir()
	ledger := filepath.Join(dir, "orch.jsonl")
	lines := []string{
		`{"ts":"2026-06-14T00:00:01Z","backend":"codex","subcommand":"companion-result","write":true,"exit_code":1,"duration_ms":10,"session_id":"task-a","counts":false}`,
		`{"ts":"2026-06-14T00:00:02Z","backend":"codex","subcommand":"companion-result","write":true,"exit_code":1,"duration_ms":11,"session_id":"task-b","counts":false}`,
		`{"ts":"2026-06-14T00:00:03Z","backend":"codex","subcommand":"companion-result","write":true,"exit_code":1,"duration_ms":12,"session_id":"task-c","counts":false}`,
	}
	if err := os.WriteFile(ledger, []byte(stringsJoin(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rules, err := ExtractFromLedger(ExtractOpts{
		RepoRoot:                dir,
		JudgmentLedgerPath:      filepath.Join(dir, "empty-judgment.jsonl"),
		OrchestrationLedgerPath: ledger,
		SchemaPath:              schemaPath(t),
	})
	if err != nil {
		t.Fatalf("ExtractFromLedger: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	if rules[0].Confidence != ConfidenceMedium {
		t.Fatalf("confidence = %q, want %q (3 occurrences)", rules[0].Confidence, ConfidenceMedium)
	}
	if rules[0].ProposedSSOTTarget != SSOTTargetPatterns {
		t.Fatalf("proposed_ssot_target = %q, want %q", rules[0].ProposedSSOTTarget, SSOTTargetPatterns)
	}
}

func TestCodifier_PromotionRequiresApproval(t *testing.T) {
	rule := Rule{
		RuleID:             "failure-test",
		Confidence:         ConfidenceLow,
		EvidenceRefs:       []string{"orchestration-ledger:test:codex"},
		ProposedSSOTTarget: SSOTTargetPatterns,
	}

	if err := AutoPromote(PromoteOpts{Rule: rule}); err == nil {
		t.Fatal("AutoPromote should return error")
	} else if !errors.Is(err, ErrAutoPromotionForbidden) {
		t.Fatalf("AutoPromote err = %v, want ErrAutoPromotionForbidden", err)
	}

	if err := Promote(PromoteOpts{Rule: rule, HumanApproved: false, DryRun: true}); err == nil {
		t.Fatal("Promote without human approval should return error")
	} else if !errors.Is(err, ErrAutoPromotionForbidden) {
		t.Fatalf("Promote err = %v, want ErrAutoPromotionForbidden", err)
	}

	if err := Promote(PromoteOpts{Rule: rule, HumanApproved: true, DryRun: true}); err == nil {
		t.Fatal("dry-run Promote should still return error (no auto-write)")
	} else if !errors.Is(err, ErrAutoPromotionForbidden) {
		t.Fatalf("dry-run Promote err = %v, want ErrAutoPromotionForbidden", err)
	}
}

func TestProposeDryRun_ReturnsJSONArray(t *testing.T) {
	dir := t.TempDir()
	data, err := ProposeDryRun(ProposeOpts{ExtractOpts: ExtractOpts{
		RepoRoot:                dir,
		JudgmentLedgerPath:      filepath.Join(dir, "missing-judgment.jsonl"),
		OrchestrationLedgerPath: filepath.Join(dir, "missing-orch.jsonl"),
		SchemaPath:              schemaPath(t),
	}})
	if err != nil {
		t.Fatalf("ProposeDryRun: %v", err)
	}
	if len(data) == 0 || data[0] != '[' {
		t.Fatalf("expected JSON array, got %q", string(data))
	}
}

func stringsJoin(lines []string, sep string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += sep
		}
		out += line
	}
	return out
}
