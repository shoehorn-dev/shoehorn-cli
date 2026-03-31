# Declarative Operations

## `apply`

Create or update entities from manifest files. If the entity exists, it's updated; if not, it's created.

```bash
shoehorn apply -f service.yaml              # single file
shoehorn apply -f catalog/                  # all YAML files in directory
shoehorn apply -f service.yaml --dry-run    # show what would change
cat service.yaml | shoehorn apply -f -      # from stdin
shoehorn apply -f service.yaml -o json      # JSON output
```

## `diff`

Compare local manifests against remote state (read-only, no changes made).

```bash
shoehorn diff -f service.yaml              # show what apply would do
shoehorn diff -f catalog/                  # diff entire directory
shoehorn diff -f service.yaml -o json      # JSON output for scripting
```

Output symbols: `+` create, `~` update (with field changes), `=` unchanged.

## `create entity`

Create a new entity from a manifest YAML file.

```bash
shoehorn create entity -f service.yaml
cat service.yaml | shoehorn create entity -f -
shoehorn create entity -f service.yaml -o json
```

## `update entity`

Update an existing entity from a manifest file.

```bash
shoehorn update entity my-service -f service.yaml
cat service.yaml | shoehorn update entity my-service -f -
```

## `delete entity`

Delete an entity (requires confirmation).

```bash
shoehorn delete entity my-service
shoehorn delete entity my-service --yes    # skip confirmation
```
