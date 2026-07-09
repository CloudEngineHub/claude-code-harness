package main

import (
	"fmt"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/plans"
)

func runPlans(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: harness plans <check-deps> [Plans.md]")
		os.Exit(1)
	}
	switch args[0] {
	case "check-deps":
		path := "Plans.md"
		if len(args) > 1 {
			path = args[1]
		}
		tasks, err := plans.ParseFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", path, err)
			os.Exit(1)
		}
		violations := plans.CheckDoneDependencies(tasks)
		if len(violations) > 0 {
			fmt.Fprintln(os.Stderr, plans.FormatDependencyViolations(violations))
			os.Exit(1)
		}
		fmt.Printf("Plans dependency check OK: %s (%d tasks)\n", path, len(tasks))
	default:
		fmt.Fprintf(os.Stderr, "Unknown plans subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
