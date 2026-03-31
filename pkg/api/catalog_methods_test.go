package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/config"
)

// newTestClient creates a client pointed at the given test server.
func newTestClient(ts *httptest.Server) *Client {
	c := NewClient(ts.URL)
	c.SetToken("test-token")
	return c
}

// --- GetMe ---

func TestGetMe_Success_FullProfile(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/me" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":        "user-123",
			"email":     "jane@example.com",
			"firstName": "Jane",
			"lastName":  "Smith",
			"tenant":    "acme-corp",
			"roles":     []string{"admin"},
			"groups":    []string{"engineering"},
			"teams":     []string{"platform"},
		})
	}))
	defer ts.Close()

	me, err := newTestClient(ts).GetMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.Email != "jane@example.com" {
		t.Errorf("Email = %q, want jane@example.com", me.Email)
	}
	if me.Name != "Jane Smith" {
		t.Errorf("Name = %q, want 'Jane Smith'", me.Name)
	}
	if me.TenantID != "acme-corp" {
		t.Errorf("TenantID = %q, want acme-corp", me.TenantID)
	}
	if len(me.Roles) != 1 || me.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin]", me.Roles)
	}
}

func TestGetMe_NumericEmail_Cleared(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "12345",
			"email": "12345", // Not a real email
			"user":  "jsmith",
		})
	}))
	defer ts.Close()

	me, err := newTestClient(ts).GetMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.Email != "" {
		t.Errorf("Email should be empty for numeric value, got %q", me.Email)
	}
	if me.Name != "jsmith" {
		t.Errorf("Name should fall back to user field, got %q", me.Name)
	}
}

func TestGetMe_NameFallsBackToUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "u-1",
			"user": "jdoe",
		})
	}))
	defer ts.Close()

	me, err := newTestClient(ts).GetMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.Name != "jdoe" {
		t.Errorf("Name = %q, want 'jdoe' (user field fallback)", me.Name)
	}
}

func TestGetMe_ServerError_ReturnsWrappedError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "internal failure"},
		})
	}))
	defer ts.Close()

	_, err := newTestClient(ts).GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !errors.Is(err, ErrServerError) {
		t.Errorf("expected ErrServerError, got %v", err)
	}
}

// --- ListEntities ---

func TestListEntities_Success_ParsesEntities(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") != "service" {
			t.Errorf("type filter not passed: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("limit not set: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"entities": []map[string]any{
				{
					"service":     map[string]any{"id": "svc-1", "name": "payment-service", "type": "service"},
					"owner":       []map[string]any{{"id": "platform-team", "type": "team"}},
					"description": "Handles payments",
					"tags":        []string{"payments"},
				},
			},
			"page": map[string]any{"total": 1, "nextCursor": ""},
		})
	}))
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{Type: "service"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 1 {
		t.Fatalf("got %d entities, want 1", len(entities))
	}
	if entities[0].Name != "payment-service" {
		t.Errorf("Name = %q, want payment-service", entities[0].Name)
	}
	if entities[0].Owner != "platform-team" {
		t.Errorf("Owner = %q, want platform-team", entities[0].Owner)
	}
}

func TestListEntities_Empty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"entities": []any{},
			"page":     map[string]any{"total": 0},
		})
	}))
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 0 {
		t.Errorf("got %d entities, want 0", len(entities))
	}
}

func TestListEntities_401_ReturnsNotAuthenticated(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "invalid token"},
		})
	}))
	defer ts.Close()

	_, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// --- GetEntity ---

func TestGetEntity_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/entities/svc-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"entity": map[string]any{
				"service":     map[string]any{"id": "svc-1", "name": "payment-service", "type": "service"},
				"owner":       "platform-team", // test string fallback
				"description": "Payments",
				"lifecycle":   "production",
				"links":       []map[string]any{{"name": "GitHub", "url": "https://github.com/example"}},
			},
		})
	}))
	defer ts.Close()

	entity, err := newTestClient(ts).GetEntity(context.Background(), "svc-1")
	if err != nil {
		t.Fatal(err)
	}
	if entity.Name != "payment-service" {
		t.Errorf("Name = %q", entity.Name)
	}
	if entity.Owner != "platform-team" {
		t.Errorf("Owner = %q, want platform-team (string fallback)", entity.Owner)
	}
	if entity.Lifecycle != "production" {
		t.Errorf("Lifecycle = %q", entity.Lifecycle)
	}
	if len(entity.Links) != 1 || entity.Links[0].Title != "GitHub" {
		t.Errorf("Links = %+v", entity.Links)
	}
}

