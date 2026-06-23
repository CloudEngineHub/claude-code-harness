# Local dogfood / release alignment snapshot - 2026-06-22

## Conclusion

Do not replace the current Claude Code local dogfood runtime with the public
release line yet if Phase 92.2.4 behavior matters. Public `v4.16.1` is newer and
already contains Codex `breezing --cursor` skill support, but it does not contain
commit `cbc4c904` (`Phase 92.2.4 - worktree-escape OS temp allowlist`).

The Codex local skill cache is a separate problem: the active Codex plugin cache
at `~/.codex/plugins/cache/claude-code-harness-local/claude-code-harness/4.12.11`
was older than both `v4.15.0` and `v4.16.1` for `breezing` / `harness-work`.
It should be refreshed from the repo's Codex skill SSOT so `--cursor` is usable
immediately in other Codex work.

## Phase 103 update - 2026-06-23

| Surface | Current state | What it means |
|---|---|---|
| Public latest | `v4.16.1` remains latest (`gh release list`, published 2026-06-19T03:42:48Z) | Public users still do not have the Phase 92.2.4 temp allowlist. |
| Release candidate | local branch `release/v4.16.2-phase103`, ahead of `origin/main` by 4 commits: `408fed5b`, `bc96f759`, `8ca46950`, `1138cbf6` | Ready locally for PR/release gate. It promotes runtimefloor/temp allowlist, bumps metadata to `4.16.2`, sanitizes release delegation wording, and rebuilds 4 platform binaries. |
| Local Claude dogfood | version label `4.15.0`, binary contains `runtimefloor` / `allowlistedTempRoots` behavior from `cbc4c904` | Still a dogfood patch, not a clean public install. Keep until `v4.16.2` or equivalent is published and verified. |
| Redesign branch | `plan/zero-base-redesign` at `cbc4c904`, `VERSION=4.15.0`, plus PR #225 release-delegation safety files patched in | No full merge/rebase from old main. Only release safety knowledge was imported so HOTL/redesign work can resume without rolling back to older specs. |

### Version / capability comparison

| Version/surface | Performance / capability profile | Release status |
|---|---|---|
| `v4.16.1` public | Official latest. Has PR #225 release workflow delegation on `origin/main`, but lacks local dogfood Phase 92.2.4 runtimefloor temp allowlist. | Released. |
| Local applied dogfood `4.15.0+cbc4c904` | Best local UX for current CC sessions: `/tmp` / cache cleanup no longer hard-stops, while real data-loss paths still escalate. Version label is stale by design. | Local-only test patch. |
| `v4.16.2` release candidate | Intended public replacement: official release-line base + runtimefloor + temp allowlist + PR #225 delegation + rebuilt binaries. | Local candidate only; push/PR/tag/public release require explicit external-release GO. |
| Redesign branch `plan/zero-base-redesign` | Design/research line for new HOTL/redesign. Keeps Phase 92.2.4 and receives PR #225 safety contract, but does not accept old HOTL/Fleet/UI defaults by merge. | Development branch, not release source. |

### Phase 103 decision

- Use `release/v4.16.2-phase103` as the release candidate surface, not `plan/zero-base-redesign`.
- Keep `plan/zero-base-redesign` on its redesign path. Import only Phase 92.2.4 runtime safety and PR #225 release delegation.
- Do not update the local dogfood install from public latest until the public artifact is verified to include Phase 92.2.4 behavior by binary/hash/behavior.
- Tag, GitHub Release, plugin update, and public cache replacement remain external-release gates.

## Evidence

