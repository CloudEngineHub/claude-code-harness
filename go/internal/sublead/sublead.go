// Package sublead implements the Mode 1 Sub-Lead layer: one lane is decomposed
// into a mini-plan and subtasks are fanned out in parallel via an inner
// breezing.Orchestrator. Results aggregate into a single companion-result.v1
// per lane. Hub-spoke only — subWorkers never receive peer results or channels.
package sublead

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Chachamaru127/claude-code-harness/go/internal/breezing"
	"github.com/Chachamaru127/claude-code-harness/go/internal/companionresult"
)

const resultCarrierPrefix = "companion-result.v1:"

// SubTask is one implementation unit inside a lane mini-plan.
type SubTask struct {
	ID     string
	Prompt string
}

// MiniPlan is the decomposed work for a single lane.
type MiniPlan struct {
	LaneID   string
	SubTasks []SubTask
}

// Planner decomposes a lane into a mini-plan. Production uses an
// orchestrator-spawned headless CLI; tests inject fixtures.
type Planner func(ctx context.Context, laneID, laneSpec string) (MiniPlan, error)

// CommandRunner executes a headless CLI process. Tests inject fakes; production
// shells out via exec.CommandContext.
type CommandRunner func(ctx context.Context, name string, args []string, stdin string) (stdout string, exitCode int, err error)

type subtaskSummary struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Summary string `json:"summary"`
}

type miniPlanJSON struct {
	LaneID   string `json:"lane_id"`
	SubTasks []struct {
		ID     string `json:"id"`
		Prompt string `json:"prompt"`
	} `json:"sub_tasks"`
}

// NewSubLeadWorker returns a WorkerFunc that accepts one lane task, plans
// subtasks, fans them out through an inner Orchestrator, and aggregates into
// one companion-result.v1 envelope for the Lead layer.
func NewSubLeadWorker(planner Planner, subWorker breezing.WorkerFunc, backend string, maxParallel int) breezing.WorkerFunc {
	return func(ctx context.Context, task *breezing.Task) breezing.TaskResult {
		laneID := task.ID
		laneSpec := task.Description

		plan, err := planner(ctx, laneID, laneSpec)
		if err != nil {
			return carryResult(failedLaneResult(backend, laneID, fmt.Sprintf("planner failed: %v", err), 1))
		}
		if len(plan.SubTasks) == 0 {
			return carryResult(failedLaneResult(backend, laneID, "empty mini-plan: no subtasks", 1))
		}

		inner := breezing.NewOrchestrator(subWorker, breezing.WithMaxParallel(maxParallel))
		for _, st := range plan.SubTasks {
			inner.AddTask(&breezing.Task{
				ID:          st.ID,
				Description: st.Prompt,
				AgentType:   "worker",
			})
		}

		if _, err := inner.Run(ctx); err != nil {
			return carryResult(failedLaneResult(backend, laneID, fmt.Sprintf("sub-orchestrator: %v", err), 1))
		}

		byID := inner.Results()
		summaries := make([]subtaskSummary, 0, len(plan.SubTasks))
		allOK := true
		for _, st := range plan.SubTasks {
			tr, ok := byID[st.ID]
			if !ok {
				allOK = false
				summaries = append(summaries, subtaskSummary{
					ID:      st.ID,
					Success: false,
					Summary: "no result recorded by sub-orchestrator",
				})
				continue
			}
			r := resultFromTaskResult(backend, st.ID, tr)
			if !r.Success {
				allOK = false
			}
			summaries = append(summaries, subtaskSummary{
				ID:      st.ID,
				Success: r.Success,
				Summary: r.Summary,
			})
		}

		summaryJSON, err := json.Marshal(summaries)
		if err != nil {
			return carryResult(failedLaneResult(backend, laneID, fmt.Sprintf("aggregate summary marshal: %v", err), 1))
		}

		lane := companionresult.New(backend, laneID)
		lane.Success = allOK
		lane.Summary = string(summaryJSON)
		if allOK {
			lane.ExitCode = 0
		} else {
			lane.ExitCode = 1
		}
		return carryResult(lane)
	}
}

// NewHeadlessCLIPlanner returns a Planner that invokes an orchestrator-spawned
// headless CLI (same harness binary as the Lead) to decompose a lane.
// The CLI must print {"lane_id":"...","sub_tasks":[{"id":"...","prompt":"..."}]}
// to stdout. Parse failure or non-zero exit is fail-loud.
func NewHeadlessCLIPlanner(runner CommandRunner) Planner {
	return func(ctx context.Context, laneID, laneSpec string) (MiniPlan, error) {
		name, err := os.Executable()
		if err != nil {
			name = "harness"
		}
		args := []string{"plan", "decompose", "--lane", laneID}
		stdout, exitCode, err := runner(ctx, name, args, laneSpec)
		if err != nil {
			return MiniPlan{}, fmt.Errorf("headless CLI planner: %w", err)
		}
		if exitCode != 0 {
			return MiniPlan{}, fmt.Errorf("headless CLI planner: exit %d", exitCode)
		}
		return parseMiniPlanJSON(stdout, laneID)
	}
}

func parseMiniPlanJSON(stdout, expectLaneID string) (MiniPlan, error) {
	var raw miniPlanJSON
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &raw); err != nil {
		return MiniPlan{}, fmt.Errorf("headless CLI planner: parse JSON: %w", err)
	}
	if raw.LaneID == "" {
		raw.LaneID = expectLaneID
	}
	if raw.LaneID != expectLaneID {
		return MiniPlan{}, fmt.Errorf("headless CLI planner: lane_id %q, want %q", raw.LaneID, expectLaneID)
	}
	if len(raw.SubTasks) == 0 {
		return MiniPlan{LaneID: raw.LaneID}, nil
	}
	out := MiniPlan{LaneID: raw.LaneID, SubTasks: make([]SubTask, 0, len(raw.SubTasks))}
	for _, st := range raw.SubTasks {
		out.SubTasks = append(out.SubTasks, SubTask{ID: st.ID, Prompt: st.Prompt})
	}
	return out, nil
}

func failedLaneResult(backend, laneID, summary string, exitCode int) companionresult.Result {
	r := companionresult.New(backend, laneID)
	r.Success = false
	r.ExitCode = exitCode
	r.Summary = summary
	return r
}

func resultFromTaskResult(backend, taskID string, tr breezing.TaskResult) companionresult.Result {
	if payload, ok := strings.CutPrefix(tr.CommitHash, resultCarrierPrefix); ok {
		if r, err := companionresult.Parse([]byte(payload)); err == nil {
			return r
		}
	}
	r := companionresult.New(backend, taskID)
	if tr.Err != nil {
		r.Success = false
		r.ExitCode = 1
		r.Summary = tr.Err.Error()
	} else {
		r.Success = true
		r.ExitCode = 0
	}
	return r
}

func carryResult(r companionresult.Result) breezing.TaskResult {
	tr := breezing.TaskResult{
		TaskID:     r.TaskID,
		CommitHash: resultCarrierPrefix + mustMarshal(r),
	}
	if !r.Success {
		tr.Err = fmt.Errorf("lane %s failed (exit %d)", r.TaskID, r.ExitCode)
	}
	return tr
}

func mustMarshal(r companionresult.Result) string {
	b, err := r.Marshal()
	if err != nil {
		return ""
	}
	return string(b)
}
