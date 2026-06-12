#!/bin/bash
# deny-baseline regression gate: repo settings must not shrink vs templates/security/deny-baseline.json
set -u
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGIN_ROOT="$(dirname "$SCRIPT_DIR")"

HARNESS_BIN="$(mktemp)"
trap 'rm -f "$HARNESS_BIN"' EXIT

if ! GO111MODULE=on go build -o "$HARNESS_BIN" "$PLUGIN_ROOT/go/cmd/harness" 2>/dev/null; then
    echo "FAIL: go build harness CLI"
    exit 1
fi

SETTINGS="$PLUGIN_ROOT/.claude-plugin/settings.json"
BASELINE="$PLUGIN_ROOT/templates/security/deny-baseline.json"

if [ ! -f "$SETTINGS" ]; then
    echo "FAIL: settings.json not found: $SETTINGS"
    exit 1
fi
if [ ! -f "$BASELINE" ]; then
    echo "FAIL: deny baseline not found: $BASELINE"
    exit 1
fi

if "$HARNESS_BIN" self-audit baseline --settings "$SETTINGS" --baseline "$BASELINE"; then
    echo "PASS: repo settings match deny baseline (no regression)"
else
    rc=$?
    echo "FAIL: deny baseline check exited $rc"
    exit "$rc"
fi

# tempdir fixture: remove one deny entry from current settings → exit 2
FIXTURE_DIR="$(mktemp -d)"
trap 'rm -rf "$FIXTURE_DIR" "$HARNESS_BIN"' EXIT

python3 - <<'PY' "$SETTINGS" "$FIXTURE_DIR/settings-trimmed.json"
import json, sys
settings_path, out_path = sys.argv[1], sys.argv[2]
with open(settings_path) as f:
    data = json.load(f)
deny = data.get("permissions", {}).get("deny", [])
if not deny:
    raise SystemExit("settings has empty deny list")
data["permissions"]["deny"] = deny[1:]
with open(out_path, "w") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PY

if "$HARNESS_BIN" self-audit baseline --settings "$FIXTURE_DIR/settings-trimmed.json" --baseline "$BASELINE"; then
    echo "FAIL: trimmed settings should exit 2 (deny regression)"
    exit 1
fi
trim_rc=$?
if [ "$trim_rc" -ne 2 ]; then
    echo "FAIL: trimmed settings exit = $trim_rc, want 2"
    exit 1
fi

echo "PASS: deny regression fixture exits 2 as expected"
exit 0
