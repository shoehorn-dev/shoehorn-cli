package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- CreateEntityFromManifest ---

func TestCreateEntityFromManifest_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/manifests/entities" {
			t.Errorf("path = %s", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["content"] == "" {
			t.Error("content is empty")
		}
		if body["source"] != "cli" {
			t.Errorf("source = %q, want 'cli'", body["source"])
		}

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"entity": map[string]any{
				"id":        "uuid-123",
				"serviceId": "my-service",
				"name":      "My Service",
				"type":      "service",
			},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	result, err := c.CreateEntityFromManifest(context.Background(), "apiVersion: shoehorn/v1\nkind: Service\nmetadata:\n  name: my-service")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Entity.ServiceID != "my-service" {
		t.Errorf("ServiceID = %q", result.Entity.ServiceID)
	}
}

func TestCreateEntityFromManifest_ValidationError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Manifest content is required",
				"code":    "VALIDATION_ERROR",
			},
		})
	}))
	defer ts.Close()

	_, err := newTestClient(ts).CreateEntityFromManifest(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestCreateEntityFromManifest_409Conflict(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(409)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "entity already exists"},
		})
	}))
	defer ts.Close()

	_, err := newTestClient(ts).CreateEntityFromManifest(context.Background(), "content")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

// --- UpdateEntityFromManifest ---

func TestUpdateEntityFromManifest_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/manifests/entities/my-service" {
			t.Errorf("path = %s", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"entity": map[string]any{
				"serviceId": "my-service",
				"name":      "My Service Updated",
			},
		})
	}))
	defer ts.Close()

	result, err := newTestClient(ts).UpdateEntityFromManifest(context.Background(), "my-service", "updated content")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestUpdateEntityFromManifest_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Entity not found"},
		})
	}))
	defer ts.Close()

	_, err := newTestClient(ts).UpdateEntityFromManifest(context.Background(), "nonexistent", "content")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- DeleteEntity ---

func TestDeleteEntity_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/manifests/entities/my-service" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer ts.Close()

	err := newTestClient(ts).DeleteEntity(context.Background(), "my-service")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteEntity_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Entity not found"},
		})
	}))
	defer ts.Close()

	err := newTestClient(ts).DeleteEntity(context.Background(), "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- Patch method ---

func TestPatch_SendsCorrectMethod(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("empty body")
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer ts.Close()

	c := newTestClient(ts)
	var result map[string]any
	err := c.Patch(context.Background(), "/test", map[string]string{"key": "val"}, &result)
	if err != nil {
		t.Fatal(err)
	}
}
