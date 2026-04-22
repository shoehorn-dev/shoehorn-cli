# Manifest Tools

## `validate`

Validate Shoehorn or Backstage manifest files. Format is auto-detected.

```bash
shoehorn validate catalog-info.yaml
shoehorn validate .shoehorn/service.yml --format json
cat catalog-info.yaml | shoehorn validate -
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `text` | Output format: `text` or `json` |

The command calls the Shoehorn validation API, so you must be authenticated (`shoehorn auth login`). Use `--profile` to validate against a specific environment. Input (file or stdin) is capped at 10 MB. Returns a non-zero exit code on validation failure.

## `validate mold`

Validate a Forge mold YAML definition **offline** — no server is contacted. Useful in pre-commit hooks and pull-request CI, including from forks where API tokens aren't available.

```bash
shoehorn validate mold .shoehorn/molds/create-repo.yaml
shoehorn validate mold my-mold.yaml --format json
shoehorn validate mold my-mold.yaml --strict
cat my-mold.yaml | shoehorn validate mold -
```

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `text` | Output format: `text` or `json` |
| `--strict` | `false` | Fail if any warnings are reported |

**Checks (errors):**

- YAML parses cleanly
- Required top-level fields: `version`, `metadata.name`
- At least one of `steps` or `actions` is defined
- Each step has `id` and `name`, and exactly one of `action` or `adapter`
- No duplicate step IDs across `steps` and `rollback.steps`
- Action IDs are at least two dot-separated parts (e.g. `github.repo.create`) and the provider is one of `github`, `deployment`, `system`, `catalog`, `repo`
- Adapter names are one of: `http`, `postgres`, `slack`, `github`, `docker`, `kubernetes`, `terraform`, `webhook`, `email`, `s3`, `gcs`, `file`, `git`, `catalog`, `log`
- Approval flow: `auto_approve_after` ≥ 3600s (1 hour), max 10 approval steps, each step has a name and 1–50 approvers

**Checks (warnings — only fail with `--strict`):**

- Missing recommended `metadata.displayName`, `metadata.description`, or `metadata.category`
- Action IDs with a valid provider that aren't on the known built-in actions list (treated as custom actions)

See the full validator reference in [docs.shoehorn.dev → CLI → Validator](https://docs.shoehorn.dev/cli/validator/).

## `convert`

Convert between Backstage and Shoehorn formats. Supports files, directories, and stdin.

```bash
# Backstage to Shoehorn
shoehorn convert catalog-info.yaml

# From stdin (pipe)
cat catalog-info.yaml | shoehorn convert - --to shoehorn

# Shoehorn to Backstage
shoehorn convert .shoehorn/my-service.yml --to backstage

# Backstage Template to Shoehorn Mold
shoehorn convert template.yaml --to mold -o mold.json

# Recursive directory conversion
shoehorn convert ./manifests -r

# Validate during conversion
shoehorn convert catalog-info.yaml --validate
```
