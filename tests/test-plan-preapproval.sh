#!/usr/bin/env bash
# Phase 108.5 — plan-time preapproval schema + secret-read runtimefloor bridge.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA="${ROOT_DIR}/templates/schemas/plan-preapproval.v1.json"
SCRIPT="${ROOT_DIR}/scripts/plan-preapproval.sh"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

pass() { echo "✓ $1"; }
fail() { echo "✗ $1" >&2; exit 1; }

[ -f "${SCHEMA}" ] || fail "schema missing: ${SCHEMA}"
[ -x "${SCRIPT}" ] || fail "script missing or not executable: ${SCRIPT}"

grep -q "## 事前確認" "${ROOT_DIR}/skills/harness-plan/SKILL.md" \
  || fail "harness-plan must generate a preapproval section"
grep -q "事項: <secret-read / external-send / destructive の具体操作>" "${ROOT_DIR}/skills/harness-plan/SKILL.md" \
  || fail "harness-plan must pin the preapproval item format"
grep -q "確認は plan 承認時の 1 回のみ" "${ROOT_DIR}/skills/harness-plan/SKILL.md" \
  || fail "harness-plan must state approval happens once at plan approval"
grep -q "work 中の宣言済み事項起因.*AskUserQuestion.*ゼロ" "${ROOT_DIR}/skills/harness-work/SKILL.md" \
  || fail "harness-work must prohibit AskUserQuestion for declared items"
grep -q "記録に無い未計画の secret-read / 外部送信 / 破壊的操作は従来どおり runtime floor / ask で停止" "${ROOT_DIR}/skills/breezing/SKILL.md" \
  || fail "breezing must preserve undeclared floor/ask behavior"
grep -q "HARNESS_RUNTIME_FLOOR_SECRET_ALLOW" "${ROOT_DIR}/docs/sandbox-allowlist-recipe.md" \
  && grep -q "plan-time 事前確認" "${ROOT_DIR}/docs/sandbox-allowlist-recipe.md" \
  || fail "docs must mention migration from broad env allow to plan-time preapproval"
pass "skill/docs preapproval contract text is present"

STATE="${TMP_DIR}/plan-preapprovals.json"
cat >"${STATE}" <<'JSON'
{
  "schema_version": "plan-preapproval.v1",
  "approved_at": "2026-07-07T09:30:00Z",
  "approvals": [
    {
      "item": "Read local dotenv for integration smoke",
      "reason": "DoD command needs a local token presence check without printing values",
      "scope": {"phase": "108", "task": "108.5"},
      "operations": ["secret-read"],
      "paths": [".env.integration"],
      "commands": ["grep -q API_TOKEN .env.integration"],
      "decision": "approved",
      "approved_at": "2026-07-07T09:30:00Z"
    },
    {
      "item": "Read undeclared production env",
      "reason": "Should not be reflected because it was denied",
      "scope": {"phase": "108", "task": "108.5"},
      "operations": ["secret-read"],
      "paths": [".env.production"],
      "decision": "denied",
      "approved_at": "2026-07-07T09:31:00Z"
    }
  ]
}
JSON

"${SCRIPT}" validate "${STATE}" >/dev/null
pass "plan-preapproval.v1 fixture validates"

PROJECT="${TMP_DIR}/project"
mkdir -p "${PROJECT}/.claude/state"
cp "${STATE}" "${PROJECT}/.claude/state/plan-preapprovals.json"

"${SCRIPT}" apply-secret-allow "${PROJECT}" >/dev/null
CONFIG="${PROJECT}/.claude-code-harness.config.json"
[ -f "${CONFIG}" ] || fail "project config was not created"

python3 - "${CONFIG}" <<'PY'
import json, sys
data = json.load(open(sys.argv[1], encoding="utf-8"))
allow = data.get("runtimefloor", {}).get("secretAllow", [])
if ".env.integration" not in allow:
    raise SystemExit("approved path was not reflected into runtimefloor.secretAllow")
if ".env.production" in allow:
    raise SystemExit("denied path leaked into runtimefloor.secretAllow")
PY
pass "approved secret-read path reflected, denied path excluded"

(
  cd "${ROOT_DIR}/go"
  GOCACHE="${TMP_DIR}/go-build" go test ./internal/runtimefloor -run 'TestCheckSecretRead_ConfigAllowlistedPathPasses|TestCheckSecretRead_UnsetEnvKeepsDenyByDefault' -count=1
) >/dev/null
pass "runtimefloor config bridge allows declared path and keeps deny-by-default"
