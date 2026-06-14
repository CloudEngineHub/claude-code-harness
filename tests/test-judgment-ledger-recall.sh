#!/usr/bin/env bash
# Phase 98.1.6 — recall layer fills judgment-card.v1 similar_past_decisions (max 3)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CARD_SCRIPT="$ROOT/scripts/judgment-card.sh"
LEDGER_SCRIPT="$ROOT/scripts/judgment-ledger.sh"
FIXTURE="$ROOT/tests/fixtures/judgment/card-tradeoff.json"

PASS=0
FAIL=0
FAIL_MESSAGES=()

pass() { PASS=$((PASS + 1)); echo "✓ $1" >&2; }
fail() { FAIL=$((FAIL + 1)); FAIL_MESSAGES+=("$1"); echo "✗ $1" >&2; }

TMP="$(mktemp -d "${TMPDIR:-/tmp}/judgment-recall-test.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

LEDGER="$TMP/judgment-ledger.jsonl"
export HARNESS_JUDGMENT_LEDGER="$LEDGER"

grep_hits="$(grep -rl 'similar_past_decisions' "$ROOT/scripts/judgment-card.sh" "$ROOT/scripts/judgment-ledger.sh" "$ROOT/go/internal/judgmentledger/recall.go" 2>/dev/null | wc -l | tr -d ' ')"
if [[ "$grep_hits" -ge 2 ]]; then
  pass "similar_past_decisions referenced in 2+ implementation files ($grep_hits)"
else
  fail "similar_past_decisions grep hits = $grep_hits, want >= 2"
fi

bash "$LEDGER_SCRIPT" append \
  --project "recall-demo" \
  --question "レート制限を Redis キャッシュに移すか、DB 最適化か？" \
  --answer "redis" \
  --rationale "latency wins" \
  --card-ref "$FIXTURE" \
  --tags "judgment-card" \
  --id "past-001" >/dev/null

bash "$LEDGER_SCRIPT" append \
  --project "recall-demo" \
  --question "Redis キャッシュのクラスタ構成は？" \
  --answer "cluster" \
  --rationale "ops familiarity" \
  --card-ref "$FIXTURE" \
  --tags "judgment-card" \
  --id "past-002" >/dev/null

recalled="$(bash "$CARD_SCRIPT" recall "$FIXTURE" --project recall-demo)"

if python3 - <<'PY' "$recalled"
import json, sys
card = json.loads(sys.argv[1])
past = card.get("similar_past_decisions")
assert isinstance(past, list), "similar_past_decisions must be array"
assert 1 <= len(past) <= 3, f"expected 1-3 items, got {len(past)}"
for item in past:
    for key in ("summary", "decision", "outcome", "decided_at", "mem_id"):
        assert key in item, f"missing {key}"
    assert item["mem_id"].startswith("judgment-ledger:")
print("ok")
PY
then
  pass "recall subcommand fills similar_past_decisions on card"
else
  fail "recall subcommand output invalid"
fi

echo ""
echo "PASS=$PASS FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  for msg in "${FAIL_MESSAGES[@]}"; do
    echo "  - $msg" >&2
  done
  exit 1
fi

echo "test-judgment-ledger-recall: ok"
