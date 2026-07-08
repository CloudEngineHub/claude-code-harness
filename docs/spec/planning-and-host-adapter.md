# Spec Sub-Spec: planning-and-host-adapter

This sub-spec is part of the `spec.md` product contract. SSOT order is `spec.md` core > `docs/spec/*` sub-specs > `Plans.md`.

## Planning Surface Contract

`/harness-plan` owns co-required planning output between the spec.md product contract and Plans.md task contract.
It must not behave as a Plans.md-only generator, and it must not flatten the
precedence order. The order remains `spec.md > sub-spec > Plans.md`:
`spec.md` is the product contract, sub-specs refine scoped domains, and
Plans.md is the task ledger.

`/harness-plan create` and product-impacting `/harness-plan add` must produce
both:

- `Spec delta` when the product contract changes, with the root `spec.md` or
  fallback spec path and the rules being added or changed.
- `Spec skip reason` when the task does not change the product contract, with
  the reason preserved in task context or the sprint contract.
- `Plans.md` task contract rows with DoD, dependencies, status, and evidence
  expectations.

For `create` and product-impacting `add`, agents must read the root `spec.md`
and produce the spec result before producing task rows. Only when a consumer
repository has no root `spec.md` may the agent fall back to an existing project
spec or `docs/spec/00-project-spec.md`.

The agent drafts the spec delta from the request, current repo evidence, memory,
and tests. The user is not expected to write a product spec from scratch before
Harness can plan. If the correct delta is ambiguous, the agent should offer the
smallest decision branch and keep unverified facts as `unknown` or
`not observed`.

Harness generates the spec result. Consumers approve or edit `Spec delta` /
`Spec skip reason`; they are not expected to write the spec from scratch.

Non-trivial planning must be team-validated before it becomes implementation
work. A request is non-trivial when it spans multiple tasks, files, sessions,
or changes product behavior, APIs, data models, permissions, billing, external
integrations, distribution, or security posture. For those requests,
`/harness-plan` must use TeamAgent or sub-agent perspectives when available.
If the runtime cannot spawn sub-agents, the plan must explicitly say
`サブエージェント未使用` and run the same checks in separated sections.
The plan must include `team_validation_mode`, one of
`not_required_lightweight`, `native`, `subagent`, `manual-pass`, or
`unavailable`. Lightweight work may use `not_required_lightweight`.
Non-trivial work must use `native`, `subagent`, or `manual-pass`; `unavailable`
cannot be marked Required.

Product, Architecture, Security, QA, and Skeptic are validation perspectives,
not required runtime `agent_type` names. Harness should pass those perspectives
to the available TeamAgent or Task mechanism rather than requiring arbitrary
agent spawning.

Every non-trivial plan must show:

- alignment with root `spec.md`, applicable sub-specs, and `Plans.md`,
- a project-scoped harness-mem / harness-recall / repo-memory wheel check,
- product-fit validation against the primary operator workflow,
- security validation for permissions, secrets, external sends, supply chain,
  branch protection, and release gates,
- works-in-practice validation that maps the plan to test, smoke, CI, review,
  and release or closeout evidence.

If any of those gates cannot pass, the plan must not mark the work Required
until it adds a spike, narrows scope, updates the product contract, or rejects
the idea.

Security validation must not require reading secrets. If a plan would need to
inspect `.env`, tokens, private keys, or customer data, it must stop at a Risk
Gate and use non-secret evidence such as guardrail rules, config shape, audit
metadata, tests, or CI/GitHub state.

## Hokage Core And Host Adapter Boundary

Harness is a single `harness` CLI binary, not a host plugin. The value-bearing
Go core is the reusable kernel; host adapters are generated shims, not
hand-written source.

The kernel is the irreducible IP and is host-agnostic. It is the set of stdlib-only
Go packages plus the embedded prompt pack:

- `go/internal/policy`: the R01-R13 guardrail rule engine (first-match-wins,
  stdlib + regexp only) plus the deny-surface self-audit baseline.
- `go/internal/gitport`: the single git-exec seam every package routes through.
- `go/internal/plans`: the `Plans.md` parser and marker tally.
- `go/internal/state`: the trimmed session/task state store.
- `go/internal/harnessmem`: the membridge to harness-mem.
- `go/internal/promptpack`: the embedded `work`/`plan`/`review`/`release`
  workflow contracts — the single source of truth for skills and agents.

The kernel must not depend on any host's hook event names, host-only tools,
host config shape, or marketplace packaging details. It defines workflow intent,
user-facing triggers, inputs/outputs, required evidence, acceptance criteria,
review/completion rules, and the R01-R13 adjudication surface. Host-specific
mechanics are derived from it, never hand-maintained alongside it.

A single descriptor, `hosts.toml`, holds every host difference: per host, the
native pre-action hook event name, the hook config path, the matcher, the deny
mechanism, the transport, and the model/effort. `harness gen` reads that one
descriptor and `go/internal/hostgen` emits each host's hook config. Host
adapters are therefore build artifacts (`harness gen` output), not tracked
source that drifts.

The boundary is one rule: **the kernel adjudicates; every generated host hook
routes each pre-action to `bin/harness hook pre-tool`.** A host adapter owns only
the thin shim that wires its native hook to that entrypoint and surfaces the
generated skills/agents; it never re-implements a rule engine and never owns a
parallel guardrail.

