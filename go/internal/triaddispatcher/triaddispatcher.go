// Package triaddispatcher routes natural-language sub-tasks to claude/codex/cursor
// backends. It integrates harness-mem read layer (Phase 95.2.3).
package triaddispatcher

import (
	"context"

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

// DispatchFor is a RED-phase stub; implementation follows in GREEN.
func DispatchFor(ctx context.Context, mem *breezingmem.Client, resolverBackend Backend, project, subtask string) Dispatch {
	_ = ctx
	_ = mem
	_ = project
	_ = subtask
	return Dispatch{Backend: resolverBackend}
}
