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

## Quick Start

```bash
# 1. Authenticate
shoehorn auth login --server http://localhost:8080 --token shp_your_token_here

# 2. Verify your identity
shoehorn whoami

# 3. Explore the catalog
shoehorn get entities
shoehorn search "payment"
shoehorn get entity payment-service --scorecard
```

## Documentation

| Guide | Description |
|-------|-------------|
| [Authentication](docs/authentication.md) | PAT login, status, logout |
| [Catalog & Search](docs/catalog.md) | Entities, teams, users, groups, ownership, scorecards, K8s, GitOps |
| [Forge Workflows](docs/forge.md) | Molds, execute workflows, manage runs |
| [Governance](docs/governance.md) | Actions, dashboard |
| [Declarative Operations](docs/declarative-ops.md) | Apply, diff, create/update/delete entities |
| [Addons](docs/addons.md) | Install, publish, develop addons |
| [Manifest Tools](docs/manifest-tools.md) | Validate and convert manifests |
| [CI/CD Checks](docs/ci-cd.md) | Gate pipelines on catalog quality |
| [Configuration](docs/configuration.md) | Config file, profiles, global flags, shell completion, TUI controls |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and debug mode |
| [Architecture](docs/architecture.md) | Project structure |

## License

[MIT](LICENSE)
