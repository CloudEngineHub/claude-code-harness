package bridge

import (
	"testing"
)

func TestCodexAdapter_NormalizesAppServerEvent(t *testing.T) {
	raw := loadFixture(t, "codex-app-server-event.json")
	adapter := NewCodexAppServerAdapter()

	ev, err := adapter.Normalize(raw)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	want := Event{
		Source:    SourceCodex,
		EventType: "thread.started",
		Payload: map[string]interface{}{
			"thread_id": "thread-xyz",
			"payload":   map[string]interface{}{"model": "gpt-5"},
		},
		TS: 1718000000000000002,
	}
	if !eventsEqual(ev, want) {
		t.Errorf("got %+v, want %+v", ev, want)
	}
	if adapter.Source() != SourceCodex {
		t.Errorf("Source() = %q, want codex", adapter.Source())
	}
}

func TestCodexAdapter_FailLoud_MissingTimestamp(t *testing.T) {
	raw := []byte(`{"type":"thread.started","thread_id":"t1"}`)
	_, err := NewCodexAppServerAdapter().Normalize(raw)
	if err == nil {
		t.Fatal("expected error when ts is missing")
	}
}

func TestCodexAdapter_FailLoud_MissingEventType(t *testing.T) {
	raw := []byte(`{"thread_id":"t1","ts":1718000000000000002}`)
	_, err := NewCodexAppServerAdapter().Normalize(raw)
	if err == nil {
		t.Fatal("expected error when type is missing")
	}
}
