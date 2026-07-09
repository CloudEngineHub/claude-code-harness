package failurecodifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, SchemaRelPath)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", wd)
		}
		dir = parent
	}
}

func schemaPath(t *testing.T) string {
	t.Helper()
	return DefaultSchemaPath(repoRoot(t))
}

func TestSchema_FailureRuleV1_RejectsMissingRuleId(t *testing.T) {
	raw := map[string]interface{}{
		"confidence":           ConfidenceMedium,
		"evidence_refs":        []interface{}{"orchestration-ledger:2026-06-14T00:00:00Z:codex"},
		"proposed_ssot_target": SSOTTargetPatterns,
	}
	if err := validateRuleMap(raw, schemaPath(t)); err == nil {
		t.Fatal("expected schema reject for missing rule_id")
	}
}

func TestSchema_FailureRuleV1_AdditionalPropertiesFalse(t *testing.T) {
	data, err := os.ReadFile(schemaPath(t))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}
	id, _ := schema["$id"].(string)
	if id == "" || !strings.Contains(id, "failure-rule.v1") {
		t.Fatalf("$id = %q, want failure-rule.v1", id)
	}
	if schema["additionalProperties"] != false {
		t.Fatalf("additionalProperties = %v, want false", schema["additionalProperties"])
	}
}

func TestSchema_FailureRuleV1_ValidRule(t *testing.T) {
	rule := Rule{
		RuleID:             "failure-orch-codex-companion-result-exit-code-1",
		Confidence:         ConfidenceHigh,
		EvidenceRefs:       []string{"orchestration-ledger:2026-06-14T00:00:00Z:codex"},
		ProposedSSOTTarget: SSOTTargetPatterns,
	}
	if err := ValidateRule(rule, schemaPath(t)); err != nil {
		t.Fatalf("valid rule rejected: %v", err)
	}
}
