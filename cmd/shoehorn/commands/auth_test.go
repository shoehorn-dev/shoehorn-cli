package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNormalizeServerURL tests URL normalization with table-driven tests.
func TestNormalizeServerURL_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"http preserved", "http://localhost:8080", "http://localhost:8080"},
		{"https preserved", "https://api.shoehorn.dev", "https://api.shoehorn.dev"},
		{"adds https when no scheme", "api.shoehorn.dev", "https://api.shoehorn.dev"},
		{"strips single trailing slash", "http://localhost:8080/", "http://localhost:8080"},
		{"strips multiple trailing slashes", "http://localhost:8080///", "http://localhost:8080"},
		{"no-scheme with trailing slash", "api.shoehorn.dev/", "https://api.shoehorn.dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeServerURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeServerURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestHasScheme tests scheme detection.
func TestHasScheme_TableDriven(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"http://localhost", true},
		{"https://api.example.com", true},
		{"api.example.com", false},
		{"ftp://other", false},
		{"", false},
		{"http://", true},
		{"h", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := hasScheme(tt.input)
			if got != tt.want {
				t.Errorf("hasScheme(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestFormatDuration tests human-readable duration formatting.
func TestFormatDuration_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		contains string
	}{
		{"seconds", 30, "seconds"},
		{"minutes", 300, "minutes"},
		{"hours", 7200, "hours"},
		{"days", 172800, "days"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := formatDuration(time.Duration(tt.seconds) * time.Second)
			if !strings.Contains(d, tt.contains) {
				t.Errorf("formatDuration(%ds) = %q, want to contain %q", tt.seconds, d, tt.contains)
			}
		})
	}
}

// ─── Security: SHOEHORN_TOKEN env var ────────────────────────────────────────

// TestResolveToken_EnvVar tests that SHOEHORN_TOKEN env var is picked up.
func TestResolveToken_EnvVar(t *testing.T) {
	t.Setenv("SHOEHORN_TOKEN", "shp_from_env")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	token, source, err := resolveToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "shp_from_env" {
		t.Errorf("resolveToken() token = %q, want %q", token, "shp_from_env")
	}
	if source != "env" {
		t.Errorf("resolveToken() source = %q, want %q", source, "env")
	}
}

// TestResolveToken_FlagOverridesEnv tests that --token flag overrides env var.
func TestResolveToken_FlagOverridesEnv(t *testing.T) {
	t.Setenv("SHOEHORN_TOKEN", "shp_from_env")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	token, source, err := resolveToken("shp_from_flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "shp_from_flag" {
		t.Errorf("resolveToken() token = %q, want %q", token, "shp_from_flag")
	}
	if source != "flag" {
		t.Errorf("resolveToken() source = %q, want %q", source, "flag")
	}
}

// TestResolveToken_Empty tests that empty flag + no env returns empty.
func TestResolveToken_Empty(t *testing.T) {
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	token, source, err := resolveToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "" {
		t.Errorf("resolveToken() token = %q, want empty", token)
	}
	if source != "none" {
		t.Errorf("resolveToken() source = %q, want %q", source, "none")
	}
}

// ─── Security: SHOEHORN_TOKEN_FILE ───────────────────────────────────────────

// TestResolveToken_TokenFile reads token from file pointed to by SHOEHORN_TOKEN_FILE.
func TestResolveToken_TokenFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tmpFile, []byte("shp_from_file\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHOEHORN_TOKEN_FILE", tmpFile)
	t.Setenv("SHOEHORN_TOKEN", "")

	token, source, err := resolveToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "shp_from_file" {
		t.Errorf("resolveToken() token = %q, want %q", token, "shp_from_file")
	}
	if source != "file" {
		t.Errorf("resolveToken() source = %q, want %q", source, "file")
	}
}

