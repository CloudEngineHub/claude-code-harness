package reviewiterate

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ScriptRunner executes a shell script with args and returns stdout.
type ScriptRunner func(ctx context.Context, script string, args ...string) (stdout string, err error)

// DefaultScriptRunner shells out to bash for production reviewer backend resolution.
func DefaultScriptRunner() ScriptRunner {
	return func(ctx context.Context, script string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "bash", append([]string{script}, args...)...)
		out, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
}

// ResolveReviewerBackend resolves the cross-CLI reviewer backend via
// scripts/resolve-impl-backend.sh --role reviewer.
func ResolveReviewerBackend(ctx context.Context, repoRoot string, runner ScriptRunner) (string, error) {
	script := repoRoot + "/scripts/resolve-impl-backend.sh"
	out, err := runner(ctx, script, "--role", "reviewer")
	if err != nil {
		return "", fmt.Errorf("resolve reviewer backend: %w", err)
	}
	backend := strings.TrimSpace(out)
	if backend == "" {
		return "", fmt.Errorf("resolve reviewer backend: empty result")
	}
	return backend, nil
}

// advisoryResponse is the JSON shape expected from a headless advisory reviewer CLI.
type advisoryResponse struct {
	Findings []string `json:"findings"`
	Refined  string   `json:"refined"`
}

// HeadlessCLIReviewer returns a fresh-context Reviewer that invokes a headless
// companion CLI. Each call is independent (no shared session state).
type HeadlessCLIReviewer struct {
	Runner          ScriptRunner
	CompanionScript string
	Lens            string
	SessionIDGen    func(lens string) string
}

// Review implements Reviewer.
func (h *HeadlessCLIReviewer) Review(ctx context.Context, lensName, workerOutput string) (Review, error) {
	prompt := fmt.Sprintf("Advisory review lens=%s\n\nWorker output:\n%s", lensName, workerOutput)
	stdout, err := h.Runner(ctx, h.CompanionScript, "task", "--read", prompt)
	if err != nil {
		return Review{}, fmt.Errorf("headless reviewer lens %s: %w", lensName, err)
	}
	_ = h.SessionIDGen(lensName) // fresh session per call when generator is wired by production
	return parseAdvisoryResponse(lensName, stdout)
}

// NewHeadlessCLIReviewerFunc wraps HeadlessCLIReviewer as a Reviewer func.
func NewHeadlessCLIReviewerFunc(h HeadlessCLIReviewer) Reviewer {
	return func(ctx context.Context, lensName, workerOutput string) (Review, error) {
		return h.Review(ctx, lensName, workerOutput)
	}
}

func parseAdvisoryResponse(lens, stdout string) (Review, error) {
	stdout = strings.TrimSpace(stdout)
	var raw advisoryResponse
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		// Best-effort: treat non-JSON stdout as a single finding line when non-empty.
		if stdout != "" {
			return Review{Lens: lens, Findings: []string{stdout}}, nil
		}
		return Review{Lens: lens}, nil
	}
	return Review{Lens: lens, Findings: raw.Findings, Refined: raw.Refined}, nil
}

// brainResponse is the JSON shape expected from the brain (claude host) verdict CLI.
type brainResponse struct {
	Verdict string `json:"verdict"`
}

// HeadlessCLIBrain returns a BrainVerdict via headless CLI (claude host).
type HeadlessCLIBrain struct {
	Runner          ScriptRunner
	CompanionScript string
}

// Verdict implements BrainVerdict.
func (b *HeadlessCLIBrain) Verdict(ctx context.Context, workerOutput string, advisories []Review) (Verdict, error) {
	payload, err := json.Marshal(map[string]any{
		"worker_output": workerOutput,
		"advisories":    advisories,
	})
	if err != nil {
		return "", err
	}
	stdout, err := b.Runner(ctx, b.CompanionScript, "task", "--read", string(payload))
	if err != nil {
		return "", fmt.Errorf("headless brain verdict: %w", err)
	}
	return parseBrainVerdict(stdout)
}

// NewHeadlessCLIBrainFunc wraps HeadlessCLIBrain as BrainVerdict.
func NewHeadlessCLIBrainFunc(b HeadlessCLIBrain) BrainVerdict {
	return func(ctx context.Context, workerOutput string, advisories []Review) (Verdict, error) {
		return b.Verdict(ctx, workerOutput, advisories)
	}
}

func parseBrainVerdict(stdout string) (Verdict, error) {
	stdout = strings.TrimSpace(stdout)
	var raw brainResponse
	if err := json.Unmarshal([]byte(stdout), &raw); err == nil && raw.Verdict != "" {
		v := Verdict(strings.ToUpper(strings.TrimSpace(raw.Verdict)))
		if v == VerdictApprove || v == VerdictRequestChanges {
			return v, nil
		}
	}
	upper := strings.ToUpper(stdout)
	switch {
	case strings.Contains(upper, "APPROVE"):
		return VerdictApprove, nil
	case strings.Contains(upper, "REQUEST_CHANGES"), strings.Contains(upper, "REQUEST CHANGES"):
		return VerdictRequestChanges, nil
	default:
		return "", fmt.Errorf("brain verdict: unrecognized output %q", stdout)
	}
}
