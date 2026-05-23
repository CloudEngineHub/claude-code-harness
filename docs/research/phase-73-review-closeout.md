# Phase 73 Review Closeout

Status: review gate complete
Checked at: 2026-05-22 JST
Phase: `Plans.md` 73.1.12

## Review Runs

Primary review:

```bash
bash scripts/codex-companion.sh review --wait --base HEAD --scope working-tree
```

Result: `REQUEST_CHANGES`.

Accepted findings:

1. `scripts/release-preflight.sh` hard-failed fixture repos when newly required
   adapter smoke scripts were absent.
2. `.codex-plugin/` was missing from adapter gate path detection.
3. `.codex-plugin/plugin.json` would be included in release archives while
   `codex/` was export-ignored, leaving a broken Codex plugin manifest.

Fixes applied:

- `tests/test-release-preflight.sh` fixture repos now include smoke stubs for
  Codex plugin, OpenCode bootstrap, and skill-trigger acceptance gates.
- `scripts/release-preflight.sh` includes `.codex-plugin/` in adapter gate
  path detection.
- `.gitattributes` export-ignores `.codex-plugin/`.
- `tests/test-distribution-archive.sh` forbids `.codex-plugin/` in release
  archives.

Second review attempt:

```bash
bash scripts/codex-companion.sh review --wait --base HEAD --scope working-tree
```

Result: stopped after it hung while attempting `harness/harness_mem_search`.
This is treated as a reviewer availability issue, not an approval. The accepted
findings from the completed review remain the review gate evidence.

## Verification After Fixes

```bash
bash tests/test-release-preflight.sh
bash tests/test-distribution-archive.sh
bash tests/test-bootstrap-skill-trigger-acceptance.sh
bash tests/test-codex-package.sh
```

All passed after the fixes.

## Residual Risk

- harness-mem MCP search is not reliable in this environment during companion
  review; do not depend on it as the only review evidence.
- Release preflight still cannot be run end-to-end on the dirty working tree
  because its normal contract includes a clean worktree check. Fixture coverage
  verifies the added adapter gates.
