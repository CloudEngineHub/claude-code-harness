#!/usr/bin/env bash
# failure-codifier-propose.sh — Failure Codifier dry-run proposals (Phase 100)
#
# Usage:
#   failure-codifier-propose.sh --dry-run
#
# Prints failure-rule.v1 JSON candidates to stdout. Never writes patterns.md or
# decisions.md — human approval is required for SSOT promotion.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

dry_run=false
for arg in "$@"; do
  case "$arg" in
    --dry-run) dry_run=true ;;
    -h|--help)
      echo "Usage: failure-codifier-propose.sh --dry-run"
      exit 0
      ;;
    *)
      echo "unknown argument: $arg" >&2
      exit 1
      ;;
  esac
done

if [[ "$dry_run" != true ]]; then
  echo "failure-codifier-propose.sh: --dry-run is required (auto-promotion forbidden)" >&2
  exit 2
fi

(cd "$ROOT/go" && go run ./cmd/failure-codifier-propose --dry-run --repo-root "$ROOT")
