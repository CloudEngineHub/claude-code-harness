#!/usr/bin/env bash
# Task 111.7.6 — release preflight host workflow smoke contract (H7 route A)
#
# Validates:
#   (a) scripts/release-preflight-host-smoke.sh exists and is invocable
#   (b) registry SSOT + REQUIRED=1 literals in host smoke script
#   (c) release-preflight.sh defines and invokes check_host_workflow_smoke
#   (d) harness-release skill documents the host smoke preflight path
#   (e) fail-closed behavioral fixture via HARNESS_PREFLIGHT_HOST_SMOKE_CMD stub
#   (f) validate-plugin.sh wires this test (via REQUIRED_INVOCATIONS pin)
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOST_SMOKE_SCRIPT="${ROOT_DIR}/scripts/release-preflight-host-smoke.sh"
PREFLIGHT_SCRIPT="${ROOT_DIR}/scripts/release-preflight.sh"
SKILL_MD="${ROOT_DIR}/skills/harness-release/SKILL.md"
VALIDATE_PLUGIN="${ROOT_DIR}/tests/validate-plugin.sh"

# shellcheck source=../scripts/lib/host-registry.sh
source "${ROOT_DIR}/scripts/lib/host-registry.sh"
HOST_REGISTRY_PATH="${ROOT_DIR}/hosts/registry.json"

fail() {
  echo "test-release-preflight-host-smoke: FAIL: $1" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local needle="$2"
  grep -Fq "$needle" "$file" || fail "missing '${needle}' in ${file}"
}

assert_file_contains_sequence() {
  local file="$1"
  local before="$2"
  local after="$3"
  awk -v before="$before" -v after="$after" '
    BEGIN { seen_before = 0; seen_after = 0; before_line = 0; after_line = 0 }
    index($0, before) { seen_before = 1; before_line = NR }
    index($0, after) { seen_after = 1; after_line = NR }
    END {
      if (!seen_before) exit 1
      if (!seen_after) exit 2
      if (before_line >= after_line) exit 3
      exit 0
    }
  ' "$file" || fail "expected '${before}' before '${after}' in ${file}"
}

# ---- (a) existence ----

if [ ! -f "$HOST_SMOKE_SCRIPT" ]; then
  fail "missing ${HOST_SMOKE_SCRIPT}"
fi
if [ ! -x "$HOST_SMOKE_SCRIPT" ]; then
  bash -n "$HOST_SMOKE_SCRIPT" 2>/dev/null || fail "host smoke script not invocable via bash"
fi

# ---- (b) registry SSOT + REQUIRED literal ----

assert_contains "$HOST_SMOKE_SCRIPT" "_WORKFLOW_SMOKE_REQUIRED=1"
assert_contains "$HOST_SMOKE_SCRIPT" "host_registry_dist_hosts"

# ---- (c) release-preflight wiring ----

assert_contains "$PREFLIGHT_SCRIPT" "check_host_workflow_smoke"
assert_file_contains_sequence "$PREFLIGHT_SCRIPT" "check_release_mirror_drift" "check_host_workflow_smoke"
assert_file_contains_sequence "$PREFLIGHT_SCRIPT" "check_host_workflow_smoke" "check_ci_status"

# ---- (d) skill documentation ----

assert_contains "$SKILL_MD" "release-preflight-host-smoke.sh"

# ---- (e) behavioral fail-closed fixture ----

first_host=""
while IFS= read -r h; do
  if [ -n "$h" ]; then
    first_host="$h"
    break
  fi
done < <(host_registry_dist_hosts)

[ -n "$first_host" ] || fail "registry returned no dist hosts for fixture"

stub_fail="$(mktemp "${TMPDIR:-/tmp}/preflight-host-smoke-fail.XXXXXX")"
stub_pass="$(mktemp "${TMPDIR:-/tmp}/preflight-host-smoke-pass.XXXXXX")"
trap 'rm -f "$stub_fail" "$stub_pass"' EXIT

cat >"$stub_fail" <<STUB
#!/usr/bin/env bash
set -euo pipefail
if [ "\$1" = "${first_host}" ]; then
  exit 1
fi
exit 0
STUB
chmod +x "$stub_fail"

cat >"$stub_pass" <<'STUB'
#!/usr/bin/env bash
set -euo pipefail
exit 0
STUB
chmod +x "$stub_pass"

dist_count=0
while IFS= read -r _h; do
  [ -n "$_h" ] && dist_count=$((dist_count + 1))
done < <(host_registry_dist_hosts)
[ "$dist_count" -gt 0 ] || fail "dist host count is zero"

