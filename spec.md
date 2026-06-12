# Claude Code Harness V2 Spec

Status: draft SSOT for Phase 72 through Phase 87
Last updated: 2026-05-28

This file is the root product contract for Claude Code Harness V2.
Plans.md is the task ledger. `spec.md` is the product contract.
`spec.md` says what must stay true.

## Purpose

V2 makes Harness a faster operator workflow without weakening evidence.

The goal is to reduce human planning and verification load by letting Harness:

- classify work before execution,
- lock the correct specification before implementation,
- require TDD evidence when behavior changes,
- route review depth by risk,
- create PR-ready evidence packs,
- preserve release-grade checks for public artifacts.
- onboard users through the tool they already use, while only claiming support
  that has adapter evidence.
- keep repo-health gates such as formatting, linting, release preflight, and
  host runtime smoke aligned with the support tier being claimed.

## Users And Workflows

Primary user:

- An operator who prefers AI to prepare the plan, implementation, comparison,
  and verification evidence, while the operator makes the final judgment.

Core workflows:

- Plan from an ambiguous request into a spec-backed `Plans.md` task contract.
- Execute `Plans.md` tasks with lane-aware effort and TDD gates.
- Review implementation against `spec.md`, `Plans.md`, tests, and evidence.
- Produce PR closeout artifacts without silently pushing or merging.
- Release only after version, tag, GitHub Release, CI, and public-surface checks.
- Strengthen lint/format and host-runtime smoke without confusing static
  compatibility with public support.

## SSOT Layers

The source-of-truth order is:

1. `spec.md`: root product contract for this repository.
2. Sub-specs for specialized domains:
   - `docs/architecture/hokage-core.md` for reusable core architecture.
   - `go/SPEC.md` for Go runtime behavior.
   - Other clearly named docs under `docs/` for scoped contracts.
3. `Plans.md`: task ledger, dependencies, DoD, status, and evidence trail.
4. Runtime evidence: tests, review artifacts, PR body, CI, release output.

`Plans.md` must not replace `spec.md`. A task can be complete only if its DoD
passes and the result does not contradict the applicable spec.

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

## Execution Backend Contract

Harness adopts the **Kernel + Prompt Pack** model. `harness work` assembles the
embedded prompt pack plus the resolved task and emits it for the host to
execute; the binary does not call an LLM and is not a self-built agent loop or a
direct-API driver. A self-driving agent loop was evaluated and rejected: native
hooks already give per-action gating, so the kernel adjudicates while the host
runs the model. ACP is not adopted — the three hosts' native pre-action hooks
provide per-action enforcement, so no cross-host protocol is required.

The three execution backends are Claude (the native host), Codex, and Cursor.
All three converge on the **same** `harness hook pre-tool` entrypoint through
their native pre-action hook:

| Backend | Native pre-action event | Deny mechanism |
|---------|-------------------------|----------------|
| Claude | `PreToolUse` | exit code 2 |
| Codex | `PreToolUse` | exit code 2 |
| Cursor | `preToolUse` | exit code 2 |

Deny is exit code 2 across all three hosts; that is the universal enforcement
contract. The kernel does not duplicate a rule engine per host — every host's
generated hook routes to one `bin/harness hook pre-tool`, which runs the R01-R13
`go/internal/policy` engine. `go/internal/hookcodec` normalizes each host's
stdin shape (`session_id` vs `conversation_id`, `tool_input` vs
`command`/`file_path`, event-name casing) into one rule-engine input, and emits
each host's deny shape (`permissionDecision` for Claude/Codex, `permission` for
Cursor) so the rule table itself never changes per host.

Non-Claude backends are driven through companions. When `harness` drives Codex
or Cursor as an execution engine (the harness-drives-the-tool direction), the
companion returns a normalized `companion-result.v1` envelope
(`go/internal/companionresult`). Those companion-produced changes are untrusted
until they pass the **FLOOR** (`go/internal/floor`): a universal pre-merge gate
that re-evaluates the candidate diff with `harness policy check` (the same
R01-R13 surface the hook calls) plus the contract greps, before any take-in.
The FLOOR is the backstop for paths a native hook cannot see (nested subagents,
in-process shells), so it runs for every backend regardless of which native
hook already fired.

Backend selection precedence (highest first): a per-command flag (e.g.
`--backend cursor`) > the `HARNESS_IMPL_BACKEND` env var > the project
`env.local` entry > the user-scope `~/.config/claude-harness/impl-backend.env`
entry > the call-site default. The global call-site default remains `claude`,
and shipped workflow skills must not hard-code Cursor as their call-site
fallback. A project or user can opt in to Cursor by setting
`HARNESS_IMPL_BACKEND=cursor`; that default must affect all Cursor-capable
surfaces, not only Breezing. Project scope overrides user scope.

**Resolver-only entry**: The sole canonical entry for choosing a backend is
`scripts/resolve-impl-backend.sh`. Direct env reads in implementations or skill
prose are forbidden; the resolver resolves precedence across env, per-run flags,
project file, and user file in one pass.

Backend is role-scoped: the implementation (worker) role uses the selected
backend. The primary review and advisor roles stay on the brain (the `claude`
host); per-role models resolve via `scripts/model-routing.sh`, and the
`HARNESS_BRAIN_MODEL` opt-in described below affects only the `deep`/`advisor`
tiers — the primary `review` tier is not changed by it. The self-review
prohibition is scoped to the producing
context, not the model family: the session that produced a diff must never
review its own output, but a fresh-context reviewer session on the same
backend (for example the cursor `review` tier, `composer-2.5-fast`) may run
an advisory pre-review pass before the brain's primary review. A pre-review
session qualifies as fresh-context only when it shares no conversation state
with the producing worker session and starts from the diff plus the task
brief. Pre-review findings are advisory input to the brain; the primary
verdict always remains with the brain reviewer, and Cursor never owns the
primary verdict.

Cursor-capable host packages must expose the Cursor namespace as first-class
commands. This surface is pre-release, so Harness does not provide legacy
aliases for the earlier `cursor-do` / `cursor-ask` naming:

