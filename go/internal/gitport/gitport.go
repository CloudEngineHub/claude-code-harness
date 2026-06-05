// Package gitport is the single subprocess seam for git operations in the harness
// runtime. All git invocations funnel through here so the policy/audit layer has
// one auditable chokepoint.
package gitport

import (
	"os/exec"
)

// gitBin is the single git binary reference for the whole runtime.
const gitBin = "git"

func newCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command(gitBin, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}

// Output runs `git <args...>` in dir and returns RAW stdout (no trimming) + error.
// Use for call sites that previously used exec.Command(...).Output(). Callers keep
// their own strings.TrimSpace if they had one.
func Output(dir string, args ...string) (string, error) {
	out, err := newCmd(dir, args...).Output()
	return string(out), err
}

// CombinedOutput runs `git <args...>` in dir and returns combined stdout+stderr + error.
// Use for sites that previously used exec.Command(...).CombinedOutput() (often embed output in errors).
func CombinedOutput(dir string, args ...string) (string, error) {
	out, err := newCmd(dir, args...).CombinedOutput()
	return string(out), err
}

// Run runs `git <args...>` in dir and returns only the error (no captured output).
// Use for sites that previously used exec.Command(...).Run().
func Run(dir string, args ...string) error {
	return newCmd(dir, args...).Run()
}
