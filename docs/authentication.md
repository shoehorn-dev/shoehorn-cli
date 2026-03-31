# Authentication

## PAT Login (recommended)

```bash
shoehorn auth login --server http://localhost:8080 --token shp_your_token_here
```

On success you'll see:

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

## Check Auth Status

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

## Logout

```bash
shoehorn auth logout
```

## Multiple Profiles

You can authenticate to multiple environments using profiles:

```bash
# Login to each environment
shoehorn --profile prod auth login --server https://api.shoehorn.dev --token shp_prod_xxx
shoehorn --profile staging auth login --server https://staging.shoehorn.dev --token shp_stg_xxx

# Use a specific profile for any command
shoehorn --profile prod get entities
shoehorn --profile staging forge molds list
```

See [Configuration](configuration.md) for more on profiles.
