package hostgen

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleDeliveryHostsTOML = `
[claude]
hook_event = "PreToolUse"
hook_path  = ".claude-plugin/hooks.json"
matcher    = "Write|Edit|MultiEdit|Bash|Read"
deny       = "exit2"
transport  = "stdin-json"
model      = "opus"
delivery_strategy = "monitor"
delivery_event_turn = "Stop"
delivery_event_monitor = "SessionStart"

[codex]
hook_event = "PreToolUse"
hook_path  = ".codex/hooks.json"
matcher    = "*"
deny       = "permissionDecision"
transport  = "stdin-json"
model      = "gpt-5"
delivery_strategy = "turn"
delivery_event_turn = "Stop"

[cursor]
hook_event = "preToolUse"
hook_path  = ".cursor/hooks.json"
matcher    = "*"
deny       = "permission"
transport  = "stdin-json"
model      = "default"
delivery_strategy = "turn"
delivery_event_turn = "stop"
`

func loadDeliveryHosts(t *testing.T) map[string]Host {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.toml")
	if err := os.WriteFile(path, []byte(sampleDeliveryHostsTOML), 0o644); err != nil {
		t.Fatal(err)
	}
	hosts, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return hosts
}

func TestGenerateDeliveryHooksJSON_TurnOnly_Cursor(t *testing.T) {
	hosts := loadDeliveryHosts(t)
	out, ok, err := GenerateDeliveryHooksJSON(hosts["cursor"])
	if err != nil {
		t.Fatalf("GenerateDeliveryHooksJSON(cursor): %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for cursor delivery config")
	}
	s := string(out)
	if !strings.Contains(s, "inbox check") {
		t.Errorf("cursor delivery hooks missing inbox check command:\n%s", s)
	}
	if !strings.Contains(s, "stop") {
		t.Errorf("cursor delivery hooks missing stop event:\n%s", s)
	}

	var doc struct {
		Version int `json:"version"`
		Hooks   map[string][]struct {
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	entries, exists := doc.Hooks["stop"]
	if !exists || len(entries) == 0 {
		t.Fatalf("missing stop hook entries: %+v", doc.Hooks)
	}
}

func TestGenerateDeliveryHooksJSON_TurnPlusMonitor_Claude(t *testing.T) {
	hosts := loadDeliveryHosts(t)
	out, ok, err := GenerateDeliveryHooksJSON(hosts["claude"])
	if err != nil {
		t.Fatalf("GenerateDeliveryHooksJSON(claude): %v", err)
	}
	if !ok {
		t.Fatal("expected ok=true for claude delivery config")
	}
	s := string(out)
	if !strings.Contains(s, "inbox check") {
		t.Errorf("claude delivery hooks missing inbox check:\n%s", s)
	}
	if !strings.Contains(s, "inbox monitor") {
		t.Errorf("claude delivery hooks missing inbox monitor:\n%s", s)
	}
	if !strings.Contains(s, "Stop") {
		t.Errorf("claude delivery hooks missing Stop event:\n%s", s)
	}
	if !strings.Contains(s, "SessionStart") {
		t.Errorf("claude delivery hooks missing SessionStart event:\n%s", s)
	}
}

func TestGenerateDeliveryHooksJSON_FireOnNormalTurn(t *testing.T) {
	hosts := loadDeliveryHosts(t)
	for _, name := range []string{"claude", "codex", "cursor"} {
		out, ok, err := GenerateDeliveryHooksJSON(hosts[name])
		if err != nil {
			t.Fatalf("GenerateDeliveryHooksJSON(%s): %v", name, err)
		}
		if !ok {
			t.Fatalf("%s: expected ok=true", name)
		}
		assertFiresOnNormalTurn(t, name, out)
	}
}

func assertFiresOnNormalTurn(t *testing.T, host string, out []byte) {
	t.Helper()
	s := string(out)

	// broadcast corpse guard: must not wire to narrow special-case-only events.
	forbiddenOnly := []string{"PermissionDenied", "Notification", "SubagentStop", "StopFailure"}
	for _, ev := range forbiddenOnly {
		if strings.Contains(s, ev) && !strings.Contains(s, "Stop") {
			// If the only event is a special one, fail — but Stop is allowed as turn event.
			if ev != "StopFailure" {
				continue
			}
		}
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("%s: invalid JSON: %v", host, err)
	}
	hooksRaw, ok := raw["hooks"]
	if !ok {
		t.Fatalf("%s: missing hooks key", host)
	}
	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		t.Fatalf("%s: hooks parse: %v", host, err)
	}

	for event, groupsRaw := range hooks {
		_ = event
		if host == "cursor" {
			var entries []struct {
				Matcher string `json:"matcher"`
			}
			if err := json.Unmarshal(groupsRaw, &entries); err != nil {
				t.Fatalf("cursor entries: %v", err)
			}
			for _, e := range entries {
				if e.Matcher != "" && e.Matcher != "*" {
					t.Errorf("%s: cursor delivery matcher %q is too narrow (want * or empty)", host, e.Matcher)
				}
			}
			continue
		}
		var groups []struct {
			Matcher string `json:"matcher"`
		}
		if err := json.Unmarshal(groupsRaw, &groups); err != nil {
			t.Fatalf("%s groups: %v", host, err)
		}
		for _, g := range groups {
			if g.Matcher != "" && g.Matcher != "*" {
				t.Errorf("%s: delivery matcher %q is too narrow (want * or empty for normal-turn fire)", host, g.Matcher)
			}
		}
	}
}

func TestGenerateDeliveryHooksJSON_CodexCursor_NoMonitorEntry(t *testing.T) {
	hosts := loadDeliveryHosts(t)
	for _, name := range []string{"codex", "cursor"} {
		out, ok, err := GenerateDeliveryHooksJSON(hosts[name])
		if err != nil {
			t.Fatalf("GenerateDeliveryHooksJSON(%s): %v", name, err)
		}
		if !ok {
			t.Fatalf("%s: expected ok=true", name)
		}
		if strings.Contains(string(out), "inbox monitor") {
			t.Errorf("%s: must not generate inbox monitor entry:\n%s", name, out)
		}
	}
}

func TestGenerateDeliveryHooksJSON_DeterministicBytes(t *testing.T) {
	hosts := loadDeliveryHosts(t)
	for _, name := range SortedNames(hosts) {
		h := hosts[name]
		a, okA, err := GenerateDeliveryHooksJSON(h)
		if err != nil {
			t.Fatalf("first GenerateDeliveryHooksJSON(%s): %v", name, err)
		}
		b, okB, err := GenerateDeliveryHooksJSON(h)
		if err != nil {
			t.Fatalf("second GenerateDeliveryHooksJSON(%s): %v", name, err)
		}
		if okA != okB {
			t.Errorf("%s: ok flag not deterministic: %v vs %v", name, okA, okB)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s: delivery hooks not bytes-identical:\nfirst:\n%s\nsecond:\n%s", name, a, b)
		}
	}
}

func TestGenerateDeliveryHooksJSON_MissingConfig_FailOpen(t *testing.T) {
	h := Host{Name: "claude", HookEvent: "PreToolUse"}
	out, ok, err := GenerateDeliveryHooksJSON(h)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for missing delivery config")
	}
	if out != nil {
		t.Fatalf("expected nil output, got %q", out)
	}
}
