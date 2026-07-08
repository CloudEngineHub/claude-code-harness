package retiredalias

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// GlobalAllowlist paths are excluded for every entry (prefix match).
var GlobalAllowlist = []string{
	"CHANGELOG.md",
	".claude/memory/archive/",
	".claude/worktrees/",
	".claude/state/",
	"out/",
	"output/",
	"benchmarks/",
	"templates/registry/retired-aliases.v1.yaml",
	"templates/schemas/retired-alias.v1.json",
	".claude/rules/retired-alias-policy.md",
	".claude/rules/migration-policy.md",
	"go/internal/retiredalias/",
	"tests/fixtures/retired-alias/",
	"TASK-97.md",
}

// ParseRegistryYAML unmarshals registry YAML bytes.
func ParseRegistryYAML(data []byte) (*Registry, error) {
	var reg Registry
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry yaml: %w", err)
	}
	return &reg, nil
}

// ValidateRegistrySchema validates registry YAML against retired-alias.v1 JSON Schema.
func ValidateRegistrySchema(registryPath, schemaPath string) error {
	registryData, err := os.ReadFile(registryPath)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	var registryDoc any
	if err := yaml.Unmarshal(registryData, &registryDoc); err != nil {
		return fmt.Errorf("yaml to document: %w", err)
	}
	registryJSON, err := json.Marshal(registryDoc)
	if err != nil {
		return fmt.Errorf("marshal registry document: %w", err)
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return fmt.Errorf("parse schema json: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	const schemaURL = "retired-alias.v1"
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	var instance any
	if err := json.Unmarshal(registryJSON, &instance); err != nil {
		return fmt.Errorf("unmarshal registry json: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}
