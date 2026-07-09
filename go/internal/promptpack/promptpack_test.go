package promptpack

import (
	"strings"
	"testing"
)

func TestGetReturnsNonEmptyForEachVerb(t *testing.T) {
	for _, v := range Verbs() {
		got, err := Get(v)
		if err != nil {
			t.Fatalf("Get(%q) returned error: %v", v, err)
		}
		if strings.TrimSpace(got) == "" {
			t.Errorf("Get(%q) returned empty content", v)
		}
	}
}

func TestGetUnknownVerbErrors(t *testing.T) {
	if _, err := Get("bogus"); err == nil {
		t.Fatal("Get(\"bogus\") expected an error, got nil")
	}
}

// TestWorkPromptHasRealContent proves the work contract is genuinely embedded
// (not a stub) by asserting load-bearing substrings are present.
func TestWorkPromptHasRealContent(t *testing.T) {
	got, err := Get("work")
	if err != nil {
		t.Fatalf("Get(\"work\") returned error: %v", err)
	}
	for _, want := range []string{"worker-report.v1", "tdd-red-evidence-attached"} {
		if !strings.Contains(got, want) {
			t.Errorf("work.md missing expected substring %q", want)
		}
	}
}
