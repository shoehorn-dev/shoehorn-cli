package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

// RouteHandler maps a "METHOD /path" key to a handler function.
type RouteHandler map[string]http.HandlerFunc

// NewTestServer creates an httptest.Server that routes requests based on
// "METHOD /path" keys. Returns 404 for unmatched routes.
func NewTestServer(routes RouteHandler) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if handler, ok := routes[key]; ok {
			handler(w, r)
			return
		}
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "not found: " + key},
		})
	}))
}

// JSONHandler returns a handler that writes the given value as JSON with 200 OK.
func JSONHandler(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}

// ErrorHandler returns a handler that writes an API error with the given status code.
func ErrorHandler(statusCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": message},
		})
	}
}
