// Package api provides a thin HTTP client for the Shoehorn API
package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/config"
	"go.uber.org/zap"
)

// Retry configuration
const (
	maxRetries    = 3
	retryBaseWait = 500 * time.Millisecond
	retryMaxWait  = 5 * time.Second
)

// maxRedirects is the maximum number of HTTP redirects the client will follow.
const maxRedirects = 3

// loadConfig is a package-level alias to avoid import cycles
var loadConfig = config.Load

// maxResponseSize is the maximum allowed API response body size (10 MB).
// Prevents denial-of-service from malicious or compromised servers.
const maxResponseSize = 10 * 1024 * 1024

// Client is a thin HTTP client for the Shoehorn API. It handles JSON
// serialization, Bearer-token authentication, and structured error responses.
// The token field uses atomic.Value for safe concurrent access (e.g. entity
// detail fetches fan out multiple goroutines sharing the same client).
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      atomic.Value // stores string
	logger     *zap.Logger
}

// NewClient creates a new API client pointed at baseURL with a 30-second
// HTTP timeout, TLS 1.2 minimum, and redirect protection that strips the
// Authorization header on cross-origin redirects to prevent credential leaks (S1, S3).
func NewClient(baseURL string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("stopped after %d redirects", maxRedirects)
				}
				// Strip Authorization header on cross-origin redirects to prevent
				// credential leaks to third-party servers.
				if req.URL.Host != via[0].URL.Host {
					req.Header.Del("Authorization")
				}
				return nil
			},
		},
		logger: zap.NewNop(),
	}
	c.token.Store("")
	return c
}

// SetLogger replaces the client's logger. Pass a non-nil *zap.Logger to enable
// debug-level HTTP request/response logging.
func (c *Client) SetLogger(l *zap.Logger) {
	if l != nil {
		c.logger = l
	}
}

// SetToken sets the Bearer token sent in the Authorization header of
// every subsequent request. Pass an empty string to clear the token.
func (c *Client) SetToken(token string) {
	c.token.Store(token)
}

// GetToken returns the current Bearer token, or an empty string if none is set.
func (c *Client) GetToken() string {
	if v, ok := c.token.Load().(string); ok {
		return v
	}
	return ""
}

// do executes an HTTP request with automatic retry for transient failures.
// Retries on: connection refused/reset/DNS/timeout, 429 (rate limit), 502, 503, 504 (gateway errors).
// Does NOT retry on: 400, 401, 403, 404, 409, 422, 500, or unknown errors.
func (c *Client) do(ctx context.Context, method, path string, body, result any) error {
	// Only retry idempotent methods. POST is not idempotent -- retrying
	// after a network error could create duplicate resources.
	if method == http.MethodPost {
		return c.doOnce(ctx, method, path, body, result)
	}

	var lastErr error
	for attempt := range maxRetries + 1 {
		lastErr = c.doOnce(ctx, method, path, body, result)
		if lastErr == nil {
			return nil
		}

		// Don't retry if context is cancelled
		if ctx.Err() != nil {
			return lastErr
		}

		// Don't retry on non-retryable errors
		if !isRetryable(lastErr) {
			return lastErr
		}

		// Don't retry after the last attempt
		if attempt >= maxRetries {
			break
		}

		// Calculate wait: exponential backoff or Retry-After header
		wait := retryWait(attempt)
		c.logger.Debug("retrying request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("attempt", attempt+1),
			zap.Duration("wait", wait),
			zap.Error(lastErr),
		)

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return lastErr
		}
	}
	return lastErr
}

