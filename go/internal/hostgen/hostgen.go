// Package hostgen generates each host's native pre-action hook configuration
// from a single descriptor file (hosts.toml at the repo root).
//
// The convergence goal of Phase 91.3: Claude, Codex, and Cursor each have a
// different hooks.json schema, but all three must invoke the SAME policy engine
// entrypoint — `bin/harness hook pre-tool` — so one R01-R13 rule kernel
// adjudicates every host. hosts.toml is the single source of cross-host
// differences (event key name, file path, deny mechanism); this package turns
// each [host] table into that host's native hooks.json bytes.
//
// Scope: this package emits HOOK configs only. The Claude PreToolUse command is
// represented for completeness/testing, but the live .claude-plugin/hooks.json
// (a hand-maintained 592-line file) is NOT overwritten until the Phase 91.8
// cutover — `harness gen` skips writing it.
//
// hostgen is a tooling package (it parses hosts.toml with BurntSushi/toml),
// not part of the pure guardrail kernel, so external deps are acceptable here.
package hostgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/BurntSushi/toml"
)

// preToolCommand is the argv tail every generated host hook appends after the
// harness binary: all three hosts converge on this single policy entrypoint.
const preToolCommand = "hook pre-tool"

// claudeBinary is the binary invocation Codex and Cursor use directly. Claude's
// own hooks.json wraps the binary in a valid_root bootstrap (see
// ClaudePreToolCommand); Codex/Cursor configs are generated minimally and
// resolve the binary from PATH / the host's plugin root.
const claudeBinary = "bin/harness"

// ClaudePreToolCommand mirrors the valid_root bootstrap wrapper used by the
// tracked .claude-plugin/hooks.json PreToolUse entry (resolve the plugin root,
// verify it owns claude-code-harness, then exec the binary with the hook args).
// It is reused verbatim so a regenerated Claude config at the Phase 91.8 cutover
// keeps the exact same launch semantics. The live file is not overwritten now.
const ClaudePreToolCommand = `/bin/bash -c 'valid_root(){ local r="${1:-}"; [ -n "$r" ] && [ -x "$r/bin/harness" ] && [ -f "$r/.claude-plugin/plugin.json" ] && /usr/bin/grep -q "\"name\"[[:space:]]*:[[:space:]]*\"claude-code-harness\"" "$r/.claude-plugin/plugin.json"; }; root="${CLAUDE_PLUGIN_ROOT:-}"; if ! valid_root "$root"; then root=""; for c in "${CLAUDE_PROJECT_DIR:-}" "$PWD" "$HOME/.claude/plugins/marketplaces/claude-code-harness-marketplace" "$HOME/.claude/plugins/cache/claude-code-harness-marketplace/claude-code-harness/"*; do if valid_root "$c"; then root="$c"; break; fi; done; fi; if ! valid_root "$root"; then echo "[claude-code-harness] plugin root not found; hook skipped" >&2; exit 0; fi; exec "$root/bin/harness" "$@"' _ ` + preToolCommand

// Host describes one host's pre-action hook capabilities, parsed from a [host]
// table in hosts.toml.
type Host struct {
	Name      string `toml:"-"`
	HookEvent string `toml:"hook_event"`
	HookPath  string `toml:"hook_path"`
	Matcher   string `toml:"matcher"`
	Deny      string `toml:"deny"`
	Transport string `toml:"transport"`
	Model     string `toml:"model"`
}

// Load parses hosts.toml and returns a map keyed by host name (claude, codex,
// cursor). The Name field of each Host is populated from its table key.
func Load(path string) (map[string]Host, error) {
	var raw map[string]Host
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, fmt.Errorf("hosts.toml: parse error: %w", err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("hosts.toml: no host tables found in %s", path)
	}
	out := make(map[string]Host, len(raw))
	for name, h := range raw {
		h.Name = name
		out[name] = h
	}
	return out, nil
}

