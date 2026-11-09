package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAddonConfig_Show prints the addon's current config as JSON.
func TestAddonConfig_Show(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/marketplace/installed/jira-sync" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"itemSlug": "jira-sync",
			"enabled":  true,
			"config":   json.RawMessage(`{"jira_base_url":"https://acme.atlassian.net"}`),
		})
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	printed := captureStdout(t, func() {
		if _, err := runRoot(t, "addon", "config", "jira-sync", "-I"); err != nil {
			t.Fatalf("addon config: %v", err)
		}
	})

	if !strings.Contains(printed, "https://acme.atlassian.net") {
		t.Errorf("output missing current config value, got: %s", printed)
	}
}

// TestAddonConfig_SetMergesWithoutClobberingOtherKeys is the regression test
// for the PUT endpoint being a full overwrite, not a merge: --set must read
// the current config first and merge locally, or setting one key would
// silently wipe out every other key already configured (through the UI or
// a previous CLI call).
func TestAddonConfig_SetMergesWithoutClobberingOtherKeys(t *testing.T) {
	var gotConfigBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/marketplace/installed/jira-sync":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"itemSlug": "jira-sync",
				"enabled":  true,
				"config": json.RawMessage(`{
					"jira_base_url": "https://acme.atlassian.net",
					"jira_email": "bot@acme.test"
				}`),
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/marketplace/jira-sync/config":
			var body struct {
				Config map[string]any `json:"config"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			gotConfigBody = body.Config
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	if _, err := runRoot(t, "addon", "config", "jira-sync", "--set", "jira_project=PROJ"); err != nil {
		t.Fatalf("addon config --set: %v", err)
	}

	if gotConfigBody == nil {
		t.Fatal("PUT /config was never called")
	}
	if gotConfigBody["jira_project"] != "PROJ" {
		t.Errorf("jira_project = %v, want %q", gotConfigBody["jira_project"], "PROJ")
	}
	if gotConfigBody["jira_base_url"] != "https://acme.atlassian.net" {
		t.Errorf("--set clobbered jira_base_url, got %v", gotConfigBody["jira_base_url"])
	}
	if gotConfigBody["jira_email"] != "bot@acme.test" {
		t.Errorf("--set clobbered jira_email, got %v", gotConfigBody["jira_email"])
	}
}

// TestAddonConfig_SetFromFile merges keys from a JSON file into the config.
func TestAddonConfig_SetFromFile(t *testing.T) {
	var gotConfigBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"itemSlug": "jira-sync", "config": json.RawMessage(`{}`)})
		case r.Method == http.MethodPut:
			var body struct {
				Config map[string]any `json:"config"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotConfigBody = body.Config
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgFile, []byte(`{"jira_base_url":"https://from-file.atlassian.net"}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	if _, err := runRoot(t, "addon", "config", "jira-sync", "--file", cfgFile); err != nil {
		t.Fatalf("addon config --file: %v", err)
	}

	if gotConfigBody["jira_base_url"] != "https://from-file.atlassian.net" {
		t.Errorf("jira_base_url = %v, want value from file", gotConfigBody["jira_base_url"])
	}
}

// TestAddonConfig_SetInvalidKeyValue rejects a --set value with no "=".
func TestAddonConfig_SetInvalidKeyValue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"itemSlug": "jira-sync", "config": json.RawMessage(`{}`)})
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	_, err := runRoot(t, "addon", "config", "jira-sync", "--set", "not-a-key-value-pair")
	if err == nil {
		t.Fatal("expected an error for a --set value with no '='")
	}
}

// ─── addon secret ───────────────────────────────────────────────────────────

func TestAddonSecretSet(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody struct{ Key, Value string }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	if _, err := runRoot(t, "addon", "secret", "set", "jira-sync", "jira_api_token", "s3cret"); err != nil {
		t.Fatalf("addon secret set: %v", err)
	}

	if gotMethod != http.MethodPut || gotPath != "/api/v1/admin/addons/jira-sync/secrets" {
		t.Errorf("request = %s %s, want PUT /api/v1/admin/addons/jira-sync/secrets", gotMethod, gotPath)
	}
	if gotBody.Key != "jira_api_token" || gotBody.Value != "s3cret" {
		t.Errorf("body = %+v, want key=jira_api_token value=s3cret", gotBody)
	}
}

func TestAddonSecretList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/admin/addons/jira-sync/secrets" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []string{"jira_api_token"}})
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	printed := captureStdout(t, func() {
		if _, err := runRoot(t, "addon", "secret", "list", "jira-sync", "-I"); err != nil {
			t.Fatalf("addon secret list: %v", err)
		}
	})

	if !strings.Contains(printed, "jira_api_token") {
		t.Errorf("output missing secret key name, got: %s", printed)
	}
}

// ─── addon reload ───────────────────────────────────────────────────────────

func TestAddonReload(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"reloaded": []string{"jira-sync"},
			"failed":   map[string]string{"other-addon": "runtime unavailable"},
		})
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	printed := captureStdout(t, func() {
		if _, err := runRoot(t, "addon", "reload"); err != nil {
			t.Fatalf("addon reload: %v", err)
		}
	})

	if gotMethod != http.MethodPost || gotPath != "/api/v1/marketplace/reload" {
		t.Errorf("request = %s %s, want POST /api/v1/marketplace/reload", gotMethod, gotPath)
	}
	if !strings.Contains(printed, "jira-sync reloaded") {
		t.Errorf("output missing reloaded addon, got: %s", printed)
	}
	if !strings.Contains(printed, "other-addon failed") {
		t.Errorf("output missing failed addon, got: %s", printed)
	}
}

func TestAddonSecretDelete(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	if _, err := runRoot(t, "addon", "secret", "delete", "jira-sync", "jira_api_token"); err != nil {
		t.Fatalf("addon secret delete: %v", err)
	}

	if gotMethod != http.MethodDelete || gotPath != "/api/v1/admin/addons/jira-sync/secrets/jira_api_token" {
		t.Errorf("request = %s %s, want DELETE /api/v1/admin/addons/jira-sync/secrets/jira_api_token", gotMethod, gotPath)
	}
}
