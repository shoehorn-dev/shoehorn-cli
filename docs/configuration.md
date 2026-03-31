# Configuration

## Config File

Location: `~/.shoehorn/config.yaml`

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

## Multiple Profiles

```bash
# Login to each environment
shoehorn --profile prod auth login --server https://api.shoehorn.dev --token shp_prod_xxx
shoehorn --profile staging auth login --server https://staging.shoehorn.dev --token shp_stg_xxx

# Use a specific profile for any command
shoehorn --profile prod get entities
shoehorn --profile staging forge molds list
```

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

## Script-Friendly Output

Any command can be piped to `jq` or used in scripts:

```bash
shoehorn get entities --output json | jq '.[] | select(.type == "service") | .name'
shoehorn get team platform-team --output json | jq '.members[].email'
shoehorn whoami --output json | jq '.tenant_id'
```

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

## Interactive TUI Controls

All table views share the same key bindings:

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `Enter` | Select item / expand details |
| `q` / `Esc` | Quit / clear filter |
| `Backspace` | Remove last filter character |
