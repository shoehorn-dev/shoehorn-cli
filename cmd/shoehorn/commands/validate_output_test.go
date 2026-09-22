package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
