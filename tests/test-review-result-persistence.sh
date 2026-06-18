#!/usr/bin/env bash
# test-review-result-persistence.sh
# Phase 94.1.1: harness-review / harness-release skill が write-review-result.sh 経由で
# verdict を .claude/state/review-result.json に永続化することの契約検証 (#218 fix part-1)
#
# 検証内容:
#   (1) skills/harness-review/SKILL.md に write-review-result.sh 呼び出し step が記載
#   (2) skills/harness-release/SKILL.md の Review Gate 配下に同 step が記載
#   (3) write-review-result.sh が APPROVE JSON を受けて review-result.json を生成
#   (4) 生成された review-result.json の verdict が APPROVE
#   (5) (regression) write-review-result.sh が REQUEST_CHANGES でも保存する

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WRITE_SCRIPT="${ROOT_DIR}/scripts/write-review-result.sh"
REVIEW_SKILL="${ROOT_DIR}/skills/harness-review/SKILL.md"
RELEASE_SKILL="${ROOT_DIR}/skills/harness-release/SKILL.md"

pass=0
fail=0
trap 'echo "[summary] pass=${pass} fail=${fail}"; [ "${fail}" -eq 0 ]' EXIT

assert() {
  local name="$1"
  local cond="$2"
  if eval "$cond"; then
    echo "  PASS  ${name}"
    pass=$((pass + 1))
  else
    echo "  FAIL  ${name}"
    fail=$((fail + 1))
  fi
}

# (1) harness-review SKILL に呼び出し step
echo "[1] harness-review/SKILL.md contains write-review-result.sh step"
assert "review skill references write-review-result.sh" \
  "grep -q 'write-review-result.sh' '${REVIEW_SKILL}'"
assert "review skill mentions Persist Verdict" \
  "grep -qE 'Persist Verdict|review-result\\.json' '${REVIEW_SKILL}'"

# (2) harness-release Review Gate に呼び出し step
echo "[2] harness-release/SKILL.md Review Gate contains persistence step"
assert "release skill references write-review-result.sh" \
  "grep -q 'write-review-result.sh' '${RELEASE_SKILL}'"
assert "release skill mentions Persist Verdict" \
  "grep -qE 'Persist Verdict|review-result\\.json' '${RELEASE_SKILL}'"

# (3) write-review-result.sh が APPROVE JSON を正規化保存
echo "[3] write-review-result.sh APPROVE round-trip"
TMPDIR_TEST="$(mktemp -d)"
INPUT_JSON="${TMPDIR_TEST}/input.json"
OUTPUT_JSON="${TMPDIR_TEST}/review-result.json"
cat > "${INPUT_JSON}" <<'JSON'
{
  "schema_version": "review-result.v1",
  "verdict": "APPROVE",
  "decision_needed": {"required": false, "ask_tool": "AskUserQuestion"},
  "accepted_findings": [],
  "rejected_findings": [],
  "acceptance_bar": {
    "critical_major_zero": true,
    "spec_alignment": "pass",
    "plans_alignment": "pass",
    "regression_safety": "pass",
    "verification_evidence": "pass"
  },
  "critical_issues": [],
  "major_issues": [],
  "observations": [],
  "recommendations": []
}
JSON

bash "${WRITE_SCRIPT}" "${INPUT_JSON}" "deadbeef" "${OUTPUT_JSON}" >/dev/null 2>&1
assert "output review-result.json exists" "[ -f '${OUTPUT_JSON}' ]"

# (4) verdict 確認
echo "[4] persisted verdict == APPROVE"
if command -v jq >/dev/null 2>&1; then
  VERDICT=$(jq -r '.verdict' "${OUTPUT_JSON}")
  assert "verdict is APPROVE" "[ '${VERDICT}' = 'APPROVE' ]"
  HASH=$(jq -r '.commit_hash // empty' "${OUTPUT_JSON}")
  assert "commit_hash propagated" "[ '${HASH}' = 'deadbeef' ]"
else
  echo "  SKIP  verdict check (jq not available)"
fi

# (5) REQUEST_CHANGES も保存される (commit guard が APPROVE 以外で fail-closed を確認)
echo "[5] write-review-result.sh REQUEST_CHANGES round-trip"
INPUT2="${TMPDIR_TEST}/input2.json"
OUTPUT2="${TMPDIR_TEST}/review-result-rc.json"
cat > "${INPUT2}" <<'JSON'
{
  "schema_version": "review-result.v1",
  "verdict": "REQUEST_CHANGES",
  "decision_needed": {"required": false, "ask_tool": "AskUserQuestion"},
  "accepted_findings": [],
  "rejected_findings": [],
  "acceptance_bar": {
    "critical_major_zero": false,
    "spec_alignment": "pass",
    "plans_alignment": "pass",
    "regression_safety": "pass",
    "verification_evidence": "pass"
  },
  "critical_issues": [],
  "major_issues": [],
  "observations": [],
  "recommendations": []
}
JSON
bash "${WRITE_SCRIPT}" "${INPUT2}" "" "${OUTPUT2}" >/dev/null 2>&1
assert "REQUEST_CHANGES output exists" "[ -f '${OUTPUT2}' ]"
if command -v jq >/dev/null 2>&1; then
  V2=$(jq -r '.verdict' "${OUTPUT2}")
  assert "REQUEST_CHANGES verdict preserved" "[ '${V2}' = 'REQUEST_CHANGES' ]"
fi

rm -rf "${TMPDIR_TEST}"
