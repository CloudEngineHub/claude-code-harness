package bridge

import "fmt"

type ccMailboxAdapter struct{}

func NewCCMailboxAdapter() Adapter { return &ccMailboxAdapter{} }

func (a *ccMailboxAdapter) Source() Source { return SourceCC }

func (a *ccMailboxAdapter) Normalize(raw []byte) (Event, error) {
	m, err := parseObject(raw)
	if err != nil {
		return Event{}, fmt.Errorf("bridge/cc: %w", err)
	}

	eventType, ok := stringField(m, "hook_event_name")
	if !ok || eventType == "" {
		return Event{}, fmt.Errorf("bridge/cc: hook_event_name required")
	}

	ts, err := requireNanos(m, "timestamp")
	if err != nil {
		return Event{}, fmt.Errorf("bridge/cc: %w", err)
	}

	payload := make(map[string]interface{})
	copyPayloadFields(payload, m, "conversation_id", "tool_name", "tool_input")
	reserved := map[string]struct{}{
		"hook_event_name": {},
		"timestamp":       {},
	}
	mergeExtraFields(payload, m, reserved)

	return Event{
		Source:    SourceCC,
		EventType: eventType,
		Payload:   payload,
		TS:        ts,
	}, nil
}
