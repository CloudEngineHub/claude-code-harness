package failurecodifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	SchemaRelPath = "templates/schemas/failure-rule.v1.json"
	schemaURL     = "failure-rule.v1"
)

// Rule is one failure-rule.v1 candidate (proposal only — never auto-written to SSOT).
type Rule struct {
	RuleID             string   `json:"rule_id"`
	Confidence         string   `json:"confidence"`
	EvidenceRefs       []string `json:"evidence_refs"`
	ProposedSSOTTarget string   `json:"proposed_ssot_target"`
	Summary            string   `json:"summary,omitempty"`
	OccurrenceCount    int      `json:"occurrence_count,omitempty"`
}

// DefaultSchemaPath returns the repo-relative schema path joined to repoRoot.
func DefaultSchemaPath(repoRoot string) string {
	return filepath.Join(repoRoot, SchemaRelPath)
}

// ValidateRule checks one rule against failure-rule.v1 JSON Schema.
func ValidateRule(rule Rule, schemaPath string) error {
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return fmt.Errorf("parse schema json: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	payload, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("marshal rule: %w", err)
	}
	var instance any
	if err := json.Unmarshal(payload, &instance); err != nil {
		return fmt.Errorf("unmarshal rule json: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}

func validateRuleMap(instance map[string]interface{}, schemaPath string) error {
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return fmt.Errorf("parse schema json: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}
