#!/usr/bin/env bash
# Phase 95.2.1 — Decision Card HTML render contract tests
#
# Validates:
#   (a) templates/html/judgment-card.html.template exists
#   (b) card-v1-extended.json renders to parseable HTML (html.parser)
#   (c) impact_score=100 (card-v1-floor.json) → #d22 in output
#   (d) past 3-item fixture → 3 visible <section class="past"> blocks
#   (e) past 2-item fixture → 1 hidden past section
#   (f) output contains zero <script> tags (static contract)
#   (g) 1280x800 fit — manual attestation only (documented in script usage)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RENDER="$ROOT/scripts/render-judgment-card.sh"
TEMPLATE="$ROOT/templates/html/judgment-card.html.template"
FIXTURE_DIR="$ROOT/tests/fixtures/judgment"

PASS=0
FAIL=0
FAIL_MESSAGES=()

pass() { PASS=$((PASS + 1)); echo "✓ $1" >&2; }
fail() { FAIL=$((FAIL + 1)); FAIL_MESSAGES+=("$1"); echo "✗ $1" >&2; }

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/test-judgment-card-render.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

assert_html_valid() {
  local label="$1"
  local html_path="$2"
  if python3 - <<'PY' "$html_path"
import sys
from html.parser import HTMLParser

path = sys.argv[1]
with open(path, encoding="utf-8") as f:
    content = f.read()

class Collector(HTMLParser):
    pass

parser = Collector()
parser.feed(content)
parser.close()

required = ("<!DOCTYPE html>", "<html", "</html>", "<body", "</body>")
for token in required:
    if token not in content:
        raise SystemExit(f"missing {token!r}")

if "{{" in content or "}}" in content:
    raise SystemExit("unreplaced placeholder markers remain")
PY
  then
    pass "$label"
  else
    fail "$label"
  fi
}

# ---- (a) template exists ----

if [[ -f "$TEMPLATE" ]]; then
  pass "(a) judgment-card.html.template exists"
else
  fail "(a) judgment-card.html.template missing: $TEMPLATE"
fi

# ---- pre-flight: render script ----

if [[ ! -x "$RENDER" ]]; then
  fail "pre-flight: render-judgment-card.sh missing or not executable"
  echo ""
  echo "PASS=$PASS FAIL=$FAIL"
  exit 1
fi
pass "pre-flight: render-judgment-card.sh exists and is executable"

# ---- (b) extended card → valid HTML ----

CARD_EXTENDED="$FIXTURE_DIR/card-v1-extended.json"
OUT_B="$TMP_DIR/extended.html"

if [[ ! -f "$CARD_EXTENDED" ]]; then
  fail "(b) missing fixture: card-v1-extended.json"
else
  set +e
  bash "$RENDER" --card "$CARD_EXTENDED" --out "$OUT_B" >/dev/null 2>&1
  render_exit=$?
  set -e
  if [[ "$render_exit" -ne 0 ]]; then
    fail "(b) render exit 0 for card-v1-extended.json (got $render_exit)"
  elif [[ ! -f "$OUT_B" ]]; then
    fail "(b) render did not produce output HTML"
  else
    pass "(b) render exit 0 for card-v1-extended.json"
    assert_html_valid "(b) output HTML is html.parser-valid" "$OUT_B"
  fi
fi

# ---- (c) impact_score=100 → #d22 ----

CARD_FLOOR="$FIXTURE_DIR/card-v1-floor.json"
OUT_C="$TMP_DIR/floor.html"

if [[ ! -f "$CARD_FLOOR" ]]; then
  fail "(c) missing fixture: card-v1-floor.json"
else
  bash "$RENDER" --card "$CARD_FLOOR" --out "$OUT_C" >/dev/null 2>&1 || true
  if [[ -f "$OUT_C" ]] && grep -q '#d22' "$OUT_C"; then
    pass "(c) impact_score=100 renders #d22 impact bar color"
  else
    fail "(c) impact_score=100 output missing #d22"
  fi
fi

# ---- (d) past 3 items → 3 visible past sections ----

PAST3="$FIXTURE_DIR/past-decisions-3.json"
OUT_D="$TMP_DIR/past3.html"

if [[ ! -f "$PAST3" ]]; then
  fail "(d) missing fixture: past-decisions-3.json"
else
  bash "$RENDER" --card "$CARD_EXTENDED" --past "$PAST3" --out "$OUT_D" >/dev/null 2>&1 || true
  if [[ ! -f "$OUT_D" ]]; then
    fail "(d) render with past 3 did not produce output"
  else
    visible_count="$(grep -c '<section class="past"' "$OUT_D" || true)"
    hidden_count="$(grep -c '<section class="past"[^>]*hidden' "$OUT_D" || true)"
    if [[ "$visible_count" -eq 3 && "$hidden_count" -eq 0 ]]; then
      pass "(d) past 3 fixture shows 3 visible past sections"
    else
      fail "(d) expected 3 visible past sections, got visible=$visible_count hidden=$hidden_count"
    fi
  fi
fi

# ---- (e) past 2 items → 1 hidden section ----

PAST2="$FIXTURE_DIR/past-decisions-2.json"
OUT_E="$TMP_DIR/past2.html"

if [[ ! -f "$PAST2" ]]; then
  fail "(e) missing fixture: past-decisions-2.json"
else
  bash "$RENDER" --card "$CARD_EXTENDED" --past "$PAST2" --out "$OUT_E" >/dev/null 2>&1 || true
  if [[ ! -f "$OUT_E" ]]; then
    fail "(e) render with past 2 did not produce output"
  else
    hidden_count="$(grep -c '<section class="past"[^>]*hidden' "$OUT_E" || true)"
    visible_count="$(grep '<section class="past"' "$OUT_E" | grep -vc hidden || true)"
    if [[ "$hidden_count" -eq 1 && "$visible_count" -eq 2 ]]; then
      pass "(e) past 2 fixture hides exactly 1 past section"
    else
      fail "(e) expected 2 visible + 1 hidden past sections, got visible=$visible_count hidden=$hidden_count"
    fi
  fi
fi

# ---- (f) zero <script> tags ----

for label_path in "$OUT_B" "$OUT_C" "$OUT_D" "$OUT_E"; do
  if [[ -f "$label_path" ]]; then
    script_count="$(grep -ci '<script' "$label_path" || true)"
    if [[ "$script_count" -eq 0 ]]; then
      pass "(f) no <script> in $(basename "$label_path")"
    else
      fail "(f) $(basename "$label_path") contains $script_count <script> tag(s)"
    fi
  fi
done

# ---- (g) manual attestation only ----

if grep -q '1280x800' "$RENDER" && grep -q 'manual attestation' "$RENDER"; then
  pass "(g) render script documents 1280x800 manual attestation (CI-exempt)"
else
  fail "(g) render script usage missing 1280x800 manual attestation note"
fi

# ---- summary ----

echo ""
echo "PASS=$PASS FAIL=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  echo "Failures:" >&2
  for msg in "${FAIL_MESSAGES[@]}"; do
    echo "  - $msg" >&2
  done
  exit 1
fi

echo "test-judgment-card-render: ok"
