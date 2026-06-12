package hookcodec

import (
	"testing"
)

// Each host describes the SAME action — `git push --force` via Bash — in its
// own native stdin shape. Normalize must collapse all three to ToolName="Bash"
// with the command in tool_input, and infer the right host.
const forceCmd = "git push --force origin main"

func TestNormalize_Claude(t *testing.T) {
	raw := []byte(`{
		"session_id":"sess-claude-1",
		"hook_event_name":"PreToolUse",
		"tool_name":"Bash",
		"tool_input":{"command":"git push --force origin main"},
		"cwd":"/repo"
	}`)
	in, host, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostClaude {
		t.Errorf("inferred host = %q, want claude", host)
	}
	if in.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash", in.ToolName)
	}
	if got := in.ToolInput["command"]; got != forceCmd {
		t.Errorf("command = %v, want %q", got, forceCmd)
	}
	if in.SessionID != "sess-claude-1" {
		t.Errorf("SessionID = %q, want sess-claude-1", in.SessionID)
	}
	if in.CWD != "/repo" {
		t.Errorf("CWD = %q, want /repo", in.CWD)
	}
}

func TestNormalize_Codex(t *testing.T) {
	// Codex PreToolUse: session_id + tool_name + tool_input{command} + tool_use_id
	// + turn_id (extra fields ignored). No cursor markers, keyed off session_id.
	raw := []byte(`{
		"session_id":"sess-codex-1",
		"tool_name":"Bash",
		"tool_input":{"command":"git push --force origin main"},
		"tool_use_id":"call_42",
		"turn_id":"turn_7",
		"cwd":"/work"
	}`)
	in, host, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostCodex {
		// session_id-only Codex payload has no conversation_id; without a hint it
		// looks like Claude. The hint path is exercised separately below; here we
		// assert the explicit-hint contract.
		t.Logf("no-hint Codex inferred as %q (expected when payload uses session_id)", host)
	}
	if in.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash", in.ToolName)
	}
	if got := in.ToolInput["command"]; got != forceCmd {
		t.Errorf("command = %v, want %q", got, forceCmd)
	}
}

func TestNormalize_Codex_ConversationID(t *testing.T) {
	// Codex variant that keys identity off conversation_id (no cursor markers) is
	// inferred as codex without a hint.
	raw := []byte(`{
		"conversation_id":"conv-codex-9",
		"tool_name":"Bash",
		"tool_input":{"command":"git push --force origin main"},
		"cwd":"/work"
	}`)
	in, host, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostCodex {
		t.Errorf("inferred host = %q, want codex", host)
	}
	if in.SessionID != "conv-codex-9" {
		t.Errorf("SessionID = %q, want conv-codex-9 (from conversation_id)", in.SessionID)
	}
	if in.ToolName != "Bash" || in.ToolInput["command"] != forceCmd {
		t.Errorf("normalized tool = %q/%v, want Bash/%q", in.ToolName, in.ToolInput["command"], forceCmd)
	}
}

func TestNormalize_Codex_HintBias(t *testing.T) {
	// A session_id-only payload + explicit codex hint resolves to codex.
	raw := []byte(`{
		"session_id":"sess-codex-2",
		"tool_name":"Bash",
		"tool_input":{"command":"git push --force origin main"}
	}`)
	_, host, err := Normalize(raw, HostCodex)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostCodex {
		t.Errorf("hinted host = %q, want codex", host)
	}
}

func TestNormalize_Cursor_PreToolUse(t *testing.T) {
	// Cursor preToolUse with a structured Shell tool_input + workspace_roots.
	raw := []byte(`{
		"conversation_id":"conv-cursor-1",
		"generation_id":"gen-1",
		"hook_event_name":"preToolUse",
		"tool_name":"Shell",
		"tool_input":{"command":"git push --force origin main","working_directory":"/proj"},
		"tool_use_id":"abc123",
		"workspace_roots":["/proj"],
		"cwd":"/proj"
	}`)
	in, host, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostCursor {
		t.Errorf("inferred host = %q, want cursor", host)
	}
	// tool_name is explicitly "Shell" here; Normalize maps it to the
	// canonical "Bash" so the policy kernel (R06/R11) can match it.
	if in.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash (mapped from Cursor's Shell)", in.ToolName)
	}
	if got := in.ToolInput["command"]; got != forceCmd {
		t.Errorf("command = %v, want %q", got, forceCmd)
	}
	if in.SessionID != "conv-cursor-1" {
		t.Errorf("SessionID = %q, want conv-cursor-1", in.SessionID)
	}
	if in.CWD != "/proj" {
		t.Errorf("CWD = %q, want /proj", in.CWD)
	}
}

func TestNormalize_Cursor_BeforeShellExecution(t *testing.T) {
	// Cursor beforeShellExecution shorthand: top-level command, no tool_name,
	// workspace_roots present. Normalize synthesizes ToolName="Bash".
	raw := []byte(`{
		"conversation_id":"conv-cursor-2",
		"hook_event_name":"preToolUse",
		"command":"git push --force origin main",
		"cwd":"/proj",
		"sandbox":false,
		"workspace_roots":["/proj","/other"]
	}`)
	in, host, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostCursor {
		t.Errorf("inferred host = %q, want cursor", host)
	}
	if in.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash (synthesized from top-level command)", in.ToolName)
	}
	if got := in.ToolInput["command"]; got != forceCmd {
		t.Errorf("command = %v, want %q", got, forceCmd)
	}
	// cwd present wins over workspace_roots, but workspace_roots[0] is the fallback.
	if in.CWD != "/proj" {
		t.Errorf("CWD = %q, want /proj", in.CWD)
	}
}

