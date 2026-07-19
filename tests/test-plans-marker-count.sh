#!/bin/bash
# Contract: shell Plans.md marker counts must match Status column only (not legend/DoD prose).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
HARNESS_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
FIXTURE="$(mktemp)"
trap 'rm -f "$FIXTURE"' EXIT

assert_eq() {
  local got="$1"
  local want="$2"
  local label="$3"
  if [ "$got" != "$want" ]; then
    echo "[FAIL] ${label}: got '${got}', want '${want}'" >&2
    exit 1
  fi
}

cat > "${FIXTURE}" <<'EOF'
# Plans fixture — marker count contract

| `cc:wip` / `cc:WIP` | 着手中 | Impl |
正規出力は pm:requested → cc:todo → cc:wip → cc:done → pm:approved

| Task | 内容 | DoD | Depends | Status |
|------|------|-----|---------|--------|
| 1.1 | cc:WIP 状態が 10 分超なら re-spawn | test PASS | - | cc:done [abc123] |
| 2.1 | canonical wip task | done when green | - | cc:wip |
| 2.2 | todo task | done when green | - | cc:todo |
| 2.3 | legacy wip task | done when green | - | cc:WIP |
| 2.4 | legacy done task | done when green | - | cc:完了 |
EOF

PLANS_FILE="${FIXTURE}"

# Prefer shared helper when present (GREEN); fall back to session-monitor count_tasks (RED).
if [ -f "${HARNESS_ROOT}/scripts/plans-marker-count.sh" ]; then
  # shellcheck source=../scripts/plans-marker-count.sh
  source "${HARNESS_ROOT}/scripts/plans-marker-count.sh"
  count_marker() { count_status_cells "$1" "${PLANS_FILE}"; }
else
  eval "$(sed -n '/^count_tasks() {/,/^}/p' "${HARNESS_ROOT}/scripts/session-monitor.sh")"
  count_marker() { count_tasks "$1"; }
fi

wip_count=$(( $(count_marker "cc:wip") + $(count_marker "cc:WIP") ))
todo_count=$(( $(count_marker "cc:todo") + $(count_marker "cc:TODO") ))
done_count=$(( $(count_marker "cc:done") + $(count_marker "cc:完了") ))

assert_eq "${wip_count}" "2" "WIP status-cell count"
assert_eq "${todo_count}" "1" "TODO status-cell count"
assert_eq "${done_count}" "2" "done status-cell count"

echo "test-plans-marker-count: ok"
