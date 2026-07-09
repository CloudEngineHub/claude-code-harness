package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func runSelfAuditBaselineCapture(args []string) (string, int) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runSelfAuditBaselineCommand(args, &stdout, &stderr)
	return stdout.String(), code
}

func writeBaselineFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLI_SelfAuditBaseline_NoRegression_Exit0(t *testing.T) {
	dir := t.TempDir()
	settings := writeBaselineFixture(t, dir, "settings.json",
		`{"permissions":{"deny":["Bash(sudo:*)","Edit(.claude/settings*)"]}}`,
	)
	baseline := writeBaselineFixture(t, dir, "baseline.json", `{
  "version": "deny-baseline.v1",
  "canonical_sha256": "placeholder",
  "entries": ["Bash(sudo:*)","Edit(.claude/settings*)"]
}`)
	// Fix baseline hash to match entries (same as settings)
	out, code := runSelfAuditBaselineCapture([]string{
		"--settings", settings,
		"--baseline", baseline,
	})
	if code != 0 {
		t.Fatalf("exit = %d, want 0; out=%s", code, out)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v; out=%s", err, out)
	}
	if ok, _ := parsed["ok"].(bool); !ok {
		t.Fatalf("ok = false in output: %s", out)
	}
}

func TestCLI_SelfAuditBaseline_DenyRemoved_Exit2(t *testing.T) {
	dir := t.TempDir()
	settings := writeBaselineFixture(t, dir, "settings.json",
		`{"permissions":{"deny":["Bash(sudo:*)"]}}`,
	)
	baseline := writeBaselineFixture(t, dir, "baseline.json", `{
  "version": "deny-baseline.v1",
  "canonical_sha256": "placeholder",
  "entries": ["Bash(sudo:*)","Edit(.claude/settings*)"]
}`)
	_, code := runSelfAuditBaselineCapture([]string{
		"--settings", settings,
		"--baseline", baseline,
	})
	if code != 2 {
		t.Fatalf("exit = %d, want 2 for deny regression", code)
	}
}

func TestCLI_SelfAuditBaseline_FileMissing_Exit1(t *testing.T) {
	dir := t.TempDir()
	settings := writeBaselineFixture(t, dir, "settings.json",
		`{"permissions":{"deny":["Bash(sudo:*)"]}}`,
	)
	missingBaseline := filepath.Join(dir, "no-such-baseline.json")
	_, code := runSelfAuditBaselineCapture([]string{
		"--settings", settings,
		"--baseline", missingBaseline,
	})
	if code != 1 {
		t.Fatalf("exit = %d, want 1 for missing baseline file", code)
	}
}
