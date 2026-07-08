package nightwatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaURL = "night-watch-report.v1"

// Report is the night-watch-report.v1 payload.
type Report struct {
	SchemaVersion   string           `json:"schema_version"`
	GeneratedAt     string           `json:"generated_at"`
	DryRun          bool             `json:"dry_run,omitempty"`
	Health          ReportHealth     `json:"health"`
	UnresolvedLoops []UnresolvedLoop `json:"unresolved_loops"`
	StaleTasks      []StaleTask      `json:"stale_tasks"`
	OpenDecisions   []OpenDecision   `json:"open_decisions"`
}

type ReportHealth struct {
	Healthy bool   `json:"healthy"`
	Reason  string `json:"reason"`
}

type UnresolvedLoop struct {
	EventID   string  `json:"event_id"`
	EventType string  `json:"event_type"`
	Source    string  `json:"source"`
	TaskID    string  `json:"task_id,omitempty"`
	AgeHours  float64 `json:"age_hours"`
}

type StaleTask struct {
	TaskID   string  `json:"task_id"`
	Status   string  `json:"status"`
	AgeHours float64 `json:"age_hours"`
}

type OpenDecision struct {
	DecisionID string  `json:"decision_id"`
	Title      string  `json:"title"`
	AgeHours   float64 `json:"age_hours"`
}

// DefaultSchemaPath returns the repo-relative schema path joined to repoRoot.
func DefaultSchemaPath(repoRoot string) string {
	return filepath.Join(repoRoot, SchemaRelPath)
}

// validateReportMap validates a decoded JSON object against the schema file.
func validateReportMap(instance map[string]interface{}, schemaPath string) error {
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

// ValidateReport checks one report against night-watch-report.v1 JSON Schema.
func ValidateReport(report Report, schemaPath string) error {
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

	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	var instance any
	if err := json.Unmarshal(payload, &instance); err != nil {
		return fmt.Errorf("unmarshal report json: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}
