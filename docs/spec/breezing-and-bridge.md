# Spec Sub-Spec: breezing-and-bridge

This sub-spec is part of the `spec.md` product contract. SSOT order is `spec.md` core > `docs/spec/*` sub-specs > `Plans.md`.


`HARNESS_AUTO_APPROVE=on` (strict literal `on` only; `true` / `1` / `ON` /
`yes` are rejected) is currently ledger instrumentation only. It records the
enablement gate and prereq check result; it does not skip approval prompts or
wire any approval-skip branch. The retained scope predicate
`autoapprove.AppliesTo(path, worktreeRoot)` returns false for any path outside
the worktree root, so the 5-category runtime floor and `wt fingerprint`
containment continue to adjudicate worktree-external attempts and remain
non-overridable by `HARNESS_AUTO_APPROVE`. The switch is gated on three
prereqs being demonstrably done (Phase 92.1.1 parallel base hygiene, 92.2.3
team dispatch hardening, 96.1.2 3-CLI hook parity); if any prereq is missing
the gate fail-safes to OFF regardless of the env value. The gate decision is
recorded on every `harness work --team` dispatch through the orchestration
ledger. Approval automation will be wired only after HOTL governance
verification (Phase 101 U0-U7 evidence).

## Breezing Brief Contract

`/breezing` may accept free-text input that does not match the existing
argument-hint surface (`all`, task ranges, `--codex`, `--cursor`,
`--reviewer-only`, `--parallel N`, `--no-commit`, `--no-discuss`, `--auto-mode`).
This contract productizes the Brief Composer and Decision Card schemas, breezing
mem write layer, and their precedence against the runtime hard floor. It builds
on Tri-Tool Parallel Collaboration Contract (team orchestrator, worktree
discipline) and Memory Contract (harness-mem when available); it does not
replace them.

### Input classification and brief-card.v1

- `scripts/breezing-brief.sh classify "<args>"` deterministically classifies
  input as `structured` or `free-text` (regex/token parse; no LLM).
- `structured` input continues on the existing team path unchanged.
- `free-text` input is decomposed by the Lead into a `brief-card.v1` card for
  user confirmation before any worktree dispatch.
- User confirmation is Yes/No. `scripts/breezing-brief.sh confirm no` emits
  `DISPATCH: 0` — zero worker executions (dry contract). Yes dispatches one
  worker per confirmed subtask onto the existing worktree-per-task team path.
- `scripts/breezing-brief.sh validate <card.json>` validates against
  `templates/schemas/brief-card.v1.json`.

Schema `brief-card.v1` required fields:

| Field | Shape | Constraints |
|-------|-------|-------------|
| `goal` | string | non-empty |
| `subtasks` | array | 3–7 items; each item `{id, title, dod}` (all non-empty strings) |
| `scope_files` | string[] | repo-relative paths in scope |
| `risk_notes` | string[] | free-form risk strings |
| `confidence` | enum | `high` \| `medium` \| `low` |

No additional properties are permitted on the card root or subtask objects.

### judgment-card.v1

When a worker or Lead needs human judgment during breezing — and the runtime
hard floor does not apply — Harness may issue one `judgment-card.v1` card.
Schema: `templates/schemas/judgment-card.v1.json`.

Issuance conditions (any one triggers a card):

1. DoD interpretation diverges (multiple valid readings of done-ness).
2. A change outside the user-approved scope is required.
3. A trade-off choice is required (mutually exclusive options).

Required fields:

| Field | Shape | Constraints |
|-------|-------|-------------|
| `question` | string | non-empty |
| `options` | array | 2–3 items; each `{id, label, consequence}` (all non-empty strings) |
| `recommendation` | string | non-empty |
| `confidence` | enum | `high` \| `medium` \| `low` |
| `impact` | string | non-empty |
| `diff_summary` | string | non-empty one-line diff summary |

No additional properties are permitted on the card root or option objects.
User answer and rationale may be recorded via `harness_mem_record_checkpoint`
when harness-mem is available (fail-open; see below).

### Breezing mem lifecycle events

