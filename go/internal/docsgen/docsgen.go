// Package docsgen regenerates the machine-managed SKILL CATALOG section of
// docs/CLAUDE-skill-catalog.md from the actual skills/*/SKILL.md frontmatter.
//
// Motivation (Phase 91.7): the hand-maintained catalog drifted badly — it listed
// skills that no longer exist (impl/, verify/, handoff/, auth/, deploy/, ui/,
// notebookLM/, principles/) and omitted many that do (harness-*, cursor-*, ci,
// memory, …). A drift-prone hand list is exactly what a generator should own.
//
// Scope: this package owns ONLY the table inside the BEGIN/END markers
// (see CatalogBeginMarker / CatalogEndMarker). Everything else in the catalog
// file — the evaluation-flow prose, the related-docs links — is left untouched
// (hand-maintained). The generator does NOT touch CLAUDE.md.
//
// docsgen is a pure, dependency-free package: it parses the small, well-formed
// YAML frontmatter of each SKILL.md with a line-based reader (name + description
// are single-line double-quoted scalars) rather than pulling in a YAML library.
package docsgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CatalogBeginMarker and CatalogEndMarker delimit the machine-managed region in
// docs/CLAUDE-skill-catalog.md. Only the bytes strictly between these two lines
// are owned by `harness gen docs`; the markers themselves are preserved verbatim.
const (
	CatalogBeginMarker = "<!-- BEGIN GENERATED SKILL CATALOG (harness gen docs) -->"
	CatalogEndMarker   = "<!-- END GENERATED SKILL CATALOG (harness gen docs) -->"
)

// CatalogRelPath is the catalog file path relative to the repo root.
const CatalogRelPath = "docs/CLAUDE-skill-catalog.md"

// skillsDirName is the source-of-truth skills directory relative to the repo root.
const skillsDirName = "skills"

// Skill is one catalog row: the skill's frontmatter name and (English) description.
type Skill struct {
	Name        string
	Description string
}

// CollectSkills walks <root>/skills/*/SKILL.md and returns the parsed catalog
// rows sorted by name. A directory without a SKILL.md is skipped (it is not a
// skill). Directories whose name starts with "test-" or "x-" are skipped because
// they are dev/experimental skills excluded from distribution (mirrors the
// exclusion list in scripts/build-opencode.js and sync-skill-mirrors.sh).
func CollectSkills(root string) ([]Skill, error) {
	skillsDir := filepath.Join(root, skillsDirName)
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("docsgen: cannot read %s: %w", skillsDir, err)
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isExcludedSkill(name) {
			continue
		}
		skillMd := filepath.Join(skillsDir, name, "SKILL.md")
		data, readErr := os.ReadFile(skillMd)
		if readErr != nil {
			// No SKILL.md → not a skill directory; skip silently.
			if os.IsNotExist(readErr) {
				continue
			}
			return nil, fmt.Errorf("docsgen: cannot read %s: %w", skillMd, readErr)
		}
		fm, parseErr := parseFrontmatter(data)
		if parseErr != nil {
			return nil, fmt.Errorf("docsgen: %s: %w", skillMd, parseErr)
		}
		fmName := fm["name"]
		if fmName == "" {
			fmName = name // fall back to directory name when frontmatter omits it
		}
		skills = append(skills, Skill{Name: fmName, Description: fm["description"]})
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, nil
}

// isExcludedSkill reports whether a skills/ subdirectory is a dev/experimental
// skill that must not appear in the distributed catalog. Kept in sync with the
// opencode/codex mirror exclusion lists.
func isExcludedSkill(name string) bool {
	return strings.HasPrefix(name, "test-") || strings.HasPrefix(name, "x-")
}

// RenderCatalog renders the generated catalog block (markers + heading + table)
// for the given skills. Output is deterministic and ends with a trailing newline.
// The table escapes pipe characters in descriptions so a `|` inside a description
// cannot break the Markdown table.
func RenderCatalog(skills []Skill) string {
	var b strings.Builder
	b.WriteString(CatalogBeginMarker)
	b.WriteString("\n")
	b.WriteString("<!-- Auto-generated from skills/*/SKILL.md frontmatter. Do not edit by hand; run `harness gen docs`. -->\n")
	b.WriteString("\n")
	b.WriteString("## スキルカタログ一覧\n")
	b.WriteString("\n")
	b.WriteString("| スキル | 説明 |\n")
	b.WriteString("|--------|------|\n")
	for _, s := range skills {
		b.WriteString("| ")
		b.WriteString(escapeTableCell(s.Name))
		b.WriteString(" | ")
		b.WriteString(escapeTableCell(s.Description))
		b.WriteString(" |\n")
	}
	b.WriteString("\n")
	b.WriteString(CatalogEndMarker)
	b.WriteString("\n")
	return b.String()
}

// escapeTableCell makes a value safe to embed in a single Markdown table cell:
// pipes are escaped and newlines collapsed to spaces (frontmatter descriptions
// are single-line, but this is defensive).
func escapeTableCell(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.TrimSpace(s)
}

