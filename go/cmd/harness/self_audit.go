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

func runSelfAudit(args []string) {
	os.Exit(runSelfAuditCommand(args, os.Stdout, os.Stderr))
}

func runSelfAuditCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness self-audit hooks --file <path>")
		return 1
	}
	switch args[0] {
	case "hooks":
		return runSelfAuditHooksCommand(args[1:], stdout, stderr)
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
