# Troubleshooting

## "not authenticated" error

```bash
shoehorn auth status
shoehorn auth login --server http://localhost:8080 --token shp_xxxx
```

## API connection refused

Check that the Shoehorn API is running:

```bash
curl http://localhost:8080/health
```

Verify the server URL in your config:

```bash
cat ~/.shoehorn/config.yaml
```

## Token rejected by server

Your PAT may have been revoked. Generate a new one in the Shoehorn UI and re-authenticate:

```bash
shoehorn auth logout
shoehorn auth login --server http://localhost:8080 --token shp_new_token
```

## Debug Mode

Enable debug logging to see HTTP requests, response codes, and timing:

```bash
shoehorn --debug get entities
# or
SHOEHORN_DEBUG=1 shoehorn get entities
```

Debug output goes to stderr, so it won't interfere with JSON/YAML piping:

```bash
shoehorn --debug get entities -o json 2>debug.log | jq .
```

## TUI not rendering correctly

Disable interactive mode if your terminal doesn't support ANSI colors:

```bash
shoehorn get entities --no-interactive
shoehorn get entities -o json
```