// GenerateHooksJSON emits the host's native hooks.json bytes wiring
// h.HookEvent → `bin/harness hook pre-tool`. Output is deterministic (stable
// key order via a fixed encoder, no map iteration over content) and ends with a
// trailing newline. The per-host JSON shape follows each vendor's documented
// schema:
//
//   - claude: {"hooks":{"<event>":[{"matcher":..,"hooks":[{"type":"command","command":<valid_root wrapper>,"timeout":10}]}]}}
//   - codex:  {"hooks":{"<event>":[{"matcher":..,"hooks":[{"type":"command","command":"bin/harness hook pre-tool","timeout":30}]}]}}
//   - cursor: {"version":1,"hooks":{"<event>":[{"command":"bin/harness hook pre-tool","timeout":30}]}}
//
// A deny is expressed by the policy engine at runtime (exit 2 + hookSpecific
// output), not by this static config, so the generated file only declares the
// wiring; the deny mechanism column in hosts.toml documents how each host reads
// that engine result.
func GenerateHooksJSON(h Host) ([]byte, error) {
	var doc interface{}
	switch h.Name {
	case "cursor":
		doc = cursorDoc(h)
	case "codex":
		doc = codexDoc(h)
	case "claude":
		doc = claudeDoc(h)
	default:
		return nil, fmt.Errorf("hostgen: unknown host %q (expected claude, codex, or cursor)", h.Name)
	}
	return marshalStable(doc)
}

// commandEntry is one `{type,command,timeout}` hook step (Claude/Codex shape,
// where steps are nested under a matcher group).
type commandEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

// matcherGroup is one `{matcher,hooks:[...]}` group used by Claude and Codex.
type matcherGroup struct {
	Matcher string         `json:"matcher"`
	Hooks   []commandEntry `json:"hooks"`
}

// cursorEntry is one Cursor hook step. Cursor uses a flatter schema: each event
// maps directly to an array of `{command,...}` entries (no matcher wrapper),
// with the matcher inlined as a sibling field.
type cursorEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Matcher string `json:"matcher"`
	Timeout int    `json:"timeout"`
}

func claudeDoc(h Host) map[string]interface{} {
	return map[string]interface{}{
		"hooks": map[string]interface{}{
			h.HookEvent: []matcherGroup{
				{
					Matcher: h.Matcher,
					Hooks: []commandEntry{
						{Type: "command", Command: ClaudePreToolCommand, Timeout: 10},
					},
				},
			},
		},
	}
}

func codexDoc(h Host) map[string]interface{} {
	return map[string]interface{}{
		"hooks": map[string]interface{}{
			h.HookEvent: []matcherGroup{
				{
					Matcher: h.Matcher,
					Hooks: []commandEntry{
						{Type: "command", Command: binCommand(), Timeout: 30},
					},
				},
			},
		},
	}
}

func cursorDoc(h Host) map[string]interface{} {
	return map[string]interface{}{
		"version": 1,
		"hooks": map[string]interface{}{
			h.HookEvent: []cursorEntry{
				{Type: "command", Command: binCommand(), Matcher: h.Matcher, Timeout: 30},
			},
		},
	}
}

// binCommand returns the harness binary invocation Codex/Cursor write into
// their command field: `bin/harness hook pre-tool`.
func binCommand() string {
	return claudeBinary + " " + preToolCommand
}

// marshalStable JSON-encodes v with sorted keys, 2-space indentation, no HTML
// escaping, and a trailing newline so generator output is byte-stable across
// runs (Go's json package already sorts map keys; the explicit settings pin the
// rest of the format).
func marshalStable(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("hostgen: marshal error: %w", err)
	}
	return buf.Bytes(), nil
}

// SortedNames returns the host names in deterministic order. Useful for callers
// that iterate hosts for stable output (e.g. `harness gen --check`).
func SortedNames(hosts map[string]Host) []string {
	names := make([]string, 0, len(hosts))
	for name := range hosts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