// --- Search ---

func TestSearch_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "payment" {
			t.Errorf("query not passed: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"id": "svc-1", "title": "payment-service", "type": "service", "score": 0.95},
			},
			"page": map[string]any{"total": 1},
		})
	}))
	defer ts.Close()

	sr, err := newTestClient(ts).Search(context.Background(), "payment")
	if err != nil {
		t.Fatal(err)
	}
	if sr.TotalCount != 1 {
		t.Errorf("TotalCount = %d, want 1", sr.TotalCount)
	}
	if sr.Hits[0].Name != "payment-service" {
		t.Errorf("Hit name = %q, want payment-service", sr.Hits[0].Name)
	}
}

// --- Teams ---

func TestListTeams_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"teams": []map[string]any{
				{"id": "t-1", "name": "Platform", "slug": "platform", "member_count": 5},
			},
		})
	}))
	defer ts.Close()

	teams, err := newTestClient(ts).ListTeams(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 1 || teams[0].Name != "Platform" {
		t.Errorf("teams = %+v", teams)
	}
}

func TestGetTeam_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"team":    map[string]any{"id": "t-1", "name": "Platform"},
			"members": []map[string]any{{"id": "u-1", "email": "jane@example.com", "role": "lead"}},
		})
	}))
	defer ts.Close()

	td, err := newTestClient(ts).GetTeam(context.Background(), "platform")
	if err != nil {
		t.Fatal(err)
	}
	if td.Name != "Platform" {
		t.Errorf("Name = %q", td.Name)
	}
	if len(td.Members) != 1 || td.Members[0].Email != "jane@example.com" {
		t.Errorf("Members = %+v", td.Members)
	}
}

// --- Users ---

func TestListUsers_NameFromFirstLast(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "u-1", "firstName": "Jane", "lastName": "Smith", "email": "jane@example.com"},
				{"id": "u-2", "username": "jdoe"}, // no first/last name
			},
		})
	}))
	defer ts.Close()

	users, err := newTestClient(ts).ListUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}
	if users[0].Name != "Jane Smith" {
		t.Errorf("users[0].Name = %q, want 'Jane Smith'", users[0].Name)
	}
	if users[1].Name != "jdoe" {
		t.Errorf("users[1].Name = %q, want 'jdoe' (username fallback)", users[1].Name)
	}
}

func TestGetUser_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": "u-1", "firstName": "Jane", "lastName": "Smith",
			"email":  "jane@example.com",
			"groups": []string{"eng"}, "teams": []string{"platform"}, "roles": []string{"admin"},
		})
	}))
	defer ts.Close()

	ud, err := newTestClient(ts).GetUser(context.Background(), "u-1")
	if err != nil {
		t.Fatal(err)
	}
	if ud.Name != "Jane Smith" {
		t.Errorf("Name = %q", ud.Name)
	}
	if len(ud.Groups) != 1 || ud.Groups[0] != "eng" {
		t.Errorf("Groups = %v", ud.Groups)
	}
}

// --- Groups ---

func TestListGroups_CountsRoles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"name":  "engineering",
					"roles": []map[string]any{{"name": "admin"}, {"name": "viewer"}},
				},
			},
		})
	}))
	defer ts.Close()

	groups, err := newTestClient(ts).ListGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].RoleCount != 2 {
		t.Errorf("groups = %+v, want RoleCount=2", groups)
	}
}

func TestGetGroupRoles_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"roles": []map[string]any{
				{"name": "admin", "description": "Full access"},
			},
		})
	}))
	defer ts.Close()

	roles, err := newTestClient(ts).GetGroupRoles(context.Background(), "engineering")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 || roles[0].Name != "admin" {
		t.Errorf("roles = %+v", roles)
	}
}

// --- K8s ---

func TestListK8sAgents_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"agents": []map[string]any{
				{"id": 1, "clusterId": "prod-east", "name": "agent-v1", "onlineStatus": "online"},
			},
		})
	}))
	defer ts.Close()

	agents, err := newTestClient(ts).ListK8sAgents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(agents))
	}
	if agents[0].ClusterName != "prod-east" {
		t.Errorf("ClusterName = %q", agents[0].ClusterName)
	}
	if agents[0].ID != "1" {
		t.Errorf("ID = %q, want '1' (converted from int)", agents[0].ID)
	}
}

// --- Forge ---

