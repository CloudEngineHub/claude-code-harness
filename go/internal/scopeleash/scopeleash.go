// Package scopeleash is a verification spike for HOTL Phase 101 U0
// (Plans.md 101.1): the in-run scope leash.
//
// It auto-infers a task's declared file scope from the plan (zero human
// hand-declaration), checks whether a write target is in scope, and flags
// declared-but-untouched scope (dropped work). All logic is deterministic
// (path matching only, no LLM). This is a standalone spike: it does not modify
// the live guardrail rule table (go/internal/policy/rules.go), which is a
// human-only class per spec invariant 6.
package scopeleash

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// knownTopDirs are the repository top-level directories a declared path may
// start under. Mirrors the sprint-contract path miner's anchor set so the
// inference matches existing harness conventions.
const knownTopDirs = `\.claude|src|app|cmd|go|lib|pkg|internal|docs|scripts|tests|agents|skills|hooks|templates|frontend|mcp-server|harness-ui`

// pathTokenRe matches a repo-relative file path token (top-dir prefix, any path
// chars, ending in a .ext). The leading group keeps it from matching mid-word.
var pathTokenRe = regexp.MustCompile(
	`(?:^|[^A-Za-z0-9._/-])((?:` + knownTopDirs + `)/[A-Za-z0-9._/-]*\.[A-Za-z0-9]+)`)

// InferScopeFromPlan derives the declared file scope for taskID from the
// Plans.md markdown with no human input. It locates the task's table row and
// mines path-like tokens from its Title and DoD columns. The returned slice is
// normalized (slash form, cleaned), de-duplicated and sorted.
func InferScopeFromPlan(plansMarkdown, taskID string) ([]string, error) {
	row, ok := findTaskRow(plansMarkdown, taskID)
	if !ok {
		return nil, fmt.Errorf("scopeleash: task %q not found in plan", taskID)
	}
	return minePaths(row), nil
}

// findTaskRow returns the first Plans.md pipe-table row whose first cell is
// exactly taskID (e.g. "| 101.7 | ... |").
func findTaskRow(markdown, taskID string) (string, bool) {
	want := strings.TrimSpace(taskID)
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := strings.Split(trimmed, "|")
		// cells[0] is empty (before leading pipe); cells[1] is the first column.
		if len(cells) < 2 {
			continue
		}
		if strings.TrimSpace(cells[1]) == want {
			return trimmed, true
		}
	}
	return "", false
}

// minePaths extracts normalized, de-duplicated, sorted path tokens from text.
func minePaths(text string) []string {
	seen := map[string]struct{}{}
	for _, m := range pathTokenRe.FindAllStringSubmatch(text, -1) {
		norm := normalize(m[1])
		if norm == "" {
			continue
		}
		seen[norm] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// normalize cleans a path to slash form without a leading "./".
func normalize(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "./")
	return p
}

// CheckWrite reports whether targetPath is within the declared scope.
// targetPath may be absolute; projectRoot (if non-empty) is stripped first so
// the comparison is repo-relative. A scope entry matches the target if it is
// equal to it, or is a directory prefix of it (entry + "/"). Out-of-scope
// (returns false) is the alarm condition.
func CheckWrite(scope []string, targetPath, projectRoot string) bool {
	target := relToRoot(targetPath, projectRoot)
	for _, entry := range scope {
		e := normalize(entry)
		if e == "" {
			continue
		}
		if target == e || strings.HasPrefix(target, e+"/") {
			return true
		}
	}
	return false
}

// relToRoot makes targetPath repo-relative by trimming projectRoot, then
// normalizes it.
func relToRoot(targetPath, projectRoot string) string {
	t := strings.TrimSpace(targetPath)
	if projectRoot != "" {
		root := strings.TrimSuffix(normalize(projectRoot), "/")
		t = normalize(t)
		if t == root {
			return ""
		}
		if strings.HasPrefix(t, root+"/") {
			return strings.TrimPrefix(t, root+"/")
		}
		return t
	}
	return normalize(t)
}

// DroppedScope returns declared scope entries that were never touched (dropped
// work). touched is the set of file paths actually edited during the run (e.g.
// read from .claude/state/changed-files.jsonl). Result is sorted.
func DroppedScope(scope, touched []string) []string {
	touchedSet := map[string]struct{}{}
	for _, t := range touched {
		if n := normalize(t); n != "" {
			touchedSet[n] = struct{}{}
		}
	}
	var dropped []string
	for _, s := range scope {
		n := normalize(s)
		if n == "" {
			continue
		}
		if _, ok := touchedSet[n]; !ok {
			dropped = append(dropped, n)
		}
	}
	sort.Strings(dropped)
	return dropped
}
