# Phase 73 + Phase 74 PR / Release Closeout

Status: PR closeout artifact
Checked at: 2026-05-22 JST
Phase: `Plans.md` 73.1.13 + 74.1.1-74.1.6

## Support Tiers

| Host | Tier | Release claim |
|---|---|---|
| Claude Code | `supported` | Claude-first plugin route remains the public product surface. |
| Codex CLI | `internal-compatible` | Direct plugin smoke and setup fallback are verified for CLI usage only; Phase 74 adds CI-required Codex CLI plugin install smoke. |
| Codex app | `candidate` | App-specific smoke is still required; no CLI parity claim. |
| OpenCode | `internal-compatible` | Bootstrap plugin and Node-level validation are verified; real binary runtime parity is not claimed. |
| Cursor | `candidate` | Existing docs are handoff integration, not adapter support. |
| GitHub Copilot CLI | `candidate` | Official CLI capability docs are research evidence only. |
| Antigravity CLI | `future/unsupported` | No end-user install route or verified adapter route in this phase. |

`not_observed != absent` remains the release wording rule for all candidate and
unsupported hosts.

## Host Smoke Evidence

Passed:

```bash
bash scripts/sync-skill-mirrors.sh --check
bash tests/test-bootstrap-routing-contract.sh
bash tests/test-tool-capability-matrix.sh
bash tests/test-codex-package.sh
node scripts/validate-opencode.js
bash tests/validate-plugin.sh
bash tests/test-codex-plugin-adapter.sh
bash tests/test-opencode-bootstrap-plugin.sh
bash tests/test-bootstrap-skill-trigger-acceptance.sh
bash tests/test-release-preflight.sh
bash tests/test-distribution-archive.sh
bash tests/test-hokage-spin-off-readiness.sh
bash tests/test-format-lint.sh
bash tests/test-shell-lint.sh
HARNESS_CODEX_PLUGIN_SMOKE_REQUIRED=1 bash tests/test-codex-plugin-adapter.sh
go test ./...
go vet ./...
```

Release preflight:

```bash
bash scripts/release-preflight.sh --check-adapters
```

Result: not release-clean in the current uncommitted worktree.

Explicit unavailable reasons:

- `working tree clean` fails because Phase 73 changes are intentionally
  uncommitted in this worktree.
- `release mirror drift` reports `opencode/AGENTS.md` and `opencode/README.md`
  because adapter docs changed in this same patch; this becomes clean only after
  the PR commit contains those generated outputs.
- CI status is unavailable because the worktree is detached/unpushed.

Phase 74 hardening added:

- `tests/test-format-lint.sh` and `validate-plugin.yml` Go format gate.
- `tests/test-shell-lint.sh` and `validate-plugin.yml` ShellCheck high-risk
  subset gate.
- `.github/workflows/release.yml` release preflight before GitHub Release asset
  creation/upload.
- CI-required Codex CLI adapter smoke with isolated `CODEX_HOME`.
- Explicit OpenCode runtime-unavailable warning while keeping
  `internal-compatible`.

Fixture coverage in `tests/test-release-preflight.sh` passed after adding
Codex plugin and OpenCode bootstrap smoke stubs to the release-preflight fixture
repo.

## Migration Impact

Existing users get a report-only migration path:

```bash
bin/harness doctor --migration-report
```

The report inventories stale Claude plugin cache entries, missing slash entries,
duplicate Codex local skills, old symlinks, Codex backup path, OpenCode backup
path, and harness-mem state. It does not delete plugin cache, local skills,
OpenCode files, symlinks, backups, or memory DB data.

## Review Findings

Accepted and fixed:

- Release-preflight fixture repos needed stubs for newly hard-gated adapter
  smoke scripts.
- `.codex-plugin/` needed to be included in adapter gate path detection.
- `.codex-plugin/` needed to be excluded from Claude release archives because
  `codex/` is export-ignored.

Details: [phase-73-review-closeout.md](phase-73-review-closeout.md).

## Residual Risk

- Companion re-review hung while attempting `harness/harness_mem_search`; the
  completed review findings were accepted and fixed, but harness-mem MCP search
  is not treated as reliable review evidence in this environment.
- OpenCode runtime native-skill smoke is not observed locally because the real
  `opencode` binary is unavailable behind the Superset wrapper.
- `bash scripts/release-preflight.sh --check-adapters` still cannot be claimed
  PASS in this dirty local worktree; it should be rerun after commit in a clean
  release context.
- Codex app, Cursor, GitHub Copilot CLI, and Antigravity CLI remain outside
  public release claims until host-specific smoke exists.

## PR Body Draft

### Summary

- Adds Phase 73 tool-first onboarding, support tiers, and host claim boundaries.
- Adds Codex CLI direct plugin smoke, OpenCode bootstrap plugin, and candidate
  evidence docs for Cursor, GitHub Copilot CLI, and Antigravity CLI.
- Adds existing-user migration reporting through
  `harness doctor --migration-report`.
- Adds bootstrap/skill-trigger acceptance and release-preflight adapter smoke
  gates.
- Adds Phase 74 repo-health gates: Go format, ShellCheck high-risk subset,
  release workflow preflight, and CI-required Codex CLI runtime smoke.

### Verification

- `bash scripts/sync-skill-mirrors.sh --check`
- `bash tests/test-bootstrap-routing-contract.sh`
- `bash tests/test-tool-capability-matrix.sh`
- `bash tests/test-codex-package.sh`
- `node scripts/validate-opencode.js`
- `bash tests/validate-plugin.sh`
- `bash tests/test-release-preflight.sh`
- `bash tests/test-distribution-archive.sh`
- `bash tests/test-format-lint.sh`
- `bash tests/test-shell-lint.sh`
- `HARNESS_CODEX_PLUGIN_SMOKE_REQUIRED=1 bash tests/test-codex-plugin-adapter.sh`
- `go test ./...`
- `go vet ./...`

### Release Boundary

Release readiness is evidence-backed for Claude Code plus internal Codex CLI /
OpenCode compatibility. Candidate and future/unsupported hosts are documented
without being promoted. Final release-ready still requires clean-tree
`bash scripts/release-preflight.sh --check-adapters` and GitHub CI on the PR/tag
context.
