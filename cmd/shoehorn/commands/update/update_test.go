package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ─── Command registration tests ────────────────────────────────────────────

func TestUpdateCmd_IsRegistered(t *testing.T) {
	if UpdateCmd == nil {
		t.Fatal("UpdateCmd should not be nil")
	}
	if UpdateCmd.Use != "update" {
		t.Errorf("UpdateCmd.Use = %q, want %q", UpdateCmd.Use, "update")
	}
	if UpdateCmd.Short == "" {
		t.Error("UpdateCmd.Short should not be empty")
	}
}

func TestEntityCmd_IsSubcommand(t *testing.T) {
	found := false
	for _, sub := range UpdateCmd.Commands() {
		if strings.HasPrefix(sub.Use, "entity") {
			found = true
			break
		}
	}
	if !found {
		t.Error("entityCmd should be registered as a subcommand of UpdateCmd")
	}
}

func TestEntityCmd_RequiresExactlyOneArg(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range UpdateCmd.Commands() {
		if strings.HasPrefix(sub.Use, "entity") {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("entity subcommand not found")
	}

	if cmd.Args == nil {
		t.Error("entity command must have Args validator (ExactArgs(1))")
	}
}

func TestEntityCmd_HasRequiredFileFlag(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range UpdateCmd.Commands() {
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

// ─── Error path: missing file ──────────────────────────────────────────────

func TestRunUpdateEntity_MissingFile_ReturnsError(t *testing.T) {
	updateEntityFile = "/nonexistent/path/manifest.yaml"

	cmd := &cobra.Command{Use: "test"}
	err := runUpdateEntity(cmd, []string{"my-entity"})
	if err == nil {
		t.Error("expected error for missing manifest file, got nil")
	}
}

func TestRunUpdateEntity_EmptyFilePath_ReturnsError(t *testing.T) {
	updateEntityFile = ""

	cmd := &cobra.Command{Use: "test"}
	err := runUpdateEntity(cmd, []string{"my-entity"})
	if err == nil {
		t.Error("expected error for empty file path, got nil")
	}
}

// ─── Error path: no config ─────────────────────────────────────────────────

func TestRunUpdateEntity_NoConfig_ReturnsError(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("HOME", cfgDir)
	t.Setenv("USERPROFILE", cfgDir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")

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

	updateEntityFile = tmpFile

	cmd := &cobra.Command{Use: "test"}
	err := runUpdateEntity(cmd, []string{"test-svc"})
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunUpdateEntity_ValidFile_FailsAtClientCreation(t *testing.T) {
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

	updateEntityFile = tmpFile

	cmd := &cobra.Command{Use: "test"}
	err := runUpdateEntity(cmd, []string{"my-api"})
	if err == nil {
		t.Error("expected error at client creation, got nil")
	}
	// Error should be about config/auth, not file reading
	if strings.Contains(err.Error(), "read file") || strings.Contains(err.Error(), "stat") {
		t.Errorf("error should be about config/auth, not file reading: %v", err)
	}
}

// ─── Property-based: command structure invariants ──────────────────────────

func TestUpdateCmd_AllSubcommands_HaveShortDescription(t *testing.T) {
	for _, sub := range UpdateCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Short == "" {
				t.Errorf("subcommand %q has empty Short description", sub.Name())
			}
		})
	}
}

func TestUpdateCmd_AllSubcommands_HaveRunE(t *testing.T) {
	for _, sub := range UpdateCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.RunE == nil && sub.Run == nil {
				t.Errorf("subcommand %q has neither RunE nor Run set", sub.Name())
			}
		})
	}
}
