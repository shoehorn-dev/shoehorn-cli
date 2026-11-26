package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListGitOpsResources_WorkloadsFilter(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"resources": []any{}})
	}))
	defer server.Close()
	client := NewClient(server.URL)

	if _, err := client.ListGitOpsResources(context.Background(), ListGitOpsResourcesOpts{Workloads: "none"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if gotQuery != "workloads=none" {
		t.Errorf("query = %q, want workloads=none", gotQuery)
	}

	if _, err := client.ListGitOpsResources(context.Background(), ListGitOpsResourcesOpts{Tool: "argocd"}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if gotQuery != "tool=argocd" {
		t.Errorf("query = %q, want only tool=argocd when workloads isn't set", gotQuery)
	}
}

func TestListGitOpsResources_DecodesNoWorkloads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"resources":[{"id":"g1","name":"leaktest-directory","no_workloads":{
			"reason":"not_deployed",
			"missing":[{"kind":"Deployment","namespace":"leaktest-never-created","name":"guestbook-ui"}],
			"missing_total":1,"kinds":["Deployment"],"namespace":"leaktest-never-created"}}]}`))
	}))
	defer server.Close()

	resources, err := NewClient(server.URL).ListGitOpsResources(context.Background(), ListGitOpsResourcesOpts{Workloads: "none"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	nw := resources[0].NoWorkloads
	if nw == nil || nw.Reason != "not_deployed" || nw.MissingTotal != 1 || nw.Namespace != "leaktest-never-created" {
		t.Fatalf("no_workloads = %+v", nw)
	}
	if len(nw.Missing) != 1 || nw.Missing[0].Name != "guestbook-ui" || nw.Missing[0].Kind != "Deployment" {
		t.Fatalf("missing = %+v", nw.Missing)
	}
}
