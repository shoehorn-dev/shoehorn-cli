package api

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── S1: Redirect credential leak protection ─────────────────────────────────

// TestClient_Redirect_StripsAuthHeader_CrossOrigin verifies that the Authorization
// header is stripped on cross-origin redirects, preventing credential leaks.
// Regression test for security finding S1 (A10 SSRF, A07 Credential Leak).
func TestClient_Redirect_StripsAuthHeader_CrossOrigin(t *testing.T) {
	var redirectedAuth string

	// Target server (different origin) that captures the Authorization header
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer target.Close()

	// Origin server that redirects to a different host
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/stolen", http.StatusFound)
	}))
	defer origin.Close()

	client := NewClient(origin.URL)
	client.SetToken("secret-bearer-token")

	// The request follows the redirect, but must NOT forward the auth header
	client.Get(context.Background(), "/api/v1/test", nil)

	if redirectedAuth != "" {
		t.Errorf("Authorization header leaked on cross-origin redirect: got %q, want empty", redirectedAuth)
	}
}

// TestClient_Redirect_PreservesAuthHeader_SameOrigin verifies that the
// Authorization header IS preserved on same-origin redirects (e.g. path
// normalization within the same API server).
func TestClient_Redirect_PreservesAuthHeader_SameOrigin(t *testing.T) {
	var redirectedAuth string
	var reqCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		if reqCount == 1 {
			// First request: redirect to a different path on the same host
			http.Redirect(w, r, "/api/v2/test", http.StatusFound)
			return
		}
		// Second request: capture the auth header
		redirectedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("keep-this-token")

	client.Get(context.Background(), "/api/v1/test", nil)

	if redirectedAuth != "Bearer keep-this-token" {
		t.Errorf("Authorization header lost on same-origin redirect: got %q, want %q",
			redirectedAuth, "Bearer keep-this-token")
	}
}

// TestClient_Redirect_MaxHops verifies that the client stops after maxRedirects.
func TestClient_Redirect_MaxHops(t *testing.T) {
	hops := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops++
		// Always redirect (infinite loop)
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Get(context.Background(), "/api/v1/test", nil)

	if err == nil {
		t.Fatal("expected error from redirect loop, got nil")
	}
	if !strings.Contains(err.Error(), "redirect") {
		t.Errorf("expected redirect error, got: %v", err)
	}
	// Should stop after maxRedirects (3) + the initial request
	if hops > maxRedirects+1 {
		t.Errorf("too many redirect hops: %d, expected <= %d", hops, maxRedirects+1)
	}
}

// ─── S3: TLS 1.2 minimum enforcement ────────────────────────────────────────

// TestClient_TLSMinVersion verifies the transport enforces TLS 1.2 minimum.
func TestClient_TLSMinVersion(t *testing.T) {
	c := NewClient("https://example.com")
	transport, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig is nil")
	}
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("TLS min version = %d, want %d (TLS 1.2)", transport.TLSClientConfig.MinVersion, tls.VersionTLS12)
	}
}

// ─── S8: Error response body sanitization ────────────────────────────────────

// TestClient_ErrorBody_Truncated verifies that raw error response bodies are
// truncated to prevent leaking server internals (stack traces, paths, etc).
// Regression test for security finding S8 (A09 Security Logging Failures).
func TestClient_ErrorBody_Truncated(t *testing.T) {
	longBody := strings.Repeat("x", 500)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		// Return non-JSON body (triggers raw body in error message)
		_, _ = w.Write([]byte(longBody))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Get(context.Background(), "/test", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	// The error must NOT contain the full 500-char body
	if strings.Contains(errMsg, longBody) {
		t.Error("error message contains full un-truncated body")
	}
	if !strings.Contains(errMsg, "truncated") {
		t.Errorf("truncated error should contain 'truncated' marker, got: %s", errMsg[:min(len(errMsg), 100)])
	}
}

// TestClient_ErrorBody_ShortBodyNotTruncated verifies short error bodies are
// preserved in full.
func TestClient_ErrorBody_ShortBodyNotTruncated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("short error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Get(context.Background(), "/test", nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "short error") {
		t.Errorf("short error body should be preserved, got: %v", err)
	}
}
