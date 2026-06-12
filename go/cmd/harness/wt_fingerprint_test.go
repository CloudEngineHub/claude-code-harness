package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/wtfingerprint"
)

func TestRunWtFingerprintCaptureAndDiff_NoChange(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	outDir := t.TempDir()
	before := filepath.Join(outDir, "before.json")
	after := filepath.Join(outDir, "after.json")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runWtFingerprintCapture([]string{"--output", before}, stdout, stderr); code != 0 {
		t.Fatalf("capture before: exit %d stderr=%s", code, stderr.String())
	}
	if code := runWtFingerprintCapture([]string{"--output", after}, stdout, stderr); code != 0 {
		t.Fatalf("capture after: exit %d stderr=%s", code, stderr.String())
	}
	if code := runWtFingerprintDiff([]string{"--before", before, "--after", after}, stdout, stderr); code != 0 {
		t.Fatalf("diff: expected exit 0, got %d stderr=%s", code, stderr.String())
	}
}

func TestRunWtFingerprintDiff_DetectsEscapeExit2(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"v":1}`), 0600); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	before := filepath.Join(outDir, "before.json")
	after := filepath.Join(outDir, "after.json")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runWtFingerprintCapture([]string{"--output", before}, stdout, stderr); code != 0 {
		t.Fatalf("capture before: exit %d", code)
	}
	if err := os.WriteFile(settings, []byte(`{"v":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if code := runWtFingerprintCapture([]string{"--output", after}, stdout, stderr); code != 0 {
		t.Fatalf("capture after: exit %d", code)
	}
	if code := runWtFingerprintDiff([]string{"--before", before, "--after", after}, stdout, stderr); code != wtFingerprintDiffExitChanged {
		t.Fatalf("diff: expected exit %d, got %d stderr=%q", wtFingerprintDiffExitChanged, code, stderr.String())
	}
	if !strings.Contains(stderr.String(), ".claude/settings.json") {
		t.Fatalf("stderr missing changed path: %q", stderr.String())
	}
}

func TestRunWtFingerprintCapture_WritesValidJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	out := filepath.Join(t.TempDir(), "snap.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if code := runWtFingerprintCapture([]string{"--output", out}, stdout, stderr); code != 0 {
		t.Fatalf("capture: exit %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var snap wtfingerprint.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if snap.Files == nil {
		t.Fatal("expected files map in snapshot json")
	}
}
