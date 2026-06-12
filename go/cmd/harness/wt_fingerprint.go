package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/wtfingerprint"
)

const wtFingerprintDiffExitChanged = 2

// runWt handles `harness wt <subcommand>`.
func runWt(args []string) {
	os.Exit(runWtCommand(args, os.Stdout, os.Stderr))
}

func runWtCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness wt fingerprint <capture|diff>")
		return 1
	}
	switch args[0] {
	case "fingerprint":
		return runWtFingerprint(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown wt subcommand: %s\n", args[0])
		return 1
	}
}

func runWtFingerprint(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness wt fingerprint <capture|diff>")
		return 1
	}
	switch args[0] {
	case "capture":
		return runWtFingerprintCapture(args[1:], stdout, stderr)
	case "diff":
		return runWtFingerprintDiff(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown fingerprint subcommand: %s\n", args[0])
		return 1
	}
}

func runWtFingerprintCapture(args []string, stdout, stderr io.Writer) int {
	var outputPath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "harness wt fingerprint capture: --output requires a path")
				return 1
			}
			i++
			outputPath = args[i]
		case "--help", "-h":
			fmt.Fprintln(stdout, "Usage: harness wt fingerprint capture --output <path>")
			return 0
		default:
			fmt.Fprintf(stderr, "harness wt fingerprint capture: unknown flag %q\n", args[i])
			return 1
		}
	}
	if outputPath == "" {
		fmt.Fprintln(stderr, "harness wt fingerprint capture: --output is required")
		return 1
	}

	snap, err := wtfingerprint.Capture(nil)
	if err != nil {
		fmt.Fprintf(stderr, "harness wt fingerprint capture: %v\n", err)
		return 1
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "harness wt fingerprint capture: marshal: %v\n", err)
		return 1
	}
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		fmt.Fprintf(stderr, "harness wt fingerprint capture: write %s: %v\n", outputPath, err)
		return 1
	}
	return 0
}

func runWtFingerprintDiff(args []string, stdout, stderr io.Writer) int {
	var beforePath, afterPath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--before":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "harness wt fingerprint diff: --before requires a path")
				return 1
			}
			i++
			beforePath = args[i]
		case "--after":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "harness wt fingerprint diff: --after requires a path")
				return 1
			}
			i++
			afterPath = args[i]
		case "--help", "-h":
			fmt.Fprintln(stdout, "Usage: harness wt fingerprint diff --before <path> --after <path>")
			return 0
		default:
			fmt.Fprintf(stderr, "harness wt fingerprint diff: unknown flag %q\n", args[i])
			return 1
		}
	}
	if beforePath == "" || afterPath == "" {
		fmt.Fprintln(stderr, "harness wt fingerprint diff: --before and --after are required")
		return 1
	}

	before, err := loadSnapshotFile(beforePath)
	if err != nil {
		fmt.Fprintf(stderr, "harness wt fingerprint diff: %v\n", err)
		return 1
	}
	after, err := loadSnapshotFile(afterPath)
	if err != nil {
		fmt.Fprintf(stderr, "harness wt fingerprint diff: %v\n", err)
		return 1
	}

	changed := wtfingerprint.Diff(before, after)
	if len(changed) == 0 {
		return 0
	}
	for _, path := range changed {
		fmt.Fprintln(stderr, path)
	}
	return wtFingerprintDiffExitChanged
}

func loadSnapshotFile(path string) (wtfingerprint.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return wtfingerprint.Snapshot{}, fmt.Errorf("read %s: %w", path, err)
	}
	var snap wtfingerprint.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return wtfingerprint.Snapshot{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if snap.Files == nil {
		snap.Files = make(map[string]string)
	}
	return snap, nil
}
