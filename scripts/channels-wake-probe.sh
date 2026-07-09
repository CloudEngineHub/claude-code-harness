#!/usr/bin/env bash
# channels-wake-probe.sh — opt-in bridge channel wake proposal (hook re-injection hint).
# Default: report health only. Wake trigger requires HARNESS_CHANNELS_WAKE_OPT_IN=1.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AUTO_APPROVE_DEFAULT=false

HARNESS_BIN="${HARNESS_BIN:-harness}"
if ! command -v "$HARNESS_BIN" >/dev/null 2>&1; then
  if [[ -x "$ROOT/bin/harness" ]]; then
    HARNESS_BIN="$ROOT/bin/harness"
  fi
fi

health_json="$("$HARNESS_BIN" channels-wake check 2>/dev/null || true)"
if [[ -z "$health_json" ]]; then
  echo '{"healthy":true,"reason":"not-configured"}'
  exit 0
fi

echo "$health_json"

healthy="$(printf '%s' "$health_json" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("healthy", True))' 2>/dev/null || echo true)"
reason="$(printf '%s' "$health_json" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("reason",""))' 2>/dev/null || echo "")"

if [[ "$healthy" == "True" || "$healthy" == "true" ]]; then
  exit 0
fi

if [[ "${HARNESS_CHANNELS_WAKE_OPT_IN:-0}" != "1" ]]; then
  echo "channels-wake: wake proposal suppressed (opt-in required; AUTO_APPROVE_DEFAULT=${AUTO_APPROVE_DEFAULT})" >&2
  exit 1
fi

echo "channels-wake: propose hook re-injection wake (reason=${reason})" >&2
exit 1
