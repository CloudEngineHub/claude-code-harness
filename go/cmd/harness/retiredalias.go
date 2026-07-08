package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/retiredalias"
)

func runRetiredAlias(args []string) {
	os.Exit(runRetiredAliasCommand(args, os.Stdout, os.Stderr))
}

func runRetiredAliasCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness retired-alias scan [root]")
		return 1
	}
	switch args[0] {
	case "scan":
		return runRetiredAliasScanCommand(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown retired-alias subcommand: %s\n", args[0])
		fmt.Fprintln(stderr, "Usage: harness retired-alias scan [root]")
		return 1
	}
}

func runRetiredAliasScanCommand(args []string, stdout, stderr io.Writer) int {
	var rootOverride string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "harness retired-alias scan: --root requires a value")
				return 1
			}
			i++
			rootOverride = args[i]
		default:
			if rootOverride == "" {
				rootOverride = args[i]
			}
		}
	}

	repoRoot, err := resolveProjectRoot(nonEmptyRootArgs(rootOverride))
	if err != nil {
		fmt.Fprintf(stderr, "harness retired-alias scan: %v\n", err)
		return 2
	}

	registryPath := retiredalias.DefaultRegistryPath(repoRoot)
	reg, err := retiredalias.LoadRegistry(registryPath)
	if err != nil {
		fmt.Fprintf(stderr, "harness retired-alias scan: load registry: %v\n", err)
		return 2
	}

	hits, err := retiredalias.Scan(repoRoot, reg, retiredalias.ScanOptions{})
	if err != nil {
		fmt.Fprintf(stderr, "harness retired-alias scan: %v\n", err)
		return 2
	}

	fmt.Fprintln(stdout, "=== Retired Alias Residue Scan ===")
	fmt.Fprintf(stdout, "Registry: %s\n", retiredalias.RegistryRelPath)
	fmt.Fprintf(stdout, "Entries: %d\n", len(reg.Entries))
	fmt.Fprintln(stdout)

	if len(hits) == 0 {
		fmt.Fprintln(stdout, "OK: 0 residue hits")
		return 0
	}

	fmt.Fprintf(stdout, "FAIL: %d residue hit(s)\n", len(hits))
	for _, hit := range hits {
		fmt.Fprintf(stdout, "  ✗ %s\n", hit.String())
	}
	return 1
}

func nonEmptyRootArgs(root string) []string {
	if root == "" {
		return nil
	}
	return []string{root}
}
