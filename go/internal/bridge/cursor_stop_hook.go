package bridge

import "fmt"

type cursorStopHookAdapter struct{}

func NewCursorStopHookAdapter() Adapter { return &cursorStopHookAdapter{} }

func (a *cursorStopHookAdapter) Source() Source { return SourceCursor }

func (a *cursorStopHookAdapter) Normalize(raw []byte) (Event, error) {
	m, err := parseObject(raw)
	if err != nil {
		return Event{}, fmt.Errorf("bridge/cursor: %w", err)
	}

	hookEventName, ok := stringField(m, "hook_event_name")
	if !ok || hookEventName == "" {
		return Event{}, fmt.Errorf("bridge/cursor: hook_event_name required")
	}

	ts, err := requireNanos(m, "ts")
	if err != nil {
		return Event{}, fmt.Errorf("bridge/cursor: %w", err)
	}

	payload := make(map[string]interface{})
	copyPayloadFields(payload, m, "conversation_id", "session_id", "message")
	reserved := map[string]struct{}{
		"hook_event_name": {},
		"ts":              {},
	}
	mergeExtraFields(payload, m, reserved)

	return Event{
		Source:    SourceCursor,
		EventType: "stop",
		Payload:   payload,
		TS:        ts,
	}, nil
}
