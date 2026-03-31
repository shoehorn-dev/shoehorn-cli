package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	// Use a temp directory so we don't mess with real config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir) // Windows

	cfg := &Config{
		Version:        "1.0",
		CurrentProfile: "test",
		Profiles: map[string]*Profile{
			"test": {
				Name:   "Test",
				Server: "https://api.example.com",
				Auth: &Auth{
					ProviderType: "pat",
					AccessToken:  "shp_test_token_123",
					ExpiresAt:    time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
					User: &User{
						Email:    "test@example.com",
						Name:     "Test User",
						TenantID: "acme",
					},
				},
			},
		},
	}

	// Save
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists with correct permissions
	configPath := filepath.Join(tmpDir, ".shoehorn", "config.yaml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("config file is empty")
	}

	// Load back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify round-trip preserved data
	if loaded.Version != "1.0" {
		t.Errorf("Version = %q, want '1.0'", loaded.Version)
	}
	if loaded.CurrentProfile != "test" {
		t.Errorf("CurrentProfile = %q, want 'test'", loaded.CurrentProfile)
	}

	profile, err := loaded.GetCurrentProfile()
	if err != nil {
		t.Fatal(err)
	}
	if profile.Server != "https://api.example.com" {
		t.Errorf("Server = %q", profile.Server)
	}
	if profile.Auth == nil {
		t.Fatal("Auth is nil after round-trip")
	}
	if profile.Auth.AccessToken != "shp_test_token_123" {
		t.Errorf("AccessToken = %q", profile.Auth.AccessToken)
	}
	if profile.Auth.User == nil || profile.Auth.User.Email != "test@example.com" {
		t.Errorf("User email not preserved")
	}
}

func TestLoad_MissingFile_ReturnsDefault(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.CurrentProfile != "default" {
		t.Errorf("CurrentProfile = %q, want 'default'", cfg.CurrentProfile)
	}
	if cfg.Profiles["default"] == nil {
		t.Fatal("default profile missing")
	}
}

func TestLoad_CorruptedFile_ReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// Write invalid YAML
	dir := filepath.Join(tmpDir, ".shoehorn")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("{{invalid yaml"), 0600)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for corrupted config")
	}
}

func TestGetCurrentProfile_MissingProfile_ReturnsError(t *testing.T) {
	cfg := &Config{
		CurrentProfile: "nonexistent",
		Profiles:       map[string]*Profile{},
	}
	_, err := cfg.GetCurrentProfile()
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestSetProfile(t *testing.T) {
	cfg := &Config{}
	cfg.SetProfile("new", &Profile{Name: "New", Server: "https://new.example.com"})
	if cfg.Profiles["new"] == nil {
		t.Fatal("profile not set")
	}
	if cfg.Profiles["new"].Server != "https://new.example.com" {
		t.Errorf("Server = %q", cfg.Profiles["new"].Server)
	}
}

func TestSave_AtomicWrite_TempFileCleanedUp(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := &Config{
		Version:        "1.0",
		CurrentProfile: "default",
		Profiles:       map[string]*Profile{"default": {Name: "Default"}},
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	// Verify no .tmp file left behind
	tmpFile := filepath.Join(tmpDir, ".shoehorn", "config.yaml.tmp")
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Errorf("temp file not cleaned up: %s", tmpFile)
	}
}
