#!/usr/bin/env bash
#
# Guard the harness-plan planning quality contract across shipped skill mirrors.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

fail() {
  echo "test-harness-plan-quality: FAIL: $*" >&2
  exit 1
}

assert_file() {
  local path="$1"
  [ -f "$path" ] || fail "missing file: $path"
}

assert_absent() {
  local path="$1"
  local needle="$2"
  if grep -qF "$needle" "$path"; then
    fail "$path should not contain: $needle"
  fi
}

assert_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -qF "$needle" "$path"; then
    fail "$path missing: $needle"
  fi
}

plan_output_contract_valid() {
  local output="$1"

  { grep -qF "Spec delta:" <<<"$output" || grep -qF "Spec skip reason:" <<<"$output"; } \
    && grep -qF "Plans.md:" <<<"$output"
}

assert_plan_output_contract_valid() {
  local label="$1"
  local output="$2"

  if ! plan_output_contract_valid "$output"; then
    fail "$label should include Spec delta or Spec skip reason plus Plans.md"
  fi
}

assert_plan_output_contract_invalid() {
  local label="$1"
  local output="$2"

  if plan_output_contract_valid "$output"; then
    fail "$label should fail without Spec delta or Spec skip reason"
  fi
}

primary_surfaces=(
  "skills/harness-plan"
  "codex/.codex/skills/harness-plan"
)

check_plan_surface() {
  local surface="$1"
  skill="$surface/SKILL.md"
  create_ref="$surface/references/create.md"
  quality_ref="$surface/references/planning-quality.md"

  assert_file "$skill"
  assert_file "$create_ref"
  assert_file "$quality_ref"

  assert_contains "$skill" "Research-backed task planning"
  assert_contains "$skill" "### 標準の計画品質契約"
  assert_contains "$skill" "See [references/planning-quality.md]"
  assert_contains "$skill" "Product / Architecture / QA / Skeptic"
  assert_contains "$skill" "Required / Recommended / Optional / Reject"
  assert_contains "$skill" "co-required planning output"
  assert_contains "$skill" "spec.md > sub-spec > Plans.md"
  assert_contains "$skill" "spec.md product contract and Plans.md task contract"
  assert_contains "$skill" '`/harness-plan create` は `Spec delta` または `Spec skip reason` と `Plans.md` task 生成をセットで返す'
  assert_contains "$skill" "Harness が生成し、consumer は承認・修正だけ"
  assert_contains "$skill" '`create` と product-impacting `add` は毎回 root `spec.md` を読む'
  assert_contains "$skill" '出力には必ず `Spec delta` または `Spec skip reason` を含める'
  assert_contains "$skill" 'consumer repo に root `spec.md` がない時だけ'
  assert_contains "$skill" "not_observed != absent"

  assert_absent "$skill" "/harness-plan maxplan"
  assert_absent "$skill" "argument-hint: \"[create|maxplan"
  assert_absent "$skill" "### maxplan"

  assert_contains "$create_ref" "## Step 3: 計画品質チェック"
  assert_contains "$create_ref" "references/planning-quality.md"
  assert_contains "$create_ref" "Product Fit、Evidence Strength、User Value、Implementation Feasibility、Regression Safety、Strategic Leverage"
  assert_contains "$create_ref" '`harness-mem` の DB は直接読まない'
  assert_contains "$create_ref" "## Step 4.4: spec.md / Plans.md 二正本チェック"
  assert_contains "$create_ref" 'root `spec.md` を毎回読む'
  assert_contains "$create_ref" "Spec delta"
  assert_contains "$create_ref" "Spec skip reason"
  assert_contains "$create_ref" "co-required planning output"
  assert_contains "$create_ref" "Harness が生成し、consumer は承認・修正だけ"
  assert_contains "$create_ref" "ユーザーに spec を一から書かせない"
  assert_contains "$create_ref" "docs-only / mechanical task"

  assert_contains "$quality_ref" "これは独立サブコマンドではない"
  assert_contains "$quality_ref" "WebSearch"
  assert_contains "$quality_ref" "cross-project 検索は、ユーザーが明示した場合だけ使う"
  assert_contains "$quality_ref" "harness-mem の DB を直接読む前提にしない"
  assert_contains "$quality_ref" '`create` と product-impacting `add` では root `spec.md` を毎回読む'
  assert_contains "$quality_ref" 'root `spec.md` がない consumer repo だけ'
  assert_contains "$quality_ref" "Spec delta"
  assert_contains "$quality_ref" "Spec skip reason"
  assert_contains "$quality_ref" "co-required planning output"
  assert_contains "$quality_ref" "not_observed != absent"
  assert_contains "$quality_ref" "Product / Strategy"
  assert_contains "$quality_ref" "Architecture / Implementation"
  assert_contains "$quality_ref" "QA / Regression"
  assert_contains "$quality_ref" "Skeptic"
  assert_contains "$quality_ref" "Implementation Feasibility"
  assert_contains "$quality_ref" "Regression Safety"
  assert_contains "$quality_ref" "導入先プロダクトの核に直結"
  assert_absent "$quality_ref" "Harness の核に直結"
  assert_contains "$quality_ref" "Evidence Strength が 2 以下なら Required 禁止"
  assert_contains "$quality_ref" "Regression Safety が 2 以下なら、先に spike / spec / test を置く"
  assert_contains "$quality_ref" '## Step 7: `$easy` 報告'
}

for surface in "${primary_surfaces[@]}"; do
  check_plan_surface "$surface"
  assert_contains "$surface/SKILL.md" "purpose: \"Maintain co-required planning output for the spec.md product contract and Plans.md task contract\""
  assert_contains "$surface/SKILL.md" "argument-hint: \"[create|add|update|sync|sync --no-retro|--ci]\""
done

opencode_surface="opencode/skills/harness-plan"
check_plan_surface "$opencode_surface"
assert_absent "$opencode_surface/SKILL.md" "purpose: \"Maintain co-required planning output for the spec.md product contract and Plans.md task contract\""
assert_absent "$opencode_surface/SKILL.md" "argument-hint: \"[create|add|update|sync|sync --no-retro|--ci]\""
node scripts/validate-opencode.js >/dev/null

assert_contains "scripts/sync-skill-mirrors.sh" '".agents/skills"'
if [ -d ".agents/skills/harness-plan" ]; then
  check_plan_surface ".agents/skills/harness-plan"
fi

assert_plan_output_contract_valid "create fixture with Spec delta" "Spec delta:
- path: spec.md
- change: add product contract
Plans.md:
| Task | 内容 | DoD | Depends | Status |"

assert_plan_output_contract_valid "add fixture with Spec skip reason" "Spec skip reason:
- path checked: spec.md
- reason: docs-only task
Plans.md:
| Task | 内容 | DoD | Depends | Status |"

assert_plan_output_contract_invalid "create fixture missing spec result" "Plans.md:
| Task | 内容 | DoD | Depends | Status |"

assert_plan_output_contract_invalid "add fixture missing spec result" "Plan:
- add a task with no spec result"

[ ! -e skills/harness-plan/references/maxplan.md ] || fail "maxplan reference must not exist in SSOT"

diff -qr --exclude='.DS_Store' skills/harness-plan codex/.codex/skills/harness-plan >/dev/null \
  || fail "codex harness-plan mirror drifted"
diff -qr --exclude='.DS_Store' skills/harness-plan/references opencode/skills/harness-plan/references >/dev/null \
  || fail "opencode harness-plan references drifted"

echo "test-harness-plan-quality: ok"
