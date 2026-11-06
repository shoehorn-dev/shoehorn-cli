package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── Search ──────────────────────────────────────────────────────────────────

// TestSearch_MapsEntityType pins the fix for shoehorn-dev/shoehorn-cli#52: the
// platform's /api/v1/search response carries the entity's specific type (a k8s
// kind like StatefulSet, or api/database/library/etc.) under entity_type,
// distinct from the generic "entity" category in type. Search() must surface it
// on SearchHit so the CLI's Type column can show it instead of a generic "entity"
// for every catalog hit.
func TestSearch_MapsEntityType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":          "checkout-db",
					"type":        "entity",
					"title":       "checkout-db",
					"description": "Primary StatefulSet",
					"score":       0.9,
					"entity_type": "statefulset",
				},
				{
					"id":          "doc-1",
					"type":        "doc",
					"title":       "Onboarding guide",
					"description": "How to get started",
					"score":       0.8,
				},
			},
			"total": 2,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	res, err := client.Search(context.Background(), "checkout", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("got %d hits, want 2", len(res.Hits))
	}
	if got := res.Hits[0].EntityType; got != "statefulset" {
		t.Errorf("Hits[0].EntityType = %q, want %q", got, "statefulset")
	}
	if got := res.Hits[0].Type; got != "entity" {
		t.Errorf("Hits[0].Type = %q, want %q (category unchanged)", got, "entity")
	}
}

// TestSearch_OmitsEntityTypeForNonEntityHits guards the scoping: a doc hit
// without entity_type in the response must leave SearchHit.EntityType empty,
// not fall back to anything.
func TestSearch_OmitsEntityTypeForNonEntityHits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"id": "doc-1", "type": "doc", "title": "Onboarding guide", "score": 0.8},
			},
			"total": 1,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	res, err := client.Search(context.Background(), "guide", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("got %d hits, want 1", len(res.Hits))
	}
	if got := res.Hits[0].EntityType; got != "" {
		t.Errorf("Hits[0].EntityType = %q, want empty for a doc hit", got)
	}
}
