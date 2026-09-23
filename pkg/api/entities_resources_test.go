package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The platform has answered GET /api/v1/entities/{id}/resources with the
// resources keyed by provider since its first commit
// (internal/handlers/k8s_handlers.go): {"resources":{"kubernetes":[...]},
// "totalInstances":n}. This client decoded "resources" as a flat list, which
// fails on every entity, so `shoehorn get entity` printed "some details could
// not be loaded" and dropped the section every time.
func TestGetEntityResources_DecodesProviderKeyedShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/entities/analytics-api/resources" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entityId":       "analytics-api",
			"serviceId":      "analytics-api",
			"totalInstances": 1,
			"resources": map[string]any{
				"kubernetes": []map[string]any{{
					"name":      "analytics-api",
					"kind":      "Deployment",
					"namespace": "prod",
					"cluster":   "prod-eu-1",
					"status":    "Running",
					"replicas":  "2/2",
				}},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resources, err := client.GetEntityResources(context.Background(), "analytics-api")
	if err != nil {
		t.Fatalf("the API's provider-keyed shape must decode, got: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(resources))
	}
	if resources[0].Name != "analytics-api" || resources[0].Kind != "Deployment" || resources[0].Namespace != "prod" {
		t.Errorf("resource = %+v, want name analytics-api, kind Deployment, namespace prod", *resources[0])
	}
}

func TestFlattenResources(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		want  []string
		fails bool
	}{
		{"providers in name order", `{"upcloud":[{"name":"bucket"}],"kubernetes":[{"name":"deploy"},{"name":"svc"}]}`, []string{"deploy", "svc", "bucket"}, false},
		{"empty map", `{}`, nil, false},
		{"absent", ``, nil, false},
		{"null", `null`, nil, false},
		{"flat list still accepted", `[{"name":"deploy"}]`, []string{"deploy"}, false},
		{"neither shape", `"kubernetes"`, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := flattenResources(json.RawMessage(tc.raw))
			if (err != nil) != tc.fails {
				t.Fatalf("err = %v, want failure %v", err, tc.fails)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d resources, want %d", len(got), len(tc.want))
			}
			for i, name := range tc.want {
				if got[i].Name != name {
					t.Errorf("resource %d = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}
}
