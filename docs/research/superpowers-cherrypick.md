# Phase 73.1.1 Superpowers Cherry-Pick Evidence

Status: research artifact only
Date checked: 2026-05-21 JST
Phase: `Plans.md` 73.1.1
Spec: `spec.md`
Implementation status: not started

## Conclusion

Harness should adopt the multi-host onboarding pattern, not the host support
claims.

The reusable idea is:

- common workflow skills,
- thin host adapters,
- first-session bootstrap guidance,
- skill-trigger/runtime smoke,
- explicit migration warnings,
- public support tiers tied to evidence.

The unsafe shortcut is:

- copying another project's support matrix into Harness,
- treating a manifest file as support,
- treating one host's plugin route as proof for another host,
- calling unobserved runtime behavior unsupported.

`not_observed != absent` remains a hard rule.

## Sources Checked

### Harness sources

- `spec.md`
  - defines `spec.md` as product contract and `Plans.md` as task ledger.
  - defines Host Adapter Boundary.
  - defines support tiers: `supported`, `internal-compatible`, `candidate`, `future/unsupported`.
  - defines Onboarding Contract and New Session Bootstrap Rule.
- `Plans.md`
  - Phase 73 requires this artifact before implementation.
  - 73.1.1 DoD requires local Superpowers evidence, Harness evidence, current docs, Codex local help, and support claim boundaries.
- `docs/plans/spec-ssot.md`
  - keeps root `spec.md` as preferred project spec.
- `docs/architecture/hokage-core.md`
  - already frames Superpowers as a pattern: common skills plus thin host-specific entrypoints.
  - rejects copying skill names, false safety parity, or unsupported host claims.
- `docs/tool-capability-matrix.md`
  - Phase 70 matrix currently targets Claude Code, Codex, OpenCode only.
  - Cursor, Gemini, and Copilot are future/unsupported in that older phase.
- `docs/hokage-spin-off-readiness.md`
  - says no public spin-off yet.
  - keeps Cursor/Gemini/Copilot unsupported for public spin-off until adapter gates pass.
- `scripts/setup-codex.sh`
  - existing Codex setup path installs to user or project mode and includes backup/legacy cleanup behavior.
- `scripts/setup-opencode.sh`, `opencode/README.md`, `opencode/opencode.json`
  - existing OpenCode surface exists, but Phase 73 must re-check runtime bootstrap before raising claims.
- `scripts/codex-companion.sh`
  - Harness Codex execution should go through companion for task/review schemas, not raw `codex exec` in harness-linked sessions.
- Tests already relevant:
  - `tests/test-spec-ssot-workflow.sh`
  - `tests/test-tool-capability-matrix.sh`
  - `tests/test-bootstrap-routing-contract.sh`
  - `tests/test-codex-package.sh`
  - `tests/test-distribution-archive.sh`

### Local Superpowers sources

Observed checkout: `/Users/tachibanashuuta/LocalWork/Code/superpowers`

Key evidence:

- `README.md`
  - exposes a tool-first quickstart for Claude Code, Codex CLI, Codex App, OpenCode, Cursor, and GitHub Copilot CLI.
  - describes a workflow of brainstorming, worktrees, planning, TDD, code review, and finish branch.
- `.claude-plugin/plugin.json`
  - Claude plugin manifest exists.
- `.claude-plugin/marketplace.json`
  - marketplace manifest exists.
- `.codex-plugin/plugin.json`
  - Codex plugin manifest exists with `skills: "./skills/"` and interface metadata.
- `.cursor-plugin/plugin.json`
  - Cursor plugin manifest exists with `skills`, `agents`, `commands`, and `hooks`.
- `.opencode/plugins/superpowers.js`
  - OpenCode plugin injects bootstrap context and registers a skills directory through plugin hooks.
  - it caches bootstrap content and avoids duplicate injection.
- `.opencode/INSTALL.md` and `docs/README.opencode.md`
  - document OpenCode plugin installation, migration away from symlinks, verification prompt, and troubleshooting.
- `hooks/session-start`
  - injects the `using-superpowers` skill content and warns about legacy custom skills.
- `tests/skill-triggering/run-all.sh`
  - tests whether prompts trigger intended skills.
- `tests/codex-plugin-sync/test-sync-to-codex-plugin.sh`
  - validates Codex plugin sync/bootstrap behavior without mutating destination on dry-run.
- `tests/opencode/run-tests.sh`
  - covers OpenCode plugin loading, bootstrap caching, tools, and priority.
- `skills/using-superpowers/references/codex-tools.md`
  - maps Claude-style tools to Codex equivalents.
- `skills/using-superpowers/references/copilot-tools.md`
  - maps Claude-style tools to Copilot CLI equivalents.

