package bridgedelivery_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/bridgedelivery"
)

func TestCursorDeliverer_PayloadShape(t *testing.T) {
	var gotPayload []byte
	del := bridgedelivery.NewCursorDeliverer(func(_ context.Context, payload []byte) error {
		gotPayload = append([]byte(nil), payload...)
		return nil
	})

	n := sampleNotice()
	n.Body = "deliver this body"
	n.ToAgent = "conv-cursor-001"

	if err := del.Deliver(context.Background(), n); err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if len(gotPayload) == 0 {
		t.Fatal("expected payload bytes")
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(gotPayload, &doc); err != nil {
		t.Fatalf("invalid JSON payload: %v\n%s", err, gotPayload)
	}
	if doc["type"] != "stop" {
		t.Fatalf("type = %v, want stop", doc["type"])
	}

	followup, ok := doc["followup_message"].(map[string]interface{})
	if !ok {
		t.Fatalf("followup_message missing or wrong type: %v", doc["followup_message"])
	}
	if followup["role"] != "assistant" {
		t.Fatalf("followup role = %v, want assistant", followup["role"])
	}
	if followup["content"] != n.Body {
		t.Fatalf("followup content = %v, want %q", followup["content"], n.Body)
	}
	if doc["conversation_id"] != n.ToAgent {
		t.Fatalf("conversation_id = %v, want %q", doc["conversation_id"], n.ToAgent)
	}
}
