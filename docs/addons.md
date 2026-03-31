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
