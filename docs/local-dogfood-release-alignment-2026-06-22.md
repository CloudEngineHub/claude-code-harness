# Local dogfood / release alignment snapshot - 2026-06-22

## Conclusion

Public latest is now `v4.16.2`. It includes the Phase 92.2.4-equivalent
worktree-escape OS temp allowlist and PR #225 release workflow delegation.

The local Claude Code user-scope plugin has been updated from `4.15.0` to
`4.16.2`. Claude Code reports that a restart is required to apply the update.
Because the plugin cache update populated manifest/skills but not `bin/`, the
local `4.16.2` cache `bin/` was filled from the published `v4.16.2` release
assets so hooks can still resolve `bin/harness` after restart.

The redesign branch remains a development surface, not a release source. It keeps
its zero-base redesign direction; the release work does not authorize a full
merge/rebase that would mechanically restore old HOTL/Fleet/UI defaults. The
knowledge from those areas is still carried forward by translating it into the
new design.

## Phase 103 closeout - 2026-06-24

| Surface | Current state | What it means |
|---|---|---|
| Public latest | `v4.16.2` is Latest (`gh release list`, published 2026-06-24T02:50:47Z) | Public users now have the release-line safety promotion and PR #225 workflow delegation. |
| Release source | PR #226 merged to `origin/main` at `78a8c3b5`; tags `v4.16.2` and `claude-code-harness--v4.16.2` pushed | Release source was `release/v4.16.2-phase103`, not `plan/zero-base-redesign`. |
| GitHub Release | `auto-release` run `28071678901` succeeded; `release-verify-publish` PASS with 4 assets | Publish was delegated to workflow; operator verification used script/API polling. |
| Local Claude applied plugin | `claude-code-harness@claude-code-harness-marketplace` user scope is `4.16.2`, enabled; restart required | Local installed version now matches public latest. Cache `bin/` was filled from published assets because hooks require `bin/harness`. |
| Redesign branch | `plan/zero-base-redesign` remains a dev surface with `VERSION=4.15.0` and PR #225 safety files patched in | Work can resume without treating old main/release specs as the redesign source of truth. HOTL/Fleet/UI lessons still carry forward as translated inputs. |

### Version / capability comparison

| Version/surface | Performance / capability profile | Release status |
|---|---|---|
| `v4.16.2` public/latest | Official release-line base + runtimefloor/temp allowlist + PR #225 delegation + 4 platform release assets. | Released and Latest. |
| Local applied Claude plugin `4.16.2` | Installed user-scope plugin is `4.16.2`; cache binaries match published release asset SHA256 digests; `bin/harness --version` returns `4.16.2 (Hokage)`. | Applied locally; Claude Code restart required. |
| Previous dogfood `4.15.0+cbc4c904` | The old local-only safety patch is now superseded by public `v4.16.2`. | Historical dogfood surface only. |
| Redesign branch `plan/zero-base-redesign` | Design/research line for new HOTL/redesign. Keeps Phase 92.2.4 and receives PR #225 safety contract. It does not accept old HOTL/Fleet/UI defaults by mechanical merge, but it does inherit their lessons by translation. | Development branch, not release source. |

### Phase 103 decision

- Completed: use `release/v4.16.2-phase103` as the release surface, not `plan/zero-base-redesign`.
- Completed: publish `v4.16.2` through workflow delegation and verify with release script/API.
- Completed: update the local Claude plugin after verifying public asset version/hash/behavior.
- Still preserved: keep `plan/zero-base-redesign` on its redesign path, import allowlisted safety/release changes directly, and translate HOTL/Fleet/UI lessons into the new design.

## Evidence

