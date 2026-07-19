#!/usr/bin/env bash
# release-preflight-host-smoke.sh
# Run host workflow smoke for all dist hosts with REQUIRED=1 (fail-closed).
#
# Usage: bash scripts/release-preflight-host-smoke.sh
#
# Test-only: set HARNESS_PREFLIGHT_HOST_SMOKE_CMD to override the per-host smoke
# command (invoked as $HARNESS_PREFLIGHT_HOST_SMOKE_CMD "$host" with REQUIRED env).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# shellcheck source=lib/host-registry.sh
source "${SCRIPT_DIR}/lib/host-registry.sh"
HOST_REGISTRY_PATH="${ROOT_DIR}/hosts/registry.json"

pass_count=0
total_count=0
any_failed=0

while IFS= read -r h; do
  [ -n "$h" ] || continue
  total_count=$((total_count + 1))
  # Per-host fail-closed: HARNESS_<HOST>_WORKFLOW_SMOKE_REQUIRED=1
  required_var="HARNESS_$(printf '%s' "$h" | tr '[:lower:]' '[:upper:]')_WORKFLOW_SMOKE_REQUIRED"
  export "${required_var}=1"

  if [ -n "${HARNESS_PREFLIGHT_HOST_SMOKE_CMD:-}" ]; then
    if "${HARNESS_PREFLIGHT_HOST_SMOKE_CMD}" "$h"; then
      echo "host-smoke ${h}: PASS"
      pass_count=$((pass_count + 1))
    else
      echo "host-smoke ${h}: FAIL"
      any_failed=1
    fi
  elif bash "${ROOT_DIR}/tests/test-host-workflow-smoke.sh" --host "$h"; then
    echo "host-smoke ${h}: PASS"
    pass_count=$((pass_count + 1))
  else
    echo "host-smoke ${h}: FAIL"
    any_failed=1
  fi
done < <(host_registry_dist_hosts)

if [ "$total_count" -eq 0 ]; then
  echo "host-smoke summary: 0/0 pass" >&2
  exit 1
fi

echo "host-smoke summary: ${pass_count}/${total_count} pass"

if [ "$any_failed" -ne 0 ]; then
  exit 1
fi

exit 0
