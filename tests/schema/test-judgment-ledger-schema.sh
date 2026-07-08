#!/usr/bin/env bash
# Phase 98.1.1 — judgment-ledger.v1 schema reject tests (TestJudgmentLedger_SchemaReject equivalent)
#
# Validates:
#   (a) schema draft-07 + judgment-ledger.v1 $id + additionalProperties:false
#   (b) valid record passes
#   (c) invalid records rejected (missing field, extra field, empty id)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCHEMA="$ROOT/templates/schemas/judgment-ledger.v1.json"

PASS=0
FAIL=0
FAIL_MESSAGES=()

pass() { PASS=$((PASS + 1)); echo "✓ $1" >&2; }
fail() { FAIL=$((FAIL + 1)); FAIL_MESSAGES+=("$1"); echo "✗ $1" >&2; }

validate_record() {
  local record_json="$1"
  python3 - <<'PY' "$SCHEMA" "$record_json"
import json
import sys

schema_path, record_json = sys.argv[1], sys.argv[2]

with open(schema_path, encoding="utf-8") as f:
    schema = json.load(f)
record = json.loads(record_json)

required = set(schema.get("required", []))
properties = schema.get("properties", {})
if schema.get("additionalProperties") is False:
    extra = set(record.keys()) - set(properties.keys())
    if extra:
        raise SystemExit(f"additional properties not allowed: {sorted(extra)}")

for key in required:
    if key not in record:
        raise SystemExit(f"missing required property: {key}")

for key in ("id", "project", "decided_at", "question", "answer", "card_ref"):
    val = record.get(key)
    if not isinstance(val, str) or not val:
        raise SystemExit(f"{key} must be a non-empty string")

if "rationale" in record and not isinstance(record["rationale"], str):
    raise SystemExit("rationale must be a string")

tags = record.get("tags")
if not isinstance(tags, list):
    raise SystemExit("tags must be an array")
for i, tag in enumerate(tags):
    if not isinstance(tag, str):
        raise SystemExit(f"tags[{i}] must be a string")

print("ok")
PY
}

assert_valid() {
  local label="$1"
  local json="$2"
  if validate_record "$json" >/dev/null 2>&1; then
    pass "$label"
  else
    fail "$label"
  fi
}

assert_invalid() {
  local label="$1"
  local json="$2"
  if validate_record "$json" >/dev/null 2>&1; then
    fail "$label (expected reject, got pass)"
  else
    pass "$label"
  fi
}

if [[ ! -f "$SCHEMA" ]]; then
  fail "pre-flight: schema missing: $SCHEMA"
  echo "PASS=$PASS FAIL=$FAIL"
  exit 1
fi
pass "pre-flight: judgment-ledger.v1 schema exists"

if python3 - <<'PY' "$SCHEMA"
import json
import sys

with open(sys.argv[1], encoding="utf-8") as f:
    schema = json.load(f)

assert schema.get("$schema") == "http://json-schema.org/draft-07/schema#"
assert "judgment-ledger.v1" in schema.get("$id", "")
assert schema.get("additionalProperties") is False
required = set(schema.get("required", []))
expected = {
    "id", "project", "decided_at", "question", "answer",
    "rationale", "card_ref", "tags",
}
assert required == expected, f"required mismatch: {required} vs {expected}"
print("ok")
PY
then
  pass "(a) schema is draft-07 with judgment-ledger.v1 \$id"
else
  fail "(a) schema structure check failed"
fi

VALID='{"id":"jl-001","project":"demo","decided_at":"2026-06-14T00:00:00Z","question":"Redis or Postgres?","answer":"redis","rationale":"scale","card_ref":"/tmp/card.json","tags":["judgment-card"]}'
assert_valid "(b) valid record passes" "$VALID"

assert_invalid "(c) missing id rejected" \
  '{"project":"demo","decided_at":"2026-06-14T00:00:00Z","question":"q","answer":"a","rationale":"","card_ref":"c.json","tags":[]}'

assert_invalid "(c) extra field rejected" \
  '{"id":"x","project":"demo","decided_at":"2026-06-14T00:00:00Z","question":"q","answer":"a","rationale":"","card_ref":"c.json","tags":[],"extra":true}'

assert_invalid "(c) empty id rejected" \
  '{"id":"","project":"demo","decided_at":"2026-06-14T00:00:00Z","question":"q","answer":"a","rationale":"","card_ref":"c.json","tags":[]}'

echo ""
echo "PASS=$PASS FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  echo "Failures:" >&2
  for msg in "${FAIL_MESSAGES[@]}"; do
    echo "  - $msg" >&2
  done
  exit 1
fi

echo "test-judgment-ledger-schema: ok"
