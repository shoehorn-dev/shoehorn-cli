package browser

import (
	"strings"
	"testing"
)

func TestValidateURL_AcceptsHTTPAndHTTPS(t *testing.T) {
	for _, u := range []string{
		"https://github.com/acme/svc",
		"http://localhost:8080/runs/abc",
	} {
		if err := validateURL(u); err != nil {
			t.Errorf("validateURL(%q) = %v, want nil", u, err)
		}
	}
}

func TestValidateURL_RejectsUnsafeSchemes(t *testing.T) {
	for _, u := range []string{
		"javascript:alert(1)",
		"file:///etc/passwd",
		"ftp://example.com",
		"data:text/html,<script>",
		"",
		"not-a-url",
		"https://",
	} {
		if err := validateURL(u); err == nil {
			t.Errorf("validateURL(%q) = nil, want error", u)
		}
	}
}

func TestCommandArgs_PerPlatform(t *testing.T) {
	url := "https://github.com/acme/svc"

	tests := []struct {
		goos      string
		wantFirst string
	}{
		{"windows", "rundll32"},
		{"darwin", "open"},
		{"linux", "xdg-open"},
	}

	for _, tt := range tests {
		args := commandArgs(tt.goos, url)
		if len(args) < 2 {
			t.Fatalf("commandArgs(%q) = %v, want executable + url", tt.goos, args)
		}
		if args[0] != tt.wantFirst {
			t.Errorf("commandArgs(%q)[0] = %q, want %q", tt.goos, args[0], tt.wantFirst)
		}
		if args[len(args)-1] != url {
			t.Errorf("commandArgs(%q) last arg = %q, want the url", tt.goos, args[len(args)-1])
		}
	}
}

func TestOpen_RejectsUnsafeURLWithoutLaunching(t *testing.T) {
	err := Open("javascript:alert(1)")
	if err == nil {
		t.Fatal("Open must refuse non-http(s) URLs")
	}
	if !strings.Contains(err.Error(), "http") {
		t.Errorf("error should explain the scheme restriction, got: %v", err)
	}
}