func TestListMolds_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"molds": []map[string]any{
				{"id": "m-1", "name": "Create Repo", "slug": "create-repo", "version": "1.0"},
			},
		})
	}))
	defer ts.Close()

	molds, err := newTestClient(ts).ListMolds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(molds) != 1 || molds[0].Slug != "create-repo" {
		t.Errorf("molds = %+v", molds)
	}
}

func TestGetMold_ParsesInputsFromSchema(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// GetMold expects {"mold": {...}} wrapper
		json.NewEncoder(w).Encode(map[string]any{
			"mold": map[string]any{
				"id": "m-1", "slug": "create-repo", "name": "Create Repo", "version": "1.0",
				"actions": []map[string]any{
					{"action": "create", "label": "Create", "primary": true},
				},
				"schema": map[string]any{
					"properties": map[string]any{
						"name":  map[string]any{"type": "string", "description": "Repo name"},
						"owner": map[string]any{"type": "string", "description": "Owner"},
					},
					"required": []any{"name"},
				},
				"inputOrder": []string{"name", "owner"},
				"defaults":   map[string]any{"owner": "my-org"},
			},
		})
	}))
	defer ts.Close()

	mold, err := newTestClient(ts).GetMold(context.Background(), "create-repo")
	if err != nil {
		t.Fatal(err)
	}
	if mold.Slug != "create-repo" {
		t.Errorf("Slug = %q", mold.Slug)
	}
	if len(mold.Actions) != 1 || !mold.Actions[0].Primary {
		t.Errorf("Actions = %+v", mold.Actions)
	}
	if len(mold.Inputs) != 2 {
		t.Fatalf("got %d inputs, want 2", len(mold.Inputs))
	}
	// Verify input order matches inputOrder field
	if mold.Inputs[0].Name != "name" {
		t.Errorf("first input = %q, want 'name'", mold.Inputs[0].Name)
	}
	if !mold.Inputs[0].Required {
		t.Error("'name' input should be required")
	}
	if mold.Inputs[1].Default != "my-org" {
		t.Errorf("'owner' default = %q, want 'my-org'", mold.Inputs[1].Default)
	}
}

func TestCreateRun_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["mold_slug"] != "create-repo" {
			t.Errorf("mold_slug = %v", body["mold_slug"])
		}
		if body["action"] != "create" {
			t.Errorf("action = %v", body["action"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{
			"run": map[string]any{
				"id": "run-123", "mold_slug": "create-repo", "action": "create", "status": "pending",
			},
		})
	}))
	defer ts.Close()

	run, err := newTestClient(ts).CreateRun(context.Background(), "create-repo", "create", map[string]any{"name": "my-repo"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if run.ID != "run-123" {
		t.Errorf("ID = %q", run.ID)
	}
	if run.Status != "pending" {
		t.Errorf("Status = %q", run.Status)
	}
}

func TestGetRun_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"run": map[string]any{
				"id": "run-123", "mold_slug": "create-repo", "status": "completed",
			},
		})
	}))
	defer ts.Close()

	run, err := newTestClient(ts).GetRun(context.Background(), "run-123")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "completed" {
		t.Errorf("Status = %q", run.Status)
	}
}

func TestListRuns_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"runs": []map[string]any{
				{"id": "run-1", "mold_slug": "create-repo", "status": "completed"},
				{"id": "run-2", "mold_slug": "delete-repo", "status": "failed"},
			},
		})
	}))
	defer ts.Close()

	resp, err := newTestClient(ts).ListRuns(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Runs) != 2 {
		t.Fatalf("got %d runs, want 2", len(resp.Runs))
	}
}

// --- parseOwner ---

func TestParseOwner_ArrayFormat(t *testing.T) {
	raw := json.RawMessage(`[{"id":"platform-team","type":"team"}]`)
	got := parseOwner(raw)
	if got != "platform-team" {
		t.Errorf("parseOwner(array) = %q, want 'platform-team'", got)
	}
}

func TestParseOwner_StringFormat(t *testing.T) {
	raw := json.RawMessage(`"platform-team"`)
	got := parseOwner(raw)
	if got != "platform-team" {
		t.Errorf("parseOwner(string) = %q, want 'platform-team'", got)
	}
}

func TestParseOwner_Empty(t *testing.T) {
	got := parseOwner(nil)
	if got != "" {
		t.Errorf("parseOwner(nil) = %q, want empty", got)
	}
}