// doOnce executes a single HTTP request attempt.
func (c *Client) doOnce(ctx context.Context, method, path string, body, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if tok := c.GetToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Debug("HTTP request failed",
			zap.String("method", method),
			zap.String("path", path),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body with size limit to prevent DoS from oversized responses
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if int64(len(respBody)) > maxResponseSize {
		return fmt.Errorf("response too large (>%d bytes)", maxResponseSize)
	}

	// Log the response
	duration := time.Since(start)
	c.logger.Debug("HTTP response",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", resp.StatusCode),
		zap.Int("size", len(respBody)),
		zap.Duration("duration", duration),
	)

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			body := string(respBody)
			if len(body) > 200 {
				body = body[:200] + "... (truncated)"
			}
			return NewAPIError(resp.StatusCode, body, "")
		}
		return NewAPIError(resp.StatusCode, errResp.Error.Message, errResp.Error.Code)
	}

	// Decode success response
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// doIgnoreStatus performs an HTTP request and decodes the body into result
// regardless of HTTP status code. Returns the status code alongside any error.
func (c *Client) doIgnoreStatus(ctx context.Context, method, path string, body, result any) (int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if tok := c.GetToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	if int64(len(respBody)) > maxResponseSize {
		return resp.StatusCode, fmt.Errorf("response too large (>%d bytes)", maxResponseSize)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return resp.StatusCode, fmt.Errorf("decode response: %w", err)
		}
	}

	return resp.StatusCode, nil
}

// Get performs a GET request to the given path and decodes the JSON response
// into result. It returns an *APIError for non-2xx status codes.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.do(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request to the given path, encoding body as JSON and
// decoding the JSON response into result. It returns an *APIError for non-2xx
// status codes.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

// Put performs a PUT request to the given path, encoding body as JSON and
// decoding the JSON response into result. It returns an *APIError for non-2xx
// status codes.
func (c *Client) Put(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPut, path, body, result)
}

// Delete performs a DELETE request to the given path. It returns an *APIError
// for non-2xx status codes.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// Patch performs a PATCH request to the given path, encoding body as JSON and
// decoding the JSON response into result. It returns an *APIError for non-2xx
// status codes.
func (c *Client) Patch(ctx context.Context, path string, body, result any) error {
	return c.do(ctx, http.MethodPatch, path, body, result)
}

// NewClientFromConfig creates an API client from the active configuration
// profile, loading the server URL and access token automatically. It returns
// ErrNotAuthenticated (wrapped) when no valid credentials are found in the
// profile, prompting the caller to run "shoehorn auth login".
// An optional *zap.Logger can be passed to enable debug HTTP logging.
func NewClientFromConfig(opts ...func(*Client)) (*Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if !cfg.IsAuthenticated() {
		return nil, fmt.Errorf("%w — run: shoehorn auth login --token <PAT>", ErrNotAuthenticated)
	}
	profile, err := cfg.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	c := NewClient(profile.Server)
	c.SetToken(profile.Auth.AccessToken)
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// WithLogger returns an option that sets the client's structured logger.
func WithLogger(l *zap.Logger) func(*Client) {
	return func(c *Client) {
		c.SetLogger(l)
	}
}

// ErrorResponse represents the standard JSON error envelope returned by the
// Shoehorn API on non-2xx responses.
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// isRetryable returns true if the error is a transient failure worth retrying.
// Network errors and specific HTTP status codes (429, 502, 503, 504) are retryable.
// Client errors (400, 401, 403, 404, 409, 422) are NOT retryable.
func isRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 429, 502, 503, 504:
			return true
		default:
			return false
		}
	}
	// Only retry known transient network errors. Default to NOT retrying
	// unknown errors (JSON marshal failures, invalid URLs, etc.).
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// Check for transient network errors (connection refused, DNS, timeout)
	msg := err.Error()
	for _, substr := range []string{"connection refused", "connection reset", "no such host", "i/o timeout", "EOF", "broken pipe"} {
		if strings.Contains(msg, substr) {
			return true
		}
	}
	return false
}

// retryWait calculates the wait duration before the next retry attempt.
// Uses exponential backoff (500ms, 1s, 2s) capped at retryMaxWait.
func retryWait(attempt int) time.Duration {
	// Cap shift to prevent overflow (retryBaseWait is 500ms = ~2^29 ns,
	// so shifting by more than 33 overflows int64).
	if attempt > 30 {
		return retryMaxWait
	}
	wait := retryBaseWait << uint(attempt)
	if wait <= 0 || wait > retryMaxWait {
		return retryMaxWait
	}
	return wait
}
