#!/bin/bash
# Fail when Go files are not formatted with gofmt.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

cd "$PROJECT_ROOT"

command -v gofmt >/dev/null 2>&1 || fail "gofmt is not available"

unformatted="$(gofmt -l go)"

if [ -n "$unformatted" ]; then
  {
    echo "FAIL: Go files need gofmt:"
    echo "$unformatted"
  } >&2
  exit 1
fi

echo "PASS: gofmt -l go is empty"
