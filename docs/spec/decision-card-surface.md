# Spec Sub-Spec: decision-card-surface

This sub-spec is part of the `spec.md` product contract. SSOT order is `spec.md` core > `docs/spec/*` sub-specs > `Plans.md`.

## Decision Card Surface Contract

Decision Cards surface human judgment during breezing with structured options,
impact scoring, and optional past-decision context from harness-mem search.
This contract productizes Phase 95 Decision Card UI and mem read layer on top of
Breezing Brief Contract (`judgment-card.v1` v0 issuance rules) and Tri-Tool
Parallel Collaboration Contract (runtime hard floor precedence).

### judgment-card.v1 v1 extension

Schema: `templates/schemas/judgment-card.v1.json`. v1 retains all v0 required
fields (`question`, `options`, `recommendation`, `confidence`, `impact`,
`diff_summary`) and adds:

| Field | Shape | Constraints |
|-------|-------|-------------|
| `impact_score` | integer | 0–100 inclusive |
| `similar_past_decisions` | array | max 3 items; each `{summary, decision, outcome, decided_at, mem_id}` (all non-empty strings) |

`impact_score` combines (i) worktree-fingerprint impact (changed file count and
line magnitude within the task worktree) and (ii) distance from the five-category
runtime hard floor (Phase 92.2.1). When any hard-floor category matches,
`impact_score` is **100**; otherwise it scales 0–99 from fingerprint impact
alone.

v0 card JSON without v1 fields validates as v1 (backward compatible input).

### Past decision reference accuracy

When harness-mem is configured and reachable, Decision Card population calls
`harness_mem_search` (or equivalent HTTP search) and returns **exactly up to three**
past decisions ranked by **similarity score** (highest first). Each
`similar_past_decisions[]` entry carries the matched observation summary,
recorded decision, known outcome, `decided_at`, and source `mem_id`.

When mem is `not-configured` or `unreachable`, `similar_past_decisions` is an
empty array (fail-open — card issuance continues without past context).

### Floor precedence over judgment cards

The five-category **runtime action hard floor** (Phase 92.2.1; money/billing,
external send/egress, credential/secret read, production deploy/publish,
destruction outside the task worktree) always takes precedence over Decision
Card surfaces:

- When any hard-floor category matches, Harness hard-stops for the human,
  sets `impact_score=100`, does **not** issue a judgment card, and surfaces
  `HARD_STOP` only.
- Judgment cards apply only to non-floor ambiguity (DoD interpretation, scope,
  trade-offs) — same rule as Breezing Brief Contract floor precedence.
- The pre-merge policy gate (`go/internal/floor`) and runtime hard floor remain
  distinct.

### Fail-open memory behavior (Decision Card read layer)

Brief Composer, Triad Dispatcher (`scripts/resolve-impl-backend.sh` evolution),
and Decision Card render paths that call harness-mem for read/search must follow
the same fail-open tri-state model:

| State | Behavior |
|-------|----------|
| `not-configured` | Silent skip — proceed with empty `similar_past_decisions` |
| `unreachable` | One stderr warning line (`breezing-mem:` or component prefix), then continue |

Mem read state never blocks brief confirmation, worker dispatch, judgment card
render, or aggregation.

### workgraph signal boundary
