package main

import (
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/pkg/hookproto"
)

// TestPolicyCheckResult_Deny verifies that an action the R01-R13 kernel denies
// (R06: git force-push) yields process exit code 2 with a non-nil deny payload.
func TestPolicyCheckResult_Deny(t *testing.T) {
	out, code := policyCheckResult(hookproto.HookInput{
		ToolName: "Bash",
		ToolInput: map[string]interface{}{
			"command": "git push --force origin main",
		},
	})
	if code != 2 {
		t.Fatalf("git force-push should deny with exit 2, got exit %d", code)
	}
	if out == nil {
		t.Fatal("deny should emit a hookSpecificOutput payload, got nil")
	}
}

// TestPolicyCheckResult_Allow verifies a benign action passes with exit 0 and no
// output payload (pure approve).
func TestPolicyCheckResult_Allow(t *testing.T) {
	out, code := policyCheckResult(hookproto.HookInput{
		ToolName: "Read",
		ToolInput: map[string]interface{}{
			"file_path": "/tmp/example.txt",
		},
	})
	if code != 0 {
		t.Fatalf("benign Read should allow with exit 0, got exit %d", code)
	}
	if out != nil {
		t.Fatalf("clean approve should emit no payload, got %v", out)
	}
}
