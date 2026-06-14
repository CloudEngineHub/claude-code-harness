#!/usr/bin/env bash
# test-night-watch-report.sh — Night Watch report dry-run contract (Phase 99.1)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

output="$(bash "$ROOT/scripts/night-watch-report.sh" --dry-run)"
if [[ $? -ne 0 ]]; then
  echo "night-watch-report --dry-run failed" >&2
  exit 1
fi

if ! printf '%s' "$output" | jq -e '.schema_version == "night-watch-report.v1"' >/dev/null; then
  echo "invalid JSON or schema_version from night-watch-report --dry-run" >&2
  printf '%s\n' "$output" >&2
  exit 1
fi

if ! printf '%s' "$output" | jq -e '.dry_run == true' >/dev/null; then
  echo "expected dry_run=true in report" >&2
  exit 1
fi

echo "OK: night-watch-report dry-run jq contract"
