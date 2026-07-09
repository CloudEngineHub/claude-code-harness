#!/usr/bin/env bash
# Print copy-paste blocks for live CLI H4 smoke (operator runbook).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOC="${ROOT_DIR}/docs/onboarding/host-live-cli-smoke.md"
HOST="${1:-}"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/print-live-cli-smoke.sh           # full runbook path + keywords
  bash scripts/print-live-cli-smoke.sh keywords  # reply phrases only
  bash scripts/print-live-cli-smoke.sh claude|codex|cursor|grok|prep|check
EOF
}

if [ ! -f "$DOC" ]; then
  echo "missing $DOC" >&2
  exit 1
fi

case "${HOST}" in
  ""|all|help|-h|--help)
    usage
    echo
    echo "Runbook: $DOC"
    echo
    echo "=== keywords (paste results as one line) ==="
    grep -E '^\| `LIVE:' "$DOC" | sed 's/^| //;s/ |.*//'
    echo
    echo "Open full copy-paste blocks: less $DOC"
    echo "Or: bash scripts/print-live-cli-smoke.sh claude"
    ;;
  keywords)
    cat <<'EOF'
LIVE: claude PASS
LIVE: codex PASS
LIVE: cursor PASS
LIVE: grok PASS
LIVE: all PASS
LIVE: status
LIVE: claude FAIL: <reason>
LIVE: codex FAIL: <reason>
LIVE: cursor FAIL: <reason>
LIVE: grok FAIL: <reason>
EOF
    ;;
  prep)
    sed -n '/## 0\. 共通 prep/,/^## 1\./p' "$DOC" | sed '$d'
    ;;
  check)
    sed -n '/## 5\. 一括チェック/,/^## 6\./p' "$DOC" | sed '$d'
    ;;
  claude)
    sed -n '/## 1\. Claude Code/,/^## 2\./p' "$DOC" | sed '$d'
    ;;
  codex)
    sed -n '/## 2\. Codex CLI/,/^## 3\./p' "$DOC" | sed '$d'
    ;;
  cursor)
    sed -n '/## 3\. Cursor/,/^## 4\./p' "$DOC" | sed '$d'
    ;;
  grok)
    sed -n '/## 4\. Grok/,/^## 5\./p' "$DOC" | sed '$d'
    ;;
  *)
    usage
    exit 2
    ;;
esac
