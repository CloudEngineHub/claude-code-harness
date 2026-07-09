package policy

import "strings"

// Protected-branch direct-push policy values. These are the canonical string
// values that RuleContext.ProtectedBranchPushPolicy may hold after the
// configuration layer (internal/guardrail) resolves the setting.
const (
	ProtectedBranchPushPolicyAsk   = "ask"
	ProtectedBranchPushPolicyDeny  = "deny"
	ProtectedBranchPushPolicyAllow = "allow"
)

// NormalizeProtectedBranchPushPolicy maps a raw configuration value to one of
// the canonical policy values (ask/deny/allow). Unknown or empty values default
// to "ask". It is exported so that the configuration resolver in
// internal/guardrail can normalize values read from env vars / harness.toml /
// project YAML using the same rules as the R12 evaluator.
func NormalizeProtectedBranchPushPolicy(value string) string {
	normalized := strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`))
	switch normalized {
	case ProtectedBranchPushPolicyAsk, "confirm":
		return ProtectedBranchPushPolicyAsk
	case ProtectedBranchPushPolicyDeny, "block":
		return ProtectedBranchPushPolicyDeny
	case ProtectedBranchPushPolicyAllow, "approve":
		return ProtectedBranchPushPolicyAllow
	default:
		return ProtectedBranchPushPolicyAsk
	}
}