`go/internal/breezingmem` posts breezing lifecycle events to harness-mem
(`POST /v1/events/record`) and ingests the confirmed brief
(`POST /v1/ingest/knowledge-file` at path `breezing/brief-card.v1.json`,
kind `decisions_md`). Event type literals (fixed):

| Event type | When |
|------------|------|
| `breezing_run_started` | Breezing team run begins |
| `breezing_brief_confirmed` | User confirms the brief card (Yes) |
| `breezing_worker_result` | One worker completes (success or failure) |
| `breezing_aggregation_completed` | Lead aggregation finishes |

This layer does not call workgraph signal APIs (`signal_send`, `signal_read`,
`signal_ack`, or `/v1/signals/*`). Durable cross-session handoff remains in
harness-mem signal store per Tri-Tool Parallel Collaboration Contract; breezing
mem events are run-scoped lifecycle telemetry only.

### Fail-open memory behavior

Breezing must never stop because harness-mem is absent or unreachable.

| State | Behavior |
|-------|----------|
| `not-configured` | Silent skip — no warning, no POST |
| `unreachable` | One stderr warning line (`breezing-mem: record skipped (unreachable)` or ingest equivalent), then continue |

Configured means `~/.harness-mem` or legacy `~/.claude-mem` exists (see
`go/internal/breezingmem` `configured()`). HTTP timeout is 1s. Mem state never
blocks brief confirmation, worker dispatch, judgment cards, or aggregation.

### Floor precedence over judgment cards

The five-category **runtime action hard floor** (Phase 92.2.1; money/billing,
external send/egress, credential/secret read, production deploy/publish,
destruction outside the task worktree) always takes precedence over
`judgment-card.v1`:

- When any hard-floor category matches, Harness hard-stops for the human and
  does **not** issue a judgment card.
- Judgment cards apply only to non-floor ambiguity (DoD, scope, trade-offs).
- The pre-merge policy gate (`go/internal/floor`) and runtime hard floor remain
  distinct; see Tri-Tool Parallel Collaboration Contract.

## Bridge Daemon Contract

The Bridge Daemon normalizes events from three CLI backends (CC Mailbox, Cursor
stop hook, Codex app-server) into one source-agnostic envelope, persists them in
a unified mailbox store, optionally ingests into harness-mem by lane, and
delivers notices to non-CC peers through host hooks. It builds on Tri-Tool
Parallel Collaboration Contract (Mode 2 live notice) and Breezing Brief Contract
(fail-open memory); it does not replace durable work-handoff in harness-mem
signal store.

### bridge-event.v1

Schema: `templates/schemas/bridge-event.v1.json`. Every normalized event uses
this envelope; no additional root properties are permitted.

| Field | Shape | Constraints |
|-------|-------|-------------|
| `source` | enum | `cc` \| `cursor` \| `codex` |
| `event_type` | string | non-empty |
| `payload` | object | source-specific fields after normalization |
| `ts` | integer | Unix nanoseconds; required (missing timestamp is fail-loud) |

Go reference: `go/internal/bridge.Event` (`Source`, `EventType`, `Payload`, `TS`).

### Source adapter contract

Three adapters implement `bridge.Adapter` (`Source()` + `Normalize(raw []byte)`).
Each maps one source-specific raw JSON input to exactly one `bridge-event.v1`
event (batching is caller responsibility).

| Adapter | Source | Raw input keys | Normalized `event_type` |
|---------|--------|----------------|-------------------------|
| CC Mailbox | `cc` | `hook_event_name`, `timestamp` | value of `hook_event_name` |
| Cursor stop hook | `cursor` | `hook_event_name`, `ts` | fixed `stop` |
| Codex app-server | `codex` | `type`, `ts` | value of `type` |

Adapter rules:

- Missing or invalid `ts` / `timestamp` is **fail-loud** (`requireNanos` error);
  the event is not appended.
- Unregistered `source` on `bridge.Registry.Normalize` returns `(Event{}, false,
  nil)` — **fail-open skip**; the caller emits one warning line (the registry
  package does not write stdout).
- Payload fields are copied per adapter (`conversation_id`, `tool_name`,
  `session_id`, `message`, `thread_id`, etc.) with non-reserved keys merged
  into `payload`.

