#!/usr/bin/env bash
# judgment-card.sh — Decision Card v0 helpers (Phase 93.3.3)
#
# Subcommands:
#   should-issue --reason <reason> [--floor-category <cat>]
#   validate <card.json>
#   record-answer <card.json> --answer <option-id> --why "<text>" --project <p> --session <s>
#   recall <card.json> --project <p>   (fill similar_past_decisions from ledger, max 3)
#   compute-impact --files-changed N --lines-changed M [--floor-category CAT]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA="${ROOT}/templates/schemas/judgment-card.v1.json"
LEDGER_SCRIPT="${ROOT}/scripts/judgment-ledger.sh"

ISSUE_REASONS=(
  dod-ambiguous
  scope-exceeded
  tradeoff
)

usage() {
  cat <<'EOF'
Usage:
  judgment-card.sh should-issue --reason <reason> [--floor-category <cat>]
  judgment-card.sh validate <card.json>
  judgment-card.sh record-answer <card.json> --answer <option-id> --why "<text>" --project <p> --session <s>
  judgment-card.sh recall <card.json> --project <p>
  judgment-card.sh compute-impact --files-changed N --lines-changed M [--floor-category CAT]
EOF
}

cmd_should_issue() {
  local reason=""
  local floor_category=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --reason)
        reason="${2:-}"
        shift 2
        ;;
      --floor-category)
        floor_category="${2:-}"
        shift 2
        ;;
      *)
        echo "should-issue: unknown argument: $1" >&2
        exit 1
        ;;
    esac
  done

  if [[ -n "$floor_category" ]]; then
    echo "HARD_STOP: floor (${floor_category})"
    exit 2
  fi

  for allowed in "${ISSUE_REASONS[@]}"; do
    if [[ "$reason" == "$allowed" ]]; then
      echo "ISSUE_CARD"
      exit 0
    fi
  done

  echo "NO_CARD: reason not in enum"
  exit 1
}

cmd_validate() {
  local card_path="${1:-}"
  if [[ -z "$card_path" || ! -f "$card_path" ]]; then
    echo "validate: file not found: ${card_path:-<missing>}" >&2
    exit 1
  fi
  if [[ ! -f "$SCHEMA" ]]; then
    echo "validate: schema not found: $SCHEMA" >&2
    exit 1
  fi

  python3 - "$card_path" "$SCHEMA" <<'PY'
import json
import sys

card_path, schema_path = sys.argv[1], sys.argv[2]

try:
    with open(card_path, encoding="utf-8") as f:
        data = json.load(f)
except json.JSONDecodeError as exc:
    print(f"invalid JSON: {exc}", file=sys.stderr)
    raise SystemExit(1)

try:
    with open(schema_path, encoding="utf-8") as f:
        schema = json.load(f)
except json.JSONDecodeError as exc:
    print(f"invalid schema JSON: {exc}", file=sys.stderr)
    raise SystemExit(1)


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


def validate_card(data, schema):
    if not isinstance(data, dict):
        fail("root must be object")

    required = schema.get("required", [])
    properties = schema.get("properties", {})
    if schema.get("additionalProperties") is False:
        extra = set(data.keys()) - set(properties.keys())
        if extra:
            fail(f"additional properties not allowed: {sorted(extra)}")

    for key in required:
        if key not in data:
            fail(f"missing required property: {key}")

    if not isinstance(data["question"], str) or not data["question"]:
        fail("question must be a non-empty string")

    options_spec = properties.get("options", {})
    options = data["options"]
    if not isinstance(options, list):
        fail("options must be an array")
    min_items = options_spec.get("minItems")
    max_items = options_spec.get("maxItems")
    if min_items is not None and len(options) < min_items:
        fail(f"options must have at least {min_items} items")
    if max_items is not None and len(options) > max_items:
        fail(f"options must have at most {max_items} items")

    item_spec = options_spec.get("items", {})
    item_required = item_spec.get("required", [])
    item_props = item_spec.get("properties", {})
    for index, item in enumerate(options):
        if not isinstance(item, dict):
            fail(f"options[{index}] must be an object")
        if item_spec.get("additionalProperties") is False:
            extra = set(item.keys()) - set(item_props.keys())
            if extra:
                fail(f"options[{index}] additional properties not allowed: {sorted(extra)}")
        for key in item_required:
            if key not in item:
                fail(f"options[{index}] missing required property: {key}")
        for key in item_required:
            if not isinstance(item[key], str) or not item[key]:
                fail(f"options[{index}].{key} must be a non-empty string")

    for key in ("recommendation", "impact", "diff_summary"):
        if not isinstance(data[key], str) or not data[key]:
            fail(f"{key} must be a non-empty string")

    confidence_spec = properties.get("confidence", {})
    allowed = confidence_spec.get("enum", [])
    if data["confidence"] not in allowed:
        fail(f"confidence must be one of: {', '.join(allowed)}")

    if "impact_score" in data:
        impact_spec = properties.get("impact_score", {})
        val = data["impact_score"]
        if not isinstance(val, int) or isinstance(val, bool):
            fail("impact_score must be an integer")
        minimum = impact_spec.get("minimum", 0)
        maximum = impact_spec.get("maximum", 100)
        if val < minimum or val > maximum:
            fail(f"impact_score must be between {minimum} and {maximum}")

    if "similar_past_decisions" in data:
        past_spec = properties.get("similar_past_decisions", {})
        arr = data["similar_past_decisions"]
        if not isinstance(arr, list):
            fail("similar_past_decisions must be an array")
        max_items = past_spec.get("maxItems")
        if max_items is not None and len(arr) > max_items:
            fail(f"similar_past_decisions must have at most {max_items} items")

        item_spec = past_spec.get("items", {})
        item_required = item_spec.get("required", [])
        item_props = item_spec.get("properties", {})
        for index, item in enumerate(arr):
            if not isinstance(item, dict):
                fail(f"similar_past_decisions[{index}] must be an object")
            if item_spec.get("additionalProperties") is False:
                extra = set(item.keys()) - set(item_props.keys())
                if extra:
                    fail(
                        f"similar_past_decisions[{index}] additional properties not allowed: {sorted(extra)}"
                    )
            for key in item_required:
                if key not in item:
                    fail(f"similar_past_decisions[{index}] missing required property: {key}")
            for key in item_required:
                if not isinstance(item[key], str):
                    fail(f"similar_past_decisions[{index}].{key} must be a string")


validate_card(data, schema)
PY
}

