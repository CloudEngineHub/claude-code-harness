#!/usr/bin/env bash
# Phase 98.1.4 — judgment-card record-answer → judgment-ledger append wiring
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CARD_SCRIPT="$ROOT/scripts/judgment-card.sh"
FIXTURE="$ROOT/tests/fixtures/judgment/card-tradeoff.json"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "✓ $1" >&2; }
fail() { FAIL=$((FAIL + 1)); echo "✗ $1" >&2; }

TMP="$(mktemp -d "${TMPDIR:-/tmp}/judgment-ledger-wiring.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

LEDGER="$TMP/judgment-ledger.jsonl"
EMPTY_HOME="$(mktemp -d)"

if grep -q 'judgment-ledger.sh append' "$CARD_SCRIPT"; then
  pass "judgment-card.sh references judgment-ledger.sh append"
else
  fail "judgment-card.sh missing judgment-ledger.sh append wiring"
fi

if [[ ! -f "$FIXTURE" ]]; then
  fail "missing fixture card-tradeoff.json"
  echo "PASS=$PASS FAIL=$FAIL"
  exit 1
fi

HOME="$EMPTY_HOME" \
  HARNESS_JUDGMENT_LEDGER="$LEDGER" \
  bash "$CARD_SCRIPT" record-answer "$FIXTURE" \
    --answer redis \
    --why "scale first" \
    --project wiring-demo \
    --session sess-wiring >/dev/null 2>&1 || true

if [[ -f "$LEDGER" ]] && grep -q 'wiring-demo' "$LEDGER"; then
  pass "record-answer appended ledger record for project"
else
  fail "record-answer did not append ledger record"
fi

if grep -q '"answer": "redis"' "$LEDGER" 2>/dev/null || grep -q '"answer":"redis"' "$LEDGER" 2>/dev/null; then
  pass "ledger record contains answer"
else
  fail "ledger record missing answer"
fi

echo ""
echo "PASS=$PASS FAIL=$FAIL"
[[ "$FAIL" -eq 0 ]]
