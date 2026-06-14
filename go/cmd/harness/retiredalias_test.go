package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRetiredAlias_ScanClean_Exit0(t *testing.T) {
	root := retiredAliasRepoRoot(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runRetiredAliasScanCommand([]string{root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "0 residue hits") {
		t.Fatalf("expected clean scan message, got: %s", stdout.String())
	}
}

func TestRunRetiredAlias_ScanDetectsResidue_Exit1(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "bad.txt")
	if err := os.WriteFile(fixture, []byte("legacy path core/src/guardrails/rules.ts remains\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	registryDir := filepath.Join(root, "templates", "registry")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := `version: 1
entries:
  - id: test-path
    kind: path
    pattern: "core/src/guardrails/rules.ts"
    removed_in: "v4.0.0"
    reason: "test fixture"
`
	if err := os.WriteFile(filepath.Join(registryDir, "retired-aliases.v1.yaml"), []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runRetiredAliasScanCommand([]string{root}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit = %d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "bad.txt") {
		t.Fatalf("expected residue listing, got: %s", stdout.String())
	}
}

func retiredAliasRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "templates", "registry", "retired-aliases.v1.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