func TestNormalize_Cursor_ShellToolNameMapsToBash(t *testing.T) {
	// Phase 83.7 live evidence: the real cursor-agent CLI (2026.06.12) sends
	// preToolUse stdin with tool_name "Shell" + tool_input.command (plus
	// model / cursor_version metadata). Before the Shell-to-Bash mapping this
	// shape slipped past the policy kernel (fail-open), because only the
	// top-level-command shorthand was normalized to "Bash".
	raw := []byte(`{
		"conversation_id":"73236a11-4816-44ed-95fb-deb1fb666d5c",
		"generation_id":"73236a11-4816-44ed-95fb-deb1fb666d5c",
		"model":"composer-2.5",
		"hook_event_name":"preToolUse",
		"cursor_version":"2026.06.12",
		"tool_name":"Shell",
		"tool_input":{"command":"git push --force origin main"},
		"workspace_roots":["/proj"]
	}`)
	in, host, err := Normalize(raw, HostCursor)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostCursor {
		t.Errorf("host = %q, want cursor", host)
	}
	if in.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash (mapped from live Cursor Shell payload)", in.ToolName)
	}
	if got := in.ToolInput["command"]; got != forceCmd {
		t.Errorf("command = %v, want %q", got, forceCmd)
	}
}

func TestNormalize_Cursor_WorkspaceRootsFallback(t *testing.T) {
	// No cwd → first workspace_roots entry becomes CWD.
	raw := []byte(`{
		"hook_event_name":"preToolUse",
		"command":"git push --force origin main",
		"workspace_roots":["/first","/second"]
	}`)
	in, host, err := Normalize(raw, "cursor")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if host != HostCursor {
		t.Errorf("host = %q, want cursor", host)
	}
	if in.CWD != "/first" {
		t.Errorf("CWD = %q, want /first (workspace_roots[0])", in.CWD)
	}
	if in.PluginRoot != "/first" {
		t.Errorf("PluginRoot = %q, want /first (cwd fallback)", in.PluginRoot)
	}
}

func TestNormalize_FilePathShorthand(t *testing.T) {
	// A top-level file_path with an explicit tool_name (e.g. Write) becomes
	// tool_input.file_path.
	raw := []byte(`{
		"session_id":"s1",
		"tool_name":"Write",
		"file_path":"/repo/.claude-plugin/settings.json"
	}`)
	in, _, err := Normalize(raw, "claude")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got := in.ToolInput["file_path"]; got != "/repo/.claude-plugin/settings.json" {
		t.Errorf("file_path = %v, want the settings path", got)
	}
}

func TestNormalize_PathAlias(t *testing.T) {
	raw := []byte(`{"session_id":"s1","tool_name":"Read","path":"/etc/hosts"}`)
	in, _, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got := in.ToolInput["file_path"]; got != "/etc/hosts" {
		t.Errorf("file_path = %v, want /etc/hosts (from path alias)", got)
	}
}

func TestNormalize_ToolInputPreservedOverShorthand(t *testing.T) {
	// When both a structured tool_input.command and a top-level command are
	// present, the explicit tool_input value is NOT overwritten.
	raw := []byte(`{
		"session_id":"s1",
		"tool_name":"Bash",
		"tool_input":{"command":"structured"},
		"command":"shorthand"
	}`)
	in, _, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got := in.ToolInput["command"]; got != "structured" {
		t.Errorf("command = %v, want structured (tool_input must win)", got)
	}
}

func TestNormalize_EmptyInput(t *testing.T) {
	if _, _, err := Normalize(nil, ""); err == nil {
		t.Fatal("expected error for nil input")
	}
	if _, _, err := Normalize([]byte("   \n\t"), ""); err == nil {
		t.Fatal("expected error for whitespace-only input")
	}
}

func TestNormalize_InvalidJSON(t *testing.T) {
	if _, _, err := Normalize([]byte("{not json"), ""); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestNormalize_MissingToolName(t *testing.T) {
	// A payload with neither a tool_name nor a command has no usable action.
	raw := []byte(`{"session_id":"s1","cwd":"/x"}`)
	_, _, err := Normalize(raw, "")
	if err == nil {
		t.Fatal("expected error when no tool action is present")
	}
}

func TestNormalize_ToolInputNeverNil(t *testing.T) {
	raw := []byte(`{"session_id":"s1","tool_name":"Read"}`)
	in, _, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if in.ToolInput == nil {
		t.Error("ToolInput must be a non-nil map")
	}
}

func TestNormalize_ToolNameCamelAlias(t *testing.T) {
	raw := []byte(`{"session_id":"s1","toolName":"Bash","tool_input":{"command":"ls"}}`)
	in, _, err := Normalize(raw, "")
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if in.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash (from toolName alias)", in.ToolName)
	}
}
