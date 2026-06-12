package bridge

import (
	"reflect"
	"testing"
)

func TestCCAdapter_NormalizesFixtureInput(t *testing.T) {
	raw := loadFixture(t, "cc-tool-use.json")
	adapter := NewCCMailboxAdapter()

	ev, err := adapter.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	want := Event{
		Source:    SourceCC,
		EventType: "PostToolUse",
		Payload: map[string]interface{}{
			"conversation_id": "conv-cc-001",
			"tool_name":       "Bash",
			"tool_input":      map[string]interface{}{"command": "go test ./..."},
		},
		TS: 1718000000000000000,
	}
	if !eventsEqual(ev, want) {
		t.Errorf("got %+v, want %+v", ev, want)
	}
	if adapter.Source() != SourceCC {
		t.Errorf("Source() = %q, want cc", adapter.Source())
	}
}

func TestCCAdapter_FailLoud_MissingTimestamp(t *testing.T) {
	raw := []byte(`{"conversation_id":"c1","hook_event_name":"PostToolUse","tool_name":"Bash"}`)
	_, err := NewCCMailboxAdapter().Normalize(raw)
	if err == nil {
		t.Fatal("expected error when timestamp is missing")
	}
}

func TestCCAdapter_FailLoud_MissingEventType(t *testing.T) {
	raw := []byte(`{"conversation_id":"c1","tool_name":"Bash","timestamp":1718000000000000000}`)
	_, err := NewCCMailboxAdapter().Normalize(raw)
	if err == nil {
		t.Fatal("expected error when hook_event_name is missing")
	}
}

func eventsEqual(a, b Event) bool {
	if a.Source != b.Source || a.EventType != b.EventType || a.TS != b.TS {
		return false
	}
	return reflect.DeepEqual(a.Payload, b.Payload)
}
