package commands

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

// ─── search command, end to end ─────────────────────────────────────────────

// TestSearch_TableShowsEntityType is the end-to-end regression test for
// shoehorn-dev/shoehorn-cli#52: `shoehorn search` must print the entity's
// specific type in the Type column, not the generic "entity" category, for a
// hit that carries entity_type. The response body here is the real shape
// shoehorn-dev/shoehorn-platform's /api/v1/search now returns (captured live
// against a local stack while fixing the platform side), not a synthetic
// shortcut.
func TestSearch_TableShowsEntityType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": [
				{
					"id": "k8s-checkout-deployment-api",
					"type": "entity",
					"title": "api",
					"description": "",
					"score": 0.79,
					"entity_type": "deployment"
				}
			],
			"total": 1
		}`))
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	var runErr error
	printed := captureStdout(t, func() {
		_, runErr = runRoot(t, "search", "checkout", "-I")
	})
	if runErr != nil {
		t.Fatalf("search: %v", runErr)
	}

	// Find the data row (the one starting with the result name) and check its
	// Type cell directly, rather than substring-matching the whole table: the
	// NAME column holds "api" too, so a loose Contains("deployment") check
	// would pass even if displayType were wired to the wrong field.
	var dataLine string
	for _, line := range strings.Split(printed, "\n") {
		if strings.HasPrefix(line, "api") {
			dataLine = line
			break
		}
	}
	if dataLine == "" {
		t.Fatalf("no data row starting with \"api\" found in table output:\n%s", printed)
	}
	fields := strings.Fields(dataLine)
	if len(fields) < 2 {
		t.Fatalf("data row has too few columns: %q", dataLine)
	}
	if got := fields[1]; got != "deployment" {
		t.Errorf("Type column = %q, want %q (the entity's specific type, not the generic category); row: %q", got, "deployment", dataLine)
	}
}

// ─── displayType ────────────────────────────────────────────────────────────

// TestDisplayType pins the fix for shoehorn-dev/shoehorn-cli#52: the Type
// column (and the TUI detail fallback) must show the entity's specific type
// when the search API provides one, instead of always showing the generic
// "entity" category.
func TestDisplayType(t *testing.T) {
	tests := []struct {
		name string
		hit  api.SearchHit
		want string
	}{
		{
			name: "entity hit with EntityType shows the specific type",
			hit:  api.SearchHit{Type: "entity", EntityType: "statefulset"},
			want: "statefulset",
		},
		{
			name: "entity hit without EntityType falls back to the category",
			hit:  api.SearchHit{Type: "entity"},
			want: "entity",
		},
		{
			name: "doc hit (never has EntityType) shows its own category",
			hit:  api.SearchHit{Type: "doc"},
			want: "doc",
		},
		{
			name: "empty EntityType string falls back to the category",
			hit:  api.SearchHit{Type: "entity", EntityType: ""},
			want: "entity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayType(tt.hit); got != tt.want {
				t.Errorf("displayType(%+v) = %q, want %q", tt.hit, got, tt.want)
			}
		})
	}
}