## External Current Docs Checked

Current means checked during this session on 2026-05-21 JST.

| Host | Official source | Observed capability | Confidence | Boundary |
|---|---|---:|---:|---|
| Cursor | `https://cursor.com/marketplace` | Marketplace exists and lists Cursor plugins with skills/MCP-style capability descriptions. | medium | This proves a plugin ecosystem exists, not that Harness has a Cursor adapter. |
| Cursor | `https://docs.cursor.com/en/context` | Search result text says Project Rules live under `.cursor/rules`, `AGENTS.md` is supported, and `.cursorrules` is legacy. Direct web open redirected to `https://cursor.com/docs` without extractable body. | medium | Use rules/AGENTS as candidate bootstrap surfaces only. Runtime smoke still required. |
| OpenCode | `https://opencode.ai/docs/plugins/` | Plugins can be local files under `.opencode/plugins/` or global files, or npm packages in `opencode.json`; plugin hooks load at startup. | high | Good adapter candidate. Support still needs Harness runtime bootstrap smoke. |
| OpenCode | `https://opencode.ai/docs/skills` | Skills are `SKILL.md` definitions loaded on demand via the native `skill` tool from `.opencode/skills`, `~/.config/opencode/skills`, `.claude/skills`, or `.agents/skills`. | high | Supports a direct skills-first Harness route, but not safety parity with Claude hooks. |
| GitHub Copilot CLI | `https://docs.github.com/en/enterprise-cloud@latest/copilot/reference/copilot-cli-reference/cli-plugin-reference` | Official plugin commands include install, list, update, uninstall, marketplace add/list/browse/remove. `plugin.json` can point to `agents`, `skills`, `commands`, `hooks`, and `mcpServers`. | high | Viable candidate adapter shape. Harness-specific bootstrap/smoke is missing. |
| GitHub Copilot CLI | `https://github.com/features/copilot/cli` | Product page says Copilot CLI supports MCP integrations, skills, plugins, and subagents/multi-agent workflows. | high | Capability exists at product level, but no Harness support claim without local CLI smoke. |
| Antigravity CLI | `https://www.antigravity.google/docs/hooks` | Official search result says hooks exist with `PreToolUse`, `PostToolUse`, `PreInvocation`, `PostInvocation`, and `Stop`; tool names include file, search, command, and subagent operations. Direct fetch returned SPA shell or compressed bundle without extractable doc body. | medium | Candidate research signal only. Do not claim support until official content is captured and local smoke passes. |
| Antigravity CLI | `https://antigravity.google/docs/gcli-migration?app=antigravity` | Official search result says Antigravity CLI supports Gemini CLI-style extensions/plugins, Agent Skills, MCP servers, hooks, subagents, `GEMINI.md`, and `AGENTS.md`, and has `agy plugin import`. Direct fetch did not expose static text. | medium | Keep `future/unsupported` until install availability, plugin route, and Harness bootstrap are observed locally. |
| Antigravity CLI | `https://antigravity.google/docs/plugins` | Official search result says plugins are namespaced bundles grouping skills, rules, MCP servers, and hooks. Direct fetch did not expose static text. | medium | Promising, but insufficient to raise support tier. |

Codex current evidence is local-first:

- `codex plugin --help`
  - commands observed: `add`, `list`, `marketplace`, `remove`.
- `codex plugin marketplace --help`
  - commands observed: `add`, `list`, `upgrade`, `remove`.
- `codex plugin add --help`
  - accepted selector shape observed: `PLUGIN@MARKETPLACE` or `PLUGIN --marketplace MARKETPLACE`.

No Codex install smoke was run in this task. Local help proves command surface,
not successful Harness install.

## Adopt / Adapt / Reject / Unknown Matrix

