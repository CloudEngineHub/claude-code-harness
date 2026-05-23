# Antigravity CLI Adapter Boundary

Status: future/unsupported evidence boundary
Checked at: 2026-05-22 JST
Phase: `Plans.md` 73.1.8

## Conclusion

Antigravity CLI remains `future/unsupported`.

Official Google docs and search snippets suggest plugin, skills, hooks,
subagents, and migration routes. In this environment, direct extraction was not
stable enough and no local CLI smoke was observed.

## Evidence

Harness evidence:

- `spec.md`, `docs/tool-capability-matrix.md`, and
  `docs/bootstrap-routing-contract.md` keep Antigravity CLI as
  `future/unsupported`.
- `docs/onboarding/install.md` keeps Antigravity out of the end-user install
  flow.

Official/current docs checked:

- `https://antigravity.google/docs/cli-features`
- `https://antigravity.google/docs/plugins?app=antigravity`
- `https://www.antigravity.google/docs/hooks`
- `https://antigravity.google/docs/gcli-migration?app=antigravity`
- `https://www.antigravity.google/docs/cli-getting-started`

Observed from search/doc snippets: plugin bundles can include skills, rules, MCP
servers, hooks, and subagents; hooks include tool/model invocation concepts; a
Gemini CLI migration route is mentioned. Direct `web.open` / `curl` extraction
was unstable or SPA-shell only in this task.

Local availability:

- `command -v agy` returned no binary.
- `command -v antigravity` returned no binary.
- No install script was executed.

## Boundary

`not_observed != absent`: missing Antigravity CLI evidence does not prove
absence. It means Harness must not publish setup docs, README support wording,
or release claims for Antigravity.

Do not run `curl ... | bash`, `agy plugin import`, or home-mutating commands in
default evidence checks. Those require a separate explicit risk gate and
isolated environment.

## Safe Availability Commands

Future investigation may start with:

```bash
command -v agy || command -v antigravity
agy --version || antigravity --version
agy --help
agy plugin --help
agy plugin list
agy plugin import --help
test -d "$HOME/.gemini/antigravity-cli" && find "$HOME/.gemini/antigravity-cli" -maxdepth 3 -type f | head
```

Use temporary `HOME` / `XDG_CONFIG_HOME` for any command that might write state.

## Promotion Conditions

Antigravity CLI can move to `candidate` only after:

- official docs are captured with stable extractable evidence,
- local install availability is verified without destructive setup,
- plugin/skills/hooks or AGENTS/GEMINI bootstrap route is confirmed,
- a Harness-specific smoke plan exists,
- README/onboarding wording still avoids support claims.
