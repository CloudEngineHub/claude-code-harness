#!/usr/bin/env bats
# Phase 98.1.3 — judgment-ledger.sh bats tests (append / search / recall)

setup() {
  ROOT="$(cd "$(dirname "${BATS_TEST_FILENAME}")/.." && pwd)"
  SCRIPT="${ROOT}/scripts/judgment-ledger.sh"
  TMP="$(mktemp -d "${TMPDIR:-/tmp}/judgment-ledger-bats.XXXXXX")"
  export HARNESS_JUDGMENT_LEDGER="${TMP}/ledger.jsonl"
}

teardown() {
  rm -rf "${TMP}"
}

@test "append writes one schema-valid JSONL line" {
  run bash "${SCRIPT}" append \
    --project "demo" \
    --question "Redis or Postgres?" \
    --answer "redis" \
    --rationale "scale" \
    --card-ref "/tmp/card.json" \
    --tags "judgment-card" \
    --id "bats-001"
  [ "$status" -eq 0 ]
  grep -q '"id":"bats-001"' "${HARNESS_JUDGMENT_LEDGER}"
}

@test "append is fail-open when ledger path is unwritable" {
  blocker="${TMP}/blockfile"
  : >"${blocker}"
  export HARNESS_JUDGMENT_LEDGER="${blocker}/sub/ledger.jsonl"
  run bash "${SCRIPT}" append \
    --project "demo" \
    --question "q" \
    --answer "a" \
    --card-ref "c.json"
  [ "$status" -eq 0 ]
  [[ "$output" == *"append skipped"* ]]
}

@test "search is project-scoped and capped at 3" {
  export HARNESS_JUDGMENT_LEDGER="${TMP}/search.jsonl"
  for i in 1 2 3 4; do
    bash "${SCRIPT}" append \
      --project "p" \
      --question "redis topic ${i}" \
      --answer "a${i}" \
      --card-ref "c${i}.json" \
      --id "s${i}" >/dev/null
  done
  bash "${SCRIPT}" append \
    --project "other" \
    --question "redis elsewhere" \
    --answer "x" \
    --card-ref "x.json" \
    --id "other" >/dev/null

  run bash "${SCRIPT}" search --project "p" --query "redis"
  [ "$status" -eq 0 ]
  [ "$(printf '%s\n' "${lines[@]}" | sed '/^$/d' | wc -l | tr -d ' ')" -eq 3 ]
}

@test "recall emits similar_past_decisions array JSON" {
  export HARNESS_JUDGMENT_LEDGER="${TMP}/recall.jsonl"
  bash "${SCRIPT}" append \
    --project "p" \
    --question "Use Redis for sessions?" \
    --answer "yes" \
    --rationale "latency" \
    --card-ref "c.json" \
    --id "r1" >/dev/null

  run bash "${SCRIPT}" recall --project "p" --question "Redis session"
  [ "$status" -eq 0 ]
  python3 - <<'PY' "${output}"
import json, sys
data = json.loads(sys.argv[1])
assert isinstance(data, list) and len(data) == 1
assert data[0]["mem_id"].startswith("judgment-ledger:")
PY
}
