package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// captureStdout returns everything written to os.Stdout while fn runs, so the
// assertion holds whether the command prints through cobra's writer or fmt.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

// Every other verb honours the root -o flag; validate has a private --format
// flag and prints "✓ <file> is valid" whatever -o says, so a CI step asking
// for JSON has nothing to parse.
func TestValidate_HonoursGlobalJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/manifests/validate" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"valid":true,"errors":[]}`))
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	file := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(file, []byte("schemaVersion: 1\nservice:\n  id: a\n  name: A\n  type: service\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	var cobraOut string
	var runErr error
	printed := captureStdout(t, func() {
		cobraOut, runErr = runRoot(t, "validate", file, "-o", "json")
	})
	if runErr != nil {
		t.Fatalf("validate -o json: %v", runErr)
	}

	out := cobraOut
	if out == "" {
		out = printed
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("validate -o json must print JSON, got %q (%v)", out, err)
	}
	if parsed["valid"] != true {
		t.Errorf("valid = %v, want true", parsed["valid"])
	}
}

// resetFormatFlags puts the shared command tree back to its startup state:
// one process runs one command, but the tests share validateCmd, so a
// `--format` from an earlier run would otherwise still read as "the user
// asked for it".
func resetFormatFlags(t *testing.T) {
	t.Helper()
	reset := func(cmd *cobra.Command) {
		if f := cmd.Flags().Lookup("format"); f != nil {
			f.Changed = false
		}
	}
	restore := func() {
		validateFormat, validateMoldFormat, outputFormat = "text", "text", ""
		reset(validateCmd)
		reset(validateMoldCmd)
	}
	restore()
	t.Cleanup(restore)
}

func TestValidateMold_HonoursGlobalJSONOutput(t *testing.T) {
	resetFormatFlags(t)
	file := filepath.Join(t.TempDir(), "mold.yaml")
	mold := "version: \"1.0.0\"\nmetadata:\n  name: test-mold\n  displayName: Test Mold\n  description: A test mold\n  category: repository\nsteps:\n  - id: s1\n    name: Create repo\n    action: github.repo.create\n"
	if err := os.WriteFile(file, []byte(mold), 0o600); err != nil {
		t.Fatalf("write mold: %v", err)
	}

	var cobraOut string
	var runErr error
	printed := captureStdout(t, func() {
		cobraOut, runErr = runRoot(t, "validate", "mold", file, "-o", "json")
	})
	if runErr != nil {
		t.Fatalf("validate mold -o json: %v", runErr)
	}

	out := cobraOut
	if out == "" {
		out = printed
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("validate mold -o json must print JSON, got %q (%v)", out, err)
	}
	if parsed["valid"] != true {
		t.Errorf("valid = %v, want true", parsed["valid"])
	}
}

// An explicit --format wins: -o names the shape of a command's data output,
// and someone who asked this verb for text gets text.
func TestValidateMold_ExplicitFormatBeatsGlobalOutput(t *testing.T) {
	resetFormatFlags(t)
	file := filepath.Join(t.TempDir(), "mold.yaml")
	mold := "version: \"1.0.0\"\nmetadata:\n  name: test-mold\n  displayName: Test Mold\n  description: A test mold\n  category: repository\nsteps:\n  - id: s1\n    name: Create repo\n    action: github.repo.create\n"
	if err := os.WriteFile(file, []byte(mold), 0o600); err != nil {
		t.Fatalf("write mold: %v", err)
	}

	printed := captureStdout(t, func() {
		if _, err := runRoot(t, "validate", "mold", file, "-o", "json", "--format", "text"); err != nil {
			t.Fatalf("validate mold --format text: %v", err)
		}
	})
	if json.Valid([]byte(printed)) && printed != "" {
		t.Errorf("--format text must win over -o json, got JSON: %q", printed)
	}
}
