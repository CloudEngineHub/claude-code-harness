# Project Spec And Plans SSOT Workflow

Plans.md is the task ledger. `spec.md` is the product contract.

This distinction matters because a task can be well written while the project
meaning is still vague. If the product behavior, domain terms, data ownership,
API contract, security boundary, or non-goal is not written anywhere stable,
implementation can drift even when every Plans.md task is completed.

`/harness-plan` therefore produces co-required planning output for two
contracts:

- `spec.md` product contract: what must stay true.
- `Plans.md` task contract: what to do, how to prove it, and current status.

Precedence stays: `spec.md` > sub-spec > `Plans.md`.
Co-required output means the planning result must include both the spec result
and the task result. It does not make Plans.md equal to or higher than the
product contract.

The agent drafts the spec delta from repo evidence, memory, tests, and the user
request. Do not make the workflow look like the user must hand-write a spec
from scratch before planning can start.

## Default Location

Use the root `spec.md` first.

For this repository, and for any consumer repository that has a root
`spec.md`, that root file is the product contract:

- `spec.md`

Only when the consumer repository has no root `spec.md`, fall back to an
existing project-level specification if one already exists:

- `docs/spec/00-project-spec.md`
- `docs/ARCHITECTURE.md`
- `docs/HANDOFF.md`
- `docs/oem/PROJECT_COMPASS.md`
- a clearly named product or domain spec under `docs/specs/`

If no root `spec.md` and no stronger local convention exists in a consumer
project, create:

```text
docs/spec/00-project-spec.md
```

For this repository, `spec.md` is the root product contract. Scoped documents
such as `docs/architecture/hokage-core.md` and `go/SPEC.md` are sub-specs and
do not replace the root contract.

## Harness-Plan Output Contract

Every `create` output and every product-impacting `add` output must include the
spec result and the task result.

Required pair:

- `Spec delta`: include this when product behavior, API, data model, permission,
  billing, integration, tenant boundary, support claim, or other product
  contract changes.
- `Spec skip reason`: include this when the existing product contract already
  covers the request or when the task is docs-only / mechanical.
- `Plans.md`: include task rows or task-contract text with DoD, dependencies,
  status marker, and evidence expectations.

Harness generates `Spec delta` and `Spec skip reason`; the consumer approves or edits them.
The consumer is not expected to write the spec from scratch.

`create` and product-impacting `add` must read root `spec.md` every time and
produce the spec result before generating tasks. A missing search result,
failed memory lookup, or unobserved file is not proof that no spec exists:

```text
not_observed != absent
```

If the task is docs-only or mechanical, still preserve `Spec skip reason` in
task context or sprint contract so later workers know the skip was intentional.

## When To Create Or Update It

Create or update a spec SSOT before implementation when any of these are true:

- The task introduces or changes user-visible product behavior.
- The task changes API, data model, permissions, billing, tenant boundaries, or
  integration contracts.
- Multiple implementation choices are plausible and would create different
  product behavior.
- A reviewer, worker, or user has already seen implementation drift from unclear
  requirements.
- The task spans several modules and needs shared terms or invariants.
- Plans.md contains what to do, but not what "correct" means for the project.

## When To Skip

Do not create a new spec for purely mechanical work:

- typo fixes
- formatting
- dependency bumps without behavior change
- local CI/test repair with no product decision
- README/CHANGELOG-only updates
- narrow refactors that preserve existing behavior and have clear tests

If skipping could surprise a later implementer, write the skip reason in the
task context or sprint contract.

## Minimum Content

Keep the first spec short. It only needs enough structure to prevent drift.

```markdown
# Project Spec

## Purpose
What this project is for, in one paragraph.

## Users And Workflows
Who uses it, and the main workflows they expect.

## Core Rules
The product rules that implementation must not violate.

## Data And Contracts
Important data shapes, API contracts, integrations, and ownership boundaries.

## Non-Goals
Things the project intentionally does not do.

## Open Decisions
Unknowns that must be resolved before implementation can rely on them.

## Links
- Plans.md task or phase:
- Related briefs:
- Related decisions:
```

## Relationship To Plans.md

Plans.md should link to the spec when a task depends on project-level behavior.

Example:

```markdown
| 4.2 | Add tenant invite flow (spec: docs/spec/00-project-spec.md#tenant-rules) | invite API rejects cross-tenant roles in tests | 4.1 | cc:TODO |
```

The spec does not replace DoD. DoD still says how the task is judged complete.
The spec says what the implementation must stay consistent with.

## Review Rule

Reviewers should treat a direct contradiction of the spec SSOT as a major issue.
If the spec is missing and the task needed one, that is a planning gap, not a
reason to invent behavior during implementation.