| Candidate | Decision | Why | Evidence | Target follow-up |
|---|---|---|---|---|
| Tool-first front door | Adopt | New users should start from the tool they already use; Harness should stop presenting Claude plugin install as the only mental path. | Superpowers `README.md`; Harness `spec.md` Onboarding Contract. | 73.1.3 docs/onboarding front door. |
| Common skills plus thin host adapters | Adopt | Matches Hokage Core boundary and prevents host-specific mechanics from leaking into core workflow. | Superpowers `.codex-plugin/plugin.json`, `.cursor-plugin/plugin.json`, `.opencode/plugins/superpowers.js`; Harness `docs/architecture/hokage-core.md`. | 73.1.2 support tier and adapter contract. |
| First-session bootstrap | Adapt | Harness needs bootstrap, but Claude hooks, Codex AGENTS, OpenCode plugin hooks, Cursor rules, Copilot plugin skills, and Antigravity hooks are not equivalent. | Superpowers `hooks/session-start`; OpenCode plugin docs; Harness `docs/bootstrap-routing-contract.md`. | Define per-host bootstrap proof, not one shared claim. |
| Skill-trigger tests | Adopt | This directly reduces the user's verification burden: the agent must prove it selects the right workflow. | Superpowers `tests/skill-triggering/run-all.sh`; Harness `tests/test-bootstrap-routing-contract.sh`. | 73.1.10 Harness skill-trigger acceptance. |
| TDD-first implementation workflow | Adapt | Harness already has lane/stage/TDD gates; import the enforcement idea, not the exact workflow text. | Superpowers `skills/test-driven-development/SKILL.md`; Harness `spec.md` Stage Gate Flow. | 73.1.4-73.1.10 TDD tests per adapter. |
| Subagent-driven planning/review | Adapt | Useful for max-quality planning, but Harness must keep subagent use explicit and scoped to task contracts. | Superpowers `skills/subagent-driven-development/`; Harness Plan quality flow. | Use in planning/review phases when explicitly invoked. |
| Migration warning and rollback | Adopt | Existing users are the riskiest group; migration must report/back up, not delete. | Superpowers `hooks/session-start`, `.opencode/INSTALL.md`; Harness `scripts/setup-codex.sh`. | 73.1.9 migration report/rollback. |
| Codex direct plugin marketplace path | Adapt | Local Codex has plugin commands, but install smoke is not run. Keep existing `scripts/setup-codex.sh` fallback. | local `codex plugin --help`; Superpowers `.codex-plugin/plugin.json`. | 73.1.4 isolated `CODEX_HOME` smoke. |
| Codex App support | Unknown | Codex CLI help does not prove app behavior. | local CLI help only; no app smoke in this task. | Separate Codex App proof artifact in 73.1.4. |
| OpenCode native plugin route | Adapt | Official docs and Superpowers plugin shape align, but Harness runtime bootstrap is not yet proven. | OpenCode plugin/skills docs; Superpowers `.opencode/plugins/superpowers.js`; Harness `scripts/setup-opencode.sh`. | 73.1.5 bootstrap smoke and validation. |
| Cursor plugin/rules route | Unknown | Cursor marketplace and rules exist, but Harness has no verified Cursor bootstrap route. | Cursor marketplace and rules docs; Superpowers `.cursor-plugin/plugin.json`. | 73.1.6 candidate investigation. |
| GitHub Copilot CLI plugin route | Unknown | Official plugin/skills/hooks/MCP fields exist, but local `copilot` CLI and Harness smoke were not observed. | GitHub CLI plugin docs; Copilot CLI product page; Superpowers `copilot-tools.md`. | 73.1.7 local CLI availability and smoke plan. |
| Antigravity plugin/skills/hooks route | Unknown | Official Google snippets indicate a route, but static doc extraction and local CLI proof are missing. | Antigravity docs search results and SPA shell fetch. | 73.1.8 official content capture and local availability check. |
| Publicly claiming all Superpowers hosts | Reject | Support claims must be Harness evidence-based. Another repo's manifest is not Harness proof. | Harness `spec.md` Support Tiers; `docs/hokage-spin-off-readiness.md`. | Keep non-proven hosts candidate or future/unsupported. |
| Copying Superpowers skill names/triggers wholesale | Reject | Harness has existing workflow names and contracts. Copying names would create false continuity and confuse docs/tests. | Harness `docs/architecture/hokage-core.md` Explicit Rejects. | Translate concepts into Harness lanes and gates. |
| Treating OpenCode mirror as runtime support | Reject | Static packaging is not runtime bootstrap proof. | Harness `docs/tool-capability-matrix.md`; OpenCode docs. | Require runtime smoke before support tier raise. |

## Host Support Claim Boundary

| Host | Current Phase 73 stance | Claim allowed now | Claim blocked until |
|---|---|---|---|
| Claude Code | `supported` for Claude-first Harness | Public Claude-first support. | Do not imply non-Claude hook parity. |
| Codex CLI | `internal-compatible` | Internal compatibility; existing setup path; direct plugin path under investigation. | Isolated marketplace install and companion smoke pass. |
| Codex App | `candidate` | Candidate under Codex adapter. | App-specific worktree/UI handoff smoke pass. |
| OpenCode | `internal-compatible` | Static/package compatibility and skills mirror wording. | Runtime bootstrap and native skill smoke pass. |
| Cursor | `candidate` | Research/candidate wording only. | Rules/plugin install, bootstrap, and workflow smoke pass. |
| GitHub Copilot CLI | `candidate` | Research/candidate wording only. | Local CLI availability, plugin install, skills/bootstrap smoke pass. |
| Antigravity CLI | `future/unsupported` for public claim | Research note only. | Official doc content captured, local install availability, plugin/skills/hooks route, and bootstrap smoke pass. |