// ReplaceManagedRegion returns content with the bytes between the BEGIN and END
// markers replaced by generatedBlock (which itself contains both markers). The
// surrounding text is preserved exactly. It errors if the markers are missing or
// out of order so a malformed catalog file fails loudly rather than silently
// appending.
func ReplaceManagedRegion(content, generatedBlock string) (string, error) {
	beginIdx := strings.Index(content, CatalogBeginMarker)
	if beginIdx < 0 {
		return "", fmt.Errorf("docsgen: begin marker %q not found in catalog", CatalogBeginMarker)
	}
	endIdx := strings.Index(content, CatalogEndMarker)
	if endIdx < 0 {
		return "", fmt.Errorf("docsgen: end marker %q not found in catalog", CatalogEndMarker)
	}
	if endIdx < beginIdx {
		return "", fmt.Errorf("docsgen: end marker appears before begin marker in catalog")
	}
	endOfBlock := endIdx + len(CatalogEndMarker)

	// generatedBlock ends with a trailing newline; strip it here so we don't
	// duplicate the newline that already followed the old end marker.
	block := strings.TrimSuffix(generatedBlock, "\n")
	return content[:beginIdx] + block + content[endOfBlock:], nil
}

// Generate reads the catalog at <root>/docs/CLAUDE-skill-catalog.md, regenerates
// the managed region from <root>/skills/*/SKILL.md, and returns the full new
// file content. It does not write to disk (callers decide write vs. --check).
func Generate(root string) (string, error) {
	catalogPath := filepath.Join(root, filepath.FromSlash(CatalogRelPath))
	current, err := os.ReadFile(catalogPath)
	if err != nil {
		return "", fmt.Errorf("docsgen: cannot read %s: %w", catalogPath, err)
	}
	skills, err := CollectSkills(root)
	if err != nil {
		return "", err
	}
	block := RenderCatalog(skills)
	updated, err := ReplaceManagedRegion(string(current), block)
	if err != nil {
		return "", err
	}
	return updated, nil
}

// Write regenerates the catalog and writes it back to disk, returning whether the
// file content changed.
func Write(root string) (changed bool, err error) {
	catalogPath := filepath.Join(root, filepath.FromSlash(CatalogRelPath))
	current, err := os.ReadFile(catalogPath)
	if err != nil {
		return false, fmt.Errorf("docsgen: cannot read %s: %w", catalogPath, err)
	}
	updated, err := Generate(root)
	if err != nil {
		return false, err
	}
	if string(current) == updated {
		return false, nil
	}
	if err := os.WriteFile(catalogPath, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("docsgen: cannot write %s: %w", catalogPath, err)
	}
	return true, nil
}

// Check regenerates the catalog in memory and compares it against the committed
// file. It returns inSync=true when they match; otherwise diff holds a minimal
// line-level diff for display.
func Check(root string) (inSync bool, diff string, err error) {
	catalogPath := filepath.Join(root, filepath.FromSlash(CatalogRelPath))
	current, err := os.ReadFile(catalogPath)
	if err != nil {
		return false, "", fmt.Errorf("docsgen: cannot read %s: %w", catalogPath, err)
	}
	updated, err := Generate(root)
	if err != nil {
		return false, "", err
	}
	if string(current) == updated {
		return true, "", nil
	}
	return false, lineDiff(string(current), updated), nil
}

// parseFrontmatter extracts the top YAML frontmatter block (delimited by the
// first two `---` lines) and returns a map of the simple string scalars it cares
// about (currently "name" and "description"). Only single-line double-quoted or
// bare scalar values are supported, which matches every shipped SKILL.md. A file
// without a leading `---` returns an empty map (no frontmatter), not an error.
func parseFrontmatter(data []byte) (map[string]string, error) {
	out := map[string]string{}
	lines := splitLines(data)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return out, nil // no frontmatter block
	}
	closed := false
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		key, val, ok := splitScalar(line)
		if !ok {
			continue
		}
		if key == "name" || key == "description" {
			// First occurrence wins (frontmatter lists each key once).
			if _, exists := out[key]; !exists {
				out[key] = val
			}
		}
	}
	if !closed {
		return nil, fmt.Errorf("unterminated frontmatter (missing closing '---')")
	}
	return out, nil
}

// splitScalar parses a `key: value` frontmatter line into key and an unquoted
// value. It only handles top-level (non-indented) keys with a single-line scalar
// value; indented lines (e.g. list items) return ok=false and are ignored.
func splitScalar(line string) (key, val string, ok bool) {
	if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
		return "", "", false
	}
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:colon])
	if key == "" || strings.ContainsAny(key, " \t") {
		// Keys are single tokens; "Use when ..." prose wrapped onto a new line
		// (no real key) is rejected here.
		return "", "", false
	}
	rawVal := strings.TrimSpace(line[colon+1:])
	val = unquoteScalar(rawVal)
	return key, val, true
}

// unquoteScalar removes surrounding double quotes from a YAML scalar and unescapes
// the two sequences that can legally appear inside a double-quoted SKILL.md
// description: \" and \\. Bare (unquoted) scalars are returned trimmed as-is.
func unquoteScalar(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner
	}
	return s
}

// splitLines splits on \n and drops a trailing \r from each line so the parser
// is newline-style agnostic.
func splitLines(data []byte) []string {
	raw := strings.Split(string(data), "\n")
	for i := range raw {
		raw[i] = strings.TrimSuffix(raw[i], "\r")
	}
	return raw
}

// lineDiff renders a minimal line-by-line diff between want (committed) and got
// (generated), marking only the lines that differ. It is intentionally simple
// (per-line, not LCS) — enough to point the operator at the drift.
func lineDiff(want, got string) string {
	wl := strings.Split(want, "\n")
	gl := strings.Split(got, "\n")
	n := len(wl)
	if len(gl) > n {
		n = len(gl)
	}
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w == g {
			continue
		}
		if i < len(wl) {
			fmt.Fprintf(&b, "  - %s\n", w)
		}
		if i < len(gl) {
			fmt.Fprintf(&b, "  + %s\n", g)
		}
	}
	return b.String()
}
