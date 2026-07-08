#!/usr/bin/env bash
# render-judgment-card.sh — static Decision Card HTML renderer (Phase 95.2.1)
#
# Usage:
#   render-judgment-card.sh --card <card-v1.json> [--past <past-decisions.json>] --out <out.html>
#
# Notes:
#   - Output is static HTML (no <script>, no PWA/WebSocket). Regenerate to refresh.
#   - 1280x800 single-screen fit is manual attestation only (not CI-verified).
set -euo pipefail

usage() {
  cat <<'USAGE' >&2
Usage: render-judgment-card.sh --card <card-v1.json> [--past <past-decisions.json>] --out <out.html>

Arguments:
  --card <path>   judgment-card.v1 JSON (required)
  --past <path>   optional mem search-similar output (JSON array of past decisions)
  --out <path>    output HTML path (required)

Static HTML contract: no embedded <script>, no external CDN, regenerate to update.
1280x800 viewport fit: manual attestation only — not validated in CI (see test (g)).
USAGE
  exit 2
}

CARD_PATH=""
PAST_PATH=""
OUT_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --card) CARD_PATH="${2:-}"; shift 2 ;;
    --past) PAST_PATH="${2:-}"; shift 2 ;;
    --out)  OUT_PATH="${2:-}";  shift 2 ;;
    -h|--help) usage ;;
    *) echo "ERROR: unknown argument: $1" >&2; usage ;;
  esac
done

if [[ -z "$CARD_PATH" || -z "$OUT_PATH" ]]; then
  echo "ERROR: --card and --out are required" >&2
  usage
fi

if [[ ! -f "$CARD_PATH" ]]; then
  echo "ERROR: card file not found: $CARD_PATH" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE="$ROOT/templates/html/judgment-card.html.template"

if [[ ! -f "$TEMPLATE" ]]; then
  echo "ERROR: template not found: $TEMPLATE" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT_PATH")"

python3 - "$CARD_PATH" "$PAST_PATH" "$TEMPLATE" "$OUT_PATH" <<'PY'
import html
import json
import sys
from pathlib import Path

card_path, past_path, template_path, out_path = sys.argv[1:5]

with open(card_path, encoding="utf-8") as f:
    card = json.load(f)

past_items = []
if past_path:
    p = Path(past_path)
    if p.is_file():
        with open(p, encoding="utf-8") as f:
            raw = json.load(f)
        if isinstance(raw, list) and raw:
            past_items = raw[:3]
        elif raw:
            raise SystemExit(f"ERROR: --past must be a JSON array, got {type(raw).__name__}")

if not past_items:
    embedded = card.get("similar_past_decisions") or []
    if isinstance(embedded, list):
        past_items = embedded[:3]

def esc(value):
    return html.escape("" if value is None else str(value), quote=True)

def impact_bar_color(score):
    if score >= 100:
        return "#d22"
    if score >= 50:
        return "#e80"
    return "#3a3"

impact_score = card.get("impact_score", 0)
try:
    impact_score = int(impact_score)
except (TypeError, ValueError):
    impact_score = 0
impact_score = max(0, min(100, impact_score))

options = card.get("options") or []
while len(options) < 3:
    options.append({})

rec_id = card.get("recommendation", "")
rec_reason = ""
for opt in card.get("options") or []:
    if opt.get("id") == rec_id:
        rec_reason = opt.get("consequence") or opt.get("label") or ""
        break
if not rec_reason:
    rec_reason = rec_id

replacements = {
    "QUESTION": esc(card.get("question", "")),
    "CONFIDENCE": esc(card.get("confidence", "medium")),
    "IMPACT_SCORE": esc(impact_score),
    "IMPACT_BAR_COLOR": impact_bar_color(impact_score),
    "DIFF_SUMMARY": esc(card.get("diff_summary", "")),
    "IMPACT": esc(card.get("impact", "")),
    "RECOMMENDATION_ID": esc(rec_id),
    "RECOMMENDATION_REASON": esc(rec_reason),
}

for idx in range(1, 4):
    opt = options[idx - 1] if idx - 1 < len(card.get("options") or []) else {}
    opt_id = opt.get("id", "")
    replacements[f"OPTION_{idx}_ID"] = esc(opt_id)
    replacements[f"OPTION_{idx}_LABEL"] = esc(opt.get("label", ""))
    replacements[f"OPTION_{idx}_CONSEQUENCE"] = esc(opt.get("consequence", ""))
    replacements[f"OPTION_{idx}_VISIBILITY"] = "" if opt_id else "hidden"

for idx in range(1, 4):
    past = past_items[idx - 1] if idx - 1 < len(past_items) else {}
    has_past = bool(past)
    replacements[f"PAST_{idx}_SUMMARY"] = esc(past.get("summary", ""))
    replacements[f"PAST_{idx}_DECISION"] = esc(past.get("decision", ""))
    replacements[f"PAST_{idx}_OUTCOME"] = esc(past.get("outcome", ""))
    replacements[f"PAST_{idx}_DECIDED_AT"] = esc(past.get("decided_at", ""))
    replacements[f"PAST_{idx}_MEM_ID"] = esc(past.get("mem_id", ""))
    replacements[f"PAST_{idx}_VISIBILITY"] = "" if has_past else "hidden"
    replacements[f"PAST_{idx}_EMPTY"] = "false" if has_past else "true"

with open(template_path, encoding="utf-8") as f:
    rendered = f.read()

for key, value in replacements.items():
    rendered = rendered.replace(f"{{{{{key}}}}}", value)

if "{{" in rendered or "}}" in rendered:
    raise SystemExit("ERROR: unreplaced template placeholders remain")

if "<script" in rendered.lower():
    raise SystemExit("ERROR: rendered HTML must not contain <script>")

out = Path(out_path)
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(rendered, encoding="utf-8")
PY

echo "render-judgment-card: wrote $OUT_PATH"
