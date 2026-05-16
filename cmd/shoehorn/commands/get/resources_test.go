package get

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
)

// resetResourcesFlags clears the package-level flag vars so tests don't leak
// state into each other.
func resetResourcesFlags() {
	resourcesCluster = ""
	resourcesTeam = ""
	resourcesKind = ""
	resourcesStatus = ""
	resourcesSearch = ""
	resourcesHasGitOps = false
	resourcesCursor = ""
	resourcesLimit = 0
}

// ─── Command registration + flags ──────────────────────────────────────────

func TestResourcesCmd_IsRegistered(t *testing.T) {
	found := false
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "resources" {
			found = true
			break
		}
	}
	if !found {
		t.Error("resources subcommand not registered on GetCmd")
	}
}

func TestResourcesCmd_HasFilterFlags(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "resources" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("resources subcommand not found")
	}

	flags := []string{"cluster", "team", "kind", "status", "search", "has-gitops", "cursor", "limit"}
	for _, name := range flags {
		t.Run(name, func(t *testing.T) {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("resources command must have --%s flag", name)
			}
		})
	}
}

func TestResourcesCmd_AcceptsOptionalArg(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "resources" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("resources subcommand not found")
	}
	if cmd.Args == nil {
		t.Error("resources command must have Args validator (MaximumNArgs(1))")
	}
}

// ─── Error path: no config ─────────────────────────────────────────────────

