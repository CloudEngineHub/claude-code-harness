#!/usr/bin/env bash
# Phase 95.2.2 — judgment-card.v1 schema extension tests
#
# Validates:
#   (a) v1 schema is valid JSON draft-07 with optional impact_score + similar_past_decisions
#   (b) v0 fixtures validate against v1 schema (backward compatibility)
#   (c) v1 extended fixture validates
#   (d) invalid v1 extension fields fail validate
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT/scripts/judgment-card.sh"
SCHEMA="$ROOT/templates/schemas/judgment-card.v1.json"
FIXTURE_DIR="$ROOT/tests/fixtures/judgment"

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

if [[ ! -f "$SCHEMA" ]]; then
  fail "pre-flight: schema missing: $SCHEMA"
  echo "PASS=$PASS FAIL=$FAIL"
  exit 1
fi
pass "pre-flight: judgment-card.v1 schema exists"

# ---- (a) schema draft-07 + v1 optional fields ----

if python3 - <<'PY' "$SCHEMA"
import json
import sys

path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    schema = json.load(f)

assert isinstance(schema, dict), "schema root must be object"
assert schema.get("$schema") == "http://json-schema.org/draft-07/schema#", "$schema must be draft-07"
assert schema.get("type") == "object", "type must be object"

required = set(schema.get("required", []))
expected_required = {
    "question",
    "options",
    "recommendation",
    "confidence",
    "impact",
    "diff_summary",
}
assert required == expected_required, f"required mismatch: {required} vs {expected_required}"
assert schema.get("additionalProperties") is False, "additionalProperties must be false"

props = schema.get("properties", {})
assert "impact_score" in props, "impact_score property missing"
impact = props["impact_score"]
assert impact.get("type") == "integer", "impact_score type must be integer"
assert impact.get("minimum") == 0, "impact_score minimum must be 0"
assert impact.get("maximum") == 100, "impact_score maximum must be 100"

assert "similar_past_decisions" in props, "similar_past_decisions property missing"
past = props["similar_past_decisions"]
assert past.get("type") == "array", "similar_past_decisions type must be array"
assert past.get("maxItems") == 3, "similar_past_decisions maxItems must be 3"
item = past.get("items", {})
assert item.get("additionalProperties") is False, "similar_past_decisions items additionalProperties must be false"
item_required = set(item.get("required", []))
assert item_required == {
    "summary",
    "decision",
    "outcome",
    "decided_at",
    "mem_id",
}, f"similar_past_decisions item required mismatch: {item_required}"

print("ok")
PY
then
  pass "(a) v1 schema is JSON draft-07 with optional extension fields"
else
  fail "(a) v1 schema structure check failed"
fi

# ---- (b) v0 backward compatibility ----

V0_CARDS=(
  "card-tradeoff.json"
  "card-dod-ambiguous.json"
)

for card_name in "${V0_CARDS[@]}"; do
  card_path="$FIXTURE_DIR/$card_name"
  if [[ ! -f "$card_path" ]]; then
    fail "(b) missing v0 fixture: $card_name"
    continue
  fi
  assert_exit "(b) v0 fixture validates as v1: $card_name" 0 bash "$SCRIPT" validate "$card_path" >/dev/null
done

# ---- (c) v1 extended fixture ----

EXTENDED="$FIXTURE_DIR/card-v1-extended.json"
if [[ ! -f "$EXTENDED" ]]; then
  fail "(c) missing card-v1-extended.json"
else
  assert_exit "(c) v1 extended fixture validates" 0 bash "$SCRIPT" validate "$EXTENDED" >/dev/null
fi

# ---- (d) invalid v1 extension fields ----

INVALID_CARDS=(
  "card-v1-invalid-impact-score.json"
  "card-v1-invalid-past-decisions-count.json"
)

for card_name in "${INVALID_CARDS[@]}"; do
  card_path="$FIXTURE_DIR/$card_name"
  if [[ ! -f "$card_path" ]]; then
    fail "(d) missing invalid fixture: $card_name"
    continue
  fi
  assert_exit "(d) invalid v1 fixture fails: $card_name" 1 bash "$SCRIPT" validate "$card_path" >/dev/null
done

TMP_INVALID="$(mktemp)"
trap 'rm -f "$TMP_INVALID"' EXIT
python3 - <<'PY' "$TMP_INVALID"
import json
import sys

path = sys.argv[1]
card = {
    "question": "invalid impact_score 101",
    "options": [
        {"id": "a", "label": "A", "consequence": "c"},
        {"id": "b", "label": "B", "consequence": "c"},
    ],
    "recommendation": "a",
    "confidence": "low",
    "impact": "test",
    "diff_summary": "test",
    "impact_score": 101,
}
with open(path, "w", encoding="utf-8") as f:
    json.dump(card, f)
PY

assert_exit "(d) impact_score=101 fails validate" 1 bash "$SCRIPT" validate "$TMP_INVALID" >/dev/null

echo ""
echo "PASS=$PASS FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  echo "Failures:" >&2
  for msg in "${FAIL_MESSAGES[@]}"; do
    echo "  - $msg" >&2
  done
  exit 1
fi

echo "test-judgment-card-v1-schema: ok"
