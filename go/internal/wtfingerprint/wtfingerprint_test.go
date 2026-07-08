package wtfingerprint_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/wtfingerprint"
)

func TestCapture_EmptyDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	snap, err := wtfingerprint.Capture([]string{dir})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if len(snap.Files) != 0 {
		t.Fatalf("expected empty snapshot, got %d files", len(snap.Files))
	}
}

func TestDiff_NoChangeReturnsEmpty(t *testing.T) {
	snap := wtfingerprint.Snapshot{Files: map[string]string{"a.txt": "fp1"}}
	changed := wtfingerprint.Diff(snap, snap)
	if len(changed) != 0 {
		t.Fatalf("expected no changes, got %v", changed)
	}
}

func TestDiff_DetectsHomeDotClaudeSettingsWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"permissions":{}}`), 0600); err != nil {
		t.Fatal(err)
	}

	before, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture before: %v", err)
	}
	if err := os.WriteFile(settings, []byte(`{"permissions":{"deny":["Bash(cursor-agent:*)"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture after: %v", err)
	}

	changed := wtfingerprint.Diff(before, after)
	if len(changed) == 0 {
		t.Fatal("expected change in ~/.claude/settings.json, got none")
	}
	found := false
	for _, p := range changed {
		if p == ".claude/settings.json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected .claude/settings.json in changes, got %v", changed)
	}
}

func TestDiff_DetectsAwsCredentialsWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	awsDir := filepath.Join(home, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		t.Fatal(err)
	}
	creds := filepath.Join(awsDir, "credentials")
	if err := os.WriteFile(creds, []byte("[default]\naws_access_key_id=OLD\n"), 0600); err != nil {
		t.Fatal(err)
	}

	before, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture before: %v", err)
	}
	if err := os.WriteFile(creds, []byte("[default]\naws_access_key_id=NEW\n"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture after: %v", err)
	}

	changed := wtfingerprint.Diff(before, after)
	if len(changed) == 0 {
		t.Fatal("expected change in ~/.aws/credentials, got none")
	}
	found := false
	for _, p := range changed {
		if p == ".aws/credentials" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected .aws/credentials in changes, got %v", changed)
	}
}

func TestDiff_DetectsSshKeyWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(sshDir, "id_rsa")
	if err := os.WriteFile(key, []byte("-----BEGIN OLD KEY-----\n"), 0600); err != nil {
		t.Fatal(err)
	}

	before, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture before: %v", err)
	}
	if err := os.WriteFile(key, []byte("-----BEGIN NEW KEY-----\n"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture after: %v", err)
	}

	changed := wtfingerprint.Diff(before, after)
	if len(changed) == 0 {
		t.Fatal("expected change in ~/.ssh/id_rsa, got none")
	}
	found := false
	for _, p := range changed {
		if p == ".ssh/id_rsa" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected .ssh/id_rsa in changes, got %v", changed)
	}
}

func TestDiff_WorktreeInternalWritePasses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	worktree := filepath.Join(home, "worktree")
	if err := os.MkdirAll(worktree, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(worktree, "src.go")
	if err := os.WriteFile(target, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	before, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture before: %v", err)
	}
	if err := os.WriteFile(target, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture after: %v", err)
	}

	changed := wtfingerprint.Diff(before, after)
	if len(changed) != 0 {
		t.Fatalf("worktree-internal write should not be detected, got %v", changed)
	}
}

func TestDiff_NonExistentPathIgnored(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	before, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture before: %v", err)
	}
	after, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture after: %v", err)
	}
	changed := wtfingerprint.Diff(before, after)
	if len(changed) != 0 {
		t.Fatalf("expected no false positives for missing paths, got %v", changed)
	}
}

func TestSymlinkSwapDetected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	awsDir := filepath.Join(home, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		t.Fatal(err)
	}
	targetA := filepath.Join(awsDir, "real-a")
	targetB := filepath.Join(awsDir, "real-b")
	if err := os.WriteFile(targetA, []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetB, []byte("b"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(awsDir, "link")
	if err := os.Symlink("real-a", link); err != nil {
		t.Fatal(err)
	}

	before, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture before: %v", err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real-b", link); err != nil {
		t.Fatal(err)
	}
	after, err := wtfingerprint.Capture(nil)
	if err != nil {
		t.Fatalf("Capture after: %v", err)
	}

	changed := wtfingerprint.Diff(before, after)
	if len(changed) == 0 {
		t.Fatal("expected symlink swap to be detected")
	}
	found := false
	for _, p := range changed {
		if p == ".aws/link" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected .aws/link in changes, got %v", changed)
	}
}
