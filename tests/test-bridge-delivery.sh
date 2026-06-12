#!/bin/bash
# Phase 95.1.3 — bridge delivery layer (Go unit tests + Cursor stop followup shape).
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_DIR="${ROOT_DIR}/go"

echo "==> go test ./internal/bridgedelivery/..."
(
  cd "${GO_DIR}"
  go test ./internal/bridgedelivery/... -count=1
)

echo "==> validate Cursor stop followup JSON fixture shape"
python3 - <<'PY'
import json
from pathlib import Path

fixture = Path(__file__).resolve().parent / "fixtures" / "bridge-delivery" / "cursor-stop-followup.json"
doc = json.loads(fixture.read_text())

assert doc.get("type") == "stop", doc
followup = doc.get("followup_message")
assert isinstance(followup, dict), doc
assert followup.get("role") == "assistant", followup
assert isinstance(followup.get("content"), str) and followup["content"], followup
assert isinstance(doc.get("conversation_id"), str) and doc["conversation_id"], doc
assert isinstance(doc.get("ts"), int), doc

json.dumps(doc)
print("cursor stop followup shape ok")
PY

echo "PASS: test-bridge-delivery.sh"
