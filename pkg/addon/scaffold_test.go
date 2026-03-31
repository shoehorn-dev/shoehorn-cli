package addon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSlug_Valid(t *testing.T) {
	valid := []string{"my-addon", "jira-sync", "postgres-manager", "abc", "a-b-c", "addon123"}
	for _, slug := range valid {
		if err := ValidateSlug(slug); err != nil {
			t.Errorf("ValidateSlug(%q) = %v, want nil", slug, err)
		}
	}
}

func TestValidateSlug_Invalid(t *testing.T) {
	invalid := []string{"", "a", "ab", "My-Addon", "-leading", "trailing-", "has spaces", "has_underscore",
		strings.Repeat("a", 52)}
	for _, slug := range invalid {
		if err := ValidateSlug(slug); err == nil {
			t.Errorf("ValidateSlug(%q) = nil, want error", slug)
		}
	}
}

func TestScaffold_Scripted_CreatesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "my-addon")

	err := Scaffold(ScaffoldConfig{
		Name: "my-addon",
		Tier: TierScripted,
		Dir:  target,
	})
	if err != nil {
		t.Fatalf("Scaffold() = %v", err)
	}

	expectedFiles := []string{
		"manifest.json",
		"package.json",
		"tsconfig.json",
		"esbuild.config.mjs",
		"src/index.ts",
		"README.md",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(target, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestScaffold_Declarative_NoTypeScript(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "my-widget")

	err := Scaffold(ScaffoldConfig{
		Name: "my-widget",
		Tier: TierDeclarative,
		Dir:  target,
	})
	if err != nil {
		t.Fatalf("Scaffold() = %v", err)
	}

	// Declarative should have manifest but no TypeScript files
	if _, err := os.Stat(filepath.Join(target, "manifest.json")); os.IsNotExist(err) {
		t.Error("expected manifest.json")
	}
	if _, err := os.Stat(filepath.Join(target, "package.json")); !os.IsNotExist(err) {
		t.Error("declarative addon should not have package.json")
	}
	if _, err := os.Stat(filepath.Join(target, "src/index.ts")); !os.IsNotExist(err) {
		t.Error("declarative addon should not have src/index.ts")
	}
}

func TestScaffold_Full_HasPostgresExample(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "postgres-mgr")

	err := Scaffold(ScaffoldConfig{
		Name: "postgres-mgr",
		Tier: TierFull,
		Dir:  target,
	})
	if err != nil {
		t.Fatalf("Scaffold() = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "src/index.ts"))
	if err != nil {
		t.Fatalf("read index.ts: %v", err)
	}

	if !strings.Contains(string(content), "ctx.postgres") {
		t.Error("full-tier addon should reference ctx.postgres in template")
	}
}

func TestScaffold_ManifestIsValidJSON(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "test-addon")

	err := Scaffold(ScaffoldConfig{
		Name: "test-addon",
		Tier: TierScripted,
		Dir:  target,
	})
	if err != nil {
		t.Fatalf("Scaffold() = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}

	var manifest map[string]any
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("manifest.json is not valid JSON: %v", err)
	}

	if manifest["kind"] != "addon" {
		t.Errorf("expected kind=addon, got %v", manifest["kind"])
	}

	metadata, ok := manifest["metadata"].(map[string]any)
	if !ok {
		t.Fatal("expected metadata object")
	}
	if metadata["slug"] != "test-addon" {
		t.Errorf("expected slug=test-addon, got %v", metadata["slug"])
	}

	addon, ok := manifest["addon"].(map[string]any)
	if !ok {
		t.Fatal("expected addon object")
	}
	if addon["tier"] != "scripted" {
		t.Errorf("expected tier=scripted, got %v", addon["tier"])
	}
}

func TestScaffold_DirectoryAlreadyExists_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "existing")
	os.MkdirAll(target, 0755)

	err := Scaffold(ScaffoldConfig{
		Name: "existing",
		Tier: TierScripted,
		Dir:  target,
	})
	if err == nil {
		t.Fatal("expected error when directory already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestScaffold_InvalidSlug_ReturnsError(t *testing.T) {
	err := Scaffold(ScaffoldConfig{
		Name: "INVALID",
		Tier: TierScripted,
	})
	if err == nil {
		t.Fatal("expected error for invalid slug")
	}
}

func TestScaffold_InvalidTier_ReturnsError(t *testing.T) {
	err := Scaffold(ScaffoldConfig{
		Name: "valid-name",
		Tier: "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid tier")
	}
}

func TestSlugToDisplayName(t *testing.T) {
	tests := []struct {
		slug string
		want string
	}{
		{"my-addon", "My Addon"},
		{"jira-sync", "Jira Sync"},
		{"postgres-manager", "Postgres Manager"},
		{"single", "Single"},
	}
	for _, tt := range tests {
		got := slugToDisplayName(tt.slug)
		if got != tt.want {
			t.Errorf("slugToDisplayName(%q) = %q, want %q", tt.slug, got, tt.want)
		}
	}
}

func TestGenerateManifestJSON_ValidOutput(t *testing.T) {
	data, err := GenerateManifestJSON(ScaffoldConfig{
		Name: "test-addon",
		Tier: TierScripted,
	})
	if err != nil {
		t.Fatalf("GenerateManifestJSON() = %v", err)
	}

	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if manifest["kind"] != "addon" {
		t.Errorf("expected kind=addon, got %v", manifest["kind"])
	}
}

// Parametric test: all tiers produce valid scaffolds
func TestScaffold_AllTiers_Parametric(t *testing.T) {
	tiers := []Tier{TierDeclarative, TierScripted, TierFull}
	for _, tier := range tiers {
		t.Run(string(tier), func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "test-addon")

			err := Scaffold(ScaffoldConfig{
				Name: "test-addon",
				Tier: tier,
				Dir:  target,
			})
			if err != nil {
				t.Fatalf("Scaffold(tier=%s) = %v", tier, err)
			}

			// All tiers must have manifest.json
			if _, err := os.Stat(filepath.Join(target, "manifest.json")); os.IsNotExist(err) {
				t.Errorf("tier=%s: missing manifest.json", tier)
			}

			// All tiers must have README.md
			if _, err := os.Stat(filepath.Join(target, "README.md")); os.IsNotExist(err) {
				t.Errorf("tier=%s: missing README.md", tier)
			}

			// manifest.json must be valid JSON with correct tier
			content, _ := os.ReadFile(filepath.Join(target, "manifest.json"))
			var manifest map[string]any
			if err := json.Unmarshal(content, &manifest); err != nil {
				t.Errorf("tier=%s: invalid manifest JSON: %v", tier, err)
			}

			addonSection := manifest["addon"].(map[string]any)
			if addonSection["tier"] != string(tier) {
				t.Errorf("tier=%s: manifest tier mismatch: %v", tier, addonSection["tier"])
			}
		})
	}
}
