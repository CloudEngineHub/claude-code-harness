#!/bin/bash
# reap-worktrees.sh — remove Harness-managed parallel worktrees and task/* branches.
#
# Usage: scripts/reap-worktrees.sh [--force]
#
# Contract (Phase 92.1.2):
#   - Removes only worktrees under .harness-worktrees/ (HarnessWorktreesRoot).
#     .claude/worktrees/ and other agent roots are never touched (see spec.md
#     "Worktree Root Discipline").
#   - Deletes task/* branches only when their worktree was successfully reaped.
#     Cherry-pick integration leaves task/* unmerged, so -D is used intentionally.
#   - Skips dirty worktrees by default; pass --force for git worktree remove --force.
#   - Refuses to run when CWD is inside a harness worktree slated for removal.
#   - Runs git worktree prune at the end.
#   - Safe no-op when no harness worktrees exist.

set -euo pipefail

FORCE=0

usage() {
  echo "Usage: reap-worktrees.sh [--force]" >&2
  exit 1
}

die() {
  echo "reap-worktrees: $1" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --force)
      FORCE=1
      shift
      ;;
    -h | --help)
      usage
      ;;
    *)
      usage
      ;;
  esac
done

if ! PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"; then
  die "must be run inside a git repository (git rev-parse --show-toplevel failed)"
fi
PROJECT_ROOT="$(cd "${PROJECT_ROOT}" && pwd -P)"

HARNESS_PREFIX="${PROJECT_ROOT}/.harness-worktrees/"
CURRENT_DIR="$(pwd -P)"

canonical_path() {
  local path="$1"
  if [[ -d "${path}" ]]; then
    (cd "${path}" && pwd -P)
  else
    echo "${path}"
  fi
}

worktree_is_dirty() {
  local wt="$1"
  [[ -n "$(git -C "${wt}" status --porcelain 2>/dev/null)" ]]
}

branch_from_worktree() {
  local wt="$1"
  git -C "${wt}" rev-parse --abbrev-ref HEAD 2>/dev/null || true
}

collect_harness_worktrees() {
  git -C "${PROJECT_ROOT}" worktree list --porcelain | awk -v root="${PROJECT_ROOT}" '
    $1 == "worktree" {
      path = $2
      if (path != root) print path
    }
  '
}

HARNESS_WORKTREES=()
while IFS= read -r wt; do
  [[ -n "${wt}" ]] || continue
  wt_canonical="$(canonical_path "${wt}")"
  [[ "${wt_canonical}" == "${HARNESS_PREFIX}"* ]] || continue
  HARNESS_WORKTREES+=("${wt}")
done < <(collect_harness_worktrees)

for wt in ${HARNESS_WORKTREES[@]+"${HARNESS_WORKTREES[@]}"}; do
  [[ -n "${wt}" ]] || continue
  wt_canonical="$(canonical_path "${wt}")"
  if [[ "${CURRENT_DIR}" == "${wt_canonical}" || "${CURRENT_DIR}" == "${wt_canonical}/"* ]]; then
    die "refusing to reap while cwd is inside harness worktree: ${wt}"
  fi
done

branches_to_delete=()

for wt in ${HARNESS_WORKTREES[@]+"${HARNESS_WORKTREES[@]}"}; do
  [[ -n "${wt}" ]] || continue

  branch="$(branch_from_worktree "${wt}")"
  if worktree_is_dirty "${wt}"; then
    if [[ "${FORCE}" -eq 1 ]]; then
      echo "reap-worktrees: removing dirty worktree (forced): ${wt}" >&2
      git -C "${PROJECT_ROOT}" worktree remove --force "${wt}"
      if [[ "${branch}" == task/* ]]; then
        branches_to_delete+=("${branch}")
      fi
    else
      echo "reap-worktrees: skipping dirty worktree: ${wt}" >&2
    fi
    continue
  fi

  git -C "${PROJECT_ROOT}" worktree remove "${wt}"
  if [[ "${branch}" == task/* ]]; then
    branches_to_delete+=("${branch}")
  fi
done

for branch in ${branches_to_delete[@]+"${branches_to_delete[@]}"}; do
  [[ -n "${branch}" ]] || continue
  if git -C "${PROJECT_ROOT}" show-ref --verify --quiet "refs/heads/${branch}"; then
    git -C "${PROJECT_ROOT}" branch -D "${branch}"
  fi
done

git -C "${PROJECT_ROOT}" worktree prune
