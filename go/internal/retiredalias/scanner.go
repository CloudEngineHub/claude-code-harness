package retiredalias

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ScanOptions tunes repository scanning behavior.
type ScanOptions struct {
	MaxFileBytes int64
}

const defaultMaxFileBytes = 2 << 20

var skipDirNames = map[string]bool{
	".git":               true,
	"node_modules":       true,
	"vendor":             true,
	".harness-worktrees": true,
}

// Scan walks repoRoot and returns non-allowlisted pattern matches.
func Scan(repoRoot string, reg *Registry, opts ScanOptions) ([]Hit, error) {
	if reg == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("abs repo root: %w", err)
	}

	maxBytes := opts.MaxFileBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxFileBytes
	}

	var hits []Hit
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirNames[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTextCandidate(path) {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}
		if info.Size() > maxBytes {
			return nil
		}

		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if !isMostlyText(content) {
			return nil
		}

		for _, entry := range reg.Entries {
			if entry.Pattern == "" {
				continue
			}
			if !strings.Contains(string(content), entry.Pattern) {
				continue
			}
			if IsAllowlisted(rel, EffectiveAllowlist(entry)) {
				continue
			}
			line, snippet := firstMatchingLine(content, entry.Pattern)
			hits = append(hits, Hit{
				EntryID: entry.ID,
				Kind:    entry.Kind,
				Pattern: entry.Pattern,
				File:    rel,
				Line:    line,
				Snippet: snippet,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk repo: %w", err)
	}
	return hits, nil
}

// EffectiveAllowlist merges global and entry-specific allowlist prefixes.
func EffectiveAllowlist(entry Entry) []string {
	seen := make(map[string]struct{}, len(GlobalAllowlist)+len(entry.Allowlist))
	out := make([]string, 0, len(GlobalAllowlist)+len(entry.Allowlist))
	for _, p := range append(GlobalAllowlist, entry.Allowlist...) {
		p = normalizeRelPath(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// IsAllowlisted reports whether relPath matches any allowlist prefix.
func IsAllowlisted(relPath string, allowlist []string) bool {
	rel := normalizeRelPath(relPath)
	for _, entry := range allowlist {
		prefix := normalizeRelPath(entry)
		if prefix == "" {
			continue
		}
		if rel == prefix || strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func normalizeRelPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "./")
	p = filepath.ToSlash(p)
	return p
}

func firstMatchingLine(content []byte, pattern string) (int, string) {
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if strings.Contains(line, pattern) {
			return lineNo, strings.TrimSpace(line)
		}
	}
	return 0, pattern
}

func isTextCandidate(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case "", ".md", ".txt", ".yaml", ".yml", ".json", ".toml", ".go", ".ts", ".tsx",
		".js", ".jsx", ".sh", ".bash", ".py", ".rb", ".html", ".css", ".scss",
		".xml", ".svg", ".agent", ".template", ".mdc":
		return true
	default:
		return false
	}
}

func isMostlyText(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	nonText := 0
	limit := len(content)
	if limit > 8192 {
		limit = 8192
	}
	for _, b := range content[:limit] {
		if b == 0 {
			return false
		}
		if b < 9 || (b > 13 && b < 32) {
			nonText++
		}
	}
	return nonText*20 < limit
}
