package gitport

import (
	"path/filepath"
	"strings"
	"testing"
)

// initRepo creates a hermetic temp git repo and returns its real (symlink-resolved)
// path so comparisons against `rev-parse --show-toplevel` are stable on macOS
// (/var -> /private/var).
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	if err := Run(dir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	return dir
}

func TestOutput_RevParseShowToplevel(t *testing.T) {
	dir := initRepo(t)

	out, err := Output(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("Output returned error: %v", err)
	}
	got := strings.TrimSpace(out)
	if real, err := filepath.EvalSymlinks(got); err == nil {
		got = real
	}
	if got != dir {
		t.Fatalf("toplevel = %q, want %q", got, dir)
	}
}

func TestRun_InvalidSubcommandReturnsError(t *testing.T) {
	dir := initRepo(t)

	if err := Run(dir, "definitely-not-a-git-subcommand"); err == nil {
		t.Fatalf("Run of invalid subcommand: expected error, got nil")
	}
}

func TestCombinedOutput_CapturesStderrOnFailure(t *testing.T) {
	dir := initRepo(t)

	out, err := CombinedOutput(dir, "definitely-not-a-git-subcommand")
	if err == nil {
		t.Fatalf("CombinedOutput of invalid subcommand: expected error, got nil")
	}
	// git writes its "is not a git command" diagnostic to stderr; CombinedOutput
	// must surface it (non-empty captured output).
	if strings.TrimSpace(out) == "" {
		t.Fatalf("CombinedOutput returned empty output on failure; expected stderr text")
	}
}

func TestOutput_EmptyDirInheritsCwd(t *testing.T) {
	// dir == "" must NOT set cmd.Dir. Running `git rev-parse --git-dir` from the
	// package's own cwd (inside this repo) should succeed.
	if _, err := Output("", "rev-parse", "--git-dir"); err != nil {
		t.Fatalf("Output with empty dir (inherit cwd) failed: %v", err)
	}
}
