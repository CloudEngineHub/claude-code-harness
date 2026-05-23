# Cursor Adapter Candidate

Status: candidate evidence boundary
Checked at: 2026-05-22 JST
Phase: `Plans.md` 73.1.6

## Conclusion

Cursor remains `candidate`.

Harness has Cursor handoff integration, but it does not have verified Cursor
adapter support. The current Cursor docs and Superpowers files are useful
adapter-shape evidence only.

## Evidence

Harness evidence:

- `docs/CURSOR_INTEGRATION.md` is PM handoff: Cursor plans/reviews/releases,
  Claude Code Harness implements/verifies.
- `spec.md`, `docs/tool-capability-matrix.md`, and
  `docs/bootstrap-routing-contract.md` keep Cursor as `candidate`.

Superpowers evidence:

- `/Users/tachibanashuuta/LocalWork/Code/superpowers/.cursor-plugin/plugin.json`
  points to `skills`, `agents`, `commands`, and `hooks`.
- `/Users/tachibanashuuta/LocalWork/Code/superpowers/hooks/hooks-cursor.json`
  contains a `sessionStart` hook shape.

Official/current docs checked:

- `https://docs.cursor.com/en/context`
- `https://cursor.com/docs/plugins`
- `https://cursor.com/cn/marketplace/hooks/sessionstart`

Observed from current docs/search evidence: Cursor rules include `.cursor/rules`
and `AGENTS.md`, and marketplace/plugin examples exist. Direct docs extraction
was limited because the docs redirect to a SPA shell in this environment.

## Boundary

`not_observed != absent`: missing Cursor runtime smoke is not proof that Cursor
cannot support Harness. It is proof that Harness must not claim support yet.

Do not add `.cursor-plugin/plugin.json`, Cursor rules, or Cursor hooks as public
adapter support until the same phase also adds setup/preflight tests and a
workflow smoke artifact.

## Promotion Conditions

Cursor can move beyond `candidate` only after:

- current official docs are captured with extractable evidence,
- a Harness-specific Cursor adapter route is added,
- setup or preflight consumes that route,
- a workflow smoke proves `harness-plan` or equivalent routing,
- README/onboarding wording still separates handoff integration from adapter
  support.
