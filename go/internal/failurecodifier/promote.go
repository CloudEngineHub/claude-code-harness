package failurecodifier

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var (
	// ErrAutoPromotionForbidden is returned when auto-promotion is attempted.
	// SSOT files (patterns.md / decisions.md) require explicit human approval.
	ErrAutoPromotionForbidden = errors.New("failurecodifier: auto-promotion forbidden; human approval required")
)

// PromoteOpts configures a promotion attempt (dry-run only in Phase 100).
type PromoteOpts struct {
	Rule           Rule
	RepoRoot       string
	HumanApproved  bool
	DryRun         bool
	PatternsPath   string
	DecisionsPath  string
}

// Promote attempts SSOT promotion. Auto-promotion is structurally forbidden:
// HumanApproved must be true AND DryRun must be false for any write path, but
// even then this package never writes — callers must apply human-reviewed edits manually.
func Promote(opts PromoteOpts) error {
	if !opts.HumanApproved {
		return fmt.Errorf("%w (human_approved=false)", ErrAutoPromotionForbidden)
	}
	if opts.DryRun {
		return fmt.Errorf("%w (dry_run=true; proposal only)", ErrAutoPromotionForbidden)
	}
	// Structural guard: even with human approval flag, codifier never auto-writes SSOT.
	return fmt.Errorf("%w (codifier is proposal-only; edit SSOT manually after review)", ErrAutoPromotionForbidden)
}

// AutoPromote is the forbidden entry point for unattended SSOT writes.
func AutoPromote(opts PromoteOpts) error {
	opts.HumanApproved = false
	opts.DryRun = false
	return Promote(opts)
}

// ProposeOpts configures dry-run proposal output.
type ProposeOpts struct {
	ExtractOpts
}

// ProposeDryRun extracts rules and returns JSON (array) without writing SSOT.
func ProposeDryRun(opts ProposeOpts) ([]byte, error) {
	rules, err := ExtractFromLedger(opts.ExtractOpts)
	if err != nil {
		return nil, err
	}
	if rules == nil {
		rules = []Rule{}
	}
	return json.Marshal(rules)
}

// WriteProposalsStdout writes dry-run JSON proposals to stdout (never touches SSOT).
func WriteProposalsStdout(opts ProposeOpts) error {
	data, err := ProposeDryRun(opts)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	if err == nil {
		_, err = os.Stdout.Write([]byte("\n"))
	}
	return err
}
