#!/bin/bash
# Verify that Plans.md task workflows also preserve a project spec SSOT when needed.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

pass() {
  echo "PASS: $1"
}

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

require_contains() {
  local file="$1"
  local pattern="$2"
  local label="$3"

  if grep -Fq "$pattern" "$file"; then
    pass "$label"
  else
    fail "$label ($file に '$pattern' がありません)"
  fi
}

SPEC_DOC="$PLUGIN_ROOT/docs/plans/spec-ssot.md"
ROOT_SPEC="$PLUGIN_ROOT/spec.md"
PLAN_SKILL="$PLUGIN_ROOT/skills/harness-plan/SKILL.md"
PLAN_CREATE_REF="$PLUGIN_ROOT/skills/harness-plan/references/create.md"
WORK_SKILL="$PLUGIN_ROOT/skills/harness-work/SKILL.md"
WORK_EXEC_REF="$PLUGIN_ROOT/skills/harness-work/references/execution-modes.md"
CODEX_WORK_SKILL="$PLUGIN_ROOT/skills-codex/harness-work/SKILL.md"
CODEX_WORK_EXEC_REF="$PLUGIN_ROOT/skills-codex/harness-work/references/execution-modes.md"
WORKER_AGENT="$PLUGIN_ROOT/agents/worker.md"
SCAFFOLDER_AGENT="$PLUGIN_ROOT/agents/scaffolder.md"
REVIEWER_AGENT="$PLUGIN_ROOT/agents/reviewer.md"
REVIEW_SKILL="$PLUGIN_ROOT/skills/harness-review/SKILL.md"

echo "=== spec SSOT workflow test ==="

[ -f "$SPEC_DOC" ] || fail "docs/plans/spec-ssot.md が見つかりません"
[ -f "$ROOT_SPEC" ] || fail "root spec.md が見つかりません"

require_contains "$SPEC_DOC" "Plans.md is the task ledger. A project spec SSOT is the product contract." "spec doc が Plans.md と仕様正本の役割を分けている"
require_contains "$SPEC_DOC" "spec.md" "spec doc が root spec.md を優先候補に含む"
require_contains "$SPEC_DOC" "docs/spec/00-project-spec.md" "spec doc が default spec path を示している"
require_contains "$SPEC_DOC" "When To Create Or Update It" "spec doc が作成/更新条件を持つ"
require_contains "$SPEC_DOC" "When To Skip" "spec doc がスキップ条件を持つ"

require_contains "$ROOT_SPEC" "Plans.md is the task ledger" "root spec が Plans.md を task ledger と定義している"
require_contains "$ROOT_SPEC" "not_observed != absent" "root spec が未観測データ契約を持つ"
require_contains "$ROOT_SPEC" "docs/architecture/hokage-core.md" "root spec が Hokage Core を sub-spec として参照する"
require_contains "$ROOT_SPEC" "go/SPEC.md" "root spec が Go runtime sub-spec を参照する"
require_contains "$ROOT_SPEC" "Host Adapter Boundary" "root spec が host adapter boundary を持つ"
require_contains "$ROOT_SPEC" "Support Tiers And Host Claims" "root spec が support tier 契約を持つ"
require_contains "$ROOT_SPEC" "Onboarding Contract" "root spec が onboarding contract を持つ"
require_contains "$ROOT_SPEC" "New Session Bootstrap Rule" "root spec が new session bootstrap rule を持つ"
require_contains "$ROOT_SPEC" "future/unsupported" "root spec が unsupported host claim を tier 管理する"

require_contains "$PLAN_SKILL" "仕様正本チェック（デフォルト）" "harness-plan が仕様正本チェックを default flow に含む"
require_contains "$PLAN_SKILL" "docs/plans/spec-ssot.md" "harness-plan が spec SSOT doc を参照する"
require_contains "$PLAN_CREATE_REF" "## Step 4.4: 仕様正本チェック" "harness-plan create reference に仕様正本ステップがある"

require_contains "$WORK_SKILL" "仕様正本 preflight" "harness-work が実装前の仕様正本 preflight を持つ"
require_contains "$WORK_SKILL" "spec_path" "harness-work が Worker / Reviewer へ spec_path を渡す"
require_contains "$WORK_EXEC_REF" "project spec SSOT" "shared execution mode が spec SSOT preflight を持つ"

require_contains "$CODEX_WORK_SKILL" "仕様正本 preflight" "Codex harness-work が仕様正本 preflight を持つ"
require_contains "$CODEX_WORK_SKILL" "spec_skip_reason" "Codex harness-work が spec_skip_reason を Worker に渡す"
require_contains "$CODEX_WORK_EXEC_REF" "project spec SSOT" "Codex execution mode が spec SSOT preflight を持つ"

require_contains "$WORKER_AGENT" "spec_path" "Worker input が spec_path を受け取る"
require_contains "$SCAFFOLDER_AGENT" "spec_required" "Scaffolder analyze が spec_required を返す"
require_contains "$REVIEWER_AGENT" "spec_path" "Reviewer input が spec_path を受け取る"
require_contains "$REVIEW_SKILL" "仕様正本 alignment check" "harness-review が spec alignment を確認する"

echo "All spec SSOT workflow checks passed."
