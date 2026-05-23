#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOC="${ROOT_DIR}/docs/onboarding/skill-trigger-acceptance.md"
COMPANION="${ROOT_DIR}/scripts/codex-companion.sh"
OPENCODE_PLUGIN="${ROOT_DIR}/opencode/plugins/harness-bootstrap.mjs"
RELEASE_PREFLIGHT="${ROOT_DIR}/scripts/release-preflight.sh"

fail() {
  echo "test-bootstrap-skill-trigger-acceptance: FAIL: $1" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local needle="$2"
  grep -Fq "$needle" "$file" || fail "missing '$needle' in $file"
}

[ -f "$DOC" ] || fail "missing $DOC"
assert_contains "$DOC" "Phase 73 treats installation as incomplete"
assert_contains "$DOC" 'scripts/codex-companion.sh task --write'
assert_contains "$DOC" 'scripts/codex-companion.sh review --base'
assert_contains "$DOC" 'raw `codex exec` is not the'
assert_contains "$DOC" "opencode/plugins/harness-bootstrap.mjs"
assert_contains "$DOC" "not_observed != absent"
assert_contains "$DOC" "Claimed hosts without smoke evidence are release blockers"

required_skills=(
  harness-plan
  harness-work
  harness-review
  harness-release
  harness-setup
  harness-sync
  breezing
)

for skill in "${required_skills[@]}"; do
  [ -f "${ROOT_DIR}/skills/${skill}/SKILL.md" ] || fail "missing Claude skill ${skill}"
  [ -f "${ROOT_DIR}/codex/.codex/skills/${skill}/SKILL.md" ] || fail "missing Codex skill ${skill}"
  [ -f "${ROOT_DIR}/opencode/skills/${skill}/SKILL.md" ] || fail "missing OpenCode skill ${skill}"
done

assert_contains "$COMPANION" 'bash scripts/codex-companion.sh task --write'
assert_contains "$COMPANION" 'bash scripts/codex-companion.sh review --base'
assert_contains "$COMPANION" 'raw `codex exec` ではなく'

assert_contains "$OPENCODE_PLUGIN" "HARNESS_BOOTSTRAP"
assert_contains "$OPENCODE_PLUGIN" "experimental.chat.messages.transform"
assert_contains "$OPENCODE_PLUGIN" "not_observed != absent"
assert_contains "$OPENCODE_PLUGIN" "internal-compatible; it is not public supported parity"

assert_contains "$RELEASE_PREFLIGHT" "tests/test-bootstrap-skill-trigger-acceptance.sh"
assert_contains "$RELEASE_PREFLIGHT" "tests/test-codex-plugin-adapter.sh"
assert_contains "$RELEASE_PREFLIGHT" "tests/test-opencode-bootstrap-plugin.sh"
assert_contains "$RELEASE_PREFLIGHT" "codex plugin adapter smoke"
assert_contains "$RELEASE_PREFLIGHT" "opencode bootstrap smoke"

echo "test-bootstrap-skill-trigger-acceptance: ok"
