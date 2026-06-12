package autoapprove

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutoApproveEnabled_UnsetReturnsFalse(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "")
	enabled, reason := AutoApproveEnabled(t.TempDir())
	if enabled {
		t.Fatal("expected disabled when unset")
	}
	if reason != "HARNESS_AUTO_APPROVE not set" {
		t.Fatalf("reason = %q, want HARNESS_AUTO_APPROVE not set", reason)
	}
}

func TestAutoApproveEnabled_OffReturnsFalse(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "off")
	enabled, reason := AutoApproveEnabled(t.TempDir())
	if enabled {
		t.Fatal("expected disabled when off")
	}
	if reason != "HARNESS_AUTO_APPROVE not on" {
		t.Fatalf("reason = %q, want HARNESS_AUTO_APPROVE not on", reason)
	}
}

func TestAutoApproveEnabled_OnButHarnessBinMissing_ReturnsFalse(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "on")
	enabled, reason := AutoApproveEnabled(t.TempDir())
	if enabled {
		t.Fatal("expected disabled when harness bin missing")
	}
	if reason != "wt fingerprint subcommand unavailable" {
		t.Fatalf("reason = %q, want wt fingerprint subcommand unavailable", reason)
	}
}

func TestAutoApproveEnabled_OnAndAllPrimitivesAvailable_ReturnsTrue(t *testing.T) {
	t.Setenv("HARNESS_AUTO_APPROVE", "on")

	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(binDir, "harness")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	orig := fingerprintProbe
	fingerprintProbe = func(repoRoot string) bool {
		return resolveHarnessBin(repoRoot) != ""
	}
	t.Cleanup(func() { fingerprintProbe = orig })

	enabled, reason := AutoApproveEnabled(root)
	if !enabled {
		t.Fatalf("expected enabled, reason=%q", reason)
	}
	if reason != "enabled" {
		t.Fatalf("reason = %q, want enabled", reason)
	}
}
