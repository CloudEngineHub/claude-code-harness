package bridge

import "fmt"

type codexAppServerAdapter struct{}

func NewCodexAppServerAdapter() Adapter { return &codexAppServerAdapter{} }

func (a *codexAppServerAdapter) Source() Source { return SourceCodex }

func (a *codexAppServerAdapter) Normalize(raw []byte) (Event, error) {
	m, err := parseObject(raw)
	if err != nil {
		return Event{}, fmt.Errorf("bridge/codex: %w", err)
	}

	eventType, ok := stringField(m, "type")
	if !ok || eventType == "" {
		return Event{}, fmt.Errorf("bridge/codex: type required")
	}

	ts, err := requireNanos(m, "ts")
	if err != nil {
		return Event{}, fmt.Errorf("bridge/codex: %w", err)
	}

	payload := make(map[string]interface{})
	copyPayloadFields(payload, m, "thread_id", "payload")
	reserved := map[string]struct{}{
		"type": {},
		"ts":   {},
	}
	mergeExtraFields(payload, m, reserved)

	return Event{
		Source:    SourceCodex,
		EventType: eventType,
		Payload:   payload,
		TS:        ts,
	}, nil
}
