package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/clientmirror"
)

func runMirror(args []string) {
	os.Exit(runMirrorCommand(args, os.Stdout, os.Stderr))
}

func runMirrorCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness mirror <status|verify> [--json] [root]")
		return 1
	}

	subcommand := args[0]
	rest := args[1:]
	jsonOut := false
	var rootOverride string
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--json":
			jsonOut = true
		case "--root":
			if i+1 >= len(rest) {
				fmt.Fprintln(stderr, "harness mirror: --root requires a value")
				return 1
			}
			i++
			rootOverride = rest[i]
		default:
			if rootOverride == "" {
				rootOverride = rest[i]
			}
		}
	}

	repoRoot, err := resolveProjectRoot(nonEmptyRootArgs(rootOverride))
	if err != nil {
		fmt.Fprintf(stderr, "harness mirror %s: %v\n", subcommand, err)
		return 2
	}
	schemaPath, err := clientmirror.FindSchemaPath(repoRoot)
	if err != nil {
		if cwd, cwdErr := os.Getwd(); cwdErr == nil {
			schemaPath, err = clientmirror.FindSchemaPath(cwd)
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "harness mirror: %v\n", err)
		return 2
	}

	switch subcommand {
	case "status":
		return writeMirrorState(stdout, stderr, repoRoot, schemaPath, jsonOut, false)
	case "verify":
		return writeMirrorState(stdout, stderr, repoRoot, schemaPath, true, true)
	default:
		fmt.Fprintf(stderr, "Unknown mirror subcommand: %s\n", subcommand)
		fmt.Fprintln(stderr, "Usage: harness mirror <status|verify> [--json] [root]")
		return 1
	}
}

func writeMirrorState(stdout, stderr io.Writer, repoRoot, schemaPath string, jsonOut, verifyMode bool) int {
	state, err := clientmirror.Scan(repoRoot, clientmirror.ScanOptions{})
	if err != nil {
		fmt.Fprintf(stderr, "harness mirror: scan failed: %v\n", err)
		return 2
	}
	if err := clientmirror.ValidateState(state, schemaPath); err != nil {
		fmt.Fprintf(stderr, "harness mirror: schema validation failed: %v\n", err)
		return 2
	}

	if jsonOut || verifyMode {
		data, err := json.Marshal(state)
		if err != nil {
			fmt.Fprintf(stderr, "harness mirror: marshal failed: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "=== Client Mirror Status ===")
		fmt.Fprintf(stdout, "Fingerprint: %s\n", state.Fingerprint)
		fmt.Fprintf(stdout, "Healthy: %v (%s)\n", state.Healthy, state.Reason)
		for _, mirror := range state.Mirrors {
			fmt.Fprintf(stdout, "  %s: %s (drifts=%d)\n", mirror.Root, mirror.Status, mirror.DriftCount)
			for _, drift := range mirror.Drifts {
				fmt.Fprintf(stdout, "    - %s\n", drift)
			}
		}
	}

	if verifyMode && !state.Healthy {
		return 1
	}
	return 0
}
