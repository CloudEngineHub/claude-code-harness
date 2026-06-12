package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

func parseObject(raw []byte) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("bridge: empty input")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]interface{}
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("bridge: invalid json: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("bridge: empty object")
	}
	return m, nil
}

func stringField(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return "", false
	}
	switch s := v.(type) {
	case string:
		return s, true
	default:
		return fmt.Sprint(v), true
	}
}

func requireNanos(m map[string]interface{}, keys ...string) (int64, error) {
	for _, key := range keys {
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		n, err := toUnixNanos(v)
		if err != nil {
			return 0, fmt.Errorf("bridge: %s: %w", key, err)
		}
		if n <= 0 {
			return 0, fmt.Errorf("bridge: %s must be positive", key)
		}
		return n, nil
	}
	return 0, fmt.Errorf("bridge: timestamp required")
}

func toUnixNanos(v interface{}) (int64, error) {
	switch n := v.(type) {
	case float64:
		return scaleToNanos(int64(n))
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			f, err := n.Float64()
			if err != nil {
				return 0, err
			}
			return scaleToNanos(int64(f))
		}
		return scaleToNanos(i)
	case int64:
		return scaleToNanos(n)
	case int:
		return scaleToNanos(int64(n))
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0, err
		}
		return scaleToNanos(i)
	default:
		return 0, fmt.Errorf("unsupported timestamp type %T", v)
	}
}

func scaleToNanos(n int64) (int64, error) {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1_000_000_000_000_000_000:
		return n, nil
	case abs >= 1_000_000_000_000_000:
		return n * 1_000, nil
	case abs >= 1_000_000_000_000:
		return n * 1_000_000, nil
	case abs >= 1_000_000_000:
		return n * 1_000_000_000, nil
	default:
		return 0, fmt.Errorf("timestamp magnitude too small")
	}
}

func copyPayloadFields(dst map[string]interface{}, src map[string]interface{}, keys ...string) {
	for _, key := range keys {
		if v, ok := src[key]; ok {
			dst[key] = v
		}
	}
}

func mergeExtraFields(dst map[string]interface{}, src map[string]interface{}, reserved map[string]struct{}) {
	for key, val := range src {
		if _, skip := reserved[key]; skip {
			continue
		}
		if _, exists := dst[key]; exists {
			continue
		}
		dst[key] = val
	}
}
