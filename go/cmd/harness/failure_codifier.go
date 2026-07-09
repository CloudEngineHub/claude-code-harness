package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/failurecodifier"
)

func runFailureCodifier(args []string) {
	os.Exit(runFailureCodifierCommand(args, os.Stdout, os.Stderr))
}

func runFailureCodifierCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness failure-codifier propose --dry-run [--repo-root <dir>]")
		return 1
	}
	switch args[0] {
	case "propose":
		return runFailureCodifierPropose(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown failure-codifier subcommand: %s\n", args[0])
		return 1
	}
}

func runFailureCodifierPropose(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("failure-codifier propose", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "proposal only (required; auto-promotion forbidden)")
	repoRoot := fs.String("repo-root", "", "repository root (default: cwd)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	// human-approval gate: 自動昇格を構造的に拒否。--dry-run なしでは提案すら出さない。
	if !*dryRun {
		fmt.Fprintln(stderr, "failure-codifier: --dry-run is required (auto-promotion forbidden)")
		return 2
	}

	root := *repoRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "failure-codifier: %v\n", err)
			return 1
		}
		root = wd
	}

	data, err := failurecodifier.ProposeDryRun(failurecodifier.ProposeOpts{
		ExtractOpts: failurecodifier.ExtractOpts{RepoRoot: root},
	})
	if err != nil {
		fmt.Fprintf(stderr, "failure-codifier: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s\n", data)
	return 0
}
