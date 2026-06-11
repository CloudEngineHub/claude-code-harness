#!/bin/bash
# spawn-parallel.sh — create parallel task worktrees from one shared base SHA.
#
# Usage: scripts/spawn-parallel.sh <task-a> <task-b> [task-c ...]
#
# Contract (Phase 92.1.1):
#   - git fetch origin
#   - BASE=$(git rev-parse HEAD) once for all tasks
#   - git worktree add -b task/$T <project>/.harness-worktrees/task-$T $BASE
#   - git config rerere.enabled true (project-local)
#   - idempotent: existing worktree with same base → no-op; different base → fail-fast

set -euo pipefail

usage() {
  echo "Usage: spawn-parallel.sh <task-a> <task-b> [task-c ...]" >&2
  exit 1
}

die() {
  echo "spawn-parallel: $1" >&2
  exit 1
}

if ! PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
  die "must be run inside a git repository (git rev-parse --show-toplevel failed)"
fi
HARNESS_WORKTREES_ROOT="${PROJECT_ROOT}/.harness-worktrees"

worktree_registered() {
  local path="$1"
  git -C "${PROJECT_ROOT}" worktree list --porcelain | awk -v target="${path}" '
    $1 == "worktree" && $2 == target { found = 1 }
    END { exit(found ? 0 : 1) }
  '
}

[[ $# -ge 1 ]] || usage

git config rerere.enabled true
git fetch origin

BASE="$(git rev-parse HEAD)"
mkdir -p "${HARNESS_WORKTREES_ROOT}"

for TASK in "$@"; do
  BRANCH="task/${TASK}"
  WT_PATH="${HARNESS_WORKTREES_ROOT}/task-${TASK}"

  if worktree_registered "${WT_PATH}"; then
    EXISTING_SHA="$(git -C "${WT_PATH}" rev-parse HEAD)"
    if [[ "${EXISTING_SHA}" != "${BASE}" ]]; then
      die "worktree ${WT_PATH} exists at base ${EXISTING_SHA}, expected ${BASE}; refusing to overwrite"
    fi
    continue
  fi

  if [[ -e "${WT_PATH}" ]]; then
    die "path ${WT_PATH} exists but is not a registered worktree; refusing to overwrite"
  fi

  if git show-ref --verify --quiet "refs/heads/${BRANCH}"; then
    BRANCH_SHA="$(git rev-parse "${BRANCH}^{commit}")"
    if [[ "${BRANCH_SHA}" != "${BASE}" ]]; then
      die "branch ${BRANCH} exists at ${BRANCH_SHA}, expected base ${BASE}; refusing to reuse"
    fi
    git worktree add "${WT_PATH}" "${BRANCH}"
  else
    git worktree add -b "${BRANCH}" "${WT_PATH}" "${BASE}"
  fi
done
