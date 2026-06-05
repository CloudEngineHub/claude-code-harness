// Package companionresult defines companion-result.v1: the normalized outcome
// of one backend (codex|cursor) sub-run.
//
// The companion shell scripts (scripts/codex-companion.sh,
// scripts/cursor-companion.sh) are a shared contract and are NOT modified.
// Normalize is the harness-side layer that maps a raw companion run
// (exit code + stdout/stderr) into a uniform Result, so both the codex and
// cursor backends "produce" companion-result.v1 identically.
package companionresult

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaID is the fixed schema identifier for a companion-result.v1 envelope.
const SchemaID = "companion-result.v1"

// summaryCap bounds the derived Summary length (~200 chars) so a runaway
// companion log line cannot blow up the envelope.
const summaryCap = 200

// Result is the normalized outcome of one backend (codex|cursor) sub-run.
// Schema id: "companion-result.v1".
type Result struct {
	Schema       string   `json:"schema"`        // always "companion-result.v1"
	Backend      string   `json:"backend"`       // "codex" | "cursor"
	TaskID       string   `json:"task_id"`       //
	Success      bool     `json:"success"`       //
	ExitCode     int      `json:"exit_code"`     //
	Summary      string   `json:"summary"`       //
	FilesChanged []string `json:"files_changed"` //
	DurationMs   int64    `json:"duration_ms"`   //
}

// New returns a Result with Schema/Backend/TaskID set and FilesChanged
// initialized to a non-nil empty slice (so JSON renders [] not null).
func New(backend, taskID string) Result {
	return Result{
		Schema:       SchemaID,
		Backend:      backend,
		TaskID:       taskID,
		FilesChanged: []string{},
	}
}

// Marshal renders the Result as stable JSON. Field order is fixed by the
// struct definition, so the output is deterministic for a given Result.
func (r Result) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Parse decodes a companion-result.v1 envelope and validates the schema id.
// A wrong or missing schema is rejected.
func Parse(b []byte) (Result, error) {
	var r Result
	if err := json.Unmarshal(b, &r); err != nil {
		return Result{}, fmt.Errorf("companionresult: invalid JSON: %w", err)
	}
	if r.Schema != SchemaID {
		return Result{}, fmt.Errorf("companionresult: unexpected schema %q, want %q", r.Schema, SchemaID)
	}
	if r.FilesChanged == nil {
		r.FilesChanged = []string{}
	}
	return r, nil
}

// Normalize maps a raw companion run into a Result WITHOUT modifying the
// companion shell scripts.
//
//   - Success    = exitCode == 0.
//   - Summary    = first non-empty trimmed line of stdout; falls back to the
//     first non-empty trimmed line of stderr; capped at ~200 chars.
//   - FilesChanged = best-effort lines from stdout that look like file paths
//     (otherwise empty — never fabricated).
func Normalize(backend, taskID string, exitCode int, stdout, stderr string, durationMs int64) Result {
	r := New(backend, taskID)
	r.ExitCode = exitCode
	r.Success = exitCode == 0
	r.DurationMs = durationMs

	summary := firstNonEmptyLine(stdout)
	if summary == "" {
		summary = firstNonEmptyLine(stderr)
	}
	r.Summary = capString(summary, summaryCap)

	r.FilesChanged = extractFilePaths(stdout)
	return r
}

// firstNonEmptyLine returns the first line of s that is non-empty after
// trimming surrounding whitespace.
func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// capString truncates s to at most max runes, appending a single-rune ellipsis
// when truncation occurs so callers can tell the summary was clipped.
func capString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// extractFilePaths scans stdout for lines that look like file paths and returns
// them de-duplicated in first-seen order. This is best-effort: companions are
// not required to print file lists, so an empty result is normal and never
// fabricated.
//
// A line is treated as a path candidate when, after trimming, it is a single
// whitespace-free token that (a) contains a "/" path separator and (b) has a
// file-like extension (a "." in the final path segment). This deliberately
// avoids matching prose (which contains spaces) or bare directory mentions.
func extractFilePaths(stdout string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, line := range strings.Split(stdout, "\n") {
		token := strings.TrimSpace(line)
		if token == "" || strings.ContainsAny(token, " \t") {
			continue // prose / multi-token line, not a bare path
		}
		if !looksLikeFilePath(token) {
			continue
		}
		if _, dup := seen[token]; dup {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

// looksLikeFilePath reports whether token resembles a relative/absolute file
// path: it must contain a separator and the final segment must carry an
// extension (a "." that is neither leading nor trailing within that segment).
func looksLikeFilePath(token string) bool {
	if !strings.Contains(token, "/") {
		return false
	}
	last := token[strings.LastIndex(token, "/")+1:]
	dot := strings.Index(last, ".")
	if dot <= 0 || dot == len(last)-1 {
		return false // no extension, dotfile, or trailing dot
	}
	return true
}
