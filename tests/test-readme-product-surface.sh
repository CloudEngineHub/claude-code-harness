#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  echo "test-readme-product-surface: FAIL: $1" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local needle="$2"
  grep -Fq "$needle" "$file" || fail "missing '$needle' in $file"
}

for file in "$ROOT_DIR/README.md" "$ROOT_DIR/README_ja.md"; do
  [ -f "$file" ] || fail "missing $file"
  assert_contains "$file" "## Quickstart"
  assert_contains "$file" "## How It Works"
  assert_contains "$file" "## Install By Tool"
  assert_contains "$file" "## First 15 Minutes"
  assert_contains "$file" "## Existing User Migration"
  assert_contains "$file" "## Advanced"
  assert_contains "$file" "docs/onboarding/index.md"
  assert_contains "$file" "docs/onboarding/migration.md"
  assert_contains "$file" "docs/onboarding/skill-trigger-acceptance.md"
  assert_contains "$file" "Codex app | \`candidate\`"
  assert_contains "$file" "Cursor | \`candidate\`"
  assert_contains "$file" "GitHub Copilot CLI | \`candidate\`"
  assert_contains "$file" "Antigravity CLI | \`future/unsupported\`"
done

echo "test-readme-product-surface: ok"
