package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMoldCmd_IsRegistered(t *testing.T) {
	// The "mold" subcommand should be registered under "validate"
	found := false
	for _, sub := range validateCmd.Commands() {
		if sub.Use == "mold [file]" || sub.Name() == "mold" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'mold' subcommand registered under 'validate'")
	}
}

func TestValidateMoldCmd_ValidFile(t *testing.T) {
	yaml := `
version: "1.0.0"
metadata:
  name: test-mold
  displayName: Test Mold
  description: A test mold
  category: repository
steps:
  - id: s1
    name: Create repo
    action: github.repo.create
`
	tmpFile := writeTempFile(t, "valid-mold", yaml)

	validateMoldFormat = "text"
	validateMoldStrict = false
	cmd := validateMoldCmd
	err := cmd.RunE(cmd, []string{tmpFile})
	if err != nil {
		t.Errorf("expected no error for valid mold, got: %v", err)
	}
}

func TestValidateMoldCmd_InvalidFile(t *testing.T) {
	yaml := `
version: ""
metadata:
  name: ""
`
	tmpFile := writeTempFile(t, "invalid-mold", yaml)

	validateMoldFormat = "text"
	validateMoldStrict = false
	cmd := validateMoldCmd
	err := cmd.RunE(cmd, []string{tmpFile})
	if err == nil {
		t.Error("expected error for invalid mold")
	}
}

func TestValidateMoldCmd_NonexistentFile(t *testing.T) {
	validateMoldFormat = "text"
	validateMoldStrict = false
	cmd := validateMoldCmd
	err := cmd.RunE(cmd, []string{"/nonexistent/path/mold.yaml"})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestValidateMoldCmd_JSONOutput(t *testing.T) {
	yaml := `
version: "1.0.0"
metadata:
  name: test
  displayName: Test
  description: Desc
  category: service
steps:
  - id: s1
    name: Step
    action: github.repo.create
`
	tmpFile := writeTempFile(t, "json-mold", yaml)

	validateMoldFormat = "json"
	validateMoldStrict = false
	cmd := validateMoldCmd
	err := cmd.RunE(cmd, []string{tmpFile})
	if err != nil {
		t.Errorf("expected no error for valid mold with JSON output, got: %v", err)
	}
}

func TestValidateMoldCmd_StrictMode_WarningsBecomErrors(t *testing.T) {
	// Missing displayName/description/category should fail in strict mode
	yaml := `
version: "1.0.0"
metadata:
  name: test
steps:
  - id: s1
    name: Step
    action: github.repo.create
`
	tmpFile := writeTempFile(t, "strict-mold", yaml)

	validateMoldFormat = "text"
	validateMoldStrict = true
	cmd := validateMoldCmd
	err := cmd.RunE(cmd, []string{tmpFile})
	if err == nil {
		t.Error("expected error in strict mode when warnings exist")
	}
}

func TestValidateMoldCmd_StdinSupport(t *testing.T) {
	yaml := `
version: "1.0.0"
metadata:
  name: test
  displayName: Test
  description: Desc
  category: service
actions:
  - action: github.repo.create
    label: Create
    primary: true
`
	// Write to a temp file and replace stdin
	tmpFile := writeTempFile(t, "stdin-mold", yaml)
	f, err := os.Open(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	oldStdin := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = oldStdin }()

	validateMoldFormat = "text"
	validateMoldStrict = false
	cmd := validateMoldCmd
	runErr := cmd.RunE(cmd, []string{"-"})
	if runErr != nil {
		t.Errorf("expected no error for valid mold from stdin, got: %v", runErr)
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name+".yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
