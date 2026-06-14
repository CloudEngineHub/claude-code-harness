package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/nightwatch"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "emit schema-valid report without side effects")
	repoRoot := flag.String("repo-root", "", "repository root (default: cwd)")
	flag.Parse()

	root := *repoRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "night-watch-report: %v\n", err)
			os.Exit(1)
		}
		root = wd
	}

	report, err := nightwatch.BuildReport(nightwatch.BuildReportOptions{
		RepoRoot: root,
		DryRun:   *dryRun,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "night-watch-report: %v\n", err)
		os.Exit(1)
	}

	data, err := json.Marshal(report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "night-watch-report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", data)
}
