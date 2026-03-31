# Governance

## `governance actions list`

List governance action items with optional filters.

```bash
shoehorn governance actions list
shoehorn governance actions list --output json
```

## `governance actions get`

Get details for a specific governance action.

```bash
shoehorn governance actions get <id>
shoehorn governance actions get <id> --output json
```

## `governance actions create`

Create a new governance action. Requires `--entity-id`, `--title`, `--priority`, and `--source-type`.

```bash
shoehorn governance actions create \
  --entity-id "service:my-service" \
  --title "Update API documentation" \
  --priority high \
  --source-type policy \
  --description "API docs are outdated" \
  --assigned-to "alice@company.com" \
  --sla-days 14
```

## `governance actions update`

Update an existing governance action (status, priority, assignment, resolution).

```bash
shoehorn governance actions update <id> --status in_progress
shoehorn governance actions update <id> --priority critical --assigned-to "bob@company.com"
shoehorn governance actions update <id> --status resolved --resolution-note "Fixed in PR #123"
```

## `governance actions delete`

Delete a governance action (requires `--yes` to confirm).

```bash
shoehorn governance actions delete <id> --yes
```

## `governance dashboard`

View governance dashboard with health scores, action summary, and documentation coverage.

```bash
shoehorn governance dashboard
shoehorn governance dashboard --output json
```
