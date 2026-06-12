package bridgedelivery

import (
	"context"
	"encoding/json"
	"fmt"
)

type cursorDeliverer struct {
	stopFollowupFn func(ctx context.Context, payload []byte) error
}

func NewCursorDeliverer(stopFollowupFn func(ctx context.Context, payload []byte) error) Deliverer {
	return &cursorDeliverer{stopFollowupFn: stopFollowupFn}
}

func (d *cursorDeliverer) Target() Target { return TargetCursor }

func (d *cursorDeliverer) Deliver(ctx context.Context, n Notice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.stopFollowupFn == nil {
		return fmt.Errorf("cursor deliverer not configured")
	}

	payload, err := json.Marshal(map[string]interface{}{
		"type": "stop",
		"followup_message": map[string]string{
			"role":    "assistant",
			"content": n.Body,
		},
		"conversation_id": n.ToAgent,
		"ts":              n.TS,
	})
	if err != nil {
		return fmt.Errorf("cursor deliverer: marshal payload: %w", err)
	}

	if err := d.stopFollowupFn(ctx, payload); err != nil {
		return err
	}
	return nil
}
