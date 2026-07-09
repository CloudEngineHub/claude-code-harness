#!/usr/bin/env bash
# Blocks public surfaces from claiming non-Claude hosts as public "supported"
# or Japanese 正式対応 / 対応済み / サポート対象 without matching tier evidence.
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
  "${ROOT_DIR}/docs/research/grok-adapter-candidate.md"
  "${ROOT_DIR}/.cursor-plugin/plugin.json"
  "${ROOT_DIR}/.grok-plugin/plugin.json"
  "${ROOT_DIR}/docs/research/github-copilot-cli-adapter.md"
  "${ROOT_DIR}/docs/research/antigravity-cli-adapter.md"
)

# Hosts that must not receive a public supported / 正式対応 claim.
# Claude Code is intentionally omitted (only public supported host today).
# Codex CLI is still internal-compatible: block bare "supported Codex" claims
# that are not clearly scoped as non-public (tests use "supported" only in
# blocked-wording tables with care — prefer "public support" phrasing).
NON_PUBLIC_HOSTS='Codex App|Codex app|Cursor|Grok|GitHub Copilot CLI|Copilot CLI|Antigravity CLI|Antigravity'
# JP phrases that map to public supported for non-Claude hosts.
JP_PUBLIC_SUPPORT='正式対応|サポート済み|サポート対象|対応済み'

fail() {
  echo "test-support-claim-wording: FAIL: $1" >&2
  exit 1
}

for file in "${PUBLIC_FILES[@]}"; do
  [ -f "$file" ] || fail "missing ${file}"

  # host ... supported / 対応済み
  if grep -Eiq "(${NON_PUBLIC_HOSTS}).{0,100}([^[:alpha:]]supported([^[:alpha:]]|$)|${JP_PUBLIC_SUPPORT})" "$file"; then
    # Allow explicit denial phrases that mention the blocked words as forbidden.
    if grep -Eiq "(${NON_PUBLIC_HOSTS}).{0,100}(no public supported|not .*supported|not a public supported|public supported claim|blocked wording|do not claim|must not claim|never claim|正式対応ではない|正式対応を主張しない|正式対応にしない)" "$file"; then
      :
    else
      # Line-level escape: blocked-wording tables and "Do not claim" columns.
      if grep -Ein "(${NON_PUBLIC_HOSTS}).{0,100}([^[:alpha:]]supported([^[:alpha:]]|$)|${JP_PUBLIC_SUPPORT})" "$file" \
        | grep -Eiv 'no public supported|not a public|not .*supported|blocked wording|Do not claim|must not|never claim|禁止|主張しない|正式対応ではない|candidate route|remains `candidate`|remains `internal-compatible`|tier stays|until .*supported|waits for|gated on|not claimed|not claim|≠|!=|ではなく' \
        | grep -Eq .; then
        fail "candidate/internal host appears publicly supported in ${file}"
      fi
    fi
  fi

  # supported / 正式対応 ... host
  if grep -Eiq "([^[:alpha:]]supported([^[:alpha:]]|$)|${JP_PUBLIC_SUPPORT}).{0,100}(${NON_PUBLIC_HOSTS})" "$file"; then
    if grep -Ein "([^[:alpha:]]supported([^[:alpha:]]|$)|${JP_PUBLIC_SUPPORT}).{0,100}(${NON_PUBLIC_HOSTS})" "$file" \
      | grep -Eiv 'no public supported|Do not claim|must not|never claim|blocked wording|禁止|主張しない|正式対応ではない|not claim|until |gated |≠|!=' \
      | grep -Eq .; then
      fail "support wording implies non-public host support in ${file}"
    fi
  fi
done

# Hard fails: clear overclaim phrases (no denial context needed).
OVERCLAIMS=(
  'supported Grok adapter'
  'supported Cursor adapter'
  'Grok is `supported`'
  'Grok is \*\*supported\*\*'
  'Cursor is `supported`'
  '正式対応.*Grok'
  'Grok.*正式対応'
  '正式対応.*Cursor'
  'Cursor.*正式対応'
)

for file in "${PUBLIC_FILES[@]}"; do
  for pat in "${OVERCLAIMS[@]}"; do
    if grep -Eqi "$pat" "$file"; then
      # allow only if the same line denies the claim
      if grep -Ein "$pat" "$file" | grep -Eiv 'Do not claim|blocked|禁止|must not|never|ではない|主張しない|no public|not a public|remains `' | grep -Eq .; then
        fail "overclaim pattern '${pat}' in ${file}"
      fi
    fi
  done
done

# Positive pins: public surfaces still name Claude as supported and others lower.
assert_contains() {
  local file="$1"
  local needle="$2"
  grep -Fq "$needle" "$file" || fail "missing '${needle}' in ${file}"
}

assert_contains "${ROOT_DIR}/README.md" "| Claude Code | \`supported\` |"
assert_contains "${ROOT_DIR}/README.md" "| Cursor | \`internal-compatible\` |"
assert_contains "${ROOT_DIR}/README.md" "| Grok | \`internal-compatible\` |"
assert_contains "${ROOT_DIR}/docs/onboarding/index.md" "| Grok | \`internal-compatible\` |"

echo "test-support-claim-wording: ok"
