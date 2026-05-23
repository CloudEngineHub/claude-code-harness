# Codex App Smoke Boundary

Status: candidate evidence boundary
Checked at: 2026-05-22 JST
Phase: `Plans.md` 73.1.4

## Conclusion

Codex app remains `candidate`.

Codex CLI direct plugin smoke can prove the CLI marketplace/install surface, but
it does not prove Codex app behavior. App proof needs its own artifact because
the app may differ in worktree handling, sandbox behavior, UI handoff, and
thread/runtime state.

## Current Evidence

- Local CLI help observed `codex plugin marketplace add`, `codex plugin list`,
  and `codex plugin add`.
- Local isolated CLI smoke is allowed to use temporary `HOME` and `CODEX_HOME`.
- Phase 74 adds CI-gated Codex CLI direct plugin marketplace/install smoke using
  an isolated `CODEX_HOME`.
- No Codex app worktree, UI handoff, screenshot, transcript, or app-specific
  sandbox proof was captured in this task.

## Required App Proof Artifact

A future Codex app smoke artifact must record:

- Codex app version or build identifier.
- Selected environment and worktree path.
- Sandbox/profile shown in the app.
- Whether the app sees the installed Harness plugin or skills.
- First prompt transcript or screenshot evidence.
- Success or failure result.
- Explicit statement that the result is not inferred from Codex CLI help output.

## Boundary

`not_observed != absent`: missing app evidence means app behavior is not observed
in the current artifact set. It does not prove the app cannot support Harness.

Until this artifact exists, Codex app must stay `candidate` and onboarding must
not describe it as supported or as equivalent to Codex CLI.
