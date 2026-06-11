#!/bin/bash
# test-reap-worktrees.sh — Phase 92.1.2 contract for scripts/reap-worktrees.sh
#
# Verifies:
#   (a)(b)(d) 3 harness worktrees → reap → trunk only + no task/* branches
#   (c) empty harness worktrees → no-op success
#   (correction 1) worktrees outside .harness-worktrees/ survive reap
#   (correction 3) dirty worktree skipped by default; --force removes

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REAP="${ROOT_DIR}/scripts/reap-worktrees.sh"
SPAWN="${ROOT_DIR}/scripts/spawn-parallel.sh"
TMP_DIR="$(mktemp -d)"
trap 'cleanup' EXIT

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

cleanup() {
  if [[ -n "${REPO:-}" && -d "${REPO}" ]]; then
    git -C "${REPO}" worktree list --porcelain 2>/dev/null | awk '/^worktree /{print $2}' | while read -r wt; do
      [[ "${wt}" == "${REPO}" ]] && continue
      git -C "${REPO}" worktree remove --force "${wt}" 2>/dev/null || true
    done
  fi
  rm -rf "${TMP_DIR}"
}

SETUP_COUNTER=0

setup_repo() {
  SETUP_COUNTER=$((SETUP_COUNTER + 1))
  REPO="${TMP_DIR}/repo-${SETUP_COUNTER}"
  ORIGIN="${TMP_DIR}/origin-${SETUP_COUNTER}.git"
  mkdir -p "${REPO}"
  (
    cd "${REPO}"
    git init -q -b main
    git config user.email test@test
    git config user.name test
    echo seed > README
    git add README
    git commit -qm seed
    git init --bare -q "${ORIGIN}"
    git remote add origin "${ORIGIN}"
    git push -q -u origin main
  )
  REPO="$(cd "${REPO}" && pwd -P)"
}

count_harness_worktrees() {
  local count=0
  local prefix="${REPO}/.harness-worktrees/"
  while read -r wt; do
    [[ -n "${wt}" ]] || continue
    local canonical
    canonical="$(cd "${wt}" 2>/dev/null && pwd -P || echo "${wt}")"
    [[ "${canonical}" == "${REPO}" ]] && continue
    [[ "${canonical}" == "${prefix}"* ]] && count=$((count + 1))
  done < <(git -C "${REPO}" worktree list --porcelain | awk '/^worktree /{print $2}')
  echo "${count}"
}

count_task_branches() {
  git -C "${REPO}" branch --list 'task/*' | wc -l | tr -d ' '
}

worktree_exists() {
  local path="$1"
  local canonical listed listed_canonical
  canonical="$(cd "${path}" 2>/dev/null && pwd -P || echo "${path}")"
  while read -r listed; do
    [[ -n "${listed}" ]] || continue
    listed_canonical="$(cd "${listed}" 2>/dev/null && pwd -P || echo "${listed}")"
    if [[ "${listed_canonical}" == "${canonical}" ]]; then
      return 0
    fi
  done < <(git -C "${REPO}" worktree list --porcelain | awk '/^worktree /{print $2}')
  return 1
}

# --- preflight ---
[[ -x "${REAP}" ]] || fail "missing executable: ${REAP}"
[[ -x "${SPAWN}" ]] || fail "missing executable: ${SPAWN}"

setup_repo

# --- (c) no harness worktrees → no-op ---
(
  cd "${REPO}"
  bash "${REAP}"
)
[[ "$(count_harness_worktrees)" == "0" ]] || fail "expected 0 harness worktrees before spawn"
[[ "$(count_task_branches)" == "0" ]] || fail "expected 0 task/* branches before spawn"

# --- (a)(b)(d) three harness worktrees → reap cleans all ---
(
  cd "${REPO}"
  bash "${SPAWN}" A B C
)

WT_A="${REPO}/.harness-worktrees/task-A"
WT_B="${REPO}/.harness-worktrees/task-B"
WT_C="${REPO}/.harness-worktrees/task-C"

for wt in "${WT_A}" "${WT_B}" "${WT_C}"; do
  worktree_exists "${wt}" || fail "setup: harness worktree missing: ${wt}"
done
[[ "$(count_task_branches)" == "3" ]] || fail "setup: expected 3 task/* branches"

(
  cd "${REPO}"
  bash "${REAP}"
)

[[ "$(count_harness_worktrees)" == "0" ]] || fail "expected 0 harness worktrees after reap (got $(count_harness_worktrees))"
[[ "$(count_task_branches)" == "0" ]] || fail "expected 0 task/* branches after reap (got $(count_task_branches))"

wt_count="$(git -C "${REPO}" worktree list | wc -l | tr -d ' ')"
[[ "${wt_count}" == "1" ]] || fail "expected single trunk worktree after reap, got ${wt_count}"

# --- correction 1: external worktree survives ---
setup_repo
EXTERNAL="${REPO}/.claude/worktrees/agent-1"
(
  cd "${REPO}"
  bash "${SPAWN}" X Y
  mkdir -p "${REPO}/.claude/worktrees"
  git worktree add -b claude/live-agent "${EXTERNAL}" HEAD
  bash "${REAP}"
)

worktree_exists "${EXTERNAL}" || fail "external worktree must survive reap: ${EXTERNAL}"
[[ "$(count_harness_worktrees)" == "0" ]] || fail "harness worktrees must be gone after reap"
[[ "$(count_task_branches)" == "0" ]] || fail "task/* branches must be gone after reap"
git -C "${REPO}" show-ref --verify --quiet refs/heads/claude/live-agent || fail "external branch must survive reap"

# --- correction 3: dirty worktree skipped; --force removes ---
setup_repo
WT_DIRTY="${REPO}/.harness-worktrees/task-dirty"
(
  cd "${REPO}"
  bash "${SPAWN}" dirty
  echo uncommitted > "${WT_DIRTY}/dirty.txt"
  bash "${REAP}" 2>"${TMP_DIR}/dirty-skip.err"
)

worktree_exists "${WT_DIRTY}" || fail "dirty worktree must be skipped by default"
[[ "$(count_task_branches)" == "1" ]] || fail "task/dirty branch must remain when worktree skipped"
grep -qi "skip" "${TMP_DIR}/dirty-skip.err" || fail "expected skip warning for dirty worktree"

(
  cd "${REPO}"
  bash "${REAP}" --force
)

worktree_exists "${WT_DIRTY}" && fail "dirty worktree must be removed with --force"
[[ "$(count_task_branches)" == "0" ]] || fail "task/dirty branch must be deleted after --force reap"

echo "test-reap-worktrees: ok"
