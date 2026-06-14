#!/usr/bin/env bash
# night-watch-report.sh — Night Watch patrol report (Phase 99.1)
#
# Usage:
#   night-watch-report.sh --dry-run
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

dry_run=false
for arg in "$@"; do
  case "$arg" in
    --dry-run) dry_run=true ;;
    -h|--help)
      echo "Usage: night-watch-report.sh [--dry-run]"
      exit 0
      ;;
    *)
      echo "unknown argument: $arg" >&2
      exit 1
      ;;
  esac
done

args=(--repo-root "$ROOT")
if [[ "$dry_run" == true ]]; then
  args+=(--dry-run)
fi

(cd "$ROOT/go" && go run ./cmd/night-watch-report "${args[@]}")
