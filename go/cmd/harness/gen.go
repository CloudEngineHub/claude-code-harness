package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chachamaru127/claude-code-harness/go/internal/hostgen"
)

// hostsDescriptorName is the single descriptor file at the repo root that drives
// `harness gen`.
const hostsDescriptorName = "hosts.toml"

// runGen handles `harness gen [hooks] [--check] [root]`.
//
// Phase 91.3 convergence core: a single hosts.toml describes each host's
// pre-action hook differences, and `harness gen` materializes each host's native
// hooks.json so Claude, Codex, and Cursor all invoke `bin/harness hook pre-tool`
// (one R01-R13 policy engine for every host).
//
//	harness gen            — write each host's generated hooks.json to its hook_path
//	harness gen hooks      — alias for the above
//	harness gen --check    — compare generated codex+cursor output against golden
//	                         fixtures byte-for-byte; exit 1 on any mismatch
//
// The tracked .claude-plugin/hooks.json (claude) is NEVER overwritten by this
// phase; it is regenerated at the Phase 91.8 cutover. `harness gen` prints a skip
// line for claude and writes only .codex/hooks.json and .cursor/hooks.json (both
// gitignored generated artifacts).
func runGen(args []string) {
	check := false
	var positional []string
	for _, a := range args {
		switch a {
		case "--check":
			check = true
		case "hooks":
			// `gen` and `gen hooks` are equivalent in this phase.
		default:
			positional = append(positional, a)
		}
	}

	root, err := resolveGenRoot(positional)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}

	if check {
		os.Exit(runGenCheck(root))
	}
	if err := runGenWrite(root); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
}

// resolveGenRoot finds the repo root that contains hosts.toml. An explicit
// positional arg wins; otherwise it walks up from the working directory (dev
// invocations run from go/ or the repo root) until hosts.toml is found, falling
// back to the working directory.
func resolveGenRoot(args []string) (string, error) {
	if len(args) > 0 {
		abs, err := filepath.Abs(args[0])
		if err != nil {
			return "", fmt.Errorf("invalid root %q: %w", args[0], err)
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	dir := cwd
	for {
		if _, statErr := os.Stat(filepath.Join(dir, hostsDescriptorName)); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd, nil
}

// generatedHooks loads hosts.toml from root and returns each host's generated
// hooks.json bytes keyed by host name. Factored out so `--check` and the tests
// can obtain the generator output without writing files or shelling out.
func generatedHooks(root string) (map[string][]byte, error) {
	hosts, err := hostgen.Load(filepath.Join(root, hostsDescriptorName))
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(hosts))
	for _, name := range hostgen.SortedNames(hosts) {
		b, genErr := hostgen.GenerateHooksJSON(hosts[name])
		if genErr != nil {
			return nil, genErr
		}
		out[name] = b
	}
	return out, nil
}

// runGenWrite writes the generated codex/cursor hooks.json to their hook_path
// and skips the tracked claude config.
func runGenWrite(root string) error {
	hosts, err := hostgen.Load(filepath.Join(root, hostsDescriptorName))
	if err != nil {
		return err
	}
	for _, name := range hostgen.SortedNames(hosts) {
		h := hosts[name]
		dest := filepath.Join(root, filepath.FromSlash(h.HookPath))
		if name == "claude" {
			fmt.Printf("gen: %-7s %s  skipped (tracked; regenerated at cutover)\n", name, h.HookPath)
			continue
		}
		data, genErr := hostgen.GenerateHooksJSON(h)
		if genErr != nil {
			return genErr
		}
		if mkErr := os.MkdirAll(filepath.Dir(dest), 0o755); mkErr != nil {
			return fmt.Errorf("gen: cannot create dir for %s: %w", h.HookPath, mkErr)
		}
		if wErr := os.WriteFile(dest, data, 0o644); wErr != nil {
			return fmt.Errorf("gen: cannot write %s: %w", h.HookPath, wErr)
		}
		fmt.Printf("gen: %-7s %s  written (%d bytes)\n", name, h.HookPath, len(data))
	}
	return nil
}

// runGenCheck regenerates codex+cursor hooks in memory and compares them against
// the golden fixtures under cmd/harness/testdata/gen/. Returns 0 when every
// fixture matches byte-for-byte and 1 otherwise (printing a line-level diff).
// The claude config is excluded because this phase does not regenerate the
// tracked file.
func runGenCheck(root string) int {
	gen, err := generatedHooks(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen --check: %v\n", err)
		return 1
	}
	fixtureDir := filepath.Join(root, "go", "cmd", "harness", "testdata", "gen")
	// When invoked from within go/ (dev default), the repo root resolver may
	// return the go/ dir itself if hosts.toml is not above it; guard by also
	// trying a path relative to the located root.
	if _, statErr := os.Stat(fixtureDir); statErr != nil {
		fixtureDir = filepath.Join(root, "cmd", "harness", "testdata", "gen")
	}

	hosts := []string{"codex", "cursor"}
	ok := true
	for _, name := range hosts {
		want, readErr := os.ReadFile(filepath.Join(fixtureDir, name+"-hooks.json"))
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "gen --check: cannot read golden fixture for %s: %v\n", name, readErr)
			ok = false
			continue
		}
		got := gen[name]
		if !bytes.Equal(want, got) {
			ok = false
			fmt.Printf("gen --check: MISMATCH for %s (golden vs generated)\n", name)
			fmt.Print(unifiedDiff(string(want), string(got)))
		} else {
			fmt.Printf("gen --check: %s OK\n", name)
		}
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "gen --check: generated output drifted from golden fixtures (run `harness gen` and regenerate fixtures)")
		return 1
	}
	fmt.Println("gen --check: all hosts match golden fixtures")
	return 0
}

// unifiedDiff renders a minimal line-by-line diff between want and got. It is
// intentionally simple (per-line markers, not an LCS algorithm) — enough to show
// where generator output diverged from a fixture.
func unifiedDiff(want, got string) string {
	wl := strings.Split(want, "\n")
	gl := strings.Split(got, "\n")
	n := len(wl)
	if len(gl) > n {
		n = len(gl)
	}
	var b strings.Builder
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