- `cursor:setup`: configure or verify the Cursor package, `cursor-agent`, and
  local/project backend defaults.
- `cursor:do`: write delegation through an isolated worktree, Lead diff review,
  and cherry-pick.
- `cursor:ask`: read-only delegation through `cursor-companion.sh task` without
  `--write`, worktree, or cherry-pick.
- `cursor:review`: read-only advisory Cursor review; primary verdict remains
  on the host brain.
- `cursor:rescue`: diagnose Cursor backend setup, resolution, and companion
  failure paths.

Skill frontmatter `name` fields remain validator-safe lowercase hyphen ids
(`cursor-do`, `cursor-ask`, `cursor-setup`, `cursor-review`, `cursor-rescue`).
Codex defaultPrompt uses those registered `$cursor-<verb>` skill ids for
name-based invocation; descriptions and prose retain the `cursor:<verb>`
namespace as the user-facing mental model.

The concrete model for any host+role is resolved by
`scripts/model-routing.sh --host <backend> --role <role>`. This contract does
not reimplement model selection. The claude-host brain tiers (`deep`,
`advisor`) default to `claude-opus-4-8`; setting `HARNESS_BRAIN_MODEL=fable`
opts those two tiers into `claude-fable-5`. Unset, empty, or `opus` keeps the
default; any other value fails with exit 2 instead of falling back silently.
The opt-in never changes the worker or review tiers and never touches the
codex/cursor catalogs.

Cursor remains `internal-compatible`, not a public `supported` claim. The
shipped `harness` CLI keeps Cursor opt-in by default; individual
local environments may set `HARNESS_IMPL_BACKEND=cursor` in env, project
`env.local`, or user-scope config to make Cursor the resolved default. If Cursor
is selected but not configured, the workflow must fail with setup guidance
rather than silently claiming Claude/Codex parity.

Containment for cursor write delegation relies on a dedicated-`.git` worktree
plus Lead diff review and cherry-pick as the real boundary. Per Cursor's
official security docs (cursor.com/docs/agent/security), file writes have no
project-folder confinement and command allowlists are "best-effort, not a
security guarantee", so containment cannot be delegated to Cursor.

## Orchestration Visibility Contract

Backend selection is invisible at runtime: a user cannot tell whether work was
delegated to Codex, Cursor, or kept on Claude. The harness must make the
session's actual backend mix observable, so a user can answer "am I really
orchestrating, or did everything fall back to Claude?" and can show the result
to others.

The contract separates recording from display. Recording is always on; display
is on demand. The harness keeps two scopes: a per-session ledger of the current
session's delegations, and a lifetime accumulator that persists cumulative
per-backend totals across sessions in `.claude/state/orchestration-totals.json`
(project-scoped). Session delegations roll up into the accumulator when the session's tasks are
all complete, and again at session end as a safety net; the rollup always runs
and is never gated behind display. The rollup must
be idempotent per `session_id` so a session counted once is never
double-counted. A user-scope total across all projects is an optional extension,
not required here.

Per-task completion never triggers display; that would spam a multi-task
session. The one allowed automatic surface is a single compact terminal summary,
emitted once when the session's tasks are all complete (the rollup runs first,
so the summary reflects the updated lifetime totals). The HTML scorecard is
never emitted automatically. Surfaces report both the current session mix and
the lifetime totals; the lifetime totals are the primary shareable figure.

The authoritative record is a companion-written ledger. `codex-companion.sh` and
`cursor-companion.sh` append one structured line per delegation to
`.claude/state/orchestration-ledger.jsonl`. Each entry carries only
non-sensitive fields: timestamp, backend, subcommand, write flag, exit code,
duration, session id, and a `counts` flag. The ledger must never contain prompt
text, file contents, or secrets. Only delegation subcommands (`task`, `review`,
`adversarial-review`) set `counts: true`; status/setup/result/cancel calls are
recorded with `counts: false` so polling does not inflate the score.

Claude-side work is not companion-driven, so the scorecard derives Claude
delegation counts from the existing worker spawn trace
(`.claude/state/agent-trace.jsonl`, role `worker`) and merges them with the
ledger. The merged result is an `orchestration-scorecard.v1` snapshot reporting,
per backend: the used count, and a tri-state status — `used` (count > 0),
`available` (backend resolves/configures but unused this session), or
`not-configured` (companion setup fails or binary absent). `not-configured` is a
neutral state, never a failure or warning. The snapshot also reports backend
diversity (how many distinct backends were actually used) and a multi-backend
utilization ratio (non-Claude delegations over total delegations) with a one-line
note stating what the ratio measures.

Two surfaces consume the snapshot: a standalone HTML scorecard rendered through
`scripts/render-html.sh` (redaction layers applied as defense-in-depth, so it is
shareable), produced only on demand; and a compact terminal summary, produced on
demand and additionally emitted once at full-session completion. Neither is
emitted on every task completion. Both report session mix plus lifetime totals.

If the ledger or trace is missing or unreadable, the scorecard degrades to
"no delegations observed" rather than erroring; absent observation is not the
same as absence of work.

## Onboarding Contract

Onboarding is not complete when files are copied. It is complete when the first
useful session can be verified.

New-user onboarding must provide:

- a tool-first front door: "which agent are you using now?",
- an install or setup route for that host,
- the first command or first prompt to try,
- what successful bootstrap looks like,
- a verification command or smoke transcript,
- the support tier and known asymmetries.

Existing-user migration must provide:

- a before-state inventory,
- backup locations outside skill scan paths,
- stale plugin/cache/residue detection,
- duplicate local skill detection,
- harness-mem state handling that never deletes memory by default,
- rollback instructions that avoid destructive cleanup unless explicitly
  confirmed.

Superpowers is the reference pattern for multi-host onboarding: common skills,
thin host adapters, bootstrap guidance, skill-trigger tests, and explicit host
tool mapping. Harness may cherry-pick that pattern, but every copied idea must
be translated into Harness lanes, Plans.md tasks, TDD/review gates, and support
tier evidence.

## Host Distribution Contract

