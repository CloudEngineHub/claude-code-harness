// Package enforcelink is a verification spike for HOTL Phase 101 U2
// (Plans.md 101.3): the rule↔doc↔test machine link.
//
// The harness already has tamper-evidence for rule *matchers* (the selfaudit
// deny-surface SHA-256 baseline in policy/selfaudit.go catches a narrowed or
// removed matcher). What is missing — confirmed by a repo-wide grep returning
// zero `@enforces` tags — is a machine link tying the three layers of a single
// rule together: its definition (rules.go), its behavioral test, and the
// CLAUDE.md promise that documents it. Without that link, silently deleting the
// doc row or the test breaks no gate.
//
// Verify checks all three legs for a given rule ID and reports any missing leg.
// A check built on it exits non-zero when a leg is dropped, turning "written"
// into "effective + tamper-evident". This package is pure and does not modify
// rules.go (human-only per spec invariant 6).
package enforcelink

import (
	"regexp"
	"sort"
)

// RuleDefined reports whether ruleID (e.g. "R06") is defined in rulesContent as
// an `ID: "R06:..."` entry.
func RuleDefined(rulesContent, ruleID string) bool {
	re := regexp.MustCompile(`ID:\s*"` + regexp.QuoteMeta(ruleID) + `[:"]`)
	return re.MatchString(rulesContent)
}

// HasBehavioralTest reports whether a Go test named TestR06* exists in testContent.
func HasBehavioralTest(testContent, ruleID string) bool {
	re := regexp.MustCompile(`func Test` + regexp.QuoteMeta(ruleID) + `[_A-Za-z0-9]*\s*\(`)
	return re.MatchString(testContent)
}

// DocPromises reports whether docContent documents the ruleID (the promise leg).
func DocPromises(docContent, ruleID string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(ruleID) + `\b`)
	return re.MatchString(docContent)
}

// HasEnforcesTag reports whether an `@enforces R06` tag exists in content. This
// is the alternative link representation (tag-in-source) to a registry entry.
func HasEnforcesTag(content, ruleID string) bool {
	re := regexp.MustCompile(`@enforces\s+` + regexp.QuoteMeta(ruleID) + `\b`)
	return re.MatchString(content)
}

// Verify checks the rule/test/doc triad for ruleID and returns the sorted names
// of any MISSING legs ("rule", "test", "doc"). Empty result == fully linked.
func Verify(ruleID, rulesContent, testContent, docContent string) []string {
	var missing []string
	if !RuleDefined(rulesContent, ruleID) {
		missing = append(missing, "rule")
	}
	if !HasBehavioralTest(testContent, ruleID) {
		missing = append(missing, "test")
	}
	if !DocPromises(docContent, ruleID) {
		missing = append(missing, "doc")
	}
	sort.Strings(missing)
	return missing
}