| Surface | Observed value | Evidence |
|---|---:|---|
| Public latest release | `v4.16.1` | `gh release list --repo Chachamaru127/claude-code-harness --limit 3` |
| Release line version files | `4.16.1` | `git show v4.16.1:VERSION`; `.claude-plugin/plugin.json`; `.codex-plugin/plugin.json`; `harness.toml` |
| Current dev branch | `plan/zero-base-redesign` at `cbc4c904` | `git rev-parse --abbrev-ref HEAD`; `git log -1 --oneline` |
| Current dev version files | `4.15.0` | `cat VERSION`; `.claude-plugin/plugin.json`; `harness.toml` |
| Local Claude plugin registry | `claude-code-harness` version `4.15.0` | `claude plugin list`; `~/.claude/plugins/installed_plugins.json` |
| Local Claude patched binary | `4.15.0` label, but contains `cbc4c904` binary behavior | binary shasum equality with repo `bin/harness-darwin-arm64`; strings include `runtimefloor`, `allowlistedTempRoots`, `worktree-escape` |
| Public release contains Codex `--cursor` skill support | yes | `git show v4.16.1:skills-codex/breezing/SKILL.md` grep `--cursor` |
| Public release contains Phase 92.2.4 temp allowlist | no | `git show v4.16.1:go/internal/runtimefloor/runtimefloor.go` grep `allowlistedTempRoots` returns 0 |
| Current branch commit in release tags | no | `git tag --contains cbc4c904...` returns 0 |
| harness-mem prior context | local cache was patched for Phase 92.2.4 dogfood | `obs_00mqo0jxv88e8ec30efe8371f2` |

## Version / capability table

| Name | Actual state | Capability / safety profile | Recommendation |
|---|---|---|---|
| Codex local cache `4.12.11` | Active Codex skill bundle before this sync. `breezing` lacked `--cursor` / backend selection text. | Oldest capability. Bad fit for current cross-tool work because `composer` / `--cursor` intent is not visible to the Codex skill. | Refresh the two Codex skill files locally now. Low risk; docs/skill-only. |
| Public release `v4.16.1` | Latest public release, version files all `4.16.1`. | Newer release line. Includes Codex `breezing --cursor` support. Does not include Phase 92.2.4 temp allowlist. | Good official line, but not a drop-in replacement for the dogfood Claude runtime if Phase 92.2.4 matters. |
| Local Claude applied runtime | Registry says `4.15.0`, but binary cache was manually patched from `cbc4c904`. | Has Phase 92.2.4 behavior: OS temp cleanup no longer hard-stops as worktree escape; real data-loss paths still stop. Version label remains `4.15.0`, so it is intentionally a dogfood patch, not a clean release install. | Keep for now until `cbc4c904` or equivalent is released. |
| Development branch `plan/zero-base-redesign` | `VERSION=4.15.0`, HEAD `cbc4c904`, ahead/behind `origin/main`. | Development/research line with redesign/HOTL work and Phase 92.2.4. Not identical to public release and not installed as a full plugin. | Resume work here after the local cache sync and review closeout. |

## Decision matrix

| Question | Answer | Why |
|---|---|---|
| 1. What should the release line update include? | Promote/backport Phase 92.2.4 into a future release if the temp-allowlist behavior is desired publicly. Codex `--cursor` is already in `v4.16.1`. | The only confirmed missing dogfood feature in `v4.16.1` is `cbc4c904`; tag grep shows `--cursor` is already present. |
| 2. Can the local dogfood runtime be updated to the public release now? | Not recommended as the default. | A plain update to `v4.16.1` would move to official bits but likely drop the Phase 92.2.4 local patch. |
| 3. How to get Codex `breezing --cursor` locally now? | Sync `skills-codex/breezing` and `skills-codex/harness-work` into the active Codex cache `4.12.11`. | Current repo and public release already contain the contract; the active cache is simply stale. |
| 4. What should be reflected in development version? | Keep the existing Codex skill support as-is; record this alignment doc and Plans.md Phase 102. | No code delta is needed in dev for `--cursor`; the repo SSOT already has it. |
| 5. When to resume development work? | After cache sync, mirror/repo checks, and review approve. | This avoids continuing HOTL/redesign work while the operator-facing tool surface remains stale. |

## Spec result

Spec skip reason:

- path checked: `spec.md`
- reason: Existing `Execution Backend Contract` already defines backend
  precedence, resolver-only backend selection, Codex/Cursor companion boundary,
  and `/breezing` argument surface including `--cursor`.
- preserve in: `Plans.md` Phase 102 and this snapshot document.
