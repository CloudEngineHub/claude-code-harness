package hostgen

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleHostsTOML = `
[claude]
hook_event = "PreToolUse"
hook_path  = ".claude-plugin/hooks.json"
matcher    = "Write|Edit|MultiEdit|Bash|Read"
deny       = "exit2"
transport  = "stdin-json"
model      = "opus"

[codex]
hook_event = "PreToolUse"
hook_path  = ".codex/hooks.json"
matcher    = "*"
deny       = "permissionDecision"
transport  = "stdin-json"
model      = "gpt-5"

[cursor]
hook_event = "preToolUse"
hook_path  = ".cursor/hooks.json"
matcher    = "*"
deny       = "permission"
transport  = "stdin-json"
model      = "default"
`

func writeSampleHosts(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hosts.toml")
	if err := os.WriteFile(path, []byte(sampleHostsTOML), 0o644); err != nil {
		t.Fatalf("write sample hosts.toml: %v", err)
	}
	return path
}

func TestLoad_ParsesThreeHosts(t *testing.T) {
	hosts, err := Load(writeSampleHosts(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(hosts) != 3 {
		t.Fatalf("expected 3 hosts, got %d (%v)", len(hosts), SortedNames(hosts))
	}

	claude, ok := hosts["claude"]
	if !ok {
		t.Fatal("missing claude host")
	}
	if claude.Name != "claude" {
		t.Errorf("claude.Name = %q, want %q", claude.Name, "claude")
	}
	if claude.HookEvent != "PreToolUse" {
		t.Errorf("claude.HookEvent = %q, want PreToolUse", claude.HookEvent)
	}
	if claude.HookPath != ".claude-plugin/hooks.json" {
		t.Errorf("claude.HookPath = %q", claude.HookPath)
	}

	codex := hosts["codex"]
	if codex.HookEvent != "PreToolUse" || codex.Deny != "permissionDecision" {
		t.Errorf("codex parsed wrong: event=%q deny=%q", codex.HookEvent, codex.Deny)
	}

	cursor := hosts["cursor"]
	if cursor.HookEvent != "preToolUse" {
		t.Errorf("cursor.HookEvent = %q, want preToolUse (lowercase p)", cursor.HookEvent)
	}
	if cursor.Deny != "permission" {
		t.Errorf("cursor.Deny = %q, want permission", cursor.Deny)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.toml")); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.toml")
	if err := os.WriteFile(path, []byte("# only a comment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for empty hosts.toml, got nil")
	}
}

func TestGenerateHooksJSON_CodexValidAndContainsPreTool(t *testing.T) {
	hosts, err := Load(writeSampleHosts(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := GenerateHooksJSON(hosts["codex"])
	if err != nil {
		t.Fatalf("GenerateHooksJSON(codex): %v", err)
	}

	if !strings.Contains(string(out), "pre-tool") {
		t.Errorf("codex hooks.json does not contain 'pre-tool':\n%s", out)
	}
	if !bytes.HasSuffix(out, []byte("\n")) {
		t.Error("codex hooks.json should end with a trailing newline")
	}

	// Round-trips as valid JSON and carries the documented Codex shape:
	// hooks.PreToolUse[].hooks[].command.
	var doc struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("codex hooks.json is not valid JSON: %v\n%s", err, out)
	}
	groups, ok := doc.Hooks["PreToolUse"]
	if !ok || len(groups) == 0 {
		t.Fatalf("codex hooks.json missing PreToolUse event: %+v", doc.Hooks)
	}
	if len(groups[0].Hooks) == 0 || !strings.Contains(groups[0].Hooks[0].Command, "hook pre-tool") {
		t.Errorf("codex command does not invoke 'hook pre-tool': %+v", groups[0].Hooks)
	}
}

func TestGenerateHooksJSON_CursorValidAndContainsPreTool(t *testing.T) {
	hosts, err := Load(writeSampleHosts(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := GenerateHooksJSON(hosts["cursor"])
	if err != nil {
		t.Fatalf("GenerateHooksJSON(cursor): %v", err)
	}

	if !strings.Contains(string(out), "pre-tool") {
		t.Errorf("cursor hooks.json does not contain 'pre-tool':\n%s", out)
	}

	// Cursor's documented shape: top-level version + flat event arrays of
	// {command,...}.
	var doc struct {
		Version int `json:"version"`
		Hooks   map[string][]struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("cursor hooks.json is not valid JSON: %v\n%s", err, out)
	}
	if doc.Version != 1 {
		t.Errorf("cursor hooks.json version = %d, want 1", doc.Version)
	}
	entries, ok := doc.Hooks["preToolUse"]
	if !ok || len(entries) == 0 {
		t.Fatalf("cursor hooks.json missing preToolUse event: %+v", doc.Hooks)
	}
	if !strings.Contains(entries[0].Command, "hook pre-tool") {
		t.Errorf("cursor command does not invoke 'hook pre-tool': %q", entries[0].Command)
	}
}

func TestGenerateHooksJSON_Deterministic(t *testing.T) {
	hosts, err := Load(writeSampleHosts(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range SortedNames(hosts) {
		h := hosts[name]
		a, err := GenerateHooksJSON(h)
		if err != nil {
			t.Fatalf("GenerateHooksJSON(%s) first call: %v", name, err)
		}
		b, err := GenerateHooksJSON(h)
		if err != nil {
			t.Fatalf("GenerateHooksJSON(%s) second call: %v", name, err)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("GenerateHooksJSON(%s) not deterministic:\nfirst:\n%s\nsecond:\n%s", name, a, b)
		}
	}
}

func TestGenerateHooksJSON_ClaudeUsesValidRootWrapper(t *testing.T) {
	hosts, err := Load(writeSampleHosts(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := GenerateHooksJSON(hosts["claude"])
	if err != nil {
		t.Fatalf("GenerateHooksJSON(claude): %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "valid_root") {
		t.Errorf("claude hooks.json should reuse the valid_root bootstrap wrapper:\n%s", s)
	}
	if !strings.Contains(s, "hook pre-tool") {
		t.Errorf("claude hooks.json should invoke 'hook pre-tool':\n%s", s)
	}
	var anyDoc map[string]interface{}
	if err := json.Unmarshal(out, &anyDoc); err != nil {
		t.Fatalf("claude hooks.json is not valid JSON: %v", err)
	}
}

func TestGenerateHooksJSON_UnknownHost(t *testing.T) {
	if _, err := GenerateHooksJSON(Host{Name: "antigravity", HookEvent: "preToolUse"}); err == nil {
		t.Fatal("expected error for unknown host, got nil")
	}
}

// TestGenerateHooksJSON_HostFlag verifies Phase 91.4 wiring: codex and cursor
// pass an explicit `--host <name>` so the codec renders that host's native deny
// shape, while claude (invoked via its valid_root wrapper) carries no --host
// flag (the codec treats the empty host as the Claude default).
func TestGenerateHooksJSON_HostFlag(t *testing.T) {
	hosts, err := Load(writeSampleHosts(t))
	if err != nil {
		t.Fatal(err)
	}

	codex, err := GenerateHooksJSON(hosts["codex"])
	if err != nil {
		t.Fatalf("GenerateHooksJSON(codex): %v", err)
	}
	if !strings.Contains(string(codex), "hook pre-tool --host codex") {
		t.Errorf("codex command should pass --host codex:\n%s", codex)
	}

	cursor, err := GenerateHooksJSON(hosts["cursor"])
	if err != nil {
		t.Fatalf("GenerateHooksJSON(cursor): %v", err)
	}
	if !strings.Contains(string(cursor), "hook pre-tool --host cursor") {
		t.Errorf("cursor command should pass --host cursor:\n%s", cursor)
	}

	claude, err := GenerateHooksJSON(hosts["claude"])
	if err != nil {
		t.Fatalf("GenerateHooksJSON(claude): %v", err)
	}
	if strings.Contains(string(claude), "--host") {
		t.Errorf("claude command must NOT carry a --host flag (codec default):\n%s", claude)
	}
}