Distribution is a single `harness` CLI binary plus the manifests and mirrors that
hosts read directly. Per-host shims — the hooks.json configs, the skill/agent
mirrors, the manifest, and the catalog docs — are generated from one source
(`hosts.toml`, `skills/`, and the embedded prompt pack), never hand-maintained,
and are committed-and-drift-checked rather than gitignored. The reason they stay
committed: the Claude plugin marketplace clones the repo and reads
`.claude-plugin/*` directly, and `scripts/setup-codex.sh` / `setup-opencode.sh`
copy the committed mirror into the host — there is no install-time generation
step, so the generated artifacts must be present in the distributed tree. Drift is
prevented by CI gates (`harness gen --check`, `sync-skill-mirrors.sh --check`),
not by regenerating on the target. There is one version: a single git tag.
Manifests and mirrors do not carry independently bumped versions, and there is no
separate per-host package to release.

Rules:

- The release unit is the `harness` binary plus `hosts.toml` and the embedded
  prompt pack. Host shims are regenerated from those by `harness gen`; they are
  never the source of truth.
- `harness gen` writes each host's native hook config to its `hook_path` from
  `hosts.toml` (`.claude-plugin/hooks.json`, `.codex/hooks.json`,
  `.cursor/hooks.json`), each routing to `bin/harness hook pre-tool`. `harness
  gen --check` diffs the generated output against golden fixtures in CI so the
  generator cannot drift. The generator writes the Codex and Cursor hook configs;
  the Claude `.claude-plugin/hooks.json` is hand-maintained across its full event
  set and is not overwritten, but `harness gen --check` verifies its PreToolUse
  guardrail group still matches `hosts.toml`, so the pre-action route shared by
  all three hosts cannot drift even though the rest of that file is not generated.
- A host's generated shim must not cross-contaminate another host: the Codex
  artifact contains only Codex hook config and the Codex skill/agent mirror, the
  Cursor artifact only Cursor's, and so on. Cross-host manifests never appear in
  a single host's generated tree.
- Generated component paths must stay inside the install package and must not use
  `..` relative paths. The generator normalizes to in-package locations
  (`./skills/`, `./agents/`, or equivalent).
- The generator normalizes component metadata so each target host actually
  surfaces it. Skills authored for the Claude slash-only convention
  (`user-invocable: true`) are dropped by Cursor, so the Cursor artifact is
  generated with `user-invocable: false` to register them as Agent-Decides
  skills invokable via `/skill-name`; the Claude artifact keeps the original
  `user-invocable: true` slash contract.
- A Cursor local install lives under `~/.cursor/plugins/local/<name>`. Cursor's
  plugins doc supports symlinking an external plugin repo into that directory
  (`ln -s`); install tooling nonetheless real-copies the generated package so the
  install is self-contained and idempotent and does not depend on an external
  build path staying present (symlinks are also unreliable on Windows).
- Codex and Cursor remain below Claude in support tier until their own workflow
  smoke and release gates pass. Generating their shims does not by itself promote
  a host to `supported`.

Two layers, not one. The `hook_path` configs above (`.codex/hooks.json`,
`.cursor/hooks.json`) are the enforcement-wiring layer `harness gen` emits to
route each host's native pre-action hook to `bin/harness hook pre-tool`; each is a
first-class standalone hook location on its host and needs no plugin to run (Codex
discovers `~/.codex/hooks.json` and `<repo>/.codex/hooks.json` next to its config
layers when trusted; Cursor reads project `.cursor/hooks.json`). The
install-delivery layer is separate: `scripts/build-host-plugin-dist.sh` packages
each host's native plugin bundle (`.claude-plugin/` / `.codex-plugin/` /
`.cursor-plugin/`) carrying the generated skills/agents, which setup installs into
the host (`~/.cursor/plugins/local/<name>`, the Claude marketplace clone,
`~/.codex/plugins/<name>` + `~/.agents/plugins/marketplace.json`). Using a host's
native plugin as the install envelope does not contradict "single `harness` CLI,
not a host plugin": the binary plus `hosts.toml` and the embedded prompt pack stay
the sole source of truth and the release unit; the plugin is only the generated,
committed-and-drift-checked envelope that carries the shims to the host.

Cutover status (Phase 91.8(b), landed as generated-and-committed): the manifests
and mirrors are generated from one source and kept committed under CI drift gates,
not untracked. Investigation of the real distribution paths settled this: both
paths consume committed files with no install-time generation — the Claude
marketplace clones the repo and reads `.claude-plugin/*`, and `setup-codex.sh` /
`setup-opencode.sh` copy the committed `codex/.codex/skills` and `opencode/skills`
mirrors — so untracking and gitignoring them (the originally sketched
"generated-on-install" model) would break installation at the distribution target.
Instead the SSOT is `skills/` + `hosts.toml` + the prompt pack: `sync-skill-mirrors.sh
--check` pins the mirrors to `skills/`, `harness gen --check` pins the Codex/Cursor
hooks.json to `hosts.toml` and verifies the committed Claude PreToolUse guardrail
group matches the descriptor, and `harness gen docs --check` pins the catalog. The
artifacts stay committed so distribution keeps working, and the gates make them as
drift-proof as gitignored build output would be. A future pure-CLI install that
regenerates on the target could revisit untracking; it is out of scope while
marketplace and setup-copy distribution is the supported path.

## Clean Mode And Compatibility Mode

Harness defines two user-facing environment profiles. These are Harness
diagnostic and guidance profiles, not host-native global toggles. Cursor may
still load Claude/Codex skill directories when the user enables host
compatibility import in the Desktop app.

| Profile | Meaning | Expected UX |
|---------|---------|-------------|
| `clean` (default) | One host, one Harness route. Cursor users should see Cursor package skills only after cleanup. | Fewer duplicate skills/plugins; explicit host-specific invocation. |
| `compatibility` | Cross-host skill import remains enabled. Harness warns about duplicates but does not force-disable host import settings. | More skills visible; Harness recommends namespaced or explicit invocation. |

Harness must not delete user home configuration by default. Environment cleanup
uses dry-run inventory first, then user-confirmed archive or disable actions.
Compatibility import in Cursor Desktop can reintroduce duplicate skills even
after clean distribution packages are installed; Harness documents that limit
and detects duplicate origins before suggesting fixes.

