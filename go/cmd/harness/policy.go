package main

import (
	"fmt"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/guardrail"
	"github.com/Chachamaru127/claude-code-harness/go/internal/hook"
	"github.com/Chachamaru127/claude-code-harness/go/internal/policy"
	"github.com/Chachamaru127/claude-code-harness/go/pkg/hookproto"
)

// runPolicy handles `harness policy <subcommand>`.
//
// `harness policy check` reads a PreToolUse-style JSON action description on
// stdin, evaluates it through the R01-R13 rule kernel, prints any deny/ask
// hookSpecificOutput on stdout, and exits 2 when the action is denied (0
// otherwise). It is the explicit gate entry used by the FLOOR re-evaluation of
// cherry-picked diffs (Phase 91.6); the per-event hook fast-path drives the
// same engine via `harness hook pre-tool`.
func runPolicy(args []string) {
	if len(args) < 1 || args[0] != "check" {
		fmt.Fprintln(os.Stderr, "Usage: harness policy check  (reads PreToolUse JSON on stdin; exit 2 = deny)")
		os.Exit(1)
	}

	// Chain-integrity precondition (Phase 91.6 FLOOR). Before this gate
	// adjudicates anything, confirm the deny surface itself has not been
	// weakened relative to the build-time baseline. A weakened chain must not
	// silently approve an untrusted change, so this fails CLOSED (exit 3) rather
	// than continuing. The happy path (surface intact) is untouched below.
	if err := policy.VerifyDenySurface(); err != nil {
		fmt.Fprintf(os.Stderr, "policy check: refusing to adjudicate — %v\n", err)
		os.Exit(3)
	}

	input, err := hook.ReadInput(os.Stdin)
	if err != nil {
		// A gate cannot verify a malformed or empty action. Unlike the hook
		// fast-path (which approves on transient parse errors to avoid blocking
		// normal operation), an explicit `policy check` fails to error (exit 1)
		// rather than silently approving an action it could not evaluate.
		fmt.Fprintf(os.Stderr, "policy check: cannot read action from stdin: %v\n", err)
		os.Exit(1)
	}

	output, exitCode := policyCheckResult(input)
	if output != nil {
		hook.WriteJSON(os.Stdout, output)
	}
	os.Exit(exitCode)
}

// policyCheckResult evaluates a PreToolUse input through the R01-R13 engine and
// returns the hookSpecificOutput payload (nil for a clean approve) plus the
// process exit code (2 = deny, 0 = allow/ask). It is separated from runPolicy
// so the decision path is unit-testable without os.Exit.
func policyCheckResult(input hookproto.HookInput) (interface{}, int) {
	return policy.FormatPreToolResult(guardrail.EvaluatePreTool(input))
}
