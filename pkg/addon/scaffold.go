// Package addon provides utilities for addon development (scaffold, build, publish).
package addon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// Tier represents the addon capability tier.
type Tier string

const (
	TierDeclarative Tier = "declarative"
	TierScripted    Tier = "scripted"
	TierFull        Tier = "full"
)

// ValidTiers is the set of valid tiers.
var ValidTiers = map[Tier]bool{
	TierDeclarative: true,
	TierScripted:    true,
	TierFull:        true,
}

// slugRegexp validates addon slugs (kebab-case, 3-50 chars).
var slugRegexp = regexp.MustCompile(`^[a-z][a-z0-9-]{1,48}[a-z0-9]$`)

// ScaffoldConfig holds the configuration for scaffolding a new addon.
type ScaffoldConfig struct {
	Name string // Addon slug (kebab-case)
	Tier Tier   // declarative, scripted, full
	Dir  string // Output directory (defaults to Name)
}

// ValidateSlug checks if a slug is valid.
func ValidateSlug(slug string) error {
	if !slugRegexp.MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must be kebab-case, 3-50 chars, start/end with letter/digit", slug)
	}
	return nil
}

// Scaffold creates a new addon project directory.
func Scaffold(cfg ScaffoldConfig) error {
	if err := ValidateSlug(cfg.Name); err != nil {
		return err
	}
	if !ValidTiers[cfg.Tier] {
		return fmt.Errorf("invalid tier %q: must be declarative, scripted, or full", cfg.Tier)
	}

	dir := cfg.Dir
	if dir == "" {
		dir = cfg.Name
	}

	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory %q already exists", dir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	files, err := scaffoldFiles(cfg)
	if err != nil {
		return fmt.Errorf("generate scaffold files: %w", err)
	}
	for relPath, content := range files {
		fullPath := filepath.Join(dir, relPath)

		// Ensure parent directory exists
		if parent := filepath.Dir(fullPath); parent != dir {
			if err := os.MkdirAll(parent, 0755); err != nil {
				return fmt.Errorf("create directory %s: %w", parent, err)
			}
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
	}

	return nil
}

// scaffoldFiles returns the map of relative path -> file content for the scaffold.
func scaffoldFiles(cfg ScaffoldConfig) (map[string]string, error) {
	data := templateData{
		Name:        cfg.Name,
		DisplayName: slugToDisplayName(cfg.Name),
		Tier:        string(cfg.Tier),
	}

	files := map[string]string{}

	manifest, err := renderTemplate(manifestTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("manifest.json: %w", err)
	}
	files["manifest.json"] = manifest

	readme, err := renderTemplate(readmeTemplate, data)
	if err != nil {
		return nil, fmt.Errorf("README.md: %w", err)
	}
	files["README.md"] = readme

	switch cfg.Tier {
	case TierDeclarative:
		// Declarative addons are YAML-only, no TypeScript
		// manifest.json is all that's needed

	case TierScripted, TierFull:
		pkg, err := renderTemplate(packageJSONTemplate, data)
		if err != nil {
			return nil, fmt.Errorf("package.json: %w", err)
		}
		files["package.json"] = pkg
		files["tsconfig.json"] = tsconfigContent
		files["esbuild.config.mjs"] = esbuildConfigContent

		tmpl := indexTSTemplate
		if cfg.Tier == TierFull {
			tmpl = indexTSFullTemplate
		}
		indexTS, err := renderTemplate(tmpl, data)
		if err != nil {
			return nil, fmt.Errorf("src/index.ts: %w", err)
		}
		files["src/index.ts"] = indexTS
	}

	return files, nil
}

type templateData struct {
	Name        string
	DisplayName string
	Tier        string
}

func slugToDisplayName(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func renderTemplate(tmplStr string, data templateData) (string, error) {
	tmpl := template.Must(template.New("").Parse(tmplStr))
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}
	return buf.String(), nil
}

// GenerateManifestJSON creates a manifest.json from ScaffoldConfig.
func GenerateManifestJSON(cfg ScaffoldConfig) ([]byte, error) {
	manifest := map[string]any{
		"schemaVersion": 1,
		"kind":          "addon",
		"metadata": map[string]any{
			"slug":     cfg.Name,
			"name":     slugToDisplayName(cfg.Name),
			"version":  "0.1.0",
			"category": "custom",
			"tier":     "free",
		},
		"addon": map[string]any{
			"tier":    string(cfg.Tier),
			"runtime": "quickjs",
		},
	}

	return json.MarshalIndent(manifest, "", "  ")
}

// ─── Templates ──────────────────────────────────────────────────────────────

var manifestTemplate = `{
  "schemaVersion": 1,
  "kind": "addon",
  "metadata": {
    "slug": "{{.Name}}",
    "name": "{{.DisplayName}}",
    "version": "0.1.0",
    "description": "A Shoehorn addon",
    "author": {
      "name": "Your Name"
    },
    "category": "custom",
    "tier": "free"
  },
  "addon": {
    "tier": "{{.Tier}}",
    "runtime": "quickjs",
    "permissions": {
      "network": [],
      "shoehorn": ["entities:read"]
    }
  }
}
`

var packageJSONTemplate = `{
  "name": "{{.Name}}",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "node esbuild.config.mjs",
    "dev": "node esbuild.config.mjs --watch",
    "typecheck": "tsc --noEmit"
  },
  "devDependencies": {
    "esbuild": "^0.21.0",
    "typescript": "^5.5.0"
  }
}
`

var tsconfigContent = `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ES2020",
    "moduleResolution": "bundler",
    "strict": true,
    "outDir": "dist",
    "rootDir": "src",
    "declaration": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src"]
}
`

var esbuildConfigContent = `import { build } from 'esbuild';

const isWatch = process.argv.includes('--watch');

// QuickJS requires IIFE format with global exports.
// globalName wraps exports; footer hoists them to global scope
// so the Shoehorn runtime can call handleRoute() directly.
const config = {
  entryPoints: ['src/index.ts'],
  bundle: true,
  outfile: 'dist/addon.js',
  format: 'iife',
  globalName: '__addon__',
  footer: { js: 'if(typeof __addon__!=="undefined"){for(var k in __addon__)globalThis[k]=__addon__[k];}' },
  target: 'es2020',
  platform: 'neutral',
  minify: !isWatch,
  sourcemap: isWatch,
};

if (isWatch) {
  const ctx = await build({ ...config, plugins: [] });
  console.log('Watching for changes...');
} else {
  await build(config);
  console.log('Build complete: dist/addon.js');
}
`

var indexTSTemplate = `/**
 * {{.DisplayName}} - Shoehorn Addon ({{.Tier}})
 *
 * Runtime contract:
 * - Functions receive JS objects as arguments (not JSON strings)
 * - Functions return JS objects (runtime wraps in JSON.stringify automatically)
 * - Host functions available: ctx.log, ctx.config, ctx.http, ctx.entities
 */

// Addon context provided by Shoehorn runtime
declare const ctx: {
  log: { info: (msg: string) => void; warn: (msg: string) => void; error: (msg: string) => void; debug: (msg: string) => void };
  config: { get: (key: string) => string };
  http: { request: (method: string, url: string, opts?: { body?: string; headers?: Record<string, string>; timeout?: number }) => { status: number; body: unknown; headers?: Record<string, string> } };
  entities: {
    list: (filter?: { type?: string; lifecycle?: string; owner?: string; limit?: number; offset?: number }) => { entities: unknown[]; total: number };
    get: (id: string) => { entity: unknown };
    upsert: (entity: { serviceId: string; name: string; type: string; description?: string; tags?: string[]; links?: { name: string; url: string; icon?: string }[] }) => { entity: unknown; status: string };
    delete: (id: string) => { status: string };
  };
  addon: { id: string; version: string; tier: string };
};

interface RouteRequest {
  method: string;
  path: string;
  headers?: Record<string, string>;
  query?: Record<string, string>;
  body?: string;
}

interface RouteResponse {
  status: number;
  body: unknown;
  headers?: Record<string, string>;
}

/**
 * Handle incoming HTTP requests routed to this addon.
 */
export function handleRoute(request: RouteRequest): RouteResponse {
  ctx.log.info('Request: ' + request.method + ' ' + request.path);

  if (request.path === '/ping') {
    return { status: 200, body: { message: 'pong', addon: '{{.Name}}' } };
  }

  return { status: 404, body: { error: 'not found' } };
}

/**
 * Sync function called on schedule (if configured in manifest).
 */
export function sync(): { synced: number } {
  ctx.log.info('Starting sync...');
  return { synced: 0 };
}
`

var indexTSFullTemplate = `/**
 * {{.DisplayName}} - Shoehorn Addon (full)
 *
 * Full-tier addon with access to external resources (Postgres, Kafka, etc.)
 * Runtime contract: functions receive/return JS objects. ctx.* host functions available.
 */

// Addon context provided by Shoehorn runtime
declare const ctx: {
  log: { info: (msg: string) => void; warn: (msg: string) => void; error: (msg: string) => void; debug: (msg: string) => void };
  config: { get: (key: string) => string };
  http: { request: (method: string, url: string, opts?: { body?: string; headers?: Record<string, string>; timeout?: number }) => { status: number; body: unknown; headers?: Record<string, string> } };
  entities: {
    list: (filter?: { type?: string; limit?: number; offset?: number }) => { entities: unknown[]; total: number };
    get: (id: string) => { entity: unknown };
    upsert: (entity: { serviceId: string; name: string; type: string; description?: string; tags?: string[]; links?: { name: string; url: string; icon?: string }[] }) => { entity: unknown; status: string };
    delete: (id: string) => { status: string };
  };
  addon: { id: string; version: string; tier: string };
};

interface RouteRequest {
  method: string;
  path: string;
  headers?: Record<string, string>;
  query?: Record<string, string>;
  body?: string;
}

interface RouteResponse {
  status: number;
  body: unknown;
  headers?: Record<string, string>;
}

/**
 * Handle incoming HTTP requests routed to this addon.
 */
export function handleRoute(request: RouteRequest): RouteResponse {
  ctx.log.info('Request: ' + request.method + ' ' + request.path);

  if (request.path === '/ping') {
    return { status: 200, body: { message: 'pong', addon: '{{.Name}}' } };
  }

  return { status: 404, body: { error: 'not found' } };
}

/**
 * Sync function called on schedule (if configured in manifest).
 * For full-tier addons, ctx.postgres and ctx.kafka are also available.
 */
export function sync(): { synced: number } {
  ctx.log.info('Starting sync...');
  return { synced: 0 };
}
`

var readmeTemplate = `# {{.DisplayName}}

A Shoehorn addon (tier: {{.Tier}}).

## Development

` + "```" + `bash
# Install dependencies (scripted/full only)
npm install

# Build the addon bundle
npm run build
# Or: shoehorn addon build

# Publish to your Shoehorn instance
shoehorn addon publish
` + "```" + `

## Structure

- ` + "`manifest.json`" + ` - Addon manifest (permissions, metadata, config)
- ` + "`src/index.ts`" + ` - Addon entry point (handleRequest, sync)
- ` + "`dist/addon.js`" + ` - Compiled bundle (generated by build)
`