## New Session Bootstrap Rule

A new agent session must be able to start from one task id without re-inventing
the plan.

Each startable task must make these visible:

- source spec path,
- current task id,
- first action,
- expected evidence artifact,
- blocked conditions,
- stop or handoff condition.

If a phase is broad, the first task must be research/evidence or plan-freeze.
Implementation must not begin until the evidence artifact narrows the files,
tests, smoke commands, and claim boundaries for the next tasks.

## Lane Taxonomy

Harness V2 uses lanes as task metadata, not as separate primary skills.

| Lane | Use When | Required Closeout |
|------|----------|-------------------|
| `[lane:fast]` | Low-risk local docs, narrow cleanup, small isolated fixes | focused checks, concise evidence pack, no full review by default |
| `[lane:gate]` | Skill, workflow, guardrail, mirror, CI, spec, or shared behavior changes | spec alignment, TDD when required, major-only or full review, re-review until clean |
| `[lane:release]` | Public artifact, version, tag, GitHub Release, CI, binary/package surface | release preflight, version sync, tags, GitHub Release, CI/latest verification |

Fast lane is not a bypass. It still needs a scope, DoD, focused verification,
and an explicit residual-risk statement.

## Stage Gate Flow

Every non-trivial V2 plan follows this path:

1. Research and verification
   - Read current repo state, relevant docs, `Plans.md`, memory, and available
     runtime evidence.
   - Treat failed searches, unavailable APIs, missing fixtures, and unseen data
     as `unknown`.
2. Implementation plan freeze
   - Record lane, scope, DoD, dependencies, TDD tag, risk gates, and evidence
     expectations in `Plans.md`.
3. Implementation with TDD
   - For `[tdd:required]`, create or update a failing test first and keep red
     evidence via red-log or literal failing output.
   - Use `[tdd:skip:<reason>]` only when the reason is explicit and reviewable.
4. Review
   - `harness-review` stays read-only by default.
   - `APPROVE` means the quality gate passed. It does not mean commit, push,
     PR, merge, or release may happen automatically.
5. PR closeout
   - PR artifacts include base/head refs, spec path, lane, stage, tests, review
     result, accepted/rejected findings, residual risk, and warnings handled.
   - Push and PR creation are external side effects and require an explicit
     flag or confirmation gate.
6. Release closeout
   - Release lane is complete only after version surfaces, tags, GitHub Release,
     CI, and public artifact checks are verified.

## Unknown Data Contract

Harness V2 must distinguish unobserved data from absent data.

Required rule:

```text
not_observed != absent
```

If an agent cannot see a file, API response, memory record, CI run, GitHub
object, fixture, or runtime output, it must report `unknown`, `unavailable`, or
`not observed`. It must not claim the data does not exist unless it has checked
the relevant source of truth.

Examples:

- Search timed out: `unknown`, not `no results exist`.
- Fixture was not loaded: `not observed`, not `fixture missing`.
- harness-mem was unavailable: `memory unavailable`, not `no memory`.
- Local tests passed: `local checks passed`, not `PR/release ready`.

## Review Contract

`harness-review` checks:

- spec alignment,
- `Plans.md` scope and DoD,
- TDD evidence when required,
- regression risk,
- accepted and rejected findings,
- unknown data handling,
- evidence pack completeness.

Critical or major findings produce `REQUEST_CHANGES`.
Minor or recommendation-only findings can still produce `APPROVE` when the
acceptance bar is met.

## PR And Release Boundary

PR closeout belongs to `harness-work`, not `harness-review`.

Release belongs to `harness-release`, not PR closeout.

Do not merge these stages:

- PR ready means the change has a reviewable branch and evidence pack.
- Release ready means the public release path has passed preflight and the
  release artifacts are verified.

## README Product Surface Contract

The root README and Japanese README are public product surfaces, not internal
closeout notes.

They must lead with:

- the user pain Harness solves,
- what changes after install,
- the fastest verified setup path,
- the first command or first prompt,
- the workflow Harness actually enforces,
- the proof boundary for supported and candidate hosts,
- links to deeper docs only after the quick path is clear.

README copy must not lead with internal code names, release archaeology,
operator-only HTML artifacts, or product-history explanations. Those may live
in architecture docs, research docs, or changelog entries when useful.

Command descriptions must explain what the command does inside in one concise
line, so a new user understands the work being delegated without reading the
skill source.

Visual assets used by README / README_ja must follow the same claim boundary:

- text-bearing images require separate English and Japanese assets,
- generated images must use the current official Claude Harness logo tone on a
  white background,
- no image may imply support tiers or host parity beyond verified evidence,
- generated prompts, source files, dimensions, and alt text must be recorded in
  an asset manifest before release,
- stale images that carry obsolete product names, dark hero styling, or
  unsupported support claims must be removed or replaced.

When multiple generated-image directions are plausible, README copy may ship
without those images, but final image generation and integration require an
explicit user approval gate for the chosen direction.

## I18n And Status Marker Contract

Harness ships with English as the default user-facing locale, while Japanese
remains available through explicit opt-in.

Status markers are both protocol values and visible user-facing text. New or
updated Plans.md rows, templates, summaries, and generated notification files
must not mix Japanese and English within the same status marker family. Writer
paths must emit the English marker family, especially `cc:done` for completed
work, alongside `cc:todo`, `cc:wip`, `pm:requested`, and `pm:approved`.

Backward compatibility is mandatory:

- existing `cc:TODO`, `cc:WIP`, `cc:完了`, `pm:依頼中`, and `pm:確認済` rows remain
  valid input,
- Japanese opt-in may preserve surrounding Japanese prose, but new and updated
  status marker writes still use the English marker family,
- readers, sync, loop, sprint-contract, and Plans validation must accept both
  legacy canonical markers and English aliases,
- bulk migration of existing Plans.md files is never implicit; it requires an
  explicit migration command or user approval.

