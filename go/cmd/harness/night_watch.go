package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/nightwatch"
)

func runNightWatch(args []string) {
	os.Exit(runNightWatchCommand(args, os.Stdout, os.Stderr))
}

func runNightWatchCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness night-watch report [--dry-run] [--repo-root <dir>]")
		return 1
	}
	switch args[0] {
	case "report":
		return runNightWatchReport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown night-watch subcommand: %s\n", args[0])
		return 1
	}
}

func runNightWatchReport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("night-watch report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dryRun := fs.Bool("dry-run", false, "emit schema-valid report without side effects")
	repoRoot := fs.String("repo-root", "", "repository root (default: cwd)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	root := *repoRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "night-watch: %v\n", err)
			return 1
		}
		root = wd
	}

	report, err := nightwatch.BuildReport(nightwatch.BuildReportOptions{
		RepoRoot: root,
		DryRun:   *dryRun,
	})
	if err != nil {
		fmt.Fprintf(stderr, "night-watch: %v\n", err)
		return 1
	}

	data, err := json.Marshal(report)
	if err != nil {
		fmt.Fprintf(stderr, "night-watch: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s\n", data)
	return 0
}
