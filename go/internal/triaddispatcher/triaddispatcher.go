// Package triaddispatcher routes natural-language sub-tasks to claude/codex/cursor
// backends. It integrates harness-mem read layer (Phase 95.2.3).
package triaddispatcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezingmem"
)

type Backend string

const (
	BackendClaude Backend = "claude"
	BackendCodex  Backend = "codex"
	BackendCursor Backend = "cursor"
)

type Dispatch struct {
	Backend            Backend
	Reason             string
	SimilarPastResults []breezingmem.SimilarPastDecision
}

// DispatchFor routes a natural-language sub-task to a backend using resolver
// defaults plus optional harness-mem similar past dispatch hints (fail-open).
func DispatchFor(ctx context.Context, mem *breezingmem.Client, resolverBackend Backend, project, subtask string) Dispatch {
	similar := []breezingmem.SimilarPastDecision{}
	if mem != nil {
		similar = mem.SearchSimilar(ctx, project, subtask)
	}

	reason := "resolved via impl-backend defaults"
	if majority, count, total := majorityBackend(similar); majority != "" && count > 1 {
		reason = fmt.Sprintf(
			"resolved via impl-backend defaults; past similar dispatches favored %s (%d/%d)",
			majority, count, total,
		)
	}

	return Dispatch{
		Backend:            resolverBackend,
		Reason:             reason,
		SimilarPastResults: similar,
	}
}

func majorityBackend(items []breezingmem.SimilarPastDecision) (backend string, count int, total int) {
	counts := map[string]int{}
	for _, item := range items {
		decision := strings.ToLower(strings.TrimSpace(item.Decision))
		if !isValidBackend(decision) {
			continue
		}
		counts[decision]++
		total++
	}
	for candidate, n := range counts {
		if n > count {
			backend = candidate
			count = n
		}
	}
	return backend, count, total
}

func isValidBackend(value string) bool {
	switch Backend(value) {
	case BackendClaude, BackendCodex, BackendCursor:
		return true
	default:
		return false
	}
}
