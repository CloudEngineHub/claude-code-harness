package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/hostgen"
)

// TestCheckClaudePreToolDrift exercises the containment contract: the committed
// .claude-plugin/hooks.json may carry several PreToolUse groups, but it must
// contain the guardrail group that hostgen generates from hosts.toml. The gate
// passes when the generated group is present (alongside other hand-maintained
// hooks) and fails when it is absent or when hosts.toml drifts from the file.
func TestCheckClaudePreToolDrift(t *testing.T) {
	dir := t.TempDir()

	writeHosts := func(matcher string) {
		toml := "[claude]\n" +
			"hook_event = \"PreToolUse\"\n" +
			"hook_path  = \".claude-plugin/hooks.json\"\n" +
			"matcher    = \"" + matcher + "\"\n" +
			"deny       = \"exit2\"\n" +
			"transport  = \"stdin-json\"\n" +
			"model      = \"opus\"\n"
		if err := os.WriteFile(filepath.Join(dir, "hosts.toml"), []byte(toml), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeCommitted := func(groupsJSON string) {
		if err := os.MkdirAll(filepath.Join(dir, ".claude-plugin"), 0o755); err != nil {
			t.Fatal(err)
		}
		doc := `{"hooks":{"PreToolUse":` + groupsJSON + `}}`
		if err := os.WriteFile(filepath.Join(dir, ".claude-plugin", "hooks.json"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Derive the generated guardrail group for the real matcher straight from the
	// generator, so the fixture cannot hand-encode a stale valid_root command.
	const realMatcher = "Write|Edit|MultiEdit|Bash|Read"
	writeHosts(realMatcher)
	hosts, err := hostgen.Load(filepath.Join(dir, "hosts.toml"))
	if err != nil {
		t.Fatal(err)
	}
	genBytes, err := hostgen.GenerateHooksJSON(hosts["claude"])
	if err != nil {
		t.Fatal(err)
	}
	groups, err := extractEventGroups(genBytes, "PreToolUse")
	if err != nil {
		t.Fatal(err)
	}
	guardrail, err := json.Marshal(groups[0])
	if err != nil {
		t.Fatal(err)
	}
	const extraHook = `{"matcher":"Bash","hooks":[{"type":"command","command":"bin/harness tdd-check","timeout":5}]}`

	// PASS: guardrail group present alongside an unrelated hand-maintained hook.
	writeCommitted("[" + string(guardrail) + "," + extraHook + "]")
	if err := checkClaudePreToolDrift(dir); err != nil {
		t.Fatalf("guardrail group present should pass containment, got: %v", err)
	}

	// FAIL: guardrail group absent (only the unrelated hook remains).
	writeCommitted("[" + extraHook + "]")
	if err := checkClaudePreToolDrift(dir); err == nil {
		t.Fatal("missing guardrail group should fail the drift check")
	}

	// FAIL: hosts.toml matcher drifts but the committed config keeps the old group.
	writeHosts(realMatcher + "|Glob")
	writeCommitted("[" + string(guardrail) + "]") // old matcher group vs new descriptor
	if err := checkClaudePreToolDrift(dir); err == nil {
		t.Fatal("matcher drift between hosts.toml and committed config should fail")
	}
}
