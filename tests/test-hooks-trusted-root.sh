#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

for hooks in "$ROOT/hooks/hooks.json" "$ROOT/.claude-plugin/hooks.json"; do
  [ -f "$hooks" ] || { echo "missing $hooks" >&2; exit 1; }
  if grep -q 'CLAUDE_PROJECT_DIR' "$hooks"; then
    echo "$hooks must not fall back to CLAUDE_PROJECT_DIR" >&2
    exit 1
  fi
  if grep -q 'for c in .*"\$PWD"' "$hooks"; then
    echo "$hooks must not fall back to PWD" >&2
    exit 1
  fi
  if ! grep -Fq '.claude-plugin/plugin.json' "$hooks" || ! grep -Fq 'claude-code-harness' "$hooks"; then
    echo "$hooks must require claude-code-harness plugin identity" >&2
    exit 1
  fi
done

cmp -s "$ROOT/hooks/hooks.json" "$ROOT/.claude-plugin/hooks.json" || {
  echo "dual hooks.json files differ" >&2
  exit 1
}

echo "test-hooks-trusted-root: ok"
