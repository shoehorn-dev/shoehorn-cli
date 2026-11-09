package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// writeTestAddonDir creates a minimal addon project directory on disk (a
// manifest.json plus built dist/ bundles) for runAddonPublish to read.
func writeTestAddonDir(t *testing.T, slug string) string {
	t.Helper()
	dir := t.TempDir()
	manifest := map[string]any{
		"schemaVersion": 1,
		"kind":          "addon",
		"metadata":      map[string]any{"slug": slug, "name": "Test Addon", "version": "0.1.0"},
		"addon":         map[string]any{"tier": "scripted", "runtime": "quickjs"},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write manifest.json: %v", err)
	}
	distDir := filepath.Join(dir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "addon.js"), []byte("var __addon__={};"), 0o644); err != nil {
		t.Fatalf("write addon.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "frontend.js"), []byte("export const widgets=[];"), 0o644); err != nil {
		t.Fatalf("write frontend.js: %v", err)
	}
	return dir
}

// TestAddonPublish_EnablesAfterBundleUpload is the regression test for a bug
// found by actually running `shoehorn addon publish` against a real local
// stack: the server auto-installs a brand-new addon as part of publishing
// its manifest, before any bundle exists (bundles are a separate, later
// step), so that install's attempt to load the addon into the runtime
// always failed with "no backend bundle uploaded" -- silently, since the
// server swallows that failure and still reports the item as published.
// The fix: after uploading the bundle, if this was a fresh publish, retry
// loading the addon via the enable endpoint. This test asserts the actual
// request order the server sees, not just that publish returns no error.
func TestAddonPublish_EnablesAfterBundleUpload(t *testing.T) {
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/marketplace/import-manifest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"slug": "test-addon", "name": "Test Addon", "created": true, "installed": true,
			})
		case r.URL.Path == "/api/v1/marketplace/test-addon/bundle":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"slug": "test-addon", "uploaded": map[string]int{"backend": 10, "frontend": 20},
			})
		case r.URL.Path == "/api/v1/marketplace/test-addon/enable":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	dir := writeTestAddonDir(t, "test-addon")
	_, err := runRoot(t, "addon", "publish", "--dir", dir)
	if err != nil {
		t.Fatalf("addon publish: %v", err)
	}

	want := []string{
		"POST /api/v1/marketplace/import-manifest",
		"POST /api/v1/marketplace/test-addon/bundle",
		"POST /api/v1/marketplace/test-addon/enable",
	}
	if len(gotPaths) != len(want) {
		t.Fatalf("request sequence = %v, want %v", gotPaths, want)
	}
	for i, p := range want {
		if gotPaths[i] != p {
			t.Errorf("request[%d] = %q, want %q (full sequence: %v)", i, gotPaths[i], p, gotPaths)
		}
	}
}

// TestAddonPublish_DoesNotEnableOnRepublish confirms the retry only fires on
// a fresh publish (created=true) -- republishing an existing addon (a
// version bump, say) must not call enable on every single publish.
func TestAddonPublish_DoesNotEnableOnRepublish(t *testing.T) {
	var gotPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/marketplace/import-manifest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"slug": "test-addon", "name": "Test Addon", "created": false, "installed": false,
			})
		case r.URL.Path == "/api/v1/marketplace/test-addon/bundle":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"slug": "test-addon", "uploaded": map[string]int{"backend": 10, "frontend": 20},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	useTestProfile(t, server.URL)

	dir := writeTestAddonDir(t, "test-addon")
	_, err := runRoot(t, "addon", "publish", "--dir", dir)
	if err != nil {
		t.Fatalf("addon publish: %v", err)
	}

	for _, p := range gotPaths {
		if p == "POST /api/v1/marketplace/test-addon/enable" {
			t.Errorf("enable was called on a republish of an already-existing addon; requests: %v", gotPaths)
		}
	}
}
