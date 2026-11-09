# Addon Management

## `addon list`

List installed addons.

```bash
shoehorn addon list
shoehorn addon list --output json
```

## `addon status`

View runtime status of an addon.

```bash
shoehorn addon status my-addon
```

## `addon install` / `addon uninstall`

```bash
shoehorn addon install my-addon
shoehorn addon uninstall my-addon
```

## `addon enable` / `addon disable`

Toggle without uninstalling.

```bash
shoehorn addon enable my-addon
shoehorn addon disable my-addon
```

## `addon reload`

Re-push every enabled addon for the current tenant into the runtime. Use
this if an addon shows status `not_loaded` after an API restart or upgrade
-- the server doesn't always resume every installed addon on its own.

```bash
shoehorn addon reload
```

## `addon config`

View or update an addon's configuration from the command line instead of
the admin UI. With no flags, prints the current config. `--set` and
`--file` merge into the existing config rather than replacing it.

```bash
shoehorn addon config my-addon
shoehorn addon config my-addon --set api_url=https://api.example.com --set sync_mode=full
shoehorn addon config my-addon --file config.json
```

## `addon secret`

Set, list, or delete an addon's secret values (API tokens, credentials).
Values are never returned once set -- only key names can be listed.

```bash
shoehorn addon secret set my-addon api_token s3cr3t
shoehorn addon secret list my-addon
shoehorn addon secret delete my-addon api_token
```

## `addon logs`

View recent logs for an addon.

```bash
shoehorn addon logs my-addon
shoehorn addon logs my-addon --limit 50
```

## `addon browse`

Browse the addon marketplace.

```bash
shoehorn addon browse
```

## `addon publish`

Publish an addon to the marketplace.

```bash
shoehorn addon publish --dir ./my-addon
```

## `addon init` / `addon build` / `addon dev`

Scaffold, build, and develop addons locally.

```bash
shoehorn addon init my-addon
shoehorn addon build
shoehorn addon build --dir ./addons/my-addon
shoehorn addon dev
```
