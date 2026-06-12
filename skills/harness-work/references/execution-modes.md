# Execution Modes

`harness-work` chooses the lightest execution mode that still preserves review
and validation.

## Shared Preflight

1. Read `Plans.md` and identify the selected task set.
2. Stop if the task table lacks `Task`, `DoD`, `Depends`, or `Status`.
3. Check whether a project spec SSOT exists when product behavior can drift.
   Prefer existing project-level docs, then `docs/spec/00-project-spec.md`.
4. If the task changes product behavior, API, data model, permissions, billing,
   integrations, or tenant boundaries and no stable spec exists, create or
   update the spec before implementation.
5. Skip spec creation only for mechanical work such as typo, formatting,
   dependency bump, docs-only, or behavior-preserving refactor tasks. Record
   the skip reason in the task context or sprint contract.
6. Resolve helper scripts through `HARNESS_PLUGIN_ROOT`, not the caller
   project's `scripts/` directory.
7. Mark only the selected task as `cc:WIP`.
8. Generate and approve a sprint contract before implementation when the task
   needs reviewable DoD checks.

## Solo

Use for one task. The parent session implements directly, validates, runs the
review loop, commits unless `--no-commit` is set, and marks `Plans.md`
`cc:完了 [hash]`.

## Parallel

Use for two or three independent tasks, or when `--parallel N` is explicit.
Workers may use isolated worktrees when file ownership can conflict. The Lead
still owns final integration and status updates.

## Codex

Use only when `--codex` is explicit. Delegate implementation to the Codex
companion entrypoint:

```bash
bash "${HARNESS_PLUGIN_ROOT}/scripts/codex-companion.sh" task --write "task"
```

Validate the result locally. Codex output is not accepted until the normal
review loop approves it.

## Breezing

Use for four or more tasks, or when `--breezing` is explicit. Lead coordinates
Workers, Advisor, and Reviewer while preserving the implementation/review
boundary.

Codex-native Breezing reads this flow through `spawn_agent`, `send_input`,
`wait_agent`, and `close_agent` rather than Claude Code `Agent` /
`SendMessage` pseudo-code.

## Lane and Stage Contract

Sprint contract generation passes `spec_path`, `lane`, `stage`, and evidence
fields to Worker / Reviewer. See `skills/harness-work/SKILL.md`「Sprint
Contract」for the full field list.

### Stage gate (5 stages)

| stage | 目的 |
|-------|------|
| `research` | 現状調査・evidence 収集。未取得データは `unknown` と報告 |
| `plan` | scope / DoD / lane を Plans に freeze |
| `impl` | TDD Red→Green 実装。`[tdd:required]` は `tdd_red_log` 必須 |
| `review` | `review_artifact`（`APPROVE` / `REQUEST_CHANGES`）を contract に載せる |
| `closeout` | `pr_closeout`（`base_ref` / `head_ref` / evidence pack）を載せる |

### Lane: what to lighten vs what to keep

| lane | 軽くする項目 | 省けない項目 |
|------|-------------|-------------|
| `fast` | full review（major-only または advisory で可）、PR body の詳細、release preflight | `spec_path`、unknown data contract（`not_observed != absent`）、focused checks（`runtime_validation` / `checks`）、`tdd_red_log` または `skip_tdd_reason`（`[tdd:required]` 時） |
| `gate` | —（軽量化なし） | spec alignment、TDD when required、major-only or full review、re-review until clean、`research_evidence` |
| `release` | —（軽量化なし） | version/tag/GitHub Release/CI 検証、`pr_closeout` + release preflight、full evidence pack |

`[tdd:required]` タスクは lane に関わらず、`tdd_red_log` または明示 `skip_tdd_reason` が sprint contract に無い限り完了扱いにしない。
