package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/Chachamaru127/claude-code-harness/go/internal/impactscore"
)

type impactScoreOutput struct {
	ImpactScore int  `json:"impact_score"`
	HardStop    bool `json:"hard_stop"`
}

func runImpactScore(args []string) {
	filesChanged := 0
	linesChanged := 0
	floorCategory := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--files-changed":
			i++
			filesChanged = parseImpactScoreInt(args, i, "--files-changed")
		case "--lines-changed":
			i++
			linesChanged = parseImpactScoreInt(args, i, "--lines-changed")
		case "--floor-category":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "impact-score: --floor-category requires a value")
				os.Exit(1)
			}
			floorCategory = args[i]
		default:
			fmt.Fprintf(os.Stderr, "impact-score: unknown argument: %s\n", args[i])
			os.Exit(1)
		}
	}

	score := impactscore.Compute(impactscore.Inputs{
		FloorCategory: floorCategory,
		FilesChanged:  filesChanged,
		LinesChanged:  linesChanged,
	})
	hardStop := impactscore.IsHardStop(score)

	out := impactScoreOutput{
		ImpactScore: score,
		HardStop:    hardStop,
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "impact-score: encode output: %v\n", err)
		os.Exit(1)
	}
	if hardStop {
		os.Exit(2)
	}
}

func parseImpactScoreInt(args []string, index int, flag string) int {
	if index >= len(args) {
		fmt.Fprintf(os.Stderr, "impact-score: %s requires a value\n", flag)
		os.Exit(1)
	}
	value, err := strconv.Atoi(args[index])
	if err != nil {
		fmt.Fprintf(os.Stderr, "impact-score: %s must be an integer: %s\n", flag, args[index])
		os.Exit(1)
	}
	return value
}
