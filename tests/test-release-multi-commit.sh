#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD="$ROOT/scripts/pretooluse-guard.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

run_guard() {
  local cwd="$1"
  local command="$2"
  (cd "$cwd" && printf '{"tool_name":"Bash","tool_input":{"command":%s},"cwd":%s}\n' \
    "$(printf '%s' "$command" | jq -Rs .)" \
    "$(printf '%s' "$cwd" | jq -Rs .)" \
    | bash "$GUARD")
}

assert_not_denied() {
  local label="$1"
  local output="$2"
  local decision
  decision="$(printf '%s' "$output" | jq -r '.hookSpecificOutput.permissionDecision // empty' 2>/dev/null || true)"
  if [ "$decision" = "deny" ]; then
    echo "$label: expected allow/no decision, got deny: $output" >&2
    exit 1
  fi
}

assert_denied() {
  local label="$1"
  local output="$2"
  local decision
  decision="$(printf '%s' "$output" | jq -r '.hookSpecificOutput.permissionDecision // empty' 2>/dev/null || true)"
  if [ "$decision" != "deny" ]; then
    echo "$label: expected deny, got: ${output:-<empty>}" >&2
    exit 1
  fi
}

cd "$TMP"
git init -q
git config user.email test@example.com
git config user.name Test
printf '0.0.0\n' > VERSION
mkdir -p .claude-plugin
printf '{"name":"claude-code-harness","version":"0.0.0"}\n' > .claude-plugin/plugin.json
printf 'x\n' > app.txt
git add VERSION .claude-plugin/plugin.json app.txt
git commit -qm seed

printf '0.0.1\n' > VERSION
git add VERSION
out="$(run_guard "$TMP" 'git commit -m "chore: bump version"')"
assert_not_denied "bookkeeping VERSION-only commit" "$out"
[ -s .claude/state/commit-cleanup-audit.jsonl ] || {
  echo "bookkeeping commit must append cleanup audit" >&2
  exit 1
}

printf 'change\n' >> app.txt
git add app.txt
out="$(run_guard "$TMP" 'git commit -m "feat: app change"')"
assert_denied "non-bookkeeping commit still requires review" "$out"

git reset -q
printf '0.0.2\n' > VERSION
git add VERSION
out="$(run_guard "$TMP" 'git add app.txt && git commit -m "chore: unsafe combined bump"')"
assert_denied "combined git add and commit is not bookkeeping-exempt" "$out"

out="$(run_guard "$TMP" 'git commit -m "chore: pathspec" VERSION')"
assert_denied "commit pathspec is not bookkeeping-exempt" "$out"

out="$(run_guard "$TMP" 'git commit -am "chore: all flag"')"
assert_denied "commit -am is not bookkeeping-exempt" "$out"

echo "test-release-multi-commit: ok"
