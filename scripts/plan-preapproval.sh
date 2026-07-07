#!/usr/bin/env bash
# plan-preapproval.sh — validate and reflect plan-time preapprovals.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA="${ROOT}/templates/schemas/plan-preapproval.v1.json"

usage() {
  cat <<'EOF'
Usage:
  plan-preapproval.sh validate <plan-preapprovals.json>
  plan-preapproval.sh apply-secret-allow <project-root> [plan-preapprovals.json]
EOF
}

cmd_validate() {
  local state="${1:-}"
  if [ -z "${state}" ] || [ ! -f "${state}" ]; then
    echo "validate: file not found: ${state:-<missing>}" >&2
    exit 1
  fi
  if [ ! -f "${SCHEMA}" ]; then
    echo "validate: schema not found: ${SCHEMA}" >&2
    exit 1
  fi

  python3 - "${state}" "${SCHEMA}" <<'PY'
import json
import sys

state_path, schema_path = sys.argv[1], sys.argv[2]
with open(state_path, encoding="utf-8") as f:
    data = json.load(f)
with open(schema_path, encoding="utf-8") as f:
    schema = json.load(f)

try:
    import jsonschema  # type: ignore
except Exception:
    jsonschema = None

if jsonschema is not None:
    jsonschema.validate(instance=data, schema=schema)
else:
    if not isinstance(data, dict):
        raise SystemExit("root must be object")
    if data.get("schema_version") != "plan-preapproval.v1":
        raise SystemExit("schema_version must be plan-preapproval.v1")
    if not isinstance(data.get("approved_at"), str) or not data["approved_at"]:
        raise SystemExit("approved_at must be a non-empty string")
    approvals = data.get("approvals")
    if not isinstance(approvals, list):
        raise SystemExit("approvals must be an array")
    allowed_ops = {"secret-read", "external-send", "destructive"}
    for idx, item in enumerate(approvals):
        if not isinstance(item, dict):
            raise SystemExit(f"approvals[{idx}] must be object")
        for key in ("item", "reason", "scope", "operations", "decision", "approved_at"):
            if key not in item:
                raise SystemExit(f"approvals[{idx}] missing {key}")
        scope = item["scope"]
        if not isinstance(scope, dict) or not scope.get("phase") or not scope.get("task"):
            raise SystemExit(f"approvals[{idx}].scope must include phase and task")
        ops = item["operations"]
        if not isinstance(ops, list) or not ops:
            raise SystemExit(f"approvals[{idx}].operations must be non-empty array")
        for op in ops:
            if op not in allowed_ops:
                raise SystemExit(f"approvals[{idx}] invalid operation {op!r}")
        if item["decision"] not in {"approved", "denied"}:
            raise SystemExit(f"approvals[{idx}].decision invalid")
        if not any(k in item for k in ("paths", "commands", "targets")):
            raise SystemExit(f"approvals[{idx}] must include paths, commands, or targets")
print("OK")
PY
}

cmd_apply_secret_allow() {
  local project_root="${1:-}"
  local state="${2:-}"
  if [ -z "${project_root}" ] || [ ! -d "${project_root}" ]; then
    echo "apply-secret-allow: project root not found: ${project_root:-<missing>}" >&2
    exit 1
  fi
  if [ -z "${state}" ]; then
    state="${project_root}/.claude/state/plan-preapprovals.json"
  fi
  cmd_validate "${state}" >/dev/null

  python3 - "${project_root}" "${state}" <<'PY'
import json
import os
import sys

project_root, state_path = sys.argv[1], sys.argv[2]
config_path = os.path.join(project_root, ".claude-code-harness.config.json")

with open(state_path, encoding="utf-8") as f:
    state = json.load(f)

approved_paths = []
for item in state.get("approvals", []):
    if item.get("decision") != "approved":
        continue
    if "secret-read" not in item.get("operations", []):
        continue
    for path in item.get("paths", []):
        path = str(path).strip()
        if path and path not in ("*", "**", "/"):
            approved_paths.append(path)

if os.path.exists(config_path):
    with open(config_path, encoding="utf-8") as f:
        config = json.load(f)
else:
    config = {}

runtimefloor = config.setdefault("runtimefloor", {})
current = runtimefloor.get("secretAllow", [])
if not isinstance(current, list):
    current = []

merged = []
for path in list(current) + approved_paths:
    if isinstance(path, str) and path and path not in merged:
        merged.append(path)
runtimefloor["secretAllow"] = merged

tmp_path = config_path + ".tmp"
with open(tmp_path, "w", encoding="utf-8") as f:
    json.dump(config, f, ensure_ascii=False, indent=2)
    f.write("\n")
os.replace(tmp_path, config_path)
print(config_path)
PY
}

case "${1:-}" in
  validate)
    shift
    cmd_validate "$@"
    ;;
  apply-secret-allow)
    shift
    cmd_apply_secret_allow "$@"
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    echo "unknown command: $1" >&2
    usage >&2
    exit 1
    ;;
esac
