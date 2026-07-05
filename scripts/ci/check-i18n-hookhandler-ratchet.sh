#!/usr/bin/env bash
# Phase 105.2: i18n contract ratchet for go/internal/hookhandler.
#
# The spec i18n contract requires user-facing runtime strings to go through the
# locale resolver (localizedHarnessMessage / resolveHarnessLocale), with English
# as the default and Japanese opt-in. A large body of legacy Japanese literals
# predates the contract; migrating all of them is tracked as a follow-up.
#
# This gate is a RATCHET: it counts "bare" Japanese string literals — Japanese
# text in a string literal on a line that does NOT go through localizedHarnessMessage
# and is NOT a comment — and fails if that count rises above the recorded baseline.
# Migrating a file lowers the count (then lower BASELINE here); adding a new bare
# Japanese user-facing string raises it and fails the gate.
#
# Exit 0 if count <= baseline, exit 1 if it exceeds the baseline.
# Note: no `set -e` — grep returning 1 (no match) in the counting pipeline is
# expected and must not abort the script.
set -uo pipefail

ROOT="${1:-$(git rev-parse --show-toplevel 2>/dev/null || pwd)}"
HANDLER_DIR="$ROOT/go/internal/hookhandler"

# Baseline recorded 2026-07-06 after migrating inbox_check.go. Only lower this
# number as more files are migrated; never raise it to accommodate new debt.
BASELINE=117

if [ ! -d "$HANDLER_DIR" ]; then
  echo "i18n ratchet: hookhandler dir not found ($HANDLER_DIR) — skipping (not-configured)"
  exit 0
fi

# Count Japanese string-literal lines that are NOT inside a localizedHarnessMessage(...)
# call span and are not comments. awk tracks paren depth so that ja arguments on
# their own line inside a multi-line localizer call are correctly excluded.
count=0
for f in "$HANDLER_DIR"/*.go; do
  case "$f" in *_test.go) continue ;; esac
  [ -f "$f" ] || continue
  n=$(awk '
    # Maintain paren depth of an active localizedHarnessMessage(...) call.
    { line = $0 }
    # Strip line comments so // 日本語 does not count.
    { sub(/\/\/.*/, "", line) }
    {
      inCall = (depth > 0)
      # If this line opens a localizer call, everything from here is localized.
      if (line ~ /localizedHarnessMessage[[:space:]]*\(/) { opensCall = 1 } else { opensCall = 0 }
      # A Japanese literal outside any localizer call is a bare (unmigrated) string.
      if (!inCall && !opensCall && line ~ /"[^"]*[ぁ-んァ-ヶ一-龠][^"]*"/) { bare++ }
      # Update paren depth across the line for multi-line localizer spans.
      if (opensCall || inCall) {
        for (i = 1; i <= length(line); i++) {
          c = substr(line, i, 1)
          if (c == "(") depth++
          else if (c == ")") { if (depth > 0) depth-- }
        }
      }
    }
    END { print bare + 0 }
  ' "$f")
  count=$((count + n))
done

if [ "$count" -gt "$BASELINE" ]; then
  echo "❌ i18n ratchet: bare Japanese literals rose to $count (baseline $BASELINE)."
  echo "   New user-facing strings must use localizedHarnessMessage(locale, en, ja)."
  exit 1
fi

if [ "$count" -lt "$BASELINE" ]; then
  echo "✅ i18n ratchet: bare Japanese literals down to $count (baseline $BASELINE) — lower BASELINE in this script."
else
  echo "✅ i18n ratchet: bare Japanese literals at baseline ($count)."
fi
exit 0
