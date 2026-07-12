#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PUBLIC_FILES=(
  "${ROOT_DIR}/README.md"
  "${ROOT_DIR}/README_ja.md"
  "${ROOT_DIR}/docs/onboarding/index.md"
  "${ROOT_DIR}/docs/onboarding/install.md"
  "${ROOT_DIR}/docs/onboarding/migration.md"
  "${ROOT_DIR}/docs/onboarding/skill-trigger-acceptance.md"
  "${ROOT_DIR}/docs/CURSOR_INTEGRATION.md"
  "${ROOT_DIR}/docs/research/cursor-adapter-candidate.md"
  "${ROOT_DIR}/docs/research/hermes-agent-candidate.md"
  "${ROOT_DIR}/.cursor-plugin/plugin.json"
  "${ROOT_DIR}/docs/research/github-copilot-cli-adapter.md"
  "${ROOT_DIR}/docs/research/antigravity-cli-adapter.md"
)

fail() {
  echo "test-support-claim-wording: FAIL: $1" >&2
  exit 1
}

NON_PUBLIC_HOSTS='Codex App|Codex app|Cursor|Hermes Agent|Hermes|GitHub Copilot CLI|Copilot CLI|Antigravity CLI|Antigravity'
SUPPORT_WORDS='[^[:alpha:]]supported([^[:alpha:]]|$)|サポート済み|サポート対象|対応済み|正式対応'
ALLOW_DENIAL='not a public|no public|not .*supported|do not claim|must not claim|never claim|Blocked|Blocked wording|blocked|Do not promote|until .*support|正式対応ではない|正式対応を主張しない|主張しない|未主張|not claimed|not proven|no public supported|public support claim|support wording|Blocked Wording'

for file in "${PUBLIC_FILES[@]}"; do
  [ -f "$file" ] || fail "missing ${file}"

  if grep -Eiq "(${NON_PUBLIC_HOSTS}).{0,100}(${SUPPORT_WORDS})|(${SUPPORT_WORDS}).{0,100}(${NON_PUBLIC_HOSTS})" "$file"; then
    if grep -Ein "(${NON_PUBLIC_HOSTS}).{0,100}(${SUPPORT_WORDS})|(${SUPPORT_WORDS}).{0,100}(${NON_PUBLIC_HOSTS})" "$file" \
      | grep -Eiv "$ALLOW_DENIAL" \
      | grep -Eq .; then
      fail "candidate/unsupported host appears supported in ${file}"
    fi
  fi
done

echo "test-support-claim-wording: ok"
