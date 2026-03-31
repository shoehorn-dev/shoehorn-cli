# Manifest Tools

## `validate`

Validate Shoehorn or Backstage manifest files.

```bash
shoehorn validate catalog-info.yaml
shoehorn validate catalog-info.yaml --format json
cat catalog-info.yaml | shoehorn validate -
```

## `convert`

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
