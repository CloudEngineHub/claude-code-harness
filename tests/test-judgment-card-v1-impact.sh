#!/usr/bin/env bash
# Phase 95.2.2 — compute-impact subcommand tests
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT/scripts/judgment-card.sh"

PASS=0
FAIL=0
FAIL_MESSAGES=()

pass() { PASS=$((PASS + 1)); echo "✓ $1" >&2; }
fail() { FAIL=$((FAIL + 1)); FAIL_MESSAGES+=("$1"); echo "✗ $1" >&2; }

assert_exit() {
  local label="$1"
  local expected_exit="$2"
  shift 2
  set +e
  local output
  output="$("$@" 2>&1)"
  local exit_code=$?
  set -e
  if [[ "$exit_code" -eq "$expected_exit" ]]; then
    pass "$label"
  else
    fail "$label (expected exit ${expected_exit}, got ${exit_code}; output: ${output})"
    printf '%s' "$output"
    return
  fi
  printf '%s' "$output"
}

if [[ ! -x "$SCRIPT" ]]; then
  fail "pre-flight: judgment-card.sh missing or not executable"
  echo "PASS=$PASS FAIL=$FAIL"
  exit 1
fi
pass "pre-flight: judgment-card.sh exists"

# ---- JSON shape + floor non-hit ----

out_no_floor="$(assert_exit "compute-impact no floor exit 0" 0 \
  bash "$SCRIPT" compute-impact --files-changed 5 --lines-changed 200)"

python3 - <<'PY' "$out_no_floor"
import json
import sys

raw = sys.argv[1].strip()
data = json.loads(raw)
assert set(data.keys()) == {"impact_score", "hard_stop"}, f"keys mismatch: {data.keys()}"
assert isinstance(data["impact_score"], int), "impact_score must be int"
assert isinstance(data["hard_stop"], bool), "hard_stop must be bool"
assert data["impact_score"] == 45, f"impact_score={data['impact_score']}, want 45"
assert data["hard_stop"] is False, "hard_stop must be false without floor"
print("ok")
PY
if [[ $? -eq 0 ]]; then
  pass "compute-impact JSON shape + score 45 for files=5 lines=200"
else
  fail "compute-impact JSON shape / score check failed"
fi

if python3 - <<'PY' "$out_no_floor"
import json, sys
data = json.loads(sys.argv[1].strip())
assert data["impact_score"] < 100
PY
then
  pass "compute-impact no floor: impact_score < 100"
else
  fail "compute-impact no floor: impact_score should be < 100"
fi

# ---- floor hit ----

out_floor="$(assert_exit "compute-impact floor exit 2" 2 \
  bash "$SCRIPT" compute-impact --files-changed 5 --lines-changed 200 --floor-category egress)"

python3 - <<'PY' "$out_floor"
import json
import sys

raw = sys.argv[1].strip()
data = json.loads(raw)
assert data["impact_score"] == 100, f"impact_score={data['impact_score']}, want 100"
assert data["hard_stop"] is True, "hard_stop must be true with floor"
print("ok")
PY
if [[ $? -eq 0 ]]; then
  pass "compute-impact floor: impact_score=100 hard_stop=true"
else
  fail "compute-impact floor output check failed"
fi

echo ""
echo "PASS=$PASS FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  echo "Failures:" >&2
  for msg in "${FAIL_MESSAGES[@]}"; do
    echo "  - $msg" >&2
  done
  exit 1
fi

echo "test-judgment-card-v1-impact: ok"
