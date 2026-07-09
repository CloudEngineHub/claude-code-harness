#!/usr/bin/env bash
# Phase 104.4 — judgment-card compute-impact delegates to Go impact-score.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

GO_BIN="$TMP_DIR/harness"
GOCACHE="$TMP_DIR/gocache" go -C "$ROOT/go" build -o "$GO_BIN" ./cmd/harness

shell_out="$(HARNESS_BIN="$GO_BIN" "$ROOT/scripts/judgment-card.sh" compute-impact --files-changed 5 --lines-changed 200)"
go_out="$("$GO_BIN" impact-score --files-changed 5 --lines-changed 200)"
if [[ "$shell_out" != "$go_out" ]]; then
  echo "compute-impact wrapper output mismatch" >&2
  echo "shell: $shell_out" >&2
  echo "go:    $go_out" >&2
  exit 1
fi

set +e
floor_out="$(HARNESS_BIN="$GO_BIN" "$ROOT/scripts/judgment-card.sh" compute-impact --files-changed 5 --lines-changed 200 --floor-category egress)"
floor_exit=$?
go_floor_out="$("$GO_BIN" impact-score --files-changed 5 --lines-changed 200 --floor-category egress)"
go_floor_exit=$?
set -e

if [[ "$floor_exit" -ne 2 || "$go_floor_exit" -ne 2 ]]; then
  echo "floor exit mismatch: shell=$floor_exit go=$go_floor_exit" >&2
  exit 1
fi
if [[ "$floor_out" != "$go_floor_out" ]]; then
  echo "floor output mismatch" >&2
  echo "shell: $floor_out" >&2
  echo "go:    $go_floor_out" >&2
  exit 1
fi

echo "PASS judgment-card compute-impact uses Go impact-score"
