package bridge

import (
	"testing"
)

func TestCursorAdapter_NormalizesStopHook(t *testing.T) {
	raw := loadFixture(t, "cursor-stop-hook.json")
	adapter := NewCursorStopHookAdapter()

	ev, err := adapter.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	want := Event{
		Source:    SourceCursor,
		EventType: "stop",
		Payload: map[string]interface{}{
			"conversation_id": "conv-cursor-001",
			"session_id":        "sess-abc",
			"message":           "task complete",
		},
		TS: 1718000000000000001,
	}
	if !eventsEqual(ev, want) {
		t.Errorf("got %+v, want %+v", ev, want)
	}
	if adapter.Source() != SourceCursor {
		t.Errorf("Source() = %q, want cursor", adapter.Source())
	}
}

func TestCursorAdapter_FailLoud_MissingTimestamp(t *testing.T) {
	raw := []byte(`{"conversation_id":"c1","hook_event_name":"stop","session_id":"s1"}`)
	_, err := NewCursorStopHookAdapter().Normalize(raw)
	if err == nil {
		t.Fatal("expected error when ts is missing")
	}
}

func TestCursorAdapter_FailLoud_MissingEventType(t *testing.T) {
	raw := []byte(`{"conversation_id":"c1","ts":1718000000000000001}`)
	_, err := NewCursorStopHookAdapter().Normalize(raw)
	if err == nil {
		t.Fatal("expected error when hook_event_name is missing")
	}
}
