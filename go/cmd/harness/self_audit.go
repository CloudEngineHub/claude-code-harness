package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/selfaudit"
)

type selfAuditHooksOutput struct {
	Known          int                   `json:"known"`
	Unknown        int                   `json:"unknown"`
	UnknownEntries []selfaudit.HookEntry `json:"unknown_entries"`
}

type selfAuditBaselineOutput struct {
	OK             bool   `json:"ok"`
	CurrentSHA256  string `json:"current_sha256"`
	BaselineSHA256 string `json:"baseline_sha256"`
	Reason         string `json:"reason"`
}

func runSelfAudit(args []string) {
	os.Exit(runSelfAuditCommand(args, os.Stdout, os.Stderr))
}

func runSelfAuditCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness self-audit hooks --file <path>")
		fmt.Fprintln(stderr, "       harness self-audit baseline --settings <path> --baseline <path>")
		return 1
	}
	switch args[0] {
	case "hooks":
		return runSelfAuditHooksCommand(args[1:], stdout, stderr)
	case "baseline":
		return runSelfAuditBaselineCommand(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown self-audit subcommand: %s\n", args[0])
		return 1
	}
}

func runSelfAuditHooksCommand(args []string, stdout, stderr io.Writer) int {
	var filePath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "harness self-audit hooks: --file requires a value")
				return 2
			}
			i++
			filePath = args[i]
		default:
			fmt.Fprintf(stderr, "harness self-audit hooks: unknown flag %q\n", args[i])
			return 1
		}
	}
	if filePath == "" {
		fmt.Fprintln(stderr, "harness self-audit hooks: --file is required")
		return 1
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit hooks: read %s: %v\n", filePath, err)
		return 2
	}

	report, err := selfaudit.Audit(data)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit hooks: audit: %v\n", err)
		return 2
	}

	out := selfAuditHooksOutput{
		Known:          len(report.Known),
		Unknown:        len(report.Unknown),
		UnknownEntries: report.Unknown,
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit hooks: marshal: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "%s\n", encoded)

	if len(report.Unknown) > 0 {
		return 1
	}
	return 0
}

func runSelfAuditBaselineCommand(args []string, stdout, stderr io.Writer) int {
	var settingsPath, baselinePath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--settings":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "harness self-audit baseline: --settings requires a value")
				return 1
			}
			i++
			settingsPath = args[i]
		case "--baseline":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "harness self-audit baseline: --baseline requires a value")
				return 1
			}
			i++
			baselinePath = args[i]
		default:
			fmt.Fprintf(stderr, "harness self-audit baseline: unknown flag %q\n", args[i])
			return 1
		}
	}
	if settingsPath == "" || baselinePath == "" {
		fmt.Fprintln(stderr, "harness self-audit baseline: --settings and --baseline are required")
		return 1
	}

	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit baseline: read settings %s: %v\n", settingsPath, err)
		return 1
	}

	baseline, loaded, err := selfaudit.LoadBaseline(baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit baseline: read baseline %s: %v\n", baselinePath, err)
		return 1
	}
	if !loaded {
		fmt.Fprintf(stderr, "harness self-audit baseline: baseline file not found: %s\n", baselinePath)
		return 1
	}

	currentHash, _, err := selfaudit.ComputeDenyHash(settingsData)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit baseline: compute current hash: %v\n", err)
		return 1
	}

	ok, reason, err := selfaudit.VerifyDenyNotRegressed(baseline, settingsData)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit baseline: verify: %v\n", err)
		return 1
	}

	out := selfAuditBaselineOutput{
		OK:             ok,
		CurrentSHA256:  currentHash,
		BaselineSHA256: baseline.CanonicalSHA256,
		Reason:         reason,
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(stderr, "harness self-audit baseline: marshal: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "%s\n", encoded)

	if !ok {
		return 2
	}
	return 0
}