User-facing runtime reasons, guardrail messages, status summaries, and generated
state notifications should follow the same locale resolver as other Harness
messages for prose. Status marker writes are the exception: new/update writer
paths use the English marker family while legacy Japanese markers remain
read-compatible.

## Supply Chain Alert Contract

Open Dependabot alerts on tracked source, tooling, benchmark, or distribution
lockfiles are repo-health findings, not release noise.

Harness must handle them with evidence:

- enumerate the live GitHub alert set before planning remediation,
- group alerts by manifest path, dependency, severity, and advisory,
- prefer supported upgrades that keep the current tool line moving forward over
  security downgrades suggested only by `npm audit fix`,
- use package-manager-native override mechanisms only when the direct owner
  package has not yet published a patched dependency range,
- verify the affected tool still starts or runs an equivalent smoke command,
- add or update Dependabot configuration and CI/audit checks when a tracked
  manifest can otherwise accumulate alerts without PR automation,
- keep GitHub alert closeout, local `npm audit`, CI, and release gates separate.

Benchmark-only manifests may use focused smoke evidence instead of full
benchmark execution when model keys, Docker, or sandbox services are unavailable,
but the unavailable part must be recorded as a residual risk rather than treated
as success.

## Memory Contract

When a planning or design decision is made, Harness should record why it was
chosen, not only what changed.

Preferred memory targets:

- `harness-mem` project-scoped ingest/search when available.
- `.claude/memory/decisions.md` and `.claude/memory/patterns.md` when present.
- `Plans.md` and spec documents as local, reviewable SSOTs.

If harness-mem is unavailable, the agent must say so and keep the local SSOT
updated instead of pretending memory was written.

## Upstream Tracking Contract

Claude Code and Codex updates must be turned into Harness changes through an
evidence gate, not by copying release notes into docs.

Every non-trivial upstream refresh must:

- compare the local installed versions with the latest official upstream
  versions,
- use official Anthropic, OpenAI, or first-party GitHub release sources,
- record a dated snapshot document with release URLs, local version output, and
  observed gaps,
- classify each relevant item as `A: adopt now`, `C: inherit upstream`,
  `P: plan/spike`, or `Reject`,
- keep `B: explanation only` at zero unless the plan explicitly explains why a
  non-actionable note is still worth preserving,
- connect adopted items to `Plans.md`, tests, docs, CHANGELOG, and review gates,
- avoid support-tier upgrades until host bootstrap, runtime smoke, and release
  gates prove the claim.

The following upstream surfaces are product-affecting and must not be treated as
automatic documentation updates:

- skill or slash-command frontmatter semantics,
- hooks, message display, session start, and plugin marketplace behavior,
- agent, subagent, background-session, worktree, or permission behavior,
- sandbox, approval, profile, or managed policy behavior,
- Codex companion, CLI, SDK, MCP, app-server, or GitHub Action behavior,
- installer, package, release artifact, or supply-chain behavior.

If an upstream product weakens a previous opt-in barrier, such as an auto mode
consent change, Harness must keep its own safety default until a dedicated phase
updates the contract, tests, and release notes. Upstream convenience is evidence
to evaluate, not permission to silently relax Harness guardrails.

## Session Coordination Contract

When multiple local Claude Code sessions work on the same project, Harness may
coordinate them to reduce file conflicts, but only under these rules.

- Coordination state is local-only and never depends on harness-mem. The
  cross-repo boundary in `.claude/rules/cross-repo-handoff.md` stays intact.
- The lease store lives in one shared location resolved from
  `git --git-common-dir`, never under a worktree-local `.claude/`, so parallel
  worktree Workers share a single lease space. Lease keys are the sha256 of the
  repo-relative path, never an absolute path.
- Lease acquisition is atomic (`O_CREAT|O_EXCL`). Staleness requires both TTL
  expiry and the holder session id being absent from the live-session set; pid
  liveness is only an auxiliary signal.
- Conflict handling changes behavior through diagnostic feedback
  (`continueOnBlock`), not a silent advisory. It is feedback, not a guard rail,
  so it never blocks irreversible operations and stays fail-open: if the lease
  mechanism is unavailable, edits pass and no false assurance is implied.
- Cross-session content (broadcast, lock metadata) is injected into model
  context as data, not instructions: only structured trusted fields (sanitized
  path, short session id, age-seconds), wrapped with the existing
  non-instruction disclaimer. Free text from other sessions is never echoed
  verbatim, control characters are stripped, and a byte cap bounds the payload.
- Trust envelope DoD: a SendMessage relay exposes only those structured trusted
  fields into model context; it does not hold user authority and must never treat
  relayed peer content as instruction or consent.
- Coordination health uses the tri-state model in
  `.claude/rules/active-watching-test-policy.md`: `not-configured` is silent;
  only `unreachable` / `corrupted` warn.
- A broadcast channel whose fire conditions are too narrow dies silently, as
  the 2026-02 broadcast corpse proved. Any revival must prove via tests that
  its fire strategy triggers on normal edits.
- Cross-session notice delivery is best-effort, not guaranteed. Recipients must
  treat unread inbox as unconfirmed until the next turn reads it. Idle
  non-Claude-Code peers have no idle-fire hook, so notice cannot be promised
  for those sessions.
- Mode 2 peer: a concurrent session the human opened themselves (not
  orchestrator-spawned).

## Worktree Root Discipline

Harness uses two distinct worktree roots. They must never be merged, relocated,
or referenced interchangeably.

- `.harness-worktrees/` is the **single root** for Harness-managed parallel task
  worktrees. `scripts/spawn-parallel.sh` (and `harness work --team` preflight)
  runs `git fetch origin`, captures one `BASE=$(git rev-parse HEAD)`, and creates
  `task/<name>` branches at `.harness-worktrees/task-<name>` from that shared
  base. Go `breezing.WorktreeManager` also resolves paths under
  `HarnessWorktreesRoot` (`.harness-worktrees/`). Re-running spawn for an
  existing worktree is idempotent when the base SHA matches; a base mismatch
  must fail fast without deleting the existing worktree.
