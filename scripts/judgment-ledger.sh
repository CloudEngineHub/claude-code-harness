#!/usr/bin/env bash
# judgment-ledger.sh — Judgment Ledger v1 helpers (Phase 98.1)
#
# Subcommands:
#   append  — append one record (fail-open: exit 0 + stderr warning on write failure)
#   search  — project-scoped string-match search (max 3 JSON lines to stdout)
#   recall  — project-scoped recall for similar_past_decisions (max 3 JSON array)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA="${ROOT}/templates/schemas/judgment-ledger.v1.json"
DEFAULT_LEDGER="${ROOT}/.claude/state/judgment-ledger.jsonl"

ledger_path() {
  if [[ -n "${HARNESS_JUDGMENT_LEDGER:-}" ]]; then
    printf '%s' "$HARNESS_JUDGMENT_LEDGER"
  else
    printf '%s' "$DEFAULT_LEDGER"
  fi
}

usage() {
  cat <<'EOF'
Usage:
  judgment-ledger.sh append --project <p> --question <q> --answer <a> \
    [--rationale <text>] --card-ref <path> [--tags t1,t2] [--id <uuid>]
  judgment-ledger.sh search --project <p> --query <text>
  judgment-ledger.sh recall --project <p> --question <text>
EOF
}

cmd_append() {
  local project="" question="" answer="" rationale="" card_ref="" tags="" record_id=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --project) project="${2:-}"; shift 2 ;;
      --question) question="${2:-}"; shift 2 ;;
      --answer) answer="${2:-}"; shift 2 ;;
      --rationale) rationale="${2:-}"; shift 2 ;;
      --card-ref) card_ref="${2:-}"; shift 2 ;;
      --tags) tags="${2:-}"; shift 2 ;;
      --id) record_id="${2:-}"; shift 2 ;;
      *)
        echo "append: unknown argument: $1" >&2
        exit 1
        ;;
    esac
  done

  if [[ -z "$project" || -z "$question" || -z "$answer" || -z "$card_ref" ]]; then
    echo "append: --project, --question, --answer, and --card-ref are required" >&2
    exit 1
  fi

  local ledger
  ledger="$(ledger_path)"

  python3 - "$SCHEMA" "$ledger" "$project" "$question" "$answer" "$rationale" "$card_ref" "$tags" "$record_id" <<'PY'
import json
import os
import sys
import uuid
from datetime import datetime, timezone

schema_path, ledger_path, project, question, answer, rationale, card_ref, tags_csv, record_id = sys.argv[1:10]

with open(schema_path, encoding="utf-8") as f:
    schema = json.load(f)

tags = [t.strip() for t in tags_csv.split(",") if t.strip()] if tags_csv else []
record = {
    "id": record_id or str(uuid.uuid4()),
    "project": project,
    "decided_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "question": question,
    "answer": answer,
    "rationale": rationale,
    "card_ref": card_ref,
    "tags": tags,
}


def validate(data, sch):
    required = set(sch.get("required", []))
    props = sch.get("properties", {})
    if sch.get("additionalProperties") is False:
        extra = set(data.keys()) - set(props.keys())
        if extra:
            raise ValueError(f"additional properties not allowed: {sorted(extra)}")
    for key in required:
        if key not in data:
            raise ValueError(f"missing required property: {key}")
    for key in ("id", "project", "decided_at", "question", "answer", "card_ref"):
        val = data.get(key)
        if not isinstance(val, str) or not val:
            raise ValueError(f"{key} must be a non-empty string")
    if not isinstance(data.get("rationale"), str):
        raise ValueError("rationale must be a string")
    if not isinstance(data.get("tags"), list):
        raise ValueError("tags must be an array")


try:
    validate(record, schema)
except ValueError as exc:
    print(f"judgment-ledger: append rejected ({exc})", file=sys.stderr)
    raise SystemExit(1)

line = json.dumps(record, ensure_ascii=False)
try:
    os.makedirs(os.path.dirname(ledger_path) or ".", exist_ok=True)
    with open(ledger_path, "a", encoding="utf-8") as f:
        f.write(line + "\n")