func TestRunGetResources_NoConfig_ReturnsError(t *testing.T) {
	resetResourcesFlags()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	if err := runGetResources(testCmd(), nil); err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetResourceDetail_NoConfig_ReturnsError(t *testing.T) {
	resetResourcesFlags()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	if err := runGetResources(testCmd(), []string{"rs_abc"}); err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

// ─── Mock server: list ─────────────────────────────────────────────────────

func TestRunGetResources_MockServer_RendersList(t *testing.T) {
	resetResourcesFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/operations/resources" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"resources": []map[string]any{
				{
					"id": "rs_1", "name": "checkout", "subtype": "Deployment", "status": "Healthy",
					"clusters": []map[string]any{
						{"cluster_id": "prod", "namespace": "shop", "replicas_ready": 3, "replicas_desired": 3},
					},
				},
			},
			"next_cursor": "", "total": 1,
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	if err := runGetResources(testCmd(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGetResources_MockServer_EmptyList(t *testing.T) {
	resetResourcesFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"resources": []any{}, "next_cursor": "", "total": 0})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	if err := runGetResources(testCmd(), nil); err != nil {
		t.Fatalf("unexpected error for empty list: %v", err)
	}
}

// TestRunGetResources_FilterFlags_Passthrough verifies every filter flag is
// forwarded to the API as a query parameter.
func TestRunGetResources_FilterFlags_Passthrough(t *testing.T) {
	resetResourcesFlags()
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"resources": []any{}, "next_cursor": "", "total": 0})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	resourcesCluster = "prod-eu"
	resourcesTeam = "team-shop"
	resourcesKind = "Deployment"
	resourcesStatus = "Failing"
	resourcesSearch = "checkout"
	resourcesHasGitOps = true
	resourcesLimit = 10
	defer resetResourcesFlags()

	if err := runGetResources(testCmd(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{
		"cluster=prod-eu", "team=team-shop", "kind=Deployment",
		"status=Failing", "search=checkout", "has_gitops=true", "limit=10",
	} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}

// TestRunGetResources_PaginationFollowsCursor verifies the --cursor flag is
// forwarded so callers can page through results.
func TestRunGetResources_PaginationFollowsCursor(t *testing.T) {
	resetResourcesFlags()
	var gotCursor string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCursor = r.URL.Query().Get("cursor")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"resources":   []map[string]any{{"id": "rs_2", "name": "two"}},
			"next_cursor": "", "total": 2,
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	resourcesCursor = "cur_page2"
	defer resetResourcesFlags()

	if err := runGetResources(testCmd(), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCursor != "cur_page2" {
		t.Errorf("cursor forwarded = %q, want %q", gotCursor, "cur_page2")
	}
}

// ─── Mock server: detail ───────────────────────────────────────────────────

func TestRunGetResourceDetail_MockServer_Found(t *testing.T) {
	resetResourcesFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"resource": map[string]any{
				"id": "rs_abc", "name": "checkout", "subtype": "Deployment", "status": "Healthy",
			},
			"events": []any{},
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	if err := runGetResources(testCmd(), []string{"rs_abc"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGetResourceDetail_MockServer_NotFound(t *testing.T) {
	resetResourcesFlags()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "not found"},
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	err := runGetResources(testCmd(), []string{"rs_missing"})
	if err == nil {
		t.Fatal("expected error for not-found resource, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
	if !strings.Contains(err.Error(), "Hint") {
		t.Errorf("not-found error should include a Hint, got: %v", err)
	}
}

// ─── Display helpers ───────────────────────────────────────────────────────

func TestResourceReplicas_SumsAcrossClusters(t *testing.T) {
	r := &api.OperationsResource{
		Clusters: []api.OperationsResourceCluster{
			{ReplicasReady: 3, ReplicasDesired: 3},
			{ReplicasReady: 2, ReplicasDesired: 4},
		},
	}
	if got := resourceReplicas(r); got != "5/7" {
		t.Errorf("resourceReplicas = %q, want %q", got, "5/7")
	}
}

func TestResourceReplicas_NoClusters(t *testing.T) {
	if got := resourceReplicas(&api.OperationsResource{}); got != "0/0" {
		t.Errorf("resourceReplicas = %q, want %q", got, "0/0")
	}
}

func TestResourceOwnerName(t *testing.T) {
	cases := []struct {
		name string
		r    *api.OperationsResource
		want string
	}{
		{"with owner", &api.OperationsResource{Owner: &api.OperationsResourceOwner{TeamName: "team-shop"}}, "team-shop"},
		{"nil owner", &api.OperationsResource{}, "—"},
		{"empty team name", &api.OperationsResource{Owner: &api.OperationsResourceOwner{}}, "—"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resourceOwnerName(tc.r); got != tc.want {
				t.Errorf("resourceOwnerName = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResourceDisplayName(t *testing.T) {
	cases := []struct {
		name string
		r    *api.OperationsResource
		want string
	}{
		{"name wins", &api.OperationsResource{Name: "checkout", Identity: api.OperationsResourceIdentity{NameHint: "co"}}, "checkout"},
		{"falls back to name hint", &api.OperationsResource{Identity: api.OperationsResourceIdentity{NameHint: "co"}}, "co"},
		{"falls back to id", &api.OperationsResource{ID: "rs_1"}, "rs_1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resourceDisplayName(tc.r); got != tc.want {
				t.Errorf("resourceDisplayName = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOrDash(t *testing.T) {
	if got := orDash(""); got != "—" {
		t.Errorf("orDash(\"\") = %q, want em-dash", got)
	}
	if got := orDash("value"); got != "value" {
		t.Errorf("orDash(\"value\") = %q, want %q", got, "value")
	}
}

func TestFormatResourceLastSeen(t *testing.T) {
	if got := formatResourceLastSeen(nil); got != "never" {
		t.Errorf("formatResourceLastSeen(nil) = %q, want %q", got, "never")
	}
	now := time.Now()
	if got := formatResourceLastSeen(&now); got != "just now" {
		t.Errorf("formatResourceLastSeen(now) = %q, want %q", got, "just now")
	}
	past := time.Now().Add(-3 * time.Hour)
	if got := formatResourceLastSeen(&past); got != "3h ago" {
		t.Errorf("formatResourceLastSeen(-3h) = %q, want %q", got, "3h ago")
	}
	old := time.Now().Add(-5 * 24 * time.Hour)
	if got := formatResourceLastSeen(&old); got != "5d ago" {
		t.Errorf("formatResourceLastSeen(-5d) = %q, want %q", got, "5d ago")
	}
}