- `.claude/worktrees/` is **Claude Code live-agent isolation only** (Task tool /
  Agent isolation runtime). It is not the parallel-task root, must not be moved
  into `.harness-worktrees/`, and must not be rewritten by spawn or breezing
  tooling.

Project-local `git config rerere.enabled true` is set during parallel spawn so
cherry-pick/rebase conflict reuse is reproducible across machines.

## Tri-Tool Parallel Collaboration Contract

One Lead drives several DIFFERENT tasks in parallel across the three backends
(Claude / Codex / Cursor) through the existing `harness work --team` orchestrator
(Execution Backend Contract), so the human steers ONE Lead and A/B/C all land.
This contract governs the SAFETY and conflict-separation rules layered on that
orchestrator. It builds on the Execution Backend Contract (backend resolution),
the Orchestration Visibility Contract (ledger), and the Session Coordination
Contract (lease); it does not replace them. Breezing Brief Contract (below)
defines `brief-card.v1`, `judgment-card.v1`, breezing mem lifecycle events, and
fail-open memory behavior for `/breezing` free-text entry layered on this
orchestrator.

- Hub-spoke, no worker-to-worker. Workers emit only `companionresult.v1` on
  stdout; they never address or message each other. All coordination is
  spoke->hub (worker result -> Lead). For v1 the Lead is Claude Code, the only
  host that can natively spawn Claude subagents; Codex and Cursor cannot fan out
  to other backends, so "any of the three as Lead" is a future goal, not a v1
  claim.
- Headless v1. The Lead drives Codex/Cursor as background workers via their
  companions; there is no live session-to-session message bus in v1. The
  `harness_session_*` and `harness_mem_signal_*` MCP tools are EXTERNAL
  (harness-mem-owned) and are not a dependency of this contract.
- Physical separation first. Before fan-out the Lead establishes ONE fresh base
  (`git fetch`; `BASE=$(git rev-parse HEAD)`) and creates branch-per-task plus
  worktree-per-branch off that single BASE, so workers cannot collide while
  running. Shared files (`Plans.md`, `CHANGELOG.md`, `spec.md`) are written as
  owner-assigned append-only blocks; `VERSION` is never bumped inside a worktree;
  generated artifacts are regenerated once on trunk after merge; `rerere.enabled`
  is set. The Lead aggregates one task at a time: rebase the task branch onto
  trunk, `cherry-pick --no-commit`, run the pre-merge policy gate, commit.
  Normative detail (3 invariants, owner-assign table, Phase 92.1.1 CHANGELOG
  collision precedent): `.claude/rules/shared-file-discipline.md`. Complements
  Worktree Root Discipline above (where worktrees live vs what workers may edit).
- Two distinct floors — do not conflate them.
  - The PRE-MERGE POLICY GATE is the existing `go/internal/floor` (`floor.Gate`):
    deny-surface integrity, R01-R13 over the changed files, and the contract
    scripts. It runs at integration and catches file-level violations.
  - The RUNTIME ACTION HARD FLOOR (the human stop) is enforced BEFORE a worker
    action runs, at the companion-invocation / pre-action layer, because a
    post-hoc file diff cannot see a runtime side effect. Five categories ALWAYS
    stop and ask the human and are non-overridable in every config: (1)
    money/billing, (2) external send / network egress to a non-allowlisted
    destination, (3) credential entry or secret read, (4) production deploy or
    publish, (5) destruction OUTSIDE the task worktree.
- Auto-approve scope. Inside a CONFINED worktree the Lead may auto-judge
  code/file/git "ask" gates; the runtime hard floor is the only escalation path.
  Auto-approve must NOT be enabled until both the runtime floor and worktree
  confinement exist.
- Worktree confinement. Codex/Cursor workers must be confined to their worktree
  path; `--workspace` is a working-directory hint, NOT a write boundary. The v1
  confinement is a pre/post worktree fingerprint: any change outside the task
  worktree ($HOME-sensitive paths, trunk, sibling worktrees) hard-stops the run
  (this is also floor category 5). OS-level confinement (`sandbox-exec` /
  `unshare`) is a later hardening, not a v1 requirement.
- Visibility. Every dispatch, result, and aggregate event is emitted to the
  orchestration ledger.

Two operating modes sit on this contract. Mode 1 is fully autonomous
orchestration (headless, no live bus); Mode 2 is human-present peer co-drive
(live notice messaging). Both honor the safety core (runtime floor + worktree
confinement) above.

Mode 1 — orchestrated Producer hierarchy. The Lead/Producer (the CLI the human
talks to; for v1 the Lead is Claude Code) delegates each lane to a Sub-Lead:
one orchestrator-spawned headless CLI per lane on the same CLI backend. The
Sub-Lead decomposes the lane into a mini-plan, delegates
implementation to Composer 2.5 (the Cursor backend) workers in parallel, then
review-iterates via the Phase 92.5.2 in-process path (`go/internal/reviewiterate`):
fresh-context parallel sub-agent review plus cross-CLI review (the session that
produced the diff never reviews its own output — self-review scope, Execution
Backend Contract). Advisory reviewers share no conversation state with the
producing worker; the primary verdict always comes from the brain (claude host)
only. Mode 1 review does **not** use Mode 2 live-notice transport (Phase 92.6.5
withdrawn — durable handoff and live notice must not mix). The Sub-Lead
re-dispatches refinement into the same worktree until the lane's DoD is met or a
max-iteration cap is hit, after which it escalates to the human. The Sub-Lead
reports up to the Lead, which aggregates. Workers still never message each other;
all coordination is spoke->hub.

Mode 2 — CCH-owned live notice messaging. Live, notice-guaranteed messaging
between concurrent human-opened peer terminals (Mode 2 peers only) is owned by
claude-code-harness, NOT the memory layer. Mode 2 transport is for peer
co-drive notice delivery only; it must not carry Mode 1 review or cross-CLI
review verdict traffic. It is a self-contained,
dependency-free SQLite (WAL, append-only event log) store plus host-hook delivery
notice, modeled on the agmsg pattern (MIT): turn = a Stop hook that reads the
inbox at each turn boundary; monitor = a SessionStart hook streaming via the
Monitor tool (Claude Code only; Codex/Cursor fall back to turn). The delivery
must prove via tests that it fires on a normal turn — the 2026-02 broadcast
corpse died precisely because store and notice were split and the notice never
fired; reuniting them in the host-hook-owning layer is the fix. A human gives the
GO; the harness drives the A->B->A loop with a max-round cap and stops for the
human on the five hard-floor categories. self-audit allowlists the delivery hooks
this layer writes into `.claude/settings.local.json` so the harness does not flag
its own messaging hook as tampering.

