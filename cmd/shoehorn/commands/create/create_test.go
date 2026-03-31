package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ─── Command registration tests ────────────────────────────────────────────

func TestCreateCmd_IsRegistered(t *testing.T) {
	if CreateCmd == nil {
		t.Fatal("CreateCmd should not be nil")
	}
	if CreateCmd.Use != "create" {
		t.Errorf("CreateCmd.Use = %q, want %q", CreateCmd.Use, "create")
	}
	if CreateCmd.Short == "" {
		t.Error("CreateCmd.Short should not be empty")
	}
}

func TestEntityCmd_IsSubcommand(t *testing.T) {
	found := false
	for _, sub := range CreateCmd.Commands() {
		if strings.HasPrefix(sub.Use, "entity") {
			found = true
			break
		}
	}
	if !found {
		t.Error("entityCmd should be registered as a subcommand of CreateCmd")
	}
}

func TestEntityCmd_HasRequiredFileFlag(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range CreateCmd.Commands() {
		if strings.HasPrefix(sub.Use, "entity") {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("entity subcommand not found")
	}

	f := cmd.Flags().Lookup("file")
	if f == nil {
		t.Fatal("entity command must have --file flag")
	}
	if f.Shorthand != "f" {
		t.Errorf("--file shorthand = %q, want %q", f.Shorthand, "f")
	}
}

// ─── Error path: missing config / not authenticated ────────────────────────

func TestRunCreateEntity_NoConfig_ReturnsError(t *testing.T) {
	// Point config to a nonexistent directory so NewClientFromConfig fails.
	cfgDir := t.TempDir()
	t.Setenv("HOME", cfgDir)
	t.Setenv("USERPROFILE", cfgDir)
	// Clear any tokens so the client won't authenticate
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")

	// Create a minimal manifest file so ReadFile succeeds
	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	content := `schemaVersion: 1
service:
  id: test-svc
  name: Test Service
  type: service
`
	if err := os.WriteFile(tmpFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	// Set the package-level flag var so the command finds a file
	createEntityFile = tmpFile

	cmd := &cobra.Command{Use: "test"}
	err := runCreateEntity(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunCreateEntity_MissingFile_ReturnsError(t *testing.T) {
	createEntityFile = "/nonexistent/path/manifest.yaml"

	cmd := &cobra.Command{Use: "test"}
	err := runCreateEntity(cmd, nil)
	if err == nil {
		t.Error("expected error for missing manifest file, got nil")
	}
}

func TestRunCreateEntity_EmptyFilePath_ReturnsError(t *testing.T) {
	createEntityFile = ""

	cmd := &cobra.Command{Use: "test"}
	err := runCreateEntity(cmd, nil)
	if err == nil {
		t.Error("expected error for empty file path, got nil")
	}
}

// ─── Manifest file reading integration ─────────────────────────────────────

func TestRunCreateEntity_ValidFile_FailsAtClientCreation(t *testing.T) {
	// Ensure there's no valid config so we fail at client creation, not file reading
	noDir := t.TempDir()
	t.Setenv("HOME", noDir)
	t.Setenv("USERPROFILE", noDir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")

	tmpFile := filepath.Join(t.TempDir(), "svc.yaml")
	content := `schemaVersion: 1
service:
  id: my-api
  name: My API
  type: service
`
	if err := os.WriteFile(tmpFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	createEntityFile = tmpFile

	cmd := &cobra.Command{Use: "test"}
	err := runCreateEntity(cmd, nil)
	if err == nil {
		t.Error("expected error at client creation, got nil")
	}
	// The error should NOT be about file reading - it should be about config/auth
	if strings.Contains(err.Error(), "read file") || strings.Contains(err.Error(), "stat") {
		t.Errorf("error should be about config/auth, not file reading: %v", err)
	}
}

// ─── Property-based: command structure invariants ──────────────────────────

func TestCreateCmd_AllSubcommands_HaveShortDescription(t *testing.T) {
	for _, sub := range CreateCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Short == "" {
				t.Errorf("subcommand %q has empty Short description", sub.Name())
			}
		})
	}
}

func TestCreateCmd_AllSubcommands_HaveRunE(t *testing.T) {
	for _, sub := range CreateCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.RunE == nil && sub.Run == nil {
				t.Errorf("subcommand %q has neither RunE nor Run set", sub.Name())
			}
		})
	}
}
