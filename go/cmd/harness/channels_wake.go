package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Chachamaru127/claude-code-harness/go/internal/channelswake"
)

func runChannelsWake(args []string) {
	os.Exit(runChannelsWakeCommand(args, os.Stdout, os.Stderr))
}

func runChannelsWakeCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: harness channels-wake check")
		return 1
	}
	switch args[0] {
	case "check":
		return writeChannelsWakeCheck(stdout)
	default:
		fmt.Fprintf(stderr, "Unknown channels-wake subcommand: %s\n", args[0])
		return 1
	}
}

func writeChannelsWakeCheck(stdout io.Writer) int {
	result, code := runChannelsWakeCheck()
	data, _ := json.Marshal(result)
	fmt.Fprintf(stdout, "%s\n", data)
	return code
}

func runChannelsWakeCheck() (channelswake.Result, int) {
	return channelswake.CheckWithExit()
}
