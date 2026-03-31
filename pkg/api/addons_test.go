package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListInstalledAddons_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/marketplace/installed" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"installations": []map[string]any{
				{"itemSlug": "jira-sync", "itemKind": "addon", "itemVersion": "1.0.0", "enabled": true, "addonStatus": "running"},
				{"itemSlug": "pagerduty", "itemKind": "addon", "itemVersion": "2.1.0", "enabled": false, "addonStatus": "stopped"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	addons, err := client.ListInstalledAddons(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addons) != 2 {
		t.Fatalf("expected 2 addons, got %d", len(addons))
	}
	if addons[0].Slug != "jira-sync" {
		t.Errorf("expected slug jira-sync, got %s", addons[0].Slug)
	}
	if !addons[0].Enabled {
		t.Error("expected first addon to be enabled")
	}
	if addons[1].AddonStatus != "stopped" {
		t.Errorf("expected status stopped, got %s", addons[1].AddonStatus)
	}
}

func TestListInstalledAddons_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"installations": []any{},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	addons, err := client.ListInstalledAddons(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addons) != 0 {
		t.Fatalf("expected 0 addons, got %d", len(addons))
	}
}

func TestGetAddonStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/addons/jira-sync/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"slug":          "jira-sync",
			"status":        "active",
			"enabled":       true,
			"execCount":     42,
			"errorCount":    3,
			"vmMemoryBytes": 1048576,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	status, err := client.GetAddonStatus(context.Background(), "jira-sync")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Slug != "jira-sync" {
		t.Errorf("expected slug jira-sync, got %s", status.Slug)
	}
	if status.ExecCount != 42 {
		t.Errorf("expected 42 execs, got %d", status.ExecCount)
	}
	if status.VMMemory != 1048576 {
		t.Errorf("expected 1MB memory, got %d", status.VMMemory)
	}
}

func TestInstallAddon_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/marketplace/install" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["slug"] != "my-addon" {
			t.Errorf("expected slug my-addon, got %s", body["slug"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"itemSlug": "my-addon", "itemKind": "addon", "enabled": true,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	addon, err := client.InstallAddon(context.Background(), "my-addon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addon.Slug != "my-addon" {
		t.Errorf("expected slug my-addon, got %s", addon.Slug)
	}
}

func TestUninstallAddon_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/marketplace/my-addon/uninstall" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.UninstallAddon(context.Background(), "my-addon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnableAddon_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/marketplace/my-addon/enable" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.EnableAddon(context.Background(), "my-addon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDisableAddon_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/marketplace/my-addon/disable" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DisableAddon(context.Background(), "my-addon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAddonLogs_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/addons/jira-sync/logs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "50" {
			t.Errorf("expected limit=50, got %s", r.URL.Query().Get("limit"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"entries": []map[string]any{
				{"timestamp": "2026-03-15T10:00:00Z", "level": "info", "message": "Sync started"},
				{"timestamp": "2026-03-15T10:00:01Z", "level": "error", "message": "Connection refused"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	entries, err := client.GetAddonLogs(context.Background(), "jira-sync", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}
	if entries[0].Level != "info" {
		t.Errorf("expected level info, got %s", entries[0].Level)
	}
	if entries[1].Message != "Connection refused" {
		t.Errorf("unexpected message: %s", entries[1].Message)
	}
}

func TestGetAddonLogs_DefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "100" {
			t.Errorf("expected default limit=100, got %s", r.URL.Query().Get("limit"))
		}
		json.NewEncoder(w).Encode(map[string]any{"entries": []any{}})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetAddonLogs(context.Background(), "test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListMarketplaceItems_WithKindFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("kind") != "addon" {
			t.Errorf("expected kind=addon, got %s", r.URL.Query().Get("kind"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"slug": "jira-sync", "kind": "addon", "name": "Jira Sync", "version": "1.0.0"},
			},
			"pagination": map[string]any{"total": 1, "limit": 50, "offset": 0},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	items, err := client.ListMarketplaceItems(context.Background(), "addon")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Slug != "jira-sync" {
		t.Errorf("expected slug jira-sync, got %s", items[0].Slug)
	}
}

func TestPublishAddonManifest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/marketplace/import-manifest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"slug": "my-addon", "name": "My Addon", "created": true,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.PublishAddonManifest(context.Background(), map[string]any{
		"schemaVersion": 1, "kind": "addon",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Slug != "my-addon" {
		t.Errorf("expected slug my-addon, got %s", result.Slug)
	}
	if !result.Created {
		t.Error("expected created=true")
	}
}

func TestInstallAddon_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "NOT_FOUND",
				"message": "Marketplace item not found",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.InstallAddon(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