func TestParseOwner_EmptyArray_ReturnsEmpty(t *testing.T) {
	raw := json.RawMessage(`[]`)
	got := parseOwner(raw)
	if got != "" {
		t.Errorf("parseOwner([]) = %q, want empty", got)
	}
}

// --- parseMoldInputs ---

func TestParseMoldInputs_EmptySchema(t *testing.T) {
	got := parseMoldInputs(nil, nil, nil)
	if got != nil {
		t.Errorf("parseMoldInputs(nil) = %v, want nil", got)
	}
}

func TestParseMoldInputs_RespectsInputOrder(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"z_field": map[string]any{"type": "string"},
			"a_field": map[string]any{"type": "string"},
		},
	}
	order := []string{"z_field", "a_field"}
	got := parseMoldInputs(schema, order, nil)
	if len(got) != 2 {
		t.Fatalf("got %d inputs, want 2", len(got))
	}
	if got[0].Name != "z_field" || got[1].Name != "a_field" {
		t.Errorf("order not respected: %s, %s", got[0].Name, got[1].Name)
	}
}

func TestParseMoldInputs_SortedWhenNoOrder(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"z_field": map[string]any{"type": "string"},
			"a_field": map[string]any{"type": "string"},
			"m_field": map[string]any{"type": "string"},
		},
	}
	got := parseMoldInputs(schema, nil, nil)
	if len(got) != 3 {
		t.Fatalf("got %d inputs, want 3", len(got))
	}
	if got[0].Name != "a_field" || got[1].Name != "m_field" || got[2].Name != "z_field" {
		t.Errorf("not sorted: %s, %s, %s", got[0].Name, got[1].Name, got[2].Name)
	}
}

func TestParseMoldInputs_RequiredAndDefaults(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"name":  map[string]any{"type": "string", "description": "The name"},
			"owner": map[string]any{"type": "string"},
		},
		"required": []any{"name"},
	}
	defaults := map[string]any{"owner": "default-org"}
	got := parseMoldInputs(schema, []string{"name", "owner"}, defaults)
	if !got[0].Required {
		t.Error("'name' should be required")
	}
	if got[1].Required {
		t.Error("'owner' should not be required")
	}
	if got[1].Default != "default-org" {
		t.Errorf("'owner' default = %q, want 'default-org'", got[1].Default)
	}
	if got[0].Description != "The name" {
		t.Errorf("'name' description = %q", got[0].Description)
	}
}

// --- formatLastSeen ---

func TestFormatLastSeen_NilReturnsNever(t *testing.T) {
	if got := formatLastSeen(nil); got != "never" {
		t.Errorf("formatLastSeen(nil) = %q, want 'never'", got)
	}
}

// --- Manifests ---

func TestValidateManifest_Valid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"valid":  true,
			"errors": []any{},
		})
	}))
	defer ts.Close()

	result, err := newTestClient(ts).ValidateManifest(context.Background(), "apiVersion: shoehorn/v1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
}

func TestValidateManifest_Invalid_Returns422(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		json.NewEncoder(w).Encode(map[string]any{
			"valid": false,
			"errors": []map[string]any{
				{"field": "name", "message": "required"},
			},
		})
	}))
	defer ts.Close()

	result, err := newTestClient(ts).ValidateManifest(context.Background(), "bad manifest")
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid {
		t.Error("expected valid=false for 422")
	}
	if len(result.Errors) != 1 {
		t.Errorf("got %d errors, want 1", len(result.Errors))
	}
}

// --- Entity sub-resources ---

func TestGetEntityResources_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/entities/svc-1/resources" {
			t.Errorf("path = %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"resources": []map[string]any{
				{"id": "r-1", "name": "payment-db", "type": "PostgreSQL", "environment": "production"},
			},
		})
	}))
	defer ts.Close()

	resources, err := newTestClient(ts).GetEntityResources(context.Background(), "svc-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].Name != "payment-db" {
		t.Errorf("resources = %+v", resources)
	}
}

func TestGetEntityStatus_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"health": "healthy", "uptime": 99.97, "incident_count": 0,
		})
	}))
	defer ts.Close()

	status, err := newTestClient(ts).GetEntityStatus(context.Background(), "svc-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Health != "healthy" {
		t.Errorf("Health = %q", status.Health)
	}
	if status.Uptime != 99.97 {
		t.Errorf("Uptime = %v", status.Uptime)
	}
}

