package enforcelink

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func readPolicyTests(t *testing.T) string {
	t.Helper()
	matches, err := filepath.Glob("../policy/*_test.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(readFile(t, m))
		sb.WriteString("\n")
	}
	return sb.String()
}

// The three legs for R06 must all be present in the live repo.
func TestEnforceLink_R06_FullyLinked(t *testing.T) {
	rules := readFile(t, "../policy/rules.go")
	tests := readPolicyTests(t)
	doc := readFile(t, "../../../CLAUDE.md")

	if missing := Verify("R06", rules, tests, doc); len(missing) != 0 {
		t.Fatalf("R06 is not fully linked; missing legs: %v", missing)
	}
}

// Red-team: dropping each leg must be detected (the gate would exit non-zero).
func TestEnforceLink_DetectsMissingDocLeg(t *testing.T) {
	rules := readFile(t, "../policy/rules.go")
	tests := readPolicyTests(t)
	missing := Verify("R06", rules, tests, "" /* doc row deleted */)
	if len(missing) != 1 || missing[0] != "doc" {
		t.Fatalf("expected missing [doc], got %v", missing)
	}
}

func TestEnforceLink_DetectsMissingTestLeg(t *testing.T) {
	rules := readFile(t, "../policy/rules.go")
	doc := readFile(t, "../../../CLAUDE.md")
	missing := Verify("R06", rules, "" /* test deleted */, doc)
	if len(missing) != 1 || missing[0] != "test" {
		t.Fatalf("expected missing [test], got %v", missing)
	}
}

func TestEnforceLink_DetectsMissingRuleLeg(t *testing.T) {
	tests := readPolicyTests(t)
	doc := readFile(t, "../../../CLAUDE.md")
	missing := Verify("R06", "" /* rule removed */, tests, doc)
	if len(missing) != 1 || missing[0] != "rule" {
		t.Fatalf("expected missing [rule], got %v", missing)
	}
}

// The @enforces tag is the alternative link representation; removing it is
// detectable.
func TestEnforceLink_EnforcesTagDetection(t *testing.T) {
	withTag := "// @enforces R06\nfunc TestR06_ForcePush(t *testing.T) {}"
	if !HasEnforcesTag(withTag, "R06") {
		t.Fatalf("expected @enforces R06 tag to be detected")
	}
	if HasEnforcesTag("func TestR06_ForcePush(t *testing.T) {}", "R06") {
		t.Fatalf("absent @enforces tag must NOT be reported present")
	}
}
