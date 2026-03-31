# Catalog & Search

## `whoami`

Show your full user profile including roles, groups, and teams.

```bash
shoehorn whoami
shoehorn whoami --output json
```

## `search`

Full-text search across all catalog entities. Results open in an interactive table — press `Enter` to expand any item.

```bash
shoehorn search "payment"
shoehorn search "kafka" --output json
```

## `get entities`

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

## `get entity`

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

## `get teams`

List all teams.

```bash
shoehorn get teams
shoehorn get teams --output json
```

## `get team`

Full detail for a team, including its members.

```bash
shoehorn get team platform-team
shoehorn get team <uuid>
```

## `get users`

List all users in the directory.

```bash
shoehorn get users
shoehorn get users --output json
```

## `get user`

Detail for a specific user: groups, teams, roles.

```bash
shoehorn get user <user-id>
```

## `get groups`

List all directory groups.

```bash
shoehorn get groups
```

## `get group`

Show roles mapped to a specific group.

```bash
shoehorn get group engineers
```

## `get owned`

List all entities owned by a specific team or user.

```bash
shoehorn get owned --by team platform-team
shoehorn get owned --by user <user-id>
```

## `get scorecard`

Scorecard breakdown for an entity with a visual score bar and per-check table.

```bash
shoehorn get scorecard payment-service
shoehorn get scorecard payment-service --output json
```

## `get k8s`

List all registered Kubernetes agents.

```bash
shoehorn get k8s
shoehorn get k8s --output json
```

## `get gitops`

List GitOps resources (ArgoCD Applications, FluxCD Kustomizations) or get details for a specific one.

```bash
shoehorn get gitops
shoehorn get gitops --tool argocd --cluster-id prod-us-east-1
shoehorn get gitops --sync-status OutOfSync
shoehorn get gitops <id>
shoehorn get gitops <id> --output json
```

Flags: `--cluster-id`, `--tool` (argocd/fluxcd), `--sync-status`, `--health-status`