Memory boundary. Durable, work-linked, ackable handoff stays in harness-mem
(`signal-store` / workgraph claim+handoff) and is load-bearing there; this Phase
does not move or modify it. Live notice messaging is the CCH-owned layer above.
The two are distinct: durable work-handoff = harness-mem signal; live notice
messaging = CCH.

## Breezing Brief Contract

`/breezing` may accept free-text input that does not match the existing
argument-hint surface (`all`, task ranges, `--codex`, `--cursor`,
`--reviewer-only`, `--parallel N`, `--no-commit`, `--no-discuss`, `--auto-mode`).
This contract productizes the Brief Composer and Decision Card schemas, breezing
mem write layer, and their precedence against the runtime hard floor. It builds
on Tri-Tool Parallel Collaboration Contract (team orchestrator, worktree
discipline) and Memory Contract (harness-mem when available); it does not
replace them.

### Input classification and brief-card.v1

- `scripts/breezing-brief.sh classify "<args>"` deterministically classifies
  input as `structured` or `free-text` (regex/token parse; no LLM).
- `structured` input continues on the existing team path unchanged.
- `free-text` input is decomposed by the Lead into a `brief-card.v1` card for
  user confirmation before any worktree dispatch.
- User confirmation is Yes/No. `scripts/breezing-brief.sh confirm no` emits
  `DISPATCH: 0` — zero worker executions (dry contract). Yes dispatches one
  worker per confirmed subtask onto the existing worktree-per-task team path.
- `scripts/breezing-brief.sh validate <card.json>` validates against
  `templates/schemas/brief-card.v1.json`.

Schema `brief-card.v1` required fields:

| Field | Shape | Constraints |
|-------|-------|-------------|
| `goal` | string | non-empty |
| `subtasks` | array | 3–7 items; each item `{id, title, dod}` (all non-empty strings) |
| `scope_files` | string[] | repo-relative paths in scope |
| `risk_notes` | string[] | free-form risk strings |
| `confidence` | enum | `high` \| `medium` \| `low` |

No additional properties are permitted on the card root or subtask objects.

### judgment-card.v1

When a worker or Lead needs human judgment during breezing — and the runtime
hard floor does not apply — Harness may issue one `judgment-card.v1` card.
Schema: `templates/schemas/judgment-card.v1.json`.

Issuance conditions (any one triggers a card):

1. DoD interpretation diverges (multiple valid readings of done-ness).
2. A change outside the user-approved scope is required.
3. A trade-off choice is required (mutually exclusive options).

Required fields:

| Field | Shape | Constraints |
|-------|-------|-------------|
| `question` | string | non-empty |
| `options` | array | 2–3 items; each `{id, label, consequence}` (all non-empty strings) |
| `recommendation` | string | non-empty |
| `confidence` | enum | `high` \| `medium` \| `low` |
| `impact` | string | non-empty |
| `diff_summary` | string | non-empty one-line diff summary |

No additional properties are permitted on the card root or option objects.
User answer and rationale may be recorded via `harness_mem_record_checkpoint`
when harness-mem is available (fail-open; see below).

### Breezing mem lifecycle events

`go/internal/breezingmem` posts breezing lifecycle events to harness-mem
(`POST /v1/events/record`) and ingests the confirmed brief
(`POST /v1/ingest/knowledge-file` at path `breezing/brief-card.v1.json`,
kind `decisions_md`). Event type literals (fixed):

| Event type | When |
|------------|------|
| `breezing_run_started` | Breezing team run begins |
| `breezing_brief_confirmed` | User confirms the brief card (Yes) |
| `breezing_worker_result` | One worker completes (success or failure) |
| `breezing_aggregation_completed` | Lead aggregation finishes |

This layer does not call workgraph signal APIs (`signal_send`, `signal_read`,
`signal_ack`, or `/v1/signals/*`). Durable cross-session handoff remains in
harness-mem signal store per Tri-Tool Parallel Collaboration Contract; breezing
mem events are run-scoped lifecycle telemetry only.

### Fail-open memory behavior

Breezing must never stop because harness-mem is absent or unreachable.

| State | Behavior |
|-------|----------|
| `not-configured` | Silent skip — no warning, no POST |
| `unreachable` | One stderr warning line (`breezing-mem: record skipped (unreachable)` or ingest equivalent), then continue |

Configured means `~/.harness-mem` or legacy `~/.claude-mem` exists (see
`go/internal/breezingmem` `configured()`). HTTP timeout is 1s. Mem state never
blocks brief confirmation, worker dispatch, judgment cards, or aggregation.

### Floor precedence over judgment cards

The five-category **runtime action hard floor** (Phase 92.2.1; money/billing,
external send/egress, credential/secret read, production deploy/publish,
destruction outside the task worktree) always takes precedence over
`judgment-card.v1`:

- When any hard-floor category matches, Harness hard-stops for the human and
  does **not** issue a judgment card.
- Judgment cards apply only to non-floor ambiguity (DoD, scope, trade-offs).
- The pre-merge policy gate (`go/internal/floor`) and runtime hard floor remain
  distinct; see Tri-Tool Parallel Collaboration Contract.

## Bridge Daemon Contract

The Bridge Daemon normalizes events from three CLI backends (CC Mailbox, Cursor
stop hook, Codex app-server) into one source-agnostic envelope, persists them in
a unified mailbox store, optionally ingests into harness-mem by lane, and
delivers notices to non-CC peers through host hooks. It builds on Tri-Tool
Parallel Collaboration Contract (Mode 2 live notice) and Breezing Brief Contract
(fail-open memory); it does not replace durable work-handoff in harness-mem
signal store.

