# Claude host livemsg delivery (Mode 2)

Turn-boundary delivery for directed live messages uses the **Stop** hook in
`hooks/hooks.json` and `.claude-plugin/hooks.json` (P29 dual-sync).

## Turn delivery (default ON)

At each turn boundary, the Stop hook runs:

```text
bin/harness inbox check --team "${HARNESS_LIVEMSG_TEAM:-default}"
```

The recipient **agent id** is resolved from (in order):

1. `--agent` flag (explicit CLI / generated hooks)
2. `HARNESS_LIVEMSG_AGENT` environment variable
3. Stop hook stdin JSON `session_id` (Claude host)

When there are **zero unread** messages, the command produces **no stdout**
(silent hook). Trust contract (sanitize, non-instruction disclaimer, 4096B cap)
is applied in `go/cmd/harness/inbox_check.go` before any inject payload is emitted.

## Live monitor (~5s poll) — opt-in, default OFF

`bin/harness inbox monitor` is **not** wired in the tracked Claude hooks.
Enable it manually (for example in `.claude/settings.local.json`) only when
you want a blocking SessionStart monitor stream. It remains on the self-audit
allowlist (`CCHKnownHooks`) when injected locally.
