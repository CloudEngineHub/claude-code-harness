package main

import (
	"strings"
	"testing"

	"github.com/Chachamaru127/claude-code-harness/go/internal/plans"
)

func TestAssembleVerbPromptWorkIncludesContractAndTask(t *testing.T) {
	task := &plans.Task{TaskID: "1.1", DoD: "build passes"}
	out, err := assembleVerbPrompt("work", task)
	if err != nil {
		t.Fatalf("assembleVerbPrompt returned error: %v", err)
	}
	// Work contract is present (load-bearing schema name).
	if !strings.Contains(out, "worker-report.v1") {
		t.Error("output missing work contract marker worker-report.v1")
	}
	// Task block is present and carries the DoD verbatim.
	if !strings.Contains(out, "## Task") {
		t.Error("output missing ## Task block")
	}
	if !strings.Contains(out, "build passes") {
		t.Error("output missing task DoD text")
	}
	if !strings.Contains(out, "1.1") {
		t.Error("output missing task ID")
	}
}

func TestAssembleVerbPromptReviewIncludesContractAndTask(t *testing.T) {
	task := &plans.Task{TaskID: "2.3", DoD: "diff matches DoD"}
	out, err := assembleVerbPrompt("review", task)
	if err != nil {
		t.Fatalf("assembleVerbPrompt returned error: %v", err)
	}
	if !strings.Contains(out, "review-result.v1") {
		t.Error("output missing review contract marker review-result.v1")
	}
	if !strings.Contains(out, "diff matches DoD") {
		t.Error("output missing task DoD text")
	}
}

func TestAssembleVerbPromptPlanHasNoTaskBlock(t *testing.T) {
	out, err := assembleVerbPrompt("plan", nil)
	if err != nil {
		t.Fatalf("assembleVerbPrompt returned error: %v", err)
	}
	// Plan contract present.
	if !strings.Contains(out, "Planner Contract") {
		t.Error("output missing plan contract")
	}
	// No injected task block when task is nil. The injected block is the exact
	// sequence "## Task\n- ID:"; assert on that rather than the bare "## Task"
	// substring (which legitimately appears as "## Task row format" in the
	// plan contract prose).
	if strings.Contains(out, "## Task\n- ID:") {
		t.Error("plan output should not contain an injected task block")
	}
}

func TestAssembleVerbPromptUnknownVerbErrors(t *testing.T) {
	if _, err := assembleVerbPrompt("bogus", nil); err == nil {
		t.Fatal("expected error for unknown verb, got nil")
	}
}
