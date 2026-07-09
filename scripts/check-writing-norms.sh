#!/usr/bin/env bash
# check-writing-norms.sh — HOTL Phase 101 U6 (Plans.md 101.7) pilot gate.
#
# Deterministic prose gate: scans the Japanese opt-in public doc surface for the
# §7 "LLM-ish banned phrase" deterministic subset of the writing-norms standard
# (k16shikano). Any hit → exit 1. This is the first end-to-end Authority
# Provenance Graph instance: rule (§7 banned phrases) ↔ check (this scanner) ↔
# execution (exit code). LLM-advisory rules, hedges and the em-dash preference
# are intentionally NOT gated (see spec.md §HOTL Governance Contract).
#
# Scope: JP opt-in surface only. EN default docs are never scanned.
# Usage:
#   scripts/check-writing-norms.sh [file ...]
# With no args it scans the default JP opt-in surface (README_ja.md).
set -euo pipefail

# §7 deterministic banned-phrase subset (~14 literals).
BANNED=(
  "重要なのは"
  "本章では"
  "掘り下げる"
  "深掘り"
  "正面から"
  "に他ならない"
  "多角的"
  "包括的"
  "総合的"
  "不可欠"
  "核心的"
  "鍵となる"
  "根本的"
  "言語化"
)

repo_root() {
  if git rev-parse --show-toplevel >/dev/null 2>&1; then
    git rev-parse --show-toplevel
  else
    cd "$(dirname "$0")/.." && pwd
  fi
}

# Resolve target files. Args override the default JP opt-in surface.
if [ "$#" -gt 0 ]; then
  FILES=("$@")
else
  ROOT="$(repo_root)"
  FILES=("$ROOT/README_ja.md")
fi

hits=0
for f in "${FILES[@]}"; do
  [ -f "$f" ] || continue   # missing optional surface file is not a failure
  for phrase in "${BANNED[@]}"; do
    while IFS= read -r line; do
      [ -n "$line" ] || continue
      echo "VIOLATION: ${f}:${line} contains banned phrase \"${phrase}\""
      hits=$((hits + 1))
    done < <(grep -Fn -- "$phrase" "$f" | cut -d: -f1)
  done
done

if [ "$hits" -gt 0 ]; then
  echo "writing-norms gate: FAIL (${hits} banned-phrase hit(s) on JP opt-in surface)"
  exit 1
fi

echo "writing-norms gate: ok (0 banned-phrase hits on JP opt-in surface)"
exit 0
