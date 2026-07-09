package clientmirror

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ValidateState validates a mirror-state.v1 payload against the JSON Schema.
func ValidateState(state State, schemaPath string) error {
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return fmt.Errorf("parse schema json: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(SchemaURL, schemaDoc); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	schema, err := compiler.Compile(SchemaURL)
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	var instance any
	if err := json.Unmarshal(payload, &instance); err != nil {
		return fmt.Errorf("unmarshal state json: %w", err)
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}