## Target Files For Future Tasks

This research task creates only this file.

Do not start implementation until 73.1.2 freezes the plan.

Expected future file targets:

- Support contract:
  - `spec.md`
  - `docs/architecture/hokage-core.md`
  - `docs/tool-capability-matrix.md`
  - `docs/hokage-spin-off-readiness.md`
- Onboarding docs:
  - `docs/onboarding/index.md`
  - `docs/onboarding/install.md`
  - `docs/onboarding/migration.md`
  - `README.md`
  - `README_ja.md`
- Codex:
  - `.codex-plugin/plugin.json` or equivalent if consumed by setup/preflight in the same phase.
  - `scripts/setup-codex.sh`
  - `scripts/codex-companion.sh`
  - `codex/AGENTS.md`
  - `codex/.codex/skills/`
  - `skills-codex/`
- OpenCode:
  - `opencode/opencode.json`
  - `opencode/AGENTS.md`
  - `opencode/README.md`
  - `opencode/skills/`
  - `scripts/setup-opencode.sh`
  - `scripts/build-opencode.js`
  - `scripts/validate-opencode.js`
- Candidate adapters:
  - Cursor: `.cursor/` or `.cursor-plugin/` only if setup/test/preflight consumes it.
  - GitHub Copilot CLI: plugin manifest only after local CLI route is verified.
  - Antigravity CLI: plugin/profile files only after official route and local availability are verified.
- Tests/preflight:
  - `tests/test-tool-capability-matrix.sh`
  - `tests/test-bootstrap-routing-contract.sh`
  - `tests/test-codex-package.sh`
  - `tests/test-distribution-archive.sh`
  - new Codex plugin smoke test.
  - new OpenCode bootstrap smoke test.
  - new premature-claim wording tests for Cursor, Copilot CLI, and Antigravity CLI.
  - `scripts/release-preflight.sh`
  - `tests/validate-plugin.sh`

## Required Tests And Smoke Plan

73.1.2 plan-freeze tests:

- `bash tests/test-spec-ssot-workflow.sh`
- `bash tests/test-tool-capability-matrix.sh`
- `bash tests/test-bootstrap-routing-contract.sh`
- claim scan: README/docs must not say supported for candidate or unsupported hosts.

73.1.4 Codex tests:

- failing test first for `.codex-plugin/plugin.json` or equivalent manifest.
- isolated `CODEX_HOME` marketplace add/list/install smoke.
- keep `scripts/setup-codex.sh --user` and `--project` fallback.
- companion smoke through `bash scripts/codex-companion.sh task/review`, not raw `codex exec`.

73.1.5 OpenCode tests:

- OpenCode plugin/skill static validation.
- no duplicate bootstrap injection.
- native `skill` loading smoke if local OpenCode is available.
- `node scripts/build-opencode.js`
- `node scripts/validate-opencode.js`

73.1.6-73.1.8 candidate tests:

- local CLI availability check for each host.
- official docs capture artifact.
- isolated install or bootstrap smoke if the host supports it.
- wording gate that fails premature README support claims.

73.1.9 migration tests:

- stale plugin/cache detection.
- duplicate local skills detection.
- backup path reporting.
- rollback proposal.
- no memory DB deletion by default.
- destructive cleanup requires explicit gate.

73.1.10 acceptance tests:

- explicit and implicit skill-trigger fixtures for Harness workflows.
- first prompt transcript fixtures.
- release preflight blocks claimed hosts without smoke evidence.

## Unknowns To Carry Forward

- Cursor direct plugin install command and local runtime behavior were not
  smoked.
- Codex App behavior was not smoked; only Codex CLI help was observed.
- GitHub Copilot CLI was not locally installed or smoked.
- Antigravity CLI official docs were discovered, but direct static text capture
  was incomplete in this session. Treat it as promising but unproven.
- Harness direct marketplace packaging shape for Codex is not finalized.
- Existing user migration should inspect real installed paths before proposing
  cleanup.

## 73.1.2 Entry Criteria

73.1.2 may start when the next agent accepts these constraints:

1. Keep `spec.md` as the support-claim authority.
2. Preserve the support tiers exactly.
3. Do not raise a host tier without install/update/bootstrap/workflow smoke.
4. Keep `not_observed != absent`.
5. Translate Superpowers patterns into Harness lanes and stage gates.
6. Do not create adapter manifest files unless setup, docs generation, tests,
   or release preflight consumes them in the same phase.
7. Keep public README wording conservative until the evidence changes.
