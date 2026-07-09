package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/clientmirror"
)

type mirrorBuffer struct {
	buf []byte
}

func (b *mirrorBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *mirrorBuffer) String() string { return string(b.buf) }

func TestRunMirror_StatusJSON(t *testing.T) {
	root := mirrorFixtureRoot(t)
	var stdout, stderr mirrorBuffer
	code := runMirrorCommand([]string{"status", "--json", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr.String())
	}

	var state clientmirror.State
	if err := json.Unmarshal(stdout.buf, &state); err != nil {
		t.Fatalf("invalid JSON: %v raw=%s", err, stdout.String())
	}
	if state.SchemaVersion != clientmirror.SchemaVersion {
		t.Fatalf("schema_version = %q", state.SchemaVersion)
	}
	if state.Fingerprint == "" {
		t.Fatal("expected fingerprint")
	}
}

func TestRunMirror_VerifyDriftExit1(t *testing.T) {
	root := t.TempDir()
	writeMirrorSkill(t, root, "skills/demo", "# SSOT\n")
	writeMirrorSkill(t, root, "codex/.codex/skills/demo", "# DRIFT\n")

	var stdout, stderr mirrorBuffer
	code := runMirrorCommand([]string{"verify", "--json", root}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), clientmirror.ReasonDrift) {
		t.Fatalf("expected drift in output: %s", stdout.String())
	}
}

func TestRunMirror_VerifyInSyncExit0(t *testing.T) {
	root := mirrorFixtureRoot(t)
	var stdout, stderr mirrorBuffer
	code := runMirrorCommand([]string{"verify", "--json", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestSyncSkillMirrorsCheck_DelegatesToMirrorVerify(t *testing.T) {
	root := mirrorRepoRoot(t)
	script := filepath.Join(root, "scripts", "sync-skill-mirrors.sh")
	if _, err := os.Stat(script); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "harness mirror verify") {
		t.Fatal("expected sync-skill-mirrors.sh to delegate --check to harness mirror verify")
	}
}

func mirrorRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "templates", "schemas", "mirror-state.v1.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}

func mirrorFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	body := "---\nname: demo\n---\n\n# Demo\n"
	writeMirrorSkill(t, root, "skills/demo", body)
	writeMirrorSkill(t, root, "codex/.codex/skills/demo", body)
	writeMirrorSkill(t, root, "opencode/skills/demo", body)
	return root
}

func writeMirrorSkill(t *testing.T, root, rel, body string) {
	t.Helper()
	dir := filepath.Join(root, rel)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
