package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// C1 (QA 2026-09-18): `shoehorn get entities` returned 100 of QA's 101 entities.
// The client asked for one page of 100 and only mentioned the rest in a debug
// log, so a script piping the output saw the first 100 as the whole catalog.

// pagedEntitiesServer serves n entities in pages of the requested limit, with
// an offset cursor, the way /api/v1/entities does.
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

// A server that keeps handing back the same cursor must not hang the CLI.
func TestListEntities_StopsOnARepeatedCursor(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entities": []any{map[string]any{"service": map[string]any{"id": "svc-1"}}},
			"page":     map[string]any{"total": 99, "nextCursor": "1"},
		})
	}))
	defer ts.Close()

	if _, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{}); err == nil {
		t.Fatal("want an error when the cursor does not advance, not an endless loop")
	}
}

// Security review 2026-09-18 (L3): a server that never stops handing out new
// cursors would keep the CLI fetching, and growing its list, forever.
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

	_, err := newTestClient(ts).ListEntities(context.Background(), ListEntitiesOpts{})
	if err == nil {
		t.Fatal("want an error once the page limit is reached, not an endless loop")
	}
	if calls != maxEntityPages {
		t.Fatalf("made %d requests, want exactly %d", calls, maxEntityPages)
	}
}

// C2: with --include-end-of-life the rows mixed live entities and tombstones,
// and nothing in the output said which was which.
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
