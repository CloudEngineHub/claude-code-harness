#!/usr/bin/env bash
# Phase 95.2.3 — harness-mem read layer integration tests
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HARNESS="$ROOT/bin/harness"
BRIEF="$ROOT/scripts/breezing-brief.sh"
GO_DIR="$ROOT/go"

PASS=0
FAIL=0
FAIL_MESSAGES=()

pass() { PASS=$((PASS + 1)); echo "✓ $1" >&2; }
fail() { FAIL=$((FAIL + 1)); FAIL_MESSAGES+=("$1"); echo "✗ $1" >&2; }

echo "Building harness binary for shell tests..." >&2
(
  cd "$GO_DIR"
  GOOS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$GOOS" in
    darwin) GOOS=darwin ;;
    linux) GOOS=linux ;;
  esac
  GOARCH="$(uname -m)"
  case "$GOARCH" in
    x86_64) GOARCH=amd64 ;;
    aarch64|arm64) GOARCH=arm64 ;;
  esac
  OUT="$ROOT/bin/harness-${GOOS}-${GOARCH}"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -o "$OUT" ./cmd/harness/
)

# ---- (a) bin/harness mem search-similar JSON shape ----

if [[ ! -x "$HARNESS" ]]; then
  fail "(a) pre-flight: bin/harness missing"
else
  pass "(a) pre-flight: bin/harness exists"
fi

out="$(HARNESS_MEM_HOME=/nonexistent-mem-home "$HARNESS" mem search-similar --project /tmp/proj --query "rate limit" --format json 2>/dev/null || true)"
if python3 - <<'PY' "$out"
import json, sys
raw = sys.argv[1].strip()
data = json.loads(raw or "[]")
assert isinstance(data, list), "output must be JSON array"
for item in data:
    assert isinstance(item, dict), "each item must be object"
    for key in ("summary", "decision", "outcome", "decided_at", "mem_id", "score"):
        assert key in item, f"missing key {key}"
print("ok")
PY
then
  pass "(a) search-similar returns valid JSON array shape"
else
  fail "(a) search-similar JSON shape invalid: ${out:-<empty>}"
fi

exit_code=0
HARNESS_MEM_HOME=/nonexistent-mem-home "$HARNESS" mem search-similar --project /tmp/proj --query "x" --format json >/dev/null 2>&1 || exit_code=$?
if [[ "$exit_code" -eq 0 ]]; then
  pass "(a) search-similar fail-open exit 0"
else
  fail "(a) search-similar expected exit 0, got $exit_code"
fi

# ---- (b) breezing-brief recall subcommand ----

if [[ ! -x "$BRIEF" ]]; then
  fail "(b) pre-flight: breezing-brief.sh missing"
else
  pass "(b) pre-flight: breezing-brief.sh exists"
fi

recall_empty="$(bash "$BRIEF" recall "" 2>/dev/null || true)"
if [[ "$recall_empty" == "[]" ]]; then
  pass "(b) recall empty input → []"
else
  fail "(b) recall empty input expected [], got: ${recall_empty:-<empty>}"
fi

recall_out="$(bash "$BRIEF" recall "implement auth" --project /tmp/proj 2>/dev/null || true)"
if python3 - <<'PY' "$recall_out"
import json, sys
data = json.loads(sys.argv[1] or "[]")
assert isinstance(data, list)
assert len(data) <= 3
print("ok")
PY
then
  pass "(b) recall returns array with max 3 items"
else
  fail "(b) recall output invalid: ${recall_out:-<empty>}"
fi

# ---- (c) workgraph grep audit ----

audit_dir() {
  local label="$1"
  local dir="$2"
  if rg -i 'workgraph|signal_send' "$dir" --glob '*.go' >/dev/null 2>&1; then
    fail "(c) $label must not reference workgraph/signal_send"
  else
    pass "(c) $label workgraph audit clean"
  fi
}

audit_dir "breezingmem" "$ROOT/go/internal/breezingmem"
audit_dir "triaddispatcher" "$ROOT/go/internal/triaddispatcher"

echo ""
echo "PASS=$PASS FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  echo "Failures:" >&2
  for msg in "${FAIL_MESSAGES[@]}"; do
    echo "  - $msg" >&2
  done
  exit 1
fi

echo "test-mem-read-layer: ok"