func TestGetEntityChangelog_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"entries": []map[string]any{
				{"id": "c-1", "title": "Deployed v2", "type": "deployment", "author": "jane"},
			},
		})
	}))
	defer ts.Close()

	entries, err := newTestClient(ts).GetEntityChangelog(context.Background(), "svc-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Title != "Deployed v2" {
		t.Errorf("entries = %+v", entries)
	}
}

func TestGetEntityScorecard_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"score": 92, "grade": "A", "max_score": 100,
			"checks": []map[string]any{
				{"name": "has-docs", "passed": true, "weight": 10},
			},
		})
	}))
	defer ts.Close()

	sc, err := newTestClient(ts).GetEntityScorecard(context.Background(), "svc-1")
	if err != nil {
		t.Fatal(err)
	}
	if sc.Score != 92 || sc.Grade != "A" {
		t.Errorf("Score=%d, Grade=%q", sc.Score, sc.Grade)
	}
	if len(sc.Checks) != 1 || !sc.Checks[0].Passed {
		t.Errorf("Checks = %+v", sc.Checks)
	}
}

// --- NewClientFromConfig ---

func TestNewClientFromConfig_NotAuthenticated(t *testing.T) {
	// Override loadConfig to return unauthenticated config
	orig := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			CurrentProfile: "default",
			Profiles: map[string]*config.Profile{
				"default": {Name: "Default", Server: "http://localhost:8080"},
			},
		}, nil
	}
	defer func() { loadConfig = orig }()

	_, err := NewClientFromConfig()
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestNewClientFromConfig_Success(t *testing.T) {
	orig := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			CurrentProfile: "default",
			Profiles: map[string]*config.Profile{
				"default": {
					Name:   "Default",
					Server: "http://localhost:8080",
					Auth:   &config.Auth{AccessToken: "shp_test"},
				},
			},
		}, nil
	}
	defer func() { loadConfig = orig }()

	c, err := NewClientFromConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.GetToken() != "shp_test" {
		t.Errorf("token = %q, want shp_test", c.GetToken())
	}
}

func TestNewClientFromConfig_WithLogger(t *testing.T) {
	orig := loadConfig
	loadConfig = func() (*config.Config, error) {
		return &config.Config{
			CurrentProfile: "default",
			Profiles: map[string]*config.Profile{
				"default": {
					Name:   "Default",
					Server: "http://localhost:8080",
					Auth:   &config.Auth{AccessToken: "shp_test"},
				},
			},
		}, nil
	}
	defer func() { loadConfig = orig }()

	c, err := NewClientFromConfig(WithLogger(nil)) // nil should be safe
	if err != nil {
		t.Fatal(err)
	}
	if c.logger == nil {
		t.Error("logger should not be nil (SetLogger ignores nil)")
	}
}

// --- UploadAddonBundle ---

func TestUploadAddonBundle_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct == "" || len(ct) < 10 {
			t.Errorf("Content-Type missing or wrong: %q", ct)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"slug": "my-addon",
			"uploaded": map[string]int{
				"backend": 1024,
			},
		})
	}))
	defer ts.Close()

	result, err := newTestClient(ts).UploadAddonBundle(context.Background(), "my-addon", map[string][]byte{
		"backend": []byte("console.log('hello')"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Slug != "my-addon" {
		t.Errorf("Slug = %q", result.Slug)
	}
}

func TestUploadAddonBundle_ServerError_TruncatesBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		// Return a long HTML error page
		w.Write([]byte("<html>" + strings.Repeat("x", 500) + "</html>"))
	}))
	defer ts.Close()

	_, err := newTestClient(ts).UploadAddonBundle(context.Background(), "my-addon", map[string][]byte{
		"backend": []byte("test"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// Error message should be truncated, not contain the full 500+ byte body
	if len(err.Error()) > 300 {
		t.Errorf("error message too long (%d chars), should be truncated", len(err.Error()))
	}
}

func TestValidateManifest_ServerError_Returns5xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer ts.Close()

	_, err := newTestClient(ts).ValidateManifest(context.Background(), "content")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestValidateManifest_Unauthorized_Returns401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	_, err := newTestClient(ts).ValidateManifest(context.Background(), "content")
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

// --- Manifests ---

func TestConvertManifest_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"content": "apiVersion: shoehorn/v1\nkind: Service",
		})
	}))
	defer ts.Close()

	result, err := newTestClient(ts).ConvertManifest(context.Background(), "apiVersion: backstage.io/v1alpha1", "shoehorn", false)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}