cmd_compute_impact() {
  local files_changed=0
  local lines_changed=0
  local floor_category=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --files-changed)
        files_changed="${2:-0}"
        shift 2
        ;;
      --lines-changed)
        lines_changed="${2:-0}"
        shift 2
        ;;
      --floor-category)
        floor_category="${2:-}"
        shift 2
        ;;
      *)
        echo "compute-impact: unknown argument: $1" >&2
        exit 1
        ;;
    esac
  done

  python3 - "$files_changed" "$lines_changed" "$floor_category" <<'PY'
import json
import sys

files = int(sys.argv[1])
lines = int(sys.argv[2])
floor = sys.argv[3]

if floor:
    score = 100
    hard_stop = True
else:
    files = max(0, files)
    lines = max(0, lines)
    score = min(99, files * 5 + lines // 10)
    hard_stop = False

print(json.dumps({"impact_score": score, "hard_stop": hard_stop}))
if hard_stop:
    raise SystemExit(2)
PY
}

cmd_record_answer() {
  local card_path="${1:-}"
  shift || true

  local answer=""
  local why=""
  local project=""
  local session_id=""

  if [[ -z "$card_path" || ! -f "$card_path" ]]; then
    echo "record-answer: file not found: ${card_path:-<missing>}" >&2
    exit 1
  fi

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --answer)
        answer="${2:-}"
        shift 2
        ;;
      --why)
        why="${2:-}"
        shift 2
        ;;
      --project)
        project="${2:-}"
        shift 2
        ;;
      --session)
        session_id="${2:-}"
        shift 2
        ;;
      *)
        echo "record-answer: unknown argument: $1" >&2
        exit 1
        ;;
    esac
  done

  if [[ -z "$answer" || -z "$project" || -z "$session_id" ]]; then
    echo "record-answer: --answer, --project, and --session are required" >&2
    exit 1
  fi

  if ! cmd_validate "$card_path" >/dev/null 2>&1; then
    echo "record-answer: invalid card" >&2
    exit 1
  fi

  python3 - "$card_path" "$answer" "$why" "$project" "$session_id" <<'PY'
import json
import os
import sys
import urllib.error
import urllib.request

card_path, answer, why, project, session_id = sys.argv[1:6]

with open(card_path, encoding="utf-8") as f:
    card = json.load(f)

home = os.environ.get("HOME", "")
harness_mem_home = os.environ.get("HARNESS_MEM_HOME", "")
if not harness_mem_home:
    harness_mem_home = os.path.join(home, ".harness-mem")
