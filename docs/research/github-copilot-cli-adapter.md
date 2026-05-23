# GitHub Copilot CLI Adapter Candidate

Status: candidate evidence boundary
Checked at: 2026-05-22 JST
Phase: `Plans.md` 73.1.7

## Conclusion

GitHub Copilot CLI remains `candidate`.

Official GitHub docs show that Copilot CLI has plugins, skills, hooks, custom
agents, and MCP configuration. That proves a possible adapter shape, not Harness
support.

## Evidence

Harness evidence:

- `spec.md`, `docs/tool-capability-matrix.md`, and
  `docs/bootstrap-routing-contract.md` keep GitHub Copilot CLI as `candidate`.
- README and onboarding do not present a Copilot install path.

Superpowers evidence:

- `/Users/tachibanashuuta/LocalWork/Code/superpowers/skills/using-superpowers/references/copilot-tools.md`
  maps Claude-style tools to Copilot CLI equivalents.
- Superpowers session-start logic has a Copilot CLI branch.

This is not Harness proof.

Official/current docs checked:

- `https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-plugin-reference`
- `https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-cli-plugins`
- `https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/overview`

Observed official capabilities include `copilot plugin install`, `uninstall`,
`list`, `update`, and marketplace commands. `plugin.json` can point to `agents`,
`skills`, `commands`, `hooks`, `mcpServers`, and related component paths.

Local availability:

- `command -v copilot` finds the Superset wrapper at
  `/Users/tachibanashuuta/.superset/bin/copilot`.
- Running `copilot --help` reports that the real `copilot` binary is not found
  in `PATH`.
- No isolated Harness CLI smoke passed in this task.

## Boundary

`not_observed != absent`: missing local CLI smoke is not proof that Copilot CLI
cannot support Harness. It is proof that Harness must keep Copilot as
`candidate`.

Manual instruction profile research is allowed. Plugin support, skill routing,
bootstrap support, hook safety, and release support are not claimed.

## Promotion Conditions

GitHub Copilot CLI can move beyond `candidate` only after:

- local real `copilot` binary availability is verified,
- an isolated home/cache smoke installs or loads a Harness-specific profile,
- bootstrap or skill routing evidence is captured,
- Superpowers evidence is kept as inspiration, not Harness proof,
- README/onboarding wording still avoids support claims.
