# Shoehorn CLI

Command-line interface for the **Shoehorn Intelligent Developer Platform**. Browse your service catalog, manage governance actions, view GitOps resources, explore teams and ownership, run Forge workflows, manage addons, and more — all from the terminal with a rich interactive TUI.

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
