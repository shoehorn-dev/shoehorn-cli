package commands

import (
	"strings"
	"testing"
)

func TestExtractOutputLinks_EmptyOutputs(t *testing.T) {
	if links := extractOutputLinks(nil); len(links) != 0 {
		t.Errorf("nil outputs should yield no links, got %v", links)
	}
	if links := extractOutputLinks(map[string]any{}); len(links) != 0 {
		t.Errorf("empty outputs should yield no links, got %v", links)
	}
}

func TestExtractOutputLinks_PrimaryKeyOrdering(t *testing.T) {
	outputs := map[string]any{
		"clone_url": "https://github.com/acme/svc.git",
		"html_url":  "https://github.com/acme/svc",
		"name":      "svc",
	}

	links := extractOutputLinks(outputs)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(links), links)
	}
	if links[0].Key != "html_url" {
		t.Errorf("html_url should be the primary link, got %q first", links[0].Key)
	}
	if links[0].URL != "https://github.com/acme/svc" {
		t.Errorf("primary URL = %q", links[0].URL)
	}
}

func TestExtractOutputLinks_IgnoresNonURLValues(t *testing.T) {
	outputs := map[string]any{
		"name":          "my-repo",
		"files_created": 3,
		"enabled":       true,
		"javascript":    "javascript:alert(1)",
		"ftp":           "ftp://example.com/file",
		"relative":      "/just/a/path",
	}

	if links := extractOutputLinks(outputs); len(links) != 0 {
		t.Errorf("non-http(s) values must not become links, got %v", links)
	}
}

func TestExtractOutputLinks_NestedMapsAndDedupe(t *testing.T) {
	outputs := map[string]any{
		"html_url": "https://github.com/acme/svc",
		"steps": map[string]any{
			"create_repo": map[string]any{
				"html_url": "https://github.com/acme/svc",
				"pr_url":   "https://github.com/acme/svc/pull/1",
			},
		},
	}

	links := extractOutputLinks(outputs)
	if len(links) != 2 {
		t.Fatalf("expected 2 deduped links, got %d: %v", len(links), links)
	}
	seen := map[string]bool{}
	for _, l := range links {
		seen[l.URL] = true
	}
	if !seen["https://github.com/acme/svc"] || !seen["https://github.com/acme/svc/pull/1"] {
		t.Errorf("missing expected URLs, got %v", links)
	}
}

func TestExtractOutputLinks_RejectsURLsWithUserinfo(t *testing.T) {
	outputs := map[string]any{
		"clone_url": "https://x-access-token:ghs_secret123@github.com/acme/svc.git",
		"trick_url": "https://github.com@evil.example/acme/svc",
	}

	if links := extractOutputLinks(outputs); len(links) != 0 {
		t.Errorf("URLs with userinfo can hide credentials or spoof hosts and must be skipped, got %v", links)
	}
}

func TestExtractOutputLinks_RecursesIntoArrays(t *testing.T) {
	outputs := map[string]any{
		"pull_requests": []any{
			map[string]any{"pr_url": "https://github.com/acme/svc/pull/1"},
			map[string]any{"pr_url": "https://github.com/acme/svc/pull/2"},
		},
	}

	links := extractOutputLinks(outputs)
	if len(links) != 2 {
		t.Errorf("URLs inside arrays should be collected, got %v", links)
	}
}

func TestIsSecretOutputKey(t *testing.T) {
	secret := []string{"token", "github_token", "API_KEY", "apiKey", "webhook_secret", "password", "credentials"}
	for _, k := range secret {
		if !isSecretOutputKey(k) {
			t.Errorf("isSecretOutputKey(%q) = false, want true", k)
		}
	}
	plain := []string{"name", "owner", "files_created", "branch"}
	for _, k := range plain {
		if isSecretOutputKey(k) {
			t.Errorf("isSecretOutputKey(%q) = true, want false", k)
		}
	}
}

func TestSanitizeTerminal_StripsControlSequences(t *testing.T) {
	in := "repo created\x1b[2K\x1b]0;owned\x07 done\r"
	got := sanitizeTerminal(in)
	for _, r := range got {
		if r < 0x20 && r != '\n' && r != '\t' {
			t.Fatalf("sanitizeTerminal left control char %q in %q", r, got)
		}
	}
	if !strings.Contains(got, "repo created") || !strings.Contains(got, "done") {
		t.Errorf("sanitizeTerminal should keep printable text, got %q", got)
	}
}

func TestExtractOutputLinks_DeterministicOrder(t *testing.T) {
	outputs := map[string]any{
		"zeta_url":  "https://example.com/z",
		"alpha_url": "https://example.com/a",
	}

	for range 10 {
		links := extractOutputLinks(outputs)
		if len(links) != 2 {
			t.Fatalf("expected 2 links, got %v", links)
		}
		if links[0].Key != "alpha_url" || links[1].Key != "zeta_url" {
			t.Fatalf("order must be deterministic (alphabetical after primaries), got %v", links)
		}
	}
}
