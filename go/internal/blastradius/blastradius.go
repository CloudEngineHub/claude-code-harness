// Package blastradius is a verification spike for HOTL Phase 101 U5
// (Plans.md 101.6): the third escalation axis.
//
// The three-axis escalation gate raises to the human on (a) final spec/UX
// change, (b) security risk, OR (c) blast-radius / irreversibility. Axes (a)/(b)
// are semantic and an agent self-classifies them unreliably; axis (c) is
// machine-detectable and is the outer backstop. This package detects axis (c)
// SYNTACTICALLY — no semantic judgment — mirroring the existing runtimefloor
// approach (which already catches recursive rm, force-push, worktree escape).
//
// This is a standalone spike proving the detection mechanism; wiring a new
// CategoryBlastRadius into the live runtimefloor is Phase 102. It does not
// modify the rule table or the floor (human-only / production paths).
package blastradius

import "regexp"

// Axis labels for the reason string.
const (
	AxisDelete       = "delete"
	AxisIrreversible = "irreversible"
	AxisCrossRepo    = "cross-repo"
	AxisFileCount    = "file-count"
)

var (
	wsRe = regexp.MustCompile(`\s+`)

	recursiveRmRe = regexp.MustCompile(`\brm\s+-[A-Za-z]*r[A-Za-z]*\b`)
	gitRmRe       = regexp.MustCompile(`\bgit\s+rm\b.*-[A-Za-z]*r`)
	findDeleteRe  = regexp.MustCompile(`\bfind\b.*\s-delete\b`)

	forcePushRe = regexp.MustCompile(`\bgit\s+push\b.*(?:--force\b|--force-with-lease\b|\s-f\b)`)
	resetHardRe = regexp.MustCompile(`\bgit\s+reset\s+--hard\b`)

	pushTagsRe    = regexp.MustCompile(`\bgit\s+push\b.*--tags\b`)
	pushVersionRe = regexp.MustCompile(`\bgit\s+push\b.*\bv\d+\.\d+`)
)

// Detect reports whether command (or the run's touched-file count) crosses the
// blast-radius axis and should escalate to the human. touchedFileCount is the
// number of files changed so far this run; threshold is the count ceiling
// (<= 0 disables the count axis). The reason names the axis that fired.
func Detect(command string, touchedFileCount, threshold int) (escalate bool, reason string) {
	cmd := wsRe.ReplaceAllString(command, " ")

	switch {
	case recursiveRmRe.MatchString(cmd), gitRmRe.MatchString(cmd), findDeleteRe.MatchString(cmd):
		return true, AxisDelete + ": recursive/bulk delete detected"
	case forcePushRe.MatchString(cmd):
		return true, AxisIrreversible + ": force-push detected"
	case resetHardRe.MatchString(cmd):
		return true, AxisIrreversible + ": git reset --hard detected"
	case pushTagsRe.MatchString(cmd), pushVersionRe.MatchString(cmd):
		return true, AxisCrossRepo + ": tag/version push detected"
	}

	if threshold > 0 && touchedFileCount > threshold {
		return true, AxisFileCount + ": touched files exceed threshold"
	}
	return false, ""
}
