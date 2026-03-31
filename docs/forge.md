# Forge Workflows

## `forge molds list`

List all available Forge workflow templates.

```bash
shoehorn forge molds list
shoehorn forge molds list --output json
```

Also available as `shoehorn get molds`.

## `forge molds get`

Detail view for a mold: actions, inputs, and steps.

```bash
shoehorn forge molds get create-empty-github-repo
shoehorn forge molds get create-empty-github-repo --output json
```

Also available as `shoehorn get mold <name>`.

## `forge execute`

Execute a mold workflow in one step. Fetches the mold, resolves the action, fills defaults, validates required inputs, and creates a run.

```bash
shoehorn forge execute create-empty-github-repo \
  --input name=my-service \
  --input owner=my-org

# Specify a non-primary action
shoehorn forge execute my-mold --action scaffold --input name=my-app

# Validate without executing
shoehorn forge execute create-empty-github-repo \
  --input name=test --input owner=my-org --dry-run

# Pass inputs as JSON
shoehorn forge execute my-mold --inputs '{"name":"my-svc","owner":"my-org"}'
```

Flags:
- `--input` — repeatable `key=value` pairs (types are coerced from the mold schema)
- `--inputs` — JSON object with all inputs
- `--action` — action name (auto-selects primary action if omitted)
- `--dry-run` — validate without executing

## `forge run list`

List all workflow runs.

```bash
shoehorn forge run list
shoehorn forge run list --output json
```

Also available as `shoehorn get runs`.

## `forge run get`

Detail for a specific run.

```bash
shoehorn forge run get <run-id>
shoehorn forge run get <run-id> --output json
```

Also available as `shoehorn get run <run-id>`.

## `forge run watch`

Watch a run until it completes (polls every 2 seconds).

```bash
shoehorn forge run watch <run-id>
shoehorn forge run watch <run-id> --interval 5
shoehorn forge run watch <run-id> -o json
```

## `forge run create`

Start a new workflow run from a mold (lower-level than `forge execute`).

```bash
shoehorn forge run create create-empty-github-repo --action create \
  --input name=my-service --input owner=my-org
```