| Surface | Observed value | Evidence |
|---|---:|---|
| Public latest release | `v4.16.2` | `gh release list --repo Chachamaru127/claude-code-harness --limit 5` |
| Release source commit | `78a8c3b596fae82422e2c167dd222eba32cfc548` | PR #226 merge commit; `gh run view 28071678901` headSha |
| Release publish workflow | success | `gh run watch 28071678901 --exit-status`; `gh run view 28071678901` |
| Public artifact verification | PASS, 4 assets | `bash scripts/release-verify-publish.sh v4.16.2 Chachamaru127/claude-code-harness` |
| Published darwin-arm64 asset | `6dd13bd4743182a293b3756d9789494c26b045d64a7c59c3a59c5a5f71dd3bd7` | `gh release view v4.16.2 --json assets`; local cache `shasum -a 256` |
| Published runtime behavior | `4.16.2 (Hokage)`, contains `allowlistedTempRoots` / `runtimefloor` | downloaded release asset `--version`; `strings ... | rg` |
| Local Claude plugin registry | `claude-code-harness` version `4.16.2` | `claude plugin list`; `~/.claude/plugins/installed_plugins.json` |
| Local Claude plugin cache | `installPath` points to `.../claude-code-harness/4.16.2` at `gitCommitSha=78a8c3b5` | `~/.claude/plugins/installed_plugins.json` |
| Current dev branch | `plan/zero-base-redesign` at `c61247ac` | `git rev-parse --abbrev-ref HEAD`; `git log -1 --oneline` |
| Current dev version files | `4.15.0` | intentionally retained as dev surface, not public release metadata |
| harness-mem prior context | local cache was patched for Phase 92.2.4 dogfood | `obs_00mqo0jxv88e8ec30efe8371f2` |

## Version / capability table

| Name | Actual state | Capability / safety profile | Recommendation |
|---|---|---|---|
| Codex local cache `4.12.11` | Previously stale; Phase 102 refreshed the relevant Codex skill files from repo SSOT. | Solves Codex `breezing --cursor` visibility without changing release source. | No new action from Phase 103. |
| Public release `v4.16.2` | Latest public release, version files and GitHub Release are `4.16.2`. | Includes Codex skill support, Phase 92.2.4-equivalent runtimefloor temp allowlist, PR #225 workflow delegation, and 4 platform assets. | Use as the public baseline. |
| Local Claude applied runtime | User-scope plugin registry says `4.16.2`; cache `bin/` matches published release asset SHA256 digests. | Clean public install surface is now aligned with the safety behavior previously tested as dogfood. Current Claude process may still need restart. | Treat local applied version as `4.16.2` after restart. |
| Development branch `plan/zero-base-redesign` | `VERSION=4.15.0`, HEAD `c61247ac`, ahead/behind `origin/main`, PR #225 safety patched in. | Development/research line with redesign/HOTL work. Not identical to public release and not the release source. | Resume HOTL/redesign here, using `v4.16.2` as public baseline evidence only. |

## Decision matrix

| Question | Answer | Why |
|---|---|---|
| 1. Was the dogfood-only safety behavior promoted? | Yes. `v4.16.2` public contains the runtimefloor/temp allowlist promotion. | PR #226 merged the release branch and release assets contain `allowlistedTempRoots` / `runtimefloor`. |
| 2. Can the local dogfood runtime be updated to public latest now? | Yes, and it has been updated to `4.16.2`. | The public asset was verified by version/hash/symbol before local update/cache repair. |
| 3. Does this mean redesign should merge old main wholesale? | No. | Public release closeout and zero-base redesign are different truth surfaces. Safety/release changes may flow directly; HOTL/Fleet/UI lessons should flow as translated design inputs, not old defaults. |
| 4. What should be reflected in development version? | Keep dev metadata (`VERSION=4.15.0`) until redesign itself intentionally bumps. | `4.16.2` metadata belongs to the release line, not the current redesign branch. |
| 5. When to resume development work? | Now, after Claude restart if relying on updated Claude plugin hooks. | Release, local install, and handoff docs are closed; remaining work is HOTL/redesign, not release gate. |

## Spec result

Spec skip reason:

- path checked: `spec.md`
- reason: Existing `Execution Backend Contract` already defines backend
  precedence, resolver-only backend selection, Codex/Cursor companion boundary,
  and `/breezing` argument surface including `--cursor`.
- preserve in: `Plans.md` Phase 102 and this snapshot document.