// TestResolveToken_TokenFilePriority verifies SHOEHORN_TOKEN_FILE > SHOEHORN_TOKEN.
func TestResolveToken_TokenFilePriority(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tmpFile, []byte("shp_from_file"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHOEHORN_TOKEN_FILE", tmpFile)
	t.Setenv("SHOEHORN_TOKEN", "shp_from_env")

	token, source, err := resolveToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "shp_from_file" {
		t.Errorf("resolveToken() token = %q, want %q (file takes priority over env)", token, "shp_from_file")
	}
	if source != "file" {
		t.Errorf("resolveToken() source = %q, want %q", source, "file")
	}
}

// TestResolveToken_FlagOverridesFile verifies --token flag > SHOEHORN_TOKEN_FILE.
func TestResolveToken_FlagOverridesFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tmpFile, []byte("shp_from_file"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHOEHORN_TOKEN_FILE", tmpFile)
	token, source, err := resolveToken("shp_from_flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "shp_from_flag" {
		t.Errorf("resolveToken() token = %q, want %q (flag takes priority)", token, "shp_from_flag")
	}
	if source != "flag" {
		t.Errorf("resolveToken() source = %q, want %q", source, "flag")
	}
}

// TestResolveToken_TokenFileNotFound verifies that a missing file returns an error
// (the user explicitly configured SHOEHORN_TOKEN_FILE, so missing is not silent).
func TestResolveToken_TokenFileNotFound(t *testing.T) {
	t.Setenv("SHOEHORN_TOKEN_FILE", "/nonexistent/path/token")
	t.Setenv("SHOEHORN_TOKEN", "")

	_, _, err := resolveToken("")
	if err == nil {
		t.Error("expected error when SHOEHORN_TOKEN_FILE points to missing file")
	}
}

// TestResolveToken_TokenFileTrimsWhitespace verifies trailing newlines are stripped.
func TestResolveToken_TokenFileTrimsWhitespace(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tmpFile, []byte("  shp_with_spaces  \n\n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHOEHORN_TOKEN_FILE", tmpFile)
	t.Setenv("SHOEHORN_TOKEN", "")

	token, _, err := resolveToken("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "shp_with_spaces" {
		t.Errorf("resolveToken() token = %q, want %q (should trim whitespace)", token, "shp_with_spaces")
	}
}

// TestResolveToken_TokenFileEmpty verifies that an empty file returns an error.
func TestResolveToken_TokenFileEmpty(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tmpFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHOEHORN_TOKEN_FILE", tmpFile)
	t.Setenv("SHOEHORN_TOKEN", "")

	_, _, err := resolveToken("")
	if err == nil {
		t.Error("expected error when token file is empty")
	}
	if err != nil && !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
}

// TestResolveToken_TokenFileWhitespaceOnly verifies that a file with only whitespace
// returns an error (treated as empty after trimming).
func TestResolveToken_TokenFileWhitespaceOnly(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tmpFile, []byte("  \n\n  \n"), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHOEHORN_TOKEN_FILE", tmpFile)
	t.Setenv("SHOEHORN_TOKEN", "")

	_, _, err := resolveToken("")
	if err == nil {
		t.Error("expected error when token file contains only whitespace")
	}
}

// TestResolveToken_TokenFileIsDirectory verifies that pointing at a directory errors.
func TestResolveToken_TokenFileIsDirectory(t *testing.T) {
	t.Setenv("SHOEHORN_TOKEN_FILE", t.TempDir())
	t.Setenv("SHOEHORN_TOKEN", "")

	_, _, err := resolveToken("")
	if err == nil {
		t.Error("expected error when SHOEHORN_TOKEN_FILE points to a directory")
	}
	if err != nil && !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %v", err)
	}
}

// TestResolveToken_TokenFileTooLarge verifies that oversized token files are rejected.
func TestResolveToken_TokenFileTooLarge(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "token")
	data := make([]byte, maxTokenFileSize+1)
	for i := range data {
		data[i] = 'x'
	}
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SHOEHORN_TOKEN_FILE", tmpFile)
	t.Setenv("SHOEHORN_TOKEN", "")

	_, _, err := resolveToken("")
	if err == nil {
		t.Error("expected error for oversized token file")
	}
}

// ─── Security: HTTP safety validation ────────────────────────────────────────

// TestValidateServerSecurity tests HTTP vs HTTPS validation for remote servers.
func TestValidateServerSecurity_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https remote is safe", "https://api.company.com", false},
		{"http localhost allowed", "http://localhost:8080", false},
		{"http 127.0.0.1 allowed", "http://127.0.0.1:8080", false},
		{"http [::1] allowed", "http://[::1]:8080", false},
		{"http remote blocked", "http://api.company.com", true},
		{"http remote with port blocked", "http://api.company.com:8080", true},
		{"empty string no error", "", false},
		{"unsupported scheme rejected", "ftp://evil.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerSecurity(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServerSecurity(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

// TestRunLoginWithPAT_ErrorReturnsNonNil is a regression test for the error
// swallowing bug where runLoginWithPAT returned nil on API failure, causing
// exit code 0 instead of non-zero.
func TestRunLoginWithPAT_ErrorReturnsNonNil(t *testing.T) {
	err := runLoginWithPAT(context.Background(), "http://127.0.0.1:1", "fake-token")
	if err == nil {
		t.Error("runLoginWithPAT with unreachable server returned nil error; want non-nil for correct exit code")
	}
}
