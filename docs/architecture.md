# Project Structure

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