### Delivery layer

Notice delivery to peers uses host hooks (Phase 92.6.2 delivery-notice pattern,
extended for bridge daemon non-CC peers):

| Host | Delivery path |
|------|---------------|
| Claude Code | Monitor blocking stream (SessionStart hook) |
| Cursor 1.7+ | Stop hook `followup_message` |
| Codex | Bash `PreToolUse` hook coupling |

When delivery fails, Harness must **not** block the run: emit one warning line,
record the failure in the orchestration ledger, and **fallback on the next
turn** (turn-boundary inbox read). Codex and Cursor fall back to turn delivery
when Monitor is unavailable.

### Mailbox unified store

`go/internal/mailbox.Store` persists normalized events in SQLite WAL mode as an
**append-only event log** (`bridge_events` table). Each append selects a lane:

| Lane | Mem ingest |
|------|------------|
| `fast` | `Record` only |
| `gate` | `Record` + `Audit` |
| `release` | `Record` + `Alert` |

Ingest runs asynchronously after append; ingest errors are logged and swallowed
(fail-open). `MemIngestor` defaults to `NoopIngestor` when harness-mem is not
wired.

This store is distinct from Phase 92.6.1 `livemsg` (Mode 2 peer messaging) and
from harness-mem signal store (durable work-handoff).

### Fail-open memory behavior (bridge read/write)

Bridge and mailbox layers must never stop a breezing or bridge run because
harness-mem is absent or unreachable. Same tri-state model as Breezing Brief
Contract:

| State | Behavior |
|-------|----------|
| `not-configured` | Silent skip — no warning, no POST |
| `unreachable` | One stderr warning line, then continue |

Configured means `~/.harness-mem` or legacy `~/.claude-mem` exists (see
`go/internal/breezingmem` `configured()`). HTTP timeout is 1s.

### Channels-Wake

Phase 98.2 adds an opt-in recovery layer for Bridge Daemon communication
channels (live-messaging / delivery / inbox hooks). It monitors channel health
and may **propose** hook re-injection when unhealthy; it does **not** restart
daemons or auto-approve actions.

Configuration lives at `~/.harness-bridge/channels.json` (override with
`HARNESS_BRIDGE_HOME`). Required keys when enabled:

| Key | Purpose |
|-----|---------|
| `socket_path` | Unix domain socket for Bridge Daemon probe |
| `mailbox_db` | SQLite WAL mailbox store (`bridge_events`) |
| `stale_after_seconds` | Max age of newest mailbox event before `corrupted` |

Tri-state health (active-watching-test-policy, D40):

| State | `reason` | `healthy` | CLI exit | Session Monitor warning |
|-------|----------|-----------|----------|-------------------------|
| Opt-in not used | `not-configured` | true | 0 | **none** |
| Daemon down / socket unreachable | `daemon-unreachable` | false | 1 | yes |
| Invalid config / stale mailbox | `corrupted` | false | 1 | yes |
| OK | `""` | true | 0 | none |

CLI: `harness channels-wake check` emits JSON `{healthy, reason}`.
Implementation: `go/internal/channelswake.Check()`.

Wake trigger scope (Lead decision, Phase 98.2): **hook re-injection proposal
only**, opt-in via `HARNESS_CHANNELS_WAKE_OPT_IN=1` in
`scripts/channels-wake-probe.sh`. `AUTO_APPROVE_DEFAULT=false` is preserved.
Risk Gate 5-category runtime floor (`money-billing`, `egress`, `secret-read`,
`prod-deploy`, `worktree-escape`) remains non-overridable.

Event schema: `templates/schemas/channel-wake-event.v1.json`
(`channel-wake-event.v1`, `additionalProperties: false`).

### workgraph signal boundary

Bridge Daemon, mailbox ingest, and breezing mem lifecycle **must not** call
workgraph signal APIs (`signal_send`, `signal_read`, `signal_ack`, or
`/v1/signals/*`). Durable cross-session handoff remains in harness-mem signal
store; bridge events are run-scoped telemetry and live-notice input only.

