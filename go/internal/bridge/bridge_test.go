package bridge

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSchema_IsValidDraft07(t *testing.T) {
	data, err := os.ReadFile(schemaPath(t))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	if got := doc["$schema"]; got != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("$schema = %v, want draft-07", got)
	}

	required, ok := doc["required"].([]interface{})
	if !ok {
		t.Fatal("required field missing or not an array")
	}
	wantRequired := []string{"source", "event_type", "payload", "ts"}
	if len(required) != len(wantRequired) {
		t.Fatalf("required length = %d, want %d", len(required), len(wantRequired))
	}
	for i, w := range wantRequired {
		if required[i] != w {
			t.Errorf("required[%d] = %v, want %v", i, required[i], w)
		}
	}

	props, ok := doc["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties missing")
	}
	source, ok := props["source"].(map[string]interface{})
	if !ok {
		t.Fatal("properties.source missing")
	}
	enum, ok := source["enum"].([]interface{})
	if !ok {
		t.Fatal("source.enum missing")
	}
	wantEnum := []string{"cc", "cursor", "codex"}
	if len(enum) != len(wantEnum) {
		t.Fatalf("source enum length = %d, want %d", len(enum), len(wantEnum))
	}
	for i, w := range wantEnum {
		if enum[i] != w {
			t.Errorf("source enum[%d] = %v, want %v", i, enum[i], w)
		}
	}

	if doc["additionalProperties"] != false {
		t.Errorf("additionalProperties = %v, want false", doc["additionalProperties"])
	}
}

func TestRegistry_UnregisteredSource_FailOpenSkip(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewCCMailboxAdapter())
	reg.Register(NewCursorStopHookAdapter())
	reg.Register(NewCodexAppServerAdapter())

	ev, ok, err := reg.Normalize(Source("unknown"), []byte(`{"timestamp":1}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for unregistered source")
	}
	if !reflect.DeepEqual(ev, Event{}) {
		t.Errorf("expected zero Event, got %+v", ev)
	}
}

func TestRegistry_StdoutQuiet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewCCMailboxAdapter())
	reg.Register(NewCursorStopHookAdapter())
	reg.Register(NewCodexAppServerAdapter())

	raw := loadFixture(t, "cc-tool-use.json")

	stdoutPath := filepath.Join(t.TempDir(), "stdout.txt")
	stderrPath := filepath.Join(t.TempDir(), "stderr.txt")
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stdoutFile.Close()
		stderrFile.Close()
	})

	oldOut := os.Stdout
	oldErr := os.Stderr
	os.Stdout = stdoutFile
	os.Stderr = stderrFile
	t.Cleanup(func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	})

	if _, _, err := reg.Normalize(SourceCC, raw); err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	_, ok, err := reg.Normalize(Source("unknown"), raw)
	if err != nil {
		t.Fatalf("unregistered Normalize: %v", err)
	}
	if ok {
		t.Fatal("expected skip for unknown source")
	}

	os.Stdout = oldOut
	os.Stderr = oldErr
	stdoutFile.Close()
	stderrFile.Close()

	if data, _ := os.ReadFile(stdoutPath); len(data) > 0 {
		t.Errorf("stdout not quiet: %q", data)
	}
	if data, _ := os.ReadFile(stderrPath); len(data) > 0 {
		t.Errorf("stderr not quiet: %q", data)
	}
}

func TestEvent_RoundTripsThroughSchema(t *testing.T) {
	schema := schemaPath(t)
	cases := []struct {
		name string
		ev   Event
	}{
		{
			name: "cc",
			ev: Event{
				Source:    SourceCC,
				EventType: "PostToolUse",
				Payload: map[string]interface{}{
					"conversation_id": "conv-cc-001",
					"tool_name":       "Bash",
					"tool_input":      map[string]interface{}{"command": "go test ./..."},
				},
				TS: 1718000000000000000,
			},
		},
		{
			name: "cursor",
			ev: Event{
				Source:    SourceCursor,
				EventType: "stop",
				Payload: map[string]interface{}{
					"conversation_id": "conv-cursor-001",
					"session_id":      "sess-abc",
					"message":         "task complete",
				},
				TS: 1718000000000000001,
			},
		},
		{
			name: "codex",
			ev: Event{
				Source:    SourceCodex,
				EventType: "thread.started",
				Payload: map[string]interface{}{
					"thread_id": "thread-xyz",
					"payload":   map[string]interface{}{"model": "gpt-5"},
				},
				TS: 1718000000000000002,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.ev)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var got map[string]interface{}
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal wire: %v", err)
			}

			want := map[string]interface{}{
				"source":     string(tc.ev.Source),
				"event_type": tc.ev.EventType,
				"payload":    tc.ev.Payload,
				"ts":         float64(tc.ev.TS),
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("wire shape mismatch:\n  got  %#v\n  want %#v", got, want)
			}

			cmd := exec.Command("python3", "-c", `
import json, sys
from jsonschema import validate
schema = json.load(open(sys.argv[1]))
instance = json.loads(sys.argv[2])
validate(instance=instance, schema=schema)
`, schema, string(b))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("schema validate failed: %v\n%s", err, out)
			}
		})
	}
}

func TestEvent_RoundTripsThroughSchema_AdapterOutputs(t *testing.T) {
	schema := schemaPath(t)
	adapters := []struct {
		name    string
		adapter Adapter
		fixture string
	}{
		{"cc", NewCCMailboxAdapter(), "cc-tool-use.json"},
		{"cursor", NewCursorStopHookAdapter(), "cursor-stop-hook.json"},
		{"codex", NewCodexAppServerAdapter(), "codex-app-server-event.json"},
	}

	for _, tc := range adapters {
		t.Run(tc.name, func(t *testing.T) {
			raw := loadFixture(t, tc.fixture)
			ev, err := tc.adapter.Normalize(raw)
			if err != nil {
				t.Fatalf("Normalize: %v", err)
			}
			b, err := json.Marshal(ev)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			cmd := exec.Command("python3", "-c", `
import json, sys
from jsonschema import validate
schema = json.load(open(sys.argv[1]))
instance = json.loads(sys.argv[2])
validate(instance=instance, schema=schema)
`, schema, string(b))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("schema validate failed: %v\n%s", err, out)
			}
		})
	}
}
