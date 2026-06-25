package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ListGovernanceActions must forward the assigned_to and closed filters (parity
// with the platform's GET /governance/actions query params).
func TestListGovernanceActions_ForwardsAssignedToAndClosed(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"actions": []any{}, "total": 0})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, _, err := client.ListGovernanceActions(context.Background(), ListGovernanceActionsOpts{
		AssignedTo: "alice",
		Closed:     true,
	})
	if err != nil {
		t.Fatalf("ListGovernanceActions: %v", err)
	}
	if !contains(gotQuery, "assigned_to=alice") {
		t.Errorf("query %q missing assigned_to=alice", gotQuery)
	}
	if !contains(gotQuery, "closed=true") {
		t.Errorf("query %q missing closed=true", gotQuery)
	}
}

// BulkUpdateGovernanceActions POSTs ids + status and returns the counts.
func TestBulkUpdateGovernanceActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/governance/actions/bulk" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			IDs    []string `json:"ids"`
			Status string   `json:"status"`
		}
		_ = json.Unmarshal(body, &req)
		if len(req.IDs) != 2 || req.Status != "resolved" {
			t.Errorf("unexpected bulk body: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"requested": 2, "updated": 2})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	res, err := client.BulkUpdateGovernanceActions(context.Background(), []string{"a", "b"}, "resolved")
	if err != nil {
		t.Fatalf("BulkUpdateGovernanceActions: %v", err)
	}
	if res.Requested != 2 || res.Updated != 2 {
		t.Errorf("unexpected result: %+v", res)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
