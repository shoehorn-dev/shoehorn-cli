# Shoehorn CLI

[![Go Report Card](https://goreportcard.com/badge/github.com/shoehorn-dev/shoehorn-cli)](https://goreportcard.com/report/github.com/shoehorn-dev/shoehorn-cli)
[![Release](https://img.shields.io/github/v/release/shoehorn-dev/shoehorn-cli)](https://github.com/shoehorn-dev/shoehorn-cli/releases/latest)

Command-line interface for the Shoehorn Intelligent Developer Platform. Browse your service catalog, manage governance actions, view GitOps resources, explore teams and ownership, run Forge workflows, manage addons, and more — all from the terminal with a rich interactive TUI.

## Installation

### Build from source

```bash
go build -o shoehorn.exe ./cmd/shoehorn
```

### Homebrew (macOS / Linux)

```bash
brew tap shoehorn-dev/tap
brew install shoehorn
```

### Mise (macOS / Linux / Windows)

Add to your `mise.toml` or `~/.config/mise/config.toml`:

```toml
[tools."github:shoehorn-dev/shoehorn-cli"]
version = "latest"
exe = "shoehorn"
```

Then run:

```bash
mise install
```

**Windows:** Ensure the mise shims directory is in your PATH:

```powershell
# Add to your PowerShell profile (one-time setup):
Add-Content $PROFILE '$env:PATH = "$env:LOCALAPPDATA\mise\shims;$env:PATH"'
```

### Scoop (Windows)

```powershell
scoop bucket add shoehorn https://github.com/shoehorn-dev/scoop-bucket
scoop install shoehorn
```

### Manual download

Download the binary for your platform from [Releases](https://github.com/shoehorn-dev/shoehorn-cli/releases), extract it, and add to your PATH.

---

## Quick Start

### 1. Authenticate with a Personal Access Token (recommended)

```bash
shoehorn auth login --server http://localhost:8080 --token shp_your_token_here
```

On success you'll see a panel like:

```
╭─ Authenticated with PAT ───────────────────────────────╮
│ ✓ Authenticated with PAT                               │
│                                                        │
│ Name      Jane Smith                                   │
│ Email     jane@example.com                             │
│ Tenant    acme-corp                                    │
│ Server    http://localhost:8080                        │
╰────────────────────────────────────────────────────────╯
```

### 2. Verify your identity

```bash
shoehorn whoami
```

### 3. Explore the catalog

```bash
shoehorn get entities
shoehorn search "payment"
shoehorn get entity payment-service --scorecard
```

---

## Authentication

### PAT login

```bash
shoehorn auth login --server http://localhost:8080 --token shp_xxxx
```

### Check auth status

```bash
shoehorn auth status
```

```
Profile: default
Server:  http://localhost:8080
Status:  Authenticated (PAT)
Email:   jane@example.com
Tenant:  acme-corp
Token:   Valid (PAT, no expiry)
Server:  Token verified with server
```

### Logout

```bash
shoehorn auth logout
```

---

## Commands

### `whoami`

Show your full user profile including roles, groups, and teams.

```bash
shoehorn whoami
shoehorn whoami --output json
```

---

### `search`

Full-text search across all catalog entities. Results open in an interactive table — press `Enter` to expand any item.

```bash
shoehorn search "payment"
shoehorn search "kafka" --output json
```

---

### `get entities`

List all catalog entities in an interactive table.

```bash
shoehorn get entities
shoehorn get entities --type service
shoehorn get entities --owner platform-team
shoehorn get entities --output json
```

Flags:
- `--type` — filter by entity type (service, library, website, etc.)
- `--owner` — filter by owning team slug

---

### `get entity`

Full detail panel for a single entity.

```bash
shoehorn get entity payment-service
shoehorn get entity payment-service --scorecard
shoehorn get entity <uuid> --output json
```

Example output:

```
╭─ payment-service ──────────────────────────────────────────╮
│ payment-service                                            │
│                                                            │
│ Type               service                                 │
│ Owner              platform-team                           │
│ Lifecycle          production                              │
│ Tier               1                                       │
│ Description        Core payment processing service         │
│ Tags               payments  core  pci                     │
│ Status             ● healthy  (99.97% uptime)              │
│ Links              GitHub  Grafana  Datadog                │
│                                                            │
│ Resources (3)                                              │
│ payment-db         PostgreSQL  production                  │
│ payment-cache      Redis       production                  │
│ payment-queue      Kafka topic production                  │
│                                                            │
│ Scorecard                                                  │
│ Grade              A  ████████████████████████░░░░ 92/100  │
╰────────────────────────────────────────────────────────────╯
```

---

### `get teams`

List all teams.

```bash
shoehorn get teams
shoehorn get teams --output json
```

---

### `get team`

Full detail for a team, including its members.

```bash
shoehorn get team platform-team
shoehorn get team <uuid>
```

---

### `get users`

List all users in the directory.

```bash
shoehorn get users
shoehorn get users --output json
```

---

### `get user`

Detail for a specific user: groups, teams, roles.

```bash
shoehorn get user <user-id>
```

---

### `get groups`

List all directory groups.

```bash
shoehorn get groups
```

---

### `get group`

Show roles mapped to a specific group.

```bash
shoehorn get group engineers
```

---

### `get owned`

List all entities owned by a specific team or user.

```bash
shoehorn get owned --by team platform-team
shoehorn get owned --by user <user-id>
```

---

### `get scorecard`

Scorecard breakdown for an entity with a visual score bar and per-check table.

```bash
shoehorn get scorecard payment-service
shoehorn get scorecard payment-service --output json
```

---

### `get k8s`

List all registered Kubernetes agents.

```bash
shoehorn get k8s
shoehorn get k8s --output json
```

---

### `get molds` / `get mold`

List all Forge molds or get details for a specific one.

```bash
shoehorn get molds
shoehorn get mold create-empty-github-repo
shoehorn get mold create-empty-github-repo --output json
```

Also available as `forge molds list` / `forge molds get`.

---

### `get runs` / `get run`

List all Forge workflow runs or get details for a specific one.

```bash
shoehorn get runs
shoehorn get run <run-id>
shoehorn get run <run-id> --output json
```

Also available as `forge run list` / `forge run get`.

---

### `get gitops`

List GitOps resources (ArgoCD Applications, FluxCD Kustomizations) or get details for a specific one.

```bash
shoehorn get gitops
shoehorn get gitops --tool argocd --cluster-id prod-us-east-1
shoehorn get gitops --sync-status OutOfSync
shoehorn get gitops <id>
shoehorn get gitops <id> --output json
```

Flags: `--cluster-id`, `--tool` (argocd/fluxcd), `--sync-status`, `--health-status`

---

### `forge molds list`

List all available Forge workflow templates.

```bash
shoehorn forge molds list
shoehorn forge molds list --output json
```

---

### `forge molds get`

Detail view for a mold: actions, inputs, and steps.

```bash
shoehorn forge molds get create-empty-github-repo
```

---

### `forge execute`

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

---

### `forge run list`

List all workflow runs.

```bash
shoehorn forge run list
shoehorn forge run list --output json
```

---

### `forge run watch`

Watch a run until it completes (polls every 2 seconds).

```bash
shoehorn forge run watch <run-id>
shoehorn forge run watch <run-id> --interval 5
shoehorn forge run watch <run-id> -o json
```

---

### `forge run get`

Detail for a specific run.

```bash
shoehorn forge run get <run-id>
shoehorn forge run get <run-id> --output json
```

---

### `forge run create`

Start a new workflow run from a mold (lower-level than `forge execute`).

```bash
shoehorn forge run create create-empty-github-repo --action create \
  --input name=my-service --input owner=my-org
```

---

## Governance

### `governance actions list`

List governance action items with optional filters.

```bash
shoehorn governance actions list
shoehorn governance actions list --output json
```

### `governance actions get`

Get details for a specific governance action.

```bash
shoehorn governance actions get <id>
shoehorn governance actions get <id> --output json
```

### `governance actions create`

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

### `governance actions update`

Update an existing governance action (status, priority, assignment, resolution).

```bash
shoehorn governance actions update <id> --status in_progress
shoehorn governance actions update <id> --priority critical --assigned-to "bob@company.com"
shoehorn governance actions update <id> --status resolved --resolution-note "Fixed in PR #123"
```

### `governance actions delete`

Delete a governance action (requires `--yes` to confirm).

```bash
shoehorn governance actions delete <id> --yes
```

### `governance dashboard`

View governance dashboard with health scores, action summary, and documentation coverage.

```bash
shoehorn governance dashboard
shoehorn governance dashboard --output json
```

---

## Declarative Operations

### `apply`

Create or update entities from manifest files. If the entity exists, it's updated; if not, it's created.

```bash
shoehorn apply -f service.yaml              # single file
shoehorn apply -f catalog/                  # all YAML files in directory
shoehorn apply -f service.yaml --dry-run    # show what would change
cat service.yaml | shoehorn apply -f -      # from stdin
shoehorn apply -f service.yaml -o json      # JSON output
```

### `diff`

Compare local manifests against remote state (read-only, no changes made).

```bash
shoehorn diff -f service.yaml              # show what apply would do
shoehorn diff -f catalog/                  # diff entire directory
shoehorn diff -f service.yaml -o json      # JSON output for scripting
```

Output symbols: `+` create, `~` update (with field changes), `=` unchanged.

---

## Write Operations

### `create entity`

Create a new entity from a manifest YAML file.

```bash
shoehorn create entity -f service.yaml
cat service.yaml | shoehorn create entity -f -
shoehorn create entity -f service.yaml -o json
```

### `update entity`

Update an existing entity from a manifest file.

```bash
shoehorn update entity my-service -f service.yaml
cat service.yaml | shoehorn update entity my-service -f -
```

### `delete entity`

Delete an entity (requires confirmation).

```bash
shoehorn delete entity my-service
shoehorn delete entity my-service --yes    # skip confirmation
```

---

## Addon Management

### `addon list`

List installed addons.

```bash
shoehorn addon list
shoehorn addon list --output json
```

### `addon status`

View runtime status of an addon.

```bash
shoehorn addon status my-addon
```

### `addon install` / `addon uninstall`

```bash
shoehorn addon install my-addon
shoehorn addon uninstall my-addon
```

### `addon enable` / `addon disable`

Toggle without uninstalling.

```bash
shoehorn addon enable my-addon
shoehorn addon disable my-addon
```

### `addon logs`

View recent logs for an addon.

```bash
shoehorn addon logs my-addon
shoehorn addon logs my-addon --limit 50
```

### `addon browse`

Browse the addon marketplace.

```bash
shoehorn addon browse
```

### `addon publish`

Publish an addon to the marketplace.

```bash
shoehorn addon publish --dir ./my-addon
```

### `addon init` / `addon build` / `addon dev`

Scaffold, build, and develop addons locally.

```bash
shoehorn addon init my-addon
shoehorn addon build
shoehorn addon build --dir ./addons/my-addon
shoehorn addon dev
```

---

## Manifest Tools

### `validate`

Validate Shoehorn or Backstage manifest files.

```bash
shoehorn validate catalog-info.yaml
shoehorn validate catalog-info.yaml --format json
cat catalog-info.yaml | shoehorn validate -
```

### `convert`

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

---

## CI/CD Checks

Gate your pipelines on catalog quality. Exit code 0 = pass, 1 = fail.

### `check scorecard`

```bash
shoehorn check scorecard my-service --min-score 70
shoehorn check scorecard my-service --min-score 80 -o json
```

### `check entity`

```bash
shoehorn check entity my-service --has-owner
shoehorn check entity my-service --has-owner --has-docs
shoehorn check entity my-service --has-owner -o json
```

---

## Shell Completion

Generate completion scripts for your shell.

```bash
# Bash
shoehorn completion bash > /etc/bash_completion.d/shoehorn

# Zsh
shoehorn completion zsh > "${fpath[1]}/_shoehorn"

# Fish
shoehorn completion fish > ~/.config/fish/completions/shoehorn.fish

# PowerShell
shoehorn completion powershell > shoehorn.ps1
```

---

## Global Flags

All commands accept these flags:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | (interactive) | Output format: `json`, `yaml`, or `table` |
| `--no-interactive` | `-I` | `false` | Disable TUI, force plain text output |
| `--interactive` | `-i` | `false` | Force interactive TUI mode |
| `--debug` | | `false` | Enable debug logging to stderr |
| `--profile` | | `default` | Auth profile to use |
| `--config` | | `~/.shoehorn/config.yaml` | Config file path |

Debug logging can also be enabled with `SHOEHORN_DEBUG=1`.

### Script-friendly output

Any command can be piped to `jq` or used in scripts:

```bash
shoehorn get entities --output json | jq '.[] | select(.type == "service") | .name'
shoehorn get team platform-team --output json | jq '.members[].email'
shoehorn whoami --output json | jq '.tenant_id'
```

---

## Interactive TUI Controls

All table views share the same key bindings:

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Select item / expand details |
| `q` / `Esc` | Quit / clear filter |
| `Backspace` | Remove last filter character |

---

## Configuration

Config file: `~/.shoehorn/config.yaml`

```yaml
version: "1.0"
current_profile: default

profiles:
  default:
    name: Default
    server: http://localhost:8080
    auth:
      provider_type: pat
      access_token: shp_xxxxxxxxxxxx
      user:
        email: jane@example.com
        name: Jane Smith
        tenant_id: acme-corp

  prod:
    name: Production
    server: https://api.shoehorn.dev
    auth:
      provider_type: pat
      access_token: shp_prod_xxxx
      user:
        email: jane@example.com
        tenant_id: acme-corp
```

### Multiple profiles

```bash
# Login to each environment
shoehorn --profile prod auth login --server https://api.shoehorn.dev --token shp_prod_xxx
shoehorn --profile staging auth login --server https://staging.shoehorn.dev --token shp_stg_xxx

# Use a specific profile for any command
shoehorn --profile prod get entities
shoehorn --profile staging forge molds list
```

---

## Project Structure

```
cli/
├── cmd/shoehorn/
│   ├── main.go                    # Entry point
│   └── commands/
│       ├── root.go                # Root command, global flags, --debug, signal context
│       ├── auth.go                # auth login/status/logout
│       ├── whoami.go              # whoami
│       ├── search.go              # search <query>
│       ├── forge.go               # forge run/molds/execute
│       ├── validate.go            # validate manifests
│       ├── convert.go             # convert between formats
│       ├── addon.go               # addon management
│       ├── addon_publish.go       # addon publish
│       ├── addon_init.go          # addon scaffolding
│       ├── addon_build.go         # addon build
│       ├── addon_dev.go           # addon dev mode
│       └── get/
│           ├── get.go             # get (parent command)
│           ├── entities.go        # get entities / get entity
│           ├── teams.go           # get teams / get team
│           ├── users.go           # get users / get user
│           ├── groups.go          # get groups / get group
│           ├── owned.go           # get owned
│           ├── scorecard.go       # get scorecard
│           └── k8s.go             # get k8s
├── pkg/
│   ├── api/
│   │   ├── client.go              # HTTP client, atomic token, debug logging
│   │   ├── auth.go                # Auth status types + methods
│   │   ├── catalog.go             # Catalog API: entities, teams, users, forge...
│   │   ├── addons.go              # Addon management API
│   │   ├── manifests.go           # Manifest validation + conversion API
│   │   └── errors.go              # Typed errors + sentinels (401-429-5xx)
│   ├── addon/
│   │   ├── scaffold.go            # Addon project scaffolding
│   │   └── builder.go             # Addon bundle building
│   ├── config/
│   │   └── config.go              # Config file, profiles, atomic save
│   ├── logging/
│   │   └── logger.go              # Zap logger factory (--debug / SHOEHORN_DEBUG)
│   ├── tui/
│   │   ├── styles.go              # Shared lipgloss styles
│   │   ├── spinner.go             # RunSpinner() helper
│   │   ├── table.go               # RunTable() interactive table
│   │   └── detail.go              # RenderDetail(), score bars, boxes
│   └── ui/
│       ├── detect.go              # Interactive vs plain mode detection
│       ├── output.go              # JSON/YAML/table rendering
│       └── exit_codes.go          # Typed exit codes (errors.Is based)
└── go.mod
```

---

## Troubleshooting

### "not authenticated" error

```bash
shoehorn auth status
shoehorn auth login --server http://localhost:8080 --token shp_xxxx
```

### API connection refused

Check that the Shoehorn API is running:

```bash
curl http://localhost:8080/health
```

Verify the server URL in your config:

```bash
cat ~/.shoehorn/config.yaml
```

### Token rejected by server

Your PAT may have been revoked. Generate a new one in the Shoehorn UI and re-authenticate:

```bash
shoehorn auth logout
shoehorn auth login --server http://localhost:8080 --token shp_new_token
```

### Debug mode

Enable debug logging to see HTTP requests, response codes, and timing:

```bash
shoehorn --debug get entities
# or
SHOEHORN_DEBUG=1 shoehorn get entities
```

Debug output goes to stderr, so it won't interfere with JSON/YAML piping:

```bash
shoehorn --debug get entities -o json 2>debug.log | jq .
```

### TUI not rendering correctly

Disable interactive mode if your terminal doesn't support ANSI colors:

```bash
shoehorn get entities --no-interactive
shoehorn get entities -o json
```