| Host | Generated shim owns | Must Not Claim |
|------|---------------------|----------------|
| Claude Code | `.claude-plugin/hooks.json` `PreToolUse` entry routing to `bin/harness hook pre-tool`, generated skill/agent surface, manifest version | A separate guardrail engine; that its hook config is hand-authored source |
| Codex | generated `.codex/hooks.json` `PreToolUse` entry routing to `bin/harness hook pre-tool --host codex`, generated skill/agent surface | A divergent rule set; that the companion path is the enforcement boundary |
| Cursor | generated `.cursor/hooks.json` `preToolUse` entry routing to `bin/harness hook pre-tool --host cursor`, generated skill/agent surface | A public `supported` claim; that file writes are confined by Cursor |

A generated host shim or support document is valid only when `harness gen`
produces it from `hosts.toml` and the prompt pack, and `harness gen --check`,
setup, docs generation, release preflight, or an adapter smoke test consumes it
in the same phase. No host artifact is written by hand as source of truth.

**Codex/Cursor hook = generated, not inline (2-layer).** Codex and Cursor are
first-class hook hosts. Their hooks are not written inline in `config.toml`;
`harness gen` materializes them into `.codex/hooks.json` / `.cursor/hooks.json`
(gitignored build artifacts), each routing to `bin/harness hook pre-tool --host
<h>`. "config.toml ships no inline hooks" means "not inside config.toml," not
"no hook exists" — do not conflate the two. There are two hook layers, with
different deployment state per layer:

- **(a) Enforcement** (`PreToolUse` → `bin/harness hook pre-tool` → R01-R13):
  wired symmetrically across all three hosts and generated by `harness gen`. The
  Codex `.codex/hooks.json` carries `PreToolUse` (verified: `harness gen` emits
  it).
- **(b) Mode 2 delivery** (inbox-check / monitor receipt): the generator
  `go/internal/hostgen.GenerateDeliveryHooksJSON` is implemented and unit-tested,
  but has **no production caller** (not wired into `harness gen`), so generated
  `.codex/hooks.json` does not contain inbox-check. Claude's inbox-check lives
  hand-written in `.claude-plugin/hooks.json` (so Claude↔Claude delivery works);
  Codex/Cursor delivery hooks are not deployed. By design Codex delivery is
  turn-boundary (Stop) receipt; live monitor (SessionStart blocking stream) is
  Claude-only.

Before concluding a host lacks a capability, materialize the `harness gen` output
and inspect it: `not_observed` is not `absent`.

GAP (follow-up, not implemented here): wiring `GenerateDeliveryHooksJSON` into
`harness gen` would add inbox-check (turn-based, not live) to the generated
Codex/Cursor hooks, extending Mode 2 delivery to those hosts.

## Support Tiers And Host Claims

Public support wording must use support tiers.

| Tier | Meaning | Claim Allowed |
|------|---------|---------------|
| `supported` | Install/update path, bootstrap proof, skill loading, one workflow smoke, compatibility checks, and release/preflight gate all pass. | Public support claim for the verified host and version range. |
| `internal-compatible` | Repo mirror, setup docs, static validation, or local tooling exists, but runtime proof is incomplete. | Internal compatibility or experimental wording only. |
| `candidate` | Current official docs and local evidence suggest a viable adapter path, but no complete smoke proof exists. | Research or spike wording only. |
| `future/unsupported` | No verified adapter path or no current proof. | No setup docs, README support claim, or release support claim. |

Current default stance:

| Host | Default Tier | Reason |
|------|--------------|--------|
| Claude Code | `supported` for Claude-first Harness | Primary product surface and distribution payload. |
| Codex CLI | `internal-compatible` until direct plugin install and companion smoke are verified together | Existing Codex mirror and setup path exist; direct plugin path must be proven separately. |
| Codex app | `candidate` under the Codex adapter | App behavior must be verified separately from CLI help output. |
| OpenCode | `internal-compatible` until runtime bootstrap smoke passes | Existing mirror/setup validation exists; runtime parity is not yet proven. |
| Cursor | `internal-compatible` | Host-specific dist build, `scripts/setup-cursor.sh` real-directory install, CI-gated package smoke, and observed Desktop skill loading (`/breezing` etc.) justify internal compatibility; CI-gated workflow smoke and runtime guard/hook parity are not proven; no public supported claim |
| GitHub Copilot CLI | `candidate` | Current CLI docs must be verified and Harness-specific bootstrap proof is missing. |
| Antigravity CLI | `future/unsupported` until an official/verified adapter route exists | No local Harness or Superpowers adapter evidence has been observed. |

Phase 73.1.2 freezes this stance from
`docs/research/superpowers-cherrypick.md`. These tiers are authoritative until
the relevant host-specific setup route, bootstrap proof, workflow smoke, and
release/preflight gate all pass in the same claim path.

README, onboarding, and release wording must not imply that `candidate` or
`future/unsupported` hosts are supported. Candidate hosts may appear only as
research, spike, or adapter-candidate work. `future/unsupported` hosts may
appear only as future scope, unsupported scope, or unknown/unobserved research.

If a host is not observed in the current runtime, Harness must say `unknown` or
`not observed`, not `unsupported`, unless the relevant source of truth was
checked.

