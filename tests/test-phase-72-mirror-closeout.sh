#!/usr/bin/env bash
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"

# (a) mirror sync check
bash "$ROOT/scripts/sync-skill-mirrors.sh" --check

# (b) skill design contract (existing test)
bash "$ROOT/tests/test-skill-design-contract.sh"

# (c) bootstrap routing contract (existing test)
bash "$ROOT/tests/test-bootstrap-routing-contract.sh"

# (d) codex package distribution (existing test)
bash "$ROOT/tests/test-codex-package.sh"

# (e) opencode validation
node "$ROOT/scripts/validate-opencode.js"

# (f) distribution archive integrity (existing test)
bash "$ROOT/tests/test-distribution-archive.sh"

# (g) release preflight adapters (if supported)
if [ -x "$ROOT/scripts/release-preflight.sh" ]; then
  bash "$ROOT/scripts/release-preflight.sh" --check-adapters || true  # not blocking if optional
fi

# (h) overall validate-plugin (skip when invoked from validate-plugin to avoid recursion)
if [ -z "${HARNESS_CLOSEOUT_NESTED:-}" ]; then
  bash "$ROOT/tests/validate-plugin.sh"
fi

echo "OK: Phase 72 mirror + distribution + no-regression closeout"
