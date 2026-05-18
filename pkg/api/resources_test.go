package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// ─── ListOperationsResources ─────────────────────────────────────────────────

func TestListOperationsResources_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/operations/resources" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"resources": []map[string]any{
				{
					"id":      "rs_abc123",
					"name":    "checkout",
					"subtype": "Deployment",
					"status":  "Healthy",
					"identity": map[string]any{
						"image_repository": "ghcr.io/acme/checkout",
						"kind":             "Deployment",
						"name_hint":        "checkout",
					},
					"clusters": []map[string]any{
						{"cluster_id": "prod-eu", "namespace": "shop", "replicas_ready": 3, "replicas_desired": 3},
						{"cluster_id": "prod-us", "namespace": "shop", "replicas_ready": 2, "replicas_desired": 3},
					},
				},
			},
			"next_cursor": "",
			"total":       1,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	res, err := client.ListOperationsResources(context.Background(), ListOperationsResourcesOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 1 {
		t.Errorf("Total = %d, want 1", res.Total)
	}
	if len(res.Resources) != 1 {
		t.Fatalf("got %d resources, want 1", len(res.Resources))
	}
	got := res.Resources[0]
	if got.ID != "rs_abc123" || got.Name != "checkout" || got.Subtype != "Deployment" {
		t.Errorf("resource fields wrong: %+v", got)
	}
	if got.Identity.ImageRepository != "ghcr.io/acme/checkout" {
		t.Errorf("ImageRepository = %q", got.Identity.ImageRepository)
	}
	if len(got.Clusters) != 2 {
		t.Errorf("got %d clusters, want 2", len(got.Clusters))
	}
}

func TestListOperationsResources_FilterPassthrough(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"resources": []any{}, "next_cursor": "", "total": 0})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	_, err := client.ListOperationsResources(context.Background(), ListOperationsResourcesOpts{
		ClusterID: "prod-eu",
		Team:      "team-shop",
		Kind:      "Deployment,Job",
		Status:    "Degraded",
		Search:    "checkout",
		HasGitOps: true,
		Cursor:    "cur_xyz",
		Limit:     25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q, err := parseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	want := map[string]string{
		"cluster":    "prod-eu",
		"team":       "team-shop",
		"kind":       "Deployment,Job",
		"status":     "Degraded",
		"search":     "checkout",
		"has_gitops": "true",
		"cursor":     "cur_xyz",
		"limit":      "25",
	}
	for k, v := range want {
		if q[k] != v {
			t.Errorf("query[%q] = %q, want %q", k, q[k], v)
		}
	}
}

func TestListOperationsResources_EmptyOpts_NoQueryString(t *testing.T) {
	var gotRawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"resources": []any{}, "next_cursor": "", "total": 0})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	if _, err := client.ListOperationsResources(context.Background(), ListOperationsResourcesOpts{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRawQuery != "" {
		t.Errorf("empty opts should produce no query string, got %q", gotRawQuery)
	}
}

func TestListOperationsResources_PaginationCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"resources":   []map[string]any{{"id": "rs_1", "name": "one"}},
			"next_cursor": "cur_page2",
			"total":       3,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	res, err := client.ListOperationsResources(context.Background(), ListOperationsResourcesOpts{Limit: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.NextCursor != "cur_page2" {
		t.Errorf("NextCursor = %q, want %q", res.NextCursor, "cur_page2")
	}
	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
}

func TestListOperationsResources_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "boom"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	if _, err := client.ListOperationsResources(context.Background(), ListOperationsResourcesOpts{}); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

// ─── GetOperationsResource ───────────────────────────────────────────────────

func TestGetOperationsResource_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/operations/resources/rs_abc123" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"resource": map[string]any{
				"id":      "rs_abc123",
				"name":    "checkout",
				"subtype": "Deployment",
				"status":  "Degraded",
			},
			"events": []map[string]any{
				{"cluster_id": "prod-eu", "reason": "BackOff", "message": "restart loop", "count": 4},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	detail, err := client.GetOperationsResource(context.Background(), "rs_abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Resource.ID != "rs_abc123" || detail.Resource.Status != "Degraded" {
		t.Errorf("resource fields wrong: %+v", detail.Resource)
	}
	if len(detail.Events) != 1 || detail.Events[0].Count != 4 {
		t.Errorf("events wrong: %+v", detail.Events)
	}
}

func TestGetOperationsResource_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "not found"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	_, err := client.GetOperationsResource(context.Background(), "rs_missing")
	if err == nil {
		t.Fatal("expected error for missing resource, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got: %v", err)
	}
}

func TestGetOperationsResource_EncodesIDInPath(t *testing.T) {
	var gotURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURI = r.RequestURI
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"resource": map[string]any{"id": "x"}, "events": []any{}})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	if _, err := client.GetOperationsResource(context.Background(), "rs_a/b c"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/api/v1/operations/resources/rs_a%2Fb%20c"
	if gotURI != want {
		t.Errorf("request URI = %q, want %q (ID path-escaped)", gotURI, want)
	}
}

// parseQuery is a tiny wrapper so the filter-passthrough test reads cleanly.
func parseQuery(raw string) (map[string]string, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(values))
	for k := range values {
		out[k] = values.Get(k)
	}
	return out, nil
}