fail_output="$(mktemp "${TMPDIR:-/tmp}/preflight-host-smoke-out.XXXXXX")"
if GITHUB_ACTIONS= HARNESS_PREFLIGHT_HOST_SMOKE_CMD="$stub_fail" bash "$HOST_SMOKE_SCRIPT" >"$fail_output" 2>&1; then
  fail "expected non-zero exit when one host stub fails"
fi
grep -Fq "FAIL" "$fail_output" || fail "expected FAIL in output when stub fails"
grep -Fq "host-smoke ${first_host}: FAIL" "$fail_output" \
  || fail "expected host-smoke ${first_host}: FAIL in output"
rm -f "$fail_output"

pass_output="$(mktemp "${TMPDIR:-/tmp}/preflight-host-smoke-out.XXXXXX")"
if ! GITHUB_ACTIONS= HARNESS_PREFLIGHT_HOST_SMOKE_CMD="$stub_pass" bash "$HOST_SMOKE_SCRIPT" >"$pass_output" 2>&1; then
  cat "$pass_output" >&2
  fail "expected exit 0 when all hosts pass via stub"
fi
grep -Fq "host-smoke summary: ${dist_count}/${dist_count} pass" "$pass_output" \
  || fail "expected summary ${dist_count}/${dist_count} pass (got: $(tail -n 1 "$pass_output"))"
rm -f "$pass_output"

# ---- (g) runner scope: GITHUB_ACTIONS=true + missing CLI -> visible SKIP, exit 0 ----
# The fail-closed consumer is the operator-machine preflight; a GitHub runner
# without host CLIs must skip loudly instead of blocking every release
# (regression fixture for the v5.3.0 run 29679591686 failure).

skip_output="$(mktemp "${TMPDIR:-/tmp}/preflight-host-smoke-out.XXXXXX")"
if ! GITHUB_ACTIONS=true HARNESS_PREFLIGHT_HOST_CLI_PROBE_CMD=/usr/bin/false \
    HARNESS_PREFLIGHT_HOST_SMOKE_CMD="$stub_fail" bash "$HOST_SMOKE_SCRIPT" >"$skip_output" 2>&1; then
  cat "$skip_output" >&2
  fail "expected exit 0 on runner when all host CLIs are absent (skip path)"
fi
grep -Fq "host-smoke ${first_host}: SKIP" "$skip_output" \
  || fail "expected SKIP line for ${first_host} on runner without CLI"
grep -Fq "(${dist_count} skipped on runner)" "$skip_output" \
  || fail "expected skip summary suffix (got: $(tail -n 1 "$skip_output"))"
rm -f "$skip_output"

# Runner with CLIs present (probe true) still runs the smoke normally.
runner_pass_output="$(mktemp "${TMPDIR:-/tmp}/preflight-host-smoke-out.XXXXXX")"
if ! GITHUB_ACTIONS=true HARNESS_PREFLIGHT_HOST_CLI_PROBE_CMD=/usr/bin/true \
    HARNESS_PREFLIGHT_HOST_SMOKE_CMD="$stub_pass" bash "$HOST_SMOKE_SCRIPT" >"$runner_pass_output" 2>&1; then
  cat "$runner_pass_output" >&2
  fail "expected exit 0 on runner when CLIs present and smoke passes"
fi
grep -Fq "host-smoke summary: ${dist_count}/${dist_count} pass" "$runner_pass_output" \
  || fail "expected full pass summary on runner with CLIs present"
rm -f "$runner_pass_output"

# Outside GITHUB_ACTIONS the probe must NOT enable skipping: fail stays fail.
local_fail_output="$(mktemp "${TMPDIR:-/tmp}/preflight-host-smoke-out.XXXXXX")"
if GITHUB_ACTIONS= HARNESS_PREFLIGHT_HOST_CLI_PROBE_CMD=/usr/bin/false \
    HARNESS_PREFLIGHT_HOST_SMOKE_CMD="$stub_fail" bash "$HOST_SMOKE_SCRIPT" >"$local_fail_output" 2>&1; then
  fail "expected non-zero exit locally even when probe reports missing CLIs"
fi
grep -Fq "host-smoke ${first_host}: FAIL" "$local_fail_output" \
  || fail "expected FAIL (not SKIP) locally for ${first_host}"
rm -f "$local_fail_output"

# ---- (f) validate-plugin wiring ----

assert_contains "$VALIDATE_PLUGIN" "tests/test-release-preflight-host-smoke.sh"

echo "test-release-preflight-host-smoke: ok"