except OSError as exc:
    print(f"judgment-ledger: append skipped ({exc})", file=sys.stderr)
    raise SystemExit(0)

raise SystemExit(0)
PY
}

load_records() {
  local ledger="$1"
  local project="$2"
  python3 - "$ledger" "$project" <<'PY'
import json
import sys

ledger_path, project = sys.argv[1], sys.argv[2]
records = []
try:
    with open(ledger_path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                rec = json.loads(line)
            except json.JSONDecodeError:
                continue
            if rec.get("project") == project:
                records.append(rec)
except FileNotFoundError:
    pass
json.dump(records, sys.stdout, ensure_ascii=False)
PY
}

cmd_search() {
  local project="" query=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --project) project="${2:-}"; shift 2 ;;
      --query) query="${2:-}"; shift 2 ;;
      *)
        echo "search: unknown argument: $1" >&2
        exit 1
        ;;
    esac
  done

  if [[ -z "$project" ]]; then
    echo "search: --project is required" >&2
    exit 1
  fi

  local ledger
  ledger="$(ledger_path)"

  python3 - <<'PY' "$(load_records "$ledger" "$project")" "$query"
import json
import sys

records = json.loads(sys.argv[1])
query = sys.argv[2].strip().lower()

def score(rec):
    if not query:
        return 1
    blob = " ".join([
        rec.get("question", ""),
        rec.get("answer", ""),
        rec.get("rationale", ""),
        " ".join(rec.get("tags") or []),
    ]).lower()
    if query in blob:
        return 10
    tokens = [t for t in query.split() if len(t) >= 2]
    if not tokens:
        return 0
    s = 0
    for token in tokens:
        if token in blob:
            s += 1
    return s

ranked = [r for r in records if score(r) > 0 or not query]
ranked.sort(key=lambda r: (score(r), r.get("decided_at", "")), reverse=True)
for rec in ranked[:3]:
    print(json.dumps(rec, ensure_ascii=False))
PY
}

cmd_recall() {
  local project="" question=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --project) project="${2:-}"; shift 2 ;;
      --question) question="${2:-}"; shift 2 ;;
      *)
        echo "recall: unknown argument: $1" >&2
        exit 1
        ;;
    esac
  done

  if [[ -z "$project" || -z "$question" ]]; then
    echo "recall: --project and --question are required" >&2
    exit 1
  fi

  local ledger
  ledger="$(ledger_path)"

  python3 - <<'PY' "$(load_records "$ledger" "$project")" "$question"
import json
import sys

records = json.loads(sys.argv[1])
question = sys.argv[2].strip().lower()

def score(rec):
    if not question:
        return 1
    blob = " ".join([
        rec.get("question", ""),
        rec.get("answer", ""),
        rec.get("rationale", ""),
        " ".join(rec.get("tags") or []),
    ]).lower()
    if question in blob:
        return 10
    tokens = [t for t in question.split() if len(t) >= 2]
    if not tokens:
        return 0
    s = 0
    for token in tokens:
        if token in blob:
            s += 1
    return s

ranked = [r for r in records if score(r) > 0]
ranked.sort(key=lambda r: (score(r), r.get("decided_at", "")), reverse=True)

out = []
for rec in ranked[:3]:
    summary = rec.get("question", "")
    if len(summary) > 120:
        summary = summary[:117] + "..."
    out.append({
        "summary": summary,
        "decision": rec.get("answer", ""),
        "outcome": rec.get("rationale") or "recorded",
        "decided_at": rec.get("decided_at", ""),
        "mem_id": f"judgment-ledger:{rec.get('id', '')}",
    })

print(json.dumps(out, ensure_ascii=False))
PY
}

main() {
  local cmd="${1:-}"
  shift || true

  case "$cmd" in
    append) cmd_append "$@" ;;
    search) cmd_search "$@" ;;
    recall) cmd_recall "$@" ;;
    ""|-h|--help|help) usage; exit 0 ;;
    *)
      echo "unknown subcommand: $cmd" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
