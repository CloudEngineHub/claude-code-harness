// Package rulecoverage is a verification spike for HOTL Phase 101 U3
// (Plans.md 101.4): the rule↔check map, gate-ified.
//
// It enumerates the guardrail rule IDs defined in policy/rules.go and
// cross-references them against the three "check" surfaces (Go behavioral
// tests, the selfaudit deny-surface baseline, the validate-plugin.sh shell
// gate). It detects two pathologies:
//   - ineffective rule: a defined rule with NO check on any surface.
//   - orphan check: a selfaudit baseline entry that maps to no defined rule.
//
// The decision (custom Go scanner vs OPA/Conftest, see evidence doc) is to use
// this in-process scanner: rules are Go closures, not data, so OPA would force a
// parallel JSON export + a Rego dialect + a new binary in the distribution.
// This package is pure (stdlib only) and adds no dependency. The emitted matrix
// is a DERIVED artifact (never a judgment basis); callers should mark it
// `derived` and keep it git-ignored per spec invariant 2.
package rulecoverage

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

var (
	// ruleIDRe captures the short ID from `ID: "R06:no-force-push"`.
	ruleIDRe = regexp.MustCompile(`ID:\s*"(R\d{2})[^"]*"`)
	// baselineIDRe captures `R06:` style references used in selfaudit entries,
	// avoiding prose like "R01-R14" (no trailing colon).
	baselineIDRe = regexp.MustCompile(`(R\d{2}):`)
)

// Coverage records, for one rule, which check surfaces reference it.
type Coverage struct {
	ID             string `json:"id"`
	BehavioralTest bool   `json:"behavioral_test"`
	SelfauditPin   bool   `json:"selfaudit_pin"`
	ShellGate      bool   `json:"shell_gate"`
}

// ExtractRuleIDs returns the sorted, unique short rule IDs defined in rulesContent.
func ExtractRuleIDs(rulesContent string) []string {
	set := map[string]struct{}{}
	for _, m := range ruleIDRe.FindAllStringSubmatch(rulesContent, -1) {
		set[m[1]] = struct{}{}
	}
	return sortedKeys(set)
}

// BuildMatrix computes coverage for every defined rule across the three surfaces.
// testContents are the concatenated bodies of the Go *_test.go files.
func BuildMatrix(rulesContent, selfauditContent, shellGateContent string, testContents ...string) []Coverage {
	ids := ExtractRuleIDs(rulesContent)
	joinedTests := strings.Join(testContents, "\n")
	out := make([]Coverage, 0, len(ids))
	for _, id := range ids {
		out = append(out, Coverage{
			ID:             id,
			BehavioralTest: strings.Contains(joinedTests, "Test"+id),
			SelfauditPin:   baselineReferences(selfauditContent, id),
			ShellGate:      strings.Contains(shellGateContent, id),
		})
	}
	return out
}

// baselineReferences reports whether id appears as a baseline entry (`R06:`).
func baselineReferences(selfauditContent, id string) bool {
	for _, m := range baselineIDRe.FindAllStringSubmatch(selfauditContent, -1) {
		if m[1] == id {
			return true
		}
	}
	return false
}

// RuleOrphans returns rules with no check on any surface (ineffective rules).
func RuleOrphans(m []Coverage) []string {
	var o []string
	for _, c := range m {
		if !c.BehavioralTest && !c.SelfauditPin && !c.ShellGate {
			o = append(o, c.ID)
		}
	}
	sort.Strings(o)
	return o
}

// SelfauditOrphans returns rule IDs referenced as baseline entries in
// selfauditContent that are NOT in definedIDs (a check guarding no live rule).
func SelfauditOrphans(selfauditContent string, definedIDs []string) []string {
	defined := map[string]struct{}{}
	for _, id := range definedIDs {
		defined[id] = struct{}{}
	}
	set := map[string]struct{}{}
	for _, m := range baselineIDRe.FindAllStringSubmatch(selfauditContent, -1) {
		if _, ok := defined[m[1]]; !ok {
			set[m[1]] = struct{}{}
		}
	}
	return sortedKeys(set)
}

// Counts summarizes the matrix.
type Counts struct {
	Total          int `json:"total"`
	BehavioralTest int `json:"behavioral_test"`
	SelfauditPin   int `json:"selfaudit_pin"`
	ShellGate      int `json:"shell_gate"`
}

// Summarize counts coverage per surface.
func Summarize(m []Coverage) Counts {
	c := Counts{Total: len(m)}
	for _, r := range m {
		if r.BehavioralTest {
			c.BehavioralTest++
		}
		if r.SelfauditPin {
			c.SelfauditPin++
		}
		if r.ShellGate {
			c.ShellGate++
		}
	}
	return c
}

// MatrixJSON renders the derived coverage map as indented JSON.
func MatrixJSON(m []Coverage) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
