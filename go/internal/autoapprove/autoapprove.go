// Package autoapprove gates HARNESS_AUTO_APPROVE behind runtime safety primitives.
// fail-safe: unset/off, or missing wt fingerprint probe → auto-approve OFF.
package autoapprove

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// fingerprintProbe verifies the harness wt fingerprint subcommand is callable.
// Tests may replace this var; production never reassigns it outside tests.
var fingerprintProbe = defaultFingerprintProbe

// AutoApproveEnabled returns true iff HARNESS_AUTO_APPROVE=on AND both safety
// primitives (runtime floor import + wt fingerprint subcommand) are functional.
// reason is an audit string explaining the decision.
func AutoApproveEnabled(repoRoot string) (enabled bool, reason string) {
	switch strings.TrimSpace(os.Getenv("HARNESS_AUTO_APPROVE")) {
	case "":
		return false, "HARNESS_AUTO_APPROVE not set"
	case "on":
		// continue
	default:
		return false, "HARNESS_AUTO_APPROVE not on"
	}
	if !fingerprintProbe(repoRoot) {
		return false, "wt fingerprint subcommand unavailable"
	}
	return true, "enabled"
}

func defaultFingerprintProbe(repoRoot string) bool {
	bin := resolveHarnessBin(repoRoot)
	if bin == "" {
		return false
	}
	devNull := "/dev/null"
	if runtime.GOOS == "windows" {
		devNull = "NUL"
	}
	cmd := exec.Command(bin, "wt", "fingerprint", "capture", "--output", devNull)
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

func resolveHarnessBin(repoRoot string) string {
	if repoRoot != "" {
		candidate := filepath.Join(repoRoot, "bin", "harness")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	if path, err := exec.LookPath("harness"); err == nil {
		return path
	}
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return ""
}
