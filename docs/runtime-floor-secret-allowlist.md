# Runtime Floor Secret-Read Allowlist

The Runtime Floor treats secret reads the same way it treats network egress:
explicit allowlist first, default deny. The egress side uses `isAllowlistedHost`
to require named hosts before outbound calls; the secret-read side requires named
project-local paths before a pipeline may read files that contain credentials,
tokens, keys, or other operator-provided secrets.

## Contract

- Declare only the specific file paths a run needs. A declaration relaxes access
  for those paths only; it does not open the project, parent directories, or the
  whole filesystem.
- Empty strings and a bare `*` are invalid. Treat any all-open style declaration
  as deny, not as a wildcard.
- The effective allowlist is the union of:
  - `HARNESS_RUNTIME_FLOOR_SECRET_ALLOW`
  - `.claude-code-harness.config.json` `runtimefloor.secretAllow`
- If project config is missing, unreadable, malformed, or `runtimefloor.secretAllow`
  is not a string array, the config contribution is fail-safe empty. Secret reads
  remain denied unless the environment declaration provides a valid path.
- Relative paths resolve under the project root. Absolute paths outside the
  project root are invalid and ignored.

## Operator Flow

Before starting work, list the secret files the task will need and declare them
once in project config or in the environment. After that, the run should not stop
mid-task for repeated secret-read approvals, because the Runtime Floor can decide
from the predeclared contract.

Use project config when the same pipeline needs the same secret files across
runs:

```json
{
  "$schema": "./claude-code-harness.config.schema.json",
  "runtimefloor": {
    "secretAllow": [
      ".env.local",
      "secrets/pipeline.key"
    ]
  }
}
```

Use the environment for one-off CI or local runs. Separate multiple paths with
commas:

```bash
export HARNESS_RUNTIME_FLOOR_SECRET_ALLOW=".env.local,secrets/pipeline.key"
```

The two sources are additive. For example, if project config declares
`.env.local` and CI exports `secrets/pipeline.key`, both paths are allowed for
that run.

## Pipeline Example

Declare the secrets before invoking the pipeline:

```bash
export HARNESS_RUNTIME_FLOOR_SECRET_ALLOW="secrets/deploy-token,config/private.env"
bash scripts/pipeline/deploy-preview.sh
```

Inside the pipeline, keep the read path identical to the declaration:

```bash
DEPLOY_TOKEN="$(cat secrets/deploy-token)"
set -a
. config/private.env
set +a
```

Prefer project-relative paths in both the declaration and the pipeline. Avoid
absolute paths; an absolute path outside the project root is ignored by contract,
so it will still be denied.

## Bad Declarations

These examples must not grant access:

```bash
export HARNESS_RUNTIME_FLOOR_SECRET_ALLOW=""
export HARNESS_RUNTIME_FLOOR_SECRET_ALLOW="*"
```

```json
{
  "runtimefloor": {
    "secretAllow": ["*", "/Users/alice/.ssh/id_rsa"]
  }
}
```

The first two are all-open or empty declarations. The JSON example combines a
bare wildcard with an absolute path outside the project. Both entries are
invalid, so the effective project-config contribution is empty.