claude_mem = os.path.join(home, ".claude-mem")

configured = os.path.isdir(harness_mem_home) or os.path.isdir(claude_mem)
if not configured:
    raise SystemExit(0)

host = os.environ.get("HARNESS_MEM_HOST", "127.0.0.1")
port = os.environ.get("HARNESS_MEM_PORT", "37888")
platform = os.environ.get("HARNESS_MEM_PLATFORM", "harness")

title = f"judgment: {card['question'][:60]}"
payload = {
    "session_id": session_id,
    "title": title,
    "content": json.dumps(
        {
            "card": card,
            "answer": answer,
            "why": why,
        },
        ensure_ascii=False,
    ),
    "platform": platform,
    "project": project,
    "tags": ["judgment-card"],
}

body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
url = f"http://{host}:{port}/v1/checkpoints/record"
req = urllib.request.Request(
    url,
    data=body,
    headers={"Content-Type": "application/json"},
    method="POST",
)
token = os.environ.get("HARNESS_MEM_ADMIN_TOKEN")
if token:
    req.add_header("Authorization", f"Bearer {token}")

try:
    with urllib.request.urlopen(req, timeout=1) as resp:
        if resp.status < 200 or resp.status >= 300:
            print("judgment-card: record skipped (unreachable)", file=sys.stderr)
except (urllib.error.URLError, TimeoutError, OSError):
    print("judgment-card: record skipped (unreachable)", file=sys.stderr)

raise SystemExit(0)
PY

  if [[ -x "$LEDGER_SCRIPT" ]]; then
    card_question="$(python3 - <<'PY' "$card_path"
import json, sys
with open(sys.argv[1], encoding="utf-8") as f:
    print(json.load(f)["question"])
PY
)"
    # judgment-ledger.sh append (fail-open)
    HARNESS_JUDGMENT_LEDGER="${HARNESS_JUDGMENT_LEDGER:-${ROOT}/.claude/state/judgment-ledger.jsonl}" \
      bash "${ROOT}/scripts/judgment-ledger.sh" append \
        --project "$project" \
        --question "$card_question" \
        --answer "$answer" \
        --rationale "$why" \
        --card-ref "$card_path" \
        --tags "judgment-card" \
      || true
  fi
}

cmd_recall() {
  local card_path="${1:-}"
  shift || true

  local project=""

  if [[ -z "$card_path" || ! -f "$card_path" ]]; then
    echo "recall: file not found: ${card_path:-<missing>}" >&2
    exit 1
  fi

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --project)
        project="${2:-}"
        shift 2
        ;;
      *)
        echo "recall: unknown argument: $1" >&2
        exit 1
        ;;
    esac
  done

  if [[ -z "$project" ]]; then
    echo "recall: --project is required" >&2
    exit 1
  fi

  if ! cmd_validate "$card_path" >/dev/null 2>&1; then
    echo "recall: invalid card" >&2
    exit 1
  fi

  if [[ ! -x "$LEDGER_SCRIPT" ]]; then
    python3 - "$card_path" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as f:
    card = json.load(f)
card["similar_past_decisions"] = []
print(json.dumps(card, ensure_ascii=False, indent=2))
PY
    return
  fi

  python3 - "$card_path" "$project" "$LEDGER_SCRIPT" <<'PY'
import json
import subprocess
import sys

card_path, project, ledger_script = sys.argv[1:4]
with open(card_path, encoding="utf-8") as f:
    card = json.load(f)

question = card.get("question", "")
proc = subprocess.run(
    [ledger_script, "recall", "--project", project, "--question", question],
    capture_output=True,
    text=True,
    check=False,
)
similar_past_decisions = []
if proc.returncode == 0 and proc.stdout.strip():
    similar_past_decisions = json.loads(proc.stdout)

card["similar_past_decisions"] = similar_past_decisions[:3]
print(json.dumps(card, ensure_ascii=False, indent=2))
PY
}

main() {
  local cmd="${1:-}"
  shift || true

  case "$cmd" in
    should-issue)
      cmd_should_issue "$@"
      ;;
    validate)
      cmd_validate "${1:-}"
      ;;
    record-answer)
      cmd_record_answer "$@"
      ;;
    recall)
      cmd_recall "$@"
      ;;
    compute-impact)
      cmd_compute_impact "$@"
      ;;
    ""|-h|--help|help)
      usage
      exit 0
      ;;
    *)
      echo "unknown subcommand: $cmd" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