### bridge-event.v1

Schema: `templates/schemas/bridge-event.v1.json`. Every normalized event uses
this envelope; no additional root properties are permitted.

| Field | Shape | Constraints |
|-------|-------|-------------|
| `source` | enum | `cc` \| `cursor` \| `codex` |
| `event_type` | string | non-empty |
| `payload` | object | source-specific fields after normalization |
| `ts` | integer | Unix nanoseconds; required (missing timestamp is fail-loud) |

Go reference: `go/internal/bridge.Event` (`Source`, `EventType`, `Payload`, `TS`).

### Source adapter contract

Three adapters implement `bridge.Adapter` (`Source()` + `Normalize(raw []byte)`).
Each maps one source-specific raw JSON input to exactly one `bridge-event.v1`
event (batching is caller responsibility).

| Adapter | Source | Raw input keys | Normalized `event_type` |
|---------|--------|----------------|-------------------------|
| CC Mailbox | `cc` | `hook_event_name`, `timestamp` | value of `hook_event_name` |
| Cursor stop hook | `cursor` | `hook_event_name`, `ts` | fixed `stop` |
| Codex app-server | `codex` | `type`, `ts` | value of `type` |

Adapter rules:

- Missing or invalid `ts` / `timestamp` is **fail-loud** (`requireNanos` error);
  the event is not appended.
- Unregistered `source` on `bridge.Registry.Normalize` returns `(Event{}, false,
  nil)` — **fail-open skip**; the caller emits one warning line (the registry
  package does not write stdout).
- Payload fields are copied per adapter (`conversation_id`, `tool_name`,
  `session_id`, `message`, `thread_id`, etc.) with non-reserved keys merged
  into `payload`.

### Delivery layer

Notice delivery to peers uses host hooks (Phase 92.6.2 delivery-notice pattern,
extended for bridge daemon non-CC peers):

| Host | Delivery path |
|------|---------------|
| Claude Code | Monitor blocking stream (SessionStart hook) |
| Cursor 1.7+ | Stop hook `followup_message` |
| Codex | Bash `PreToolUse` hook coupling |

When delivery fails, Harness must **not** block the run: emit one warning line,
record the failure in the orchestration ledger, and **fallback on the next
turn** (turn-boundary inbox read). Codex and Cursor fall back to turn delivery
when Monitor is unavailable.

### Mailbox unified store

`go/internal/mailbox.Store` persists normalized events in SQLite WAL mode as an
**append-only event log** (`bridge_events` table). Each append selects a lane:

| Lane | Mem ingest |
|------|------------|
| `fast` | `Record` only |
| `gate` | `Record` + `Audit` |
| `release` | `Record` + `Alert` |

Ingest runs asynchronously after append; ingest errors are logged and swallowed
(fail-open). `MemIngestor` defaults to `NoopIngestor` when harness-mem is not
wired.

This store is distinct from Phase 92.6.1 `livemsg` (Mode 2 peer messaging) and
from harness-mem signal store (durable work-handoff).

### Fail-open memory behavior (bridge read/write)

Bridge and mailbox layers must never stop a breezing or bridge run because
harness-mem is absent or unreachable. Same tri-state model as Breezing Brief
Contract:

| State | Behavior |
|-------|----------|
| `not-configured` | Silent skip — no warning, no POST |
| `unreachable` | One stderr warning line, then continue |

Configured means `~/.harness-mem` or legacy `~/.claude-mem` exists (see
`go/internal/breezingmem` `configured()`). HTTP timeout is 1s.

### workgraph signal boundary

Bridge Daemon, mailbox ingest, and breezing mem lifecycle **must not** call
workgraph signal APIs (`signal_send`, `signal_read`, `signal_ack`, or
`/v1/signals/*`). Durable cross-session handoff remains in harness-mem signal
store; bridge events are run-scoped telemetry and live-notice input only.

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

Decision Card mem read layer **must not** call workgraph signal APIs. Past
decision lookup uses search/ingest HTTP only; durable handoff stays in
harness-mem signal store per Tri-Tool Parallel Collaboration Contract.

## Non-Goals

V2 does not:

- replace `Plans.md` with `spec.md`,
- split Fast/Gate/Release into three new primary skills,
- make every task a heavy Gate lane task,
- break existing `cc:TODO`, `cc:WIP`, or `cc:完了` marker compatibility,
- let review auto-commit, push, PR, merge, or release,
- treat local green checks as PR-ready or release-ready by themselves,
- weaken release verification to save time.
- claim Cursor, GitHub Copilot CLI, Antigravity CLI, or generic cross-host
  support before support-tier evidence exists.
- copy Superpowers slogans or trigger rules when they conflict with Harness
  verbs, lanes, or guardrails.

## Open Decisions

- Exact PR closeout command shape: `harness-work --pr` vs
  `scripts/harness-pr-closeout.sh`.
- Final machine-readable evidence schema for PR bodies and closeout artifacts.
- Whether `.claude/memory/decisions.md` and `.claude/memory/patterns.md` should
  be created in this repository or remain optional memory surfaces.
- Whether Codex direct plugin installation becomes the default Codex path or
  stays secondary to `scripts/setup-codex.sh --user`.
- Which CI-gated Desktop workflow smoke is sufficient to promote Cursor from
  `internal-compatible` to `supported`, and which host docs and smoke tests are
  sufficient to promote GitHub Copilot CLI or Antigravity CLI from `candidate` /
  `future/unsupported`.

## Links

- Task ledger: `Plans.md` Phase 72 through Phase 76.
- Phase 76 closes the `harness-plan` planning-surface portion of Phase 72.1.2.
  `harness-work`, review, and PR closeout follow-up remains in Phase 72.
- Spec workflow policy: `docs/plans/spec-ssot.md`.
- Review operating model: `docs/harness-review-operating-model.md`.
- Architecture sub-spec: `docs/architecture/hokage-core.md`.
- Host capability matrix: `docs/tool-capability-matrix.md`.
- Spin-off readiness: `docs/hokage-spin-off-readiness.md`.
- Go runtime sub-spec: `go/SPEC.md`.
