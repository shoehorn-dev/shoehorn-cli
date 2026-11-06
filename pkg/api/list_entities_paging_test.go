package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// pagedEntitiesServer serves n entities with an offset cursor, like /api/v1/entities.
func pagedEntitiesServer(t *testing.T, n int, calls *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
		end := min(offset+limit, n)
		items := make([]map[string]any, 0, end-offset)
		for i := offset; i < end; i++ {
			lifecycle := "production"
			if i%10 == 0 {
				lifecycle = "end-of-life"
			}
			items = append(items, map[string]any{
				"service":   map[string]any{"id": fmt.Sprintf("svc-%03d", i), "name": fmt.Sprintf("Service %d", i), "type": "service"},
				"lifecycle": lifecycle,
			})
		}
		page := map[string]any{"total": n}
		if end < n {
			page["nextCursor"] = strconv.Itoa(end)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entities": items, "page": page})
	}))
}

func TestListEntities_ReturnsEveryPage(t *testing.T) {
	for _, n := range []int{0, 1, 100, 101, 250} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			calls := 0
			ts := pagedEntitiesServer(t, n, &calls)
			defer ts.Close()

			entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
			if err != nil {
				t.Fatal(err)
			}
			if len(entities) != n {
				t.Fatalf("got %d entities, want all %d", len(entities), n)
			}
			seen := map[string]bool{}
			for _, e := range entities {
				if seen[e.ID] {
					t.Fatalf("%s returned twice", e.ID)
				}
				seen[e.ID] = true
			}
		})
	}
}

func TestListEntities_StopsOnARepeatedCursor(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entities": []any{map[string]any{"service": map[string]any{"id": "svc-1"}}},
			"page":     map[string]any{"total": 99, "nextCursor": "1"},
		})
	}))
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
	if err == nil {
		t.Fatal("want an error when the cursor does not advance, not an endless loop")
	}
	// The fixture never advances past "svc-1", so it's genuinely fetched twice before the repeat is caught.
	if len(entities) != 2 || entities[0].ID != "svc-1" || entities[1].ID != "svc-1" {
		t.Fatalf("want the 2 entities fetched before the repeat returned alongside the error, got %v", entities)
	}
}

func TestListEntities_StopsAfterMaxPages(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entities": []any{map[string]any{"service": map[string]any{"id": fmt.Sprintf("svc-%d", calls)}}},
			"page":     map[string]any{"nextCursor": strconv.Itoa(calls)},
		})
	}))
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
	if err == nil {
		t.Fatal("want an error once the page limit is reached, not an endless loop")
	}
	if calls != maxEntityPages {
		t.Fatalf("made %d requests, want exactly %d", calls, maxEntityPages)
	}
	if len(entities) != maxEntityPages {
		t.Fatalf("want all %d entities fetched before the cap returned alongside the error, got %d", maxEntityPages, len(entities))
	}
}

func TestListEntities_ReturnsEmptySliceNotNil(t *testing.T) {
	calls := 0
	ts := pagedEntitiesServer(t, 0, &calls)
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if entities == nil {
		t.Fatal("want a non-nil empty slice for zero results, got nil")
	}
	raw, err := json.Marshal(entities)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "[]" {
		t.Fatalf("JSON output = %s, want []", raw)
	}
}

func TestListEntities_RespectsLimit(t *testing.T) {
	calls := 0
	ts := pagedEntitiesServer(t, 250, &calls)
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 5 {
		t.Fatalf("got %d entities, want exactly the limit of 5", len(entities))
	}
	if calls != 1 {
		t.Fatalf("made %d requests, want exactly 1 (limit reached within the first page)", calls)
	}
}

func TestListEntities_LimitAbovePageSize_StaysPagedAt100PerRequest(t *testing.T) {
	calls := 0
	ts := pagedEntitiesServer(t, 250, &calls)
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{Limit: 150})
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 150 {
		t.Fatalf("got %d entities, want exactly the limit of 150", len(entities))
	}
	if entities[149].ID != "svc-149" {
		t.Fatalf("last entity = %q, want svc-149", entities[149].ID)
	}
	if calls != 2 {
		t.Fatalf("made %d requests, want exactly 2 (100 + 50)", calls)
	}
}

func TestListEntities_SmallLimit_RequestsOnlyThatManyPerPage(t *testing.T) {
	var gotPageSize string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPageSize = r.URL.Query().Get("limit")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entities": []any{map[string]any{"service": map[string]any{"id": "svc-1"}}},
			"page":     map[string]any{"total": 1},
		})
	}))
	defer ts.Close()

	if _, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{Limit: 5}); err != nil {
		t.Fatal(err)
	}
	if gotPageSize != "5" {
		t.Fatalf("server-requested page size = %q, want %q (don't fetch 100 to use only 5)", gotPageSize, "5")
	}
}

func TestListEntities_ReturnsPartialOnMidWalkTransportError(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entities": []any{map[string]any{"service": map[string]any{"id": "svc-1"}}},
				"page":     map[string]any{"total": 2, "nextCursor": "1"},
			})
			return
		}
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
	if err == nil {
		t.Fatal("want an error when a page request fails")
	}
	if len(entities) != 1 || entities[0].ID != "svc-1" {
		t.Fatalf("want the 1 entity fetched before the transport error returned alongside it, got %v", entities)
	}
	if !strings.Contains(err.Error(), "after 1 entities") {
		t.Fatalf("error should say the list is incomplete and how many were fetched, got: %v", err)
	}
}

func TestEntityJSON_EmptyLifecycleStillHasKey(t *testing.T) {
	raw, err := json.Marshal(Entity{ID: "svc-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsKey(raw, "lifecycle") {
		t.Fatalf("JSON output %s has no lifecycle key for an entity with no lifecycle set", raw)
	}
}

func TestListEntities_CarriesLifecycle(t *testing.T) {
	calls := 0
	ts := pagedEntitiesServer(t, 3, &calls)
	defer ts.Close()

	entities, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{IncludeEndOfLife: true})
	if err != nil {
		t.Fatal(err)
	}
	if entities[0].Lifecycle != "end-of-life" || entities[1].Lifecycle != "production" {
		t.Fatalf("lifecycles = %q, %q; want end-of-life, production", entities[0].Lifecycle, entities[1].Lifecycle)
	}
	raw, _ := json.Marshal(entities[0])
	if !json.Valid(raw) || !containsKey(raw, "lifecycle") {
		t.Fatalf("JSON output %s has no lifecycle key", raw)
	}
}

func containsKey(raw []byte, key string) bool {
	var m map[string]any
	return json.Unmarshal(raw, &m) == nil && m[key] != nil
}
