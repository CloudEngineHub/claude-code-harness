package hookhandler

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillMirrorDriftHook_DetectsUnsyncedEdit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_PROJECT_ROOT", root)

	ssotBody := "---\nname: demo\n---\n\n# SSOT\n"
	mirrorBody := "---\nname: demo\n---\n\n# Mirror stale\n"
	writeSkillMirrorFixture(t, root, "skills/demo", ssotBody)
	writeSkillMirrorFixture(t, root, "codex/.codex/skills/demo", mirrorBody)
	writeSkillMirrorFixture(t, root, "opencode/skills/demo", mirrorBody)

	input := `{"tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(root, "skills/demo/SKILL.md") + `"}}`
	var out bytes.Buffer
	if err := HandleSkillMirrorDrift(strings.NewReader(input), &out); err != nil {
		t.Fatalf("HandleSkillMirrorDrift: %v", err)
	}

	var payload skillMirrorDriftOutput
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid output JSON: %v raw=%s", err, out.String())
	}
	if payload.HookSpecificOutput.AdditionalContext == "" {
		t.Fatal("expected drift warning context")
	}
	if !strings.Contains(payload.HookSpecificOutput.AdditionalContext, "mirror-state.v1") {
		t.Fatalf("expected mirror-state.v1 mention, got: %s", payload.HookSpecificOutput.AdditionalContext)
	}
}

func TestSkillMirrorDriftHook_SkipsNonSkillsPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_PROJECT_ROOT", root)
	input := `{"tool_name":"Edit","tool_input":{"file_path":"` + filepath.Join(root, "src/main.go") + `"}}`
	var out bytes.Buffer
	if err := HandleSkillMirrorDrift(strings.NewReader(input), &out); err != nil {
		t.Fatalf("HandleSkillMirrorDrift: %v", err)
	}
	if strings.TrimSpace(out.String()) != "{}" {
		t.Fatalf("expected empty approve output, got %s", out.String())
	}
}

func writeSkillMirrorFixture(t *testing.T, root, rel, body string) {
	t.Helper()
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
