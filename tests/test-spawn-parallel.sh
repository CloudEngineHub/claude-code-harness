#!/bin/bash
# test-spawn-parallel.sh — Phase 92.1.1 contract for scripts/spawn-parallel.sh
#
# Verifies:
#   (a) N tasks → N worktrees on N branches, all sharing one base SHA
#   (b) worktrees live under .harness-worktrees/ only
#   (c) rerere.enabled is set in project config
#   (d) re-run is idempotent (no destroy, same-base no-op, diff-base fail-fast)

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SPAWN="${ROOT_DIR}/scripts/spawn-parallel.sh"
TMP_DIR="$(mktemp -d)"
trap 'cleanup' EXIT

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

cleanup() {
  if [[ -d "${REPO:-}" ]]; then
    git -C "${REPO}" worktree list --porcelain 2>/dev/null | awk '/^worktree /{print $2}' | while read -r wt; do
      [[ "${wt}" == "${REPO}" ]] && continue
      git -C "${REPO}" worktree remove --force "${wt}" 2>/dev/null || true
    done
  fi
  rm -rf "${TMP_DIR}"
}

setup_repo() {
  REPO="${TMP_DIR}/repo"
  ORIGIN="${TMP_DIR}/origin.git"
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
}

assert_same_base_sha() {
  local base_sha="$1"
  shift
  for wt in "$@"; do
    local got
    got="$(git -C "${wt}" rev-parse HEAD)"
    [[ "${got}" == "${base_sha}" ]] || fail "base SHA mismatch for ${wt}: got ${got}, want ${base_sha}"
  done
}

# --- preflight ---
[[ -x "${SPAWN}" ]] || fail "missing executable: ${SPAWN}"

setup_repo
cd "${REPO}"
BASE_SHA="$(git rev-parse HEAD)"

# --- (a)(b) three tasks, three branches, one base ---
bash "${SPAWN}" A B C

WT_A="${REPO}/.harness-worktrees/task-A"
WT_B="${REPO}/.harness-worktrees/task-B"
WT_C="${REPO}/.harness-worktrees/task-C"

for wt in "${WT_A}" "${WT_B}" "${WT_C}"; do
  [[ -d "${wt}" ]] || fail "worktree missing: ${wt}"
  git -C "${wt}" rev-parse --is-inside-work-tree >/dev/null 2>&1 || fail "not a git worktree: ${wt}"
done

for branch in task/A task/B task/C; do
  git -C "${REPO}" show-ref --verify --quiet "refs/heads/${branch}" || fail "branch missing: ${branch}"
done

assert_same_base_sha "${BASE_SHA}" "${WT_A}" "${WT_B}" "${WT_C}"

# stray paths under .claude/worktrees must not be created by spawn-parallel
if [[ -d "${REPO}/.claude/worktrees" ]]; then
  count="$(find "${REPO}/.claude/worktrees" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l | tr -d ' ')"
  [[ "${count}" == "0" ]] || fail "spawn-parallel must not create .claude/worktrees entries (found ${count})"
fi

# --- (c) rerere.enabled ---
rerere_val="$(git -C "${REPO}" config --get rerere.enabled || true)"
[[ "${rerere_val}" == "true" ]] || fail "rerere.enabled expected true, got '${rerere_val}'"

# --- (d) idempotent re-run (same base → no-op) ---
bash "${SPAWN}" A B C
assert_same_base_sha "${BASE_SHA}" "${WT_A}" "${WT_B}" "${WT_C}"

# --- (d) diff base → fail-fast, worktrees preserved ---
echo drift > README
git add README
git commit -qm drift
NEW_BASE="$(git rev-parse HEAD)"
[[ "${NEW_BASE}" != "${BASE_SHA}" ]] || fail "test setup: expected new base after drift commit"

set +e
bash "${SPAWN}" A B C 2>"${TMP_DIR}/diff-base.err"
rc=$?
set -e
[[ "${rc}" -ne 0 ]] || fail "expected non-zero exit when existing worktree base differs"
grep -q "base" "${TMP_DIR}/diff-base.err" || fail "expected base mismatch error message"
assert_same_base_sha "${BASE_SHA}" "${WT_A}" "${WT_B}" "${WT_C}"

echo "test-spawn-parallel: ok"
