#!/usr/bin/env bash
# test-night-watch-install.sh — Night Watch opt-in install contract (Phase 99.1)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/night-watch-install.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

TARGET="${TMP}/settings.json"
echo '{}' >"$TARGET"

if ! bash "$ROOT/scripts/night-watch-install.sh" --target "$TARGET" --enable; then
  echo "install failed" >&2
  exit 1
fi

if ! grep -q '"NIGHT_WATCH_ENABLED": "true"' "$TARGET"; then
  echo "expected NIGHT_WATCH_ENABLED=true in fixture settings" >&2
  exit 1
fi

if ! bash "$ROOT/scripts/night-watch-install.sh" --target "$TARGET" --disable; then
  echo "disable failed" >&2
  exit 1
fi

if ! grep -q '"NIGHT_WATCH_ENABLED": "false"' "$TARGET"; then
  echo "expected NIGHT_WATCH_ENABLED=false after disable" >&2
  exit 1
fi

if ! grep -q 'NIGHT_WATCH_ENABLED=false' "$ROOT/templates/night-watch-cron.template"; then
  echo "cron template must contain NIGHT_WATCH_ENABLED=false literal" >&2
  exit 1
fi

if bash "$ROOT/scripts/night-watch-install.sh" --target "$HOME/.claude/settings.json" --enable 2>/dev/null; then
  echo "install must refuse real ~/.claude/settings.json" >&2
  exit 1
fi

echo "OK: night-watch install fixture contract"
