package policy

import (
	"github.com/Chachamaru127/claude-code-harness/go/pkg/hookproto"
)

// PreToolToOutput converts a HookResult to the official PreToolUse hookSpecificOutput.
func PreToolToOutput(result hookproto.HookResult) *hookproto.PreToolOutput {
	// Only convert deny/ask decisions to hookSpecificOutput.
	// approve with no systemMessage needs no output (exit 0 with empty stdout).
	if result.Decision == hookproto.DecisionApprove && result.SystemMessage == "" {
		return nil
	}

	inner := hookproto.PreToolHookSpecific{
		HookEventName: "PreToolUse",
	}

	switch result.Decision {
	case hookproto.DecisionDeny:
		inner.PermissionDecision = "deny"
		inner.PermissionDecisionReason = result.Reason
	case hookproto.DecisionAsk:
		inner.PermissionDecision = "ask"
		inner.PermissionDecisionReason = result.Reason
	case hookproto.DecisionApprove:
		inner.PermissionDecision = "allow"
		if result.SystemMessage != "" {
			inner.AdditionalContext = result.SystemMessage
		}
	case hookproto.DecisionDefer:
		// CC 2.1.89: DecisionDefer passes the decision to CC for human review.
		inner.PermissionDecision = "defer"
		inner.PermissionDecisionReason = result.Reason
	}

	return &hookproto.PreToolOutput{HookSpecificOutput: inner}
}

// FormatPreToolResult converts a HookResult to the appropriate output for PreToolUse.
// Returns (json bytes or nil, exit code).
//   - deny → hookSpecificOutput JSON, exit 2
//   - ask → hookSpecificOutput JSON, exit 0
//   - approve with systemMessage → hookSpecificOutput JSON, exit 0
//   - approve without message → nil, exit 0
func FormatPreToolResult(result hookproto.HookResult) (output interface{}, exitCode int) {
	// deny always blocks
	if result.Decision == hookproto.DecisionDeny {
		return PreToolToOutput(result), 2
	}

	out := PreToolToOutput(result)
	if out != nil {
		return out, 0
	}

	// Pure approve — empty output, exit 0
	return nil, 0
}

// matchesWriteEditMultiEdit checks if tool name is Write, Edit, or MultiEdit.
func matchesWriteEditMultiEdit(toolName string) bool {
	return toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit"
}

// getStringField safely extracts a string field from tool_input.
func getStringField(input map[string]interface{}, key string) (string, bool) {
	v, ok := input[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

// getChangedContent extracts the changed content from Write (content) or Edit (new_string).
func getChangedContent(input map[string]interface{}) string {
	if content, ok := getStringField(input, "content"); ok {
		return content
	}
	if newStr, ok := getStringField(input, "new_string"); ok {
		return newStr
	}
	return ""
}
