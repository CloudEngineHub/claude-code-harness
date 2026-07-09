package channelswake

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestChannelWakeEventSchema_AdditionalPropertiesFalse(t *testing.T) {
	schemaPath := schemaPath(t)
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}

	id, _ := schema["$id"].(string)
	if id == "" || !strings.Contains(id, "channel-wake-event.v1") {
		t.Fatalf("$id = %q, want channel-wake-event.v1", id)
	}
	if schema["additionalProperties"] != false {
		t.Fatalf("additionalProperties = %v, want false", schema["additionalProperties"])
	}
}

func schemaPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	return filepath.Join(repoRoot, "templates", "schemas", "channel-wake-event.v1.json")
}
