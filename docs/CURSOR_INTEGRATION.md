# Cursor Integration

Last updated: 2026-07-19

Scope: **PM handoff integration** and **supported Cursor adapter route**. Adapter
evidence, bootstrap route, and promotion gates live in
`docs/research/cursor-adapter-candidate.md` and `.cursor/AGENTS.md`.

## Goal

Use Cursor as the PM side and Claude Code Harness as the implementation side
without losing task ownership or verification discipline.

When the operator wants to stay inside Cursor for Harness workflows, use the
adapter route under `.cursor-plugin/` and `.cursor/AGENTS.md`. Cursor is
`supported` after live H4 (2026-07-17) and H7 release-preflight fail-closed
(2026-07-19), with harness-side containment (below).

## Containment disclosure

Cursor adapter support is `supported`, but **containment is harness-side**, not
Cursor-native sandboxing.

(a) **No traditional sandbox:** `cursor-agent` has no filesystem jail comparable
to Codex `--sandbox`. File edits are **not** confined by `--sandbox` and can
write outside the workspace.

(b) **Allowlists are best-effort:** Cursor permission allowlists are convenience
controls, not a security boundary (per Cursor's own documentation).

(c) **Harness-side containment is the effective boundary:** dedicated `.git`
worktree isolation, Lead diff review, cherry-pick through R01–R13 policy
engine, plus `.cursorignore` for secret reads.

(d) **No bypass modes:** `--force` / "Run Everything" is never used in Harness
Cursor workflows.

See also `docs/known-limitations.md`.

## Role Split

- Cursor: planning, review sign-off, release judgment (handoff mode)
- Claude Code Harness: implementation, local verification, handoff back to PM

This split works best when both sides share the same repository, branch, and
`Plans.md`.

## Recommended Workflow

### 1. Plan in Cursor

Use the Cursor-side command templates to create or refine `Plans.md`:

- `templates/cursor/commands/start-session.md`
- `templates/cursor/commands/plan-with-cc.md`
- `templates/cursor/commands/handoff-to-claude.md`
- `templates/cursor/commands/review-cc-work.md`

Adapter route (same repo, `supported` tier with harness-side containment):

- Read `.cursor/AGENTS.md` for plan/work/review routing guidance
- Invoke `harness-plan`, `harness-work`, or `harness-review` skills when available

### 2. Implement in Claude Code

Inside Claude Code, run the Harness loop:

```bash
/harness-setup
/harness-plan
/harness-work
/harness-review
```

Use `/harness-work all` only after the plan is approved and only if you are
comfortable with the evidence-pack contract described in
`docs/evidence/work-all.md`.

### 3. Handoff Back to Cursor

For a PM-style return path, use:

```bash
/handoff-to-cursor
```

If you prefer the unified release path, `/harness-release handoff` can also be
used when the implementation and review loop is complete.

## Plans.md Markers

The safest shared contract is:

- `pm:依頼中` / `cc:TODO`
- `cc:WIP`
- `cc:完了`
- `pm:確認済`

Cursor should own PM markers. Claude Code should own worker markers.

## Guardrails

- Do not let Cursor and Claude Code edit the same task block at the same time.
- Keep one source of truth for acceptance criteria: `Plans.md`.
- Treat production deployment judgment as the PM side's responsibility.
- If the worker side fails the same issue three times, stop and escalate instead
  of widening fallback logic.
- Do not claim Claude-identical FS jail or full-tool PreToolUse parity for Cursor.

## Minimum Sanity Check

Before starting a shared session, confirm:

1. Both tools point at the same git branch.
2. Both tools can see the same `Plans.md`.
3. The implementation request includes acceptance criteria and expected
   verification commands.
4. The PM side knows whether release is in or out of scope.
5. Support wording treats Cursor as `supported` with harness-side containment only.

## Adapter Verification

Static contract (required):

```bash
bash tests/test-cursor-adapter-candidate.sh
```

This verifies manifest shape, AGENTS bootstrap routing, and supported-tier
wording with containment caveats. It does not prove full Breezing multitask parity.
