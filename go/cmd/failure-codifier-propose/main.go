// Command failure-codifier-propose prints dry-run failure-rule.v1 proposals as JSON.
// Invoked by scripts/failure-codifier-propose.sh — never writes SSOT files.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/failurecodifier"
)

func main() {
	dryRun := false
	repoRoot := ""
	flag.BoolVar(&dryRun, "dry-run", false, "proposal only (required)")
	flag.StringVar(&repoRoot, "repo-root", "", "repository root")
	flag.Parse()

	if !dryRun {
		fmt.Fprintln(os.Stderr, "failure-codifier-propose: --dry-run is required (auto-promotion forbidden)")
		os.Exit(2)
	}
	if repoRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		repoRoot = wd
	}

	if err := failurecodifier.WriteProposalsStdout(failurecodifier.ProposeOpts{
		ExtractOpts: failurecodifier.ExtractOpts{RepoRoot: repoRoot},
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
