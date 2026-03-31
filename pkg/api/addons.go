package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// ─── Addon Types ──────────────────────────────────────────────────────────────

// Addon represents an installed addon (from marketplace installations).
type Addon struct {
	ID          string `json:"id"`
	Slug        string `json:"itemSlug"`
	Kind        string `json:"itemKind"`
	Version     string `json:"itemVersion"`
	Enabled     bool   `json:"enabled"`
	SyncStatus  string `json:"syncStatus,omitempty"`
	LastSyncAt  string `json:"lastSyncAt,omitempty"`
	LastError   string `json:"lastError,omitempty"`
	InstalledBy string `json:"installedBy,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`

	// Addon runtime fields
	AddonStatus string `json:"addonStatus,omitempty"`
	VMMemory    int64  `json:"vmMemoryBytes,omitempty"`
}

// AddonStatus represents the runtime status of an addon.
type AddonStatus struct {
	Slug       string `json:"slug"`
	Status     string `json:"status"`
	Enabled    bool   `json:"enabled"`
	LastSyncAt string `json:"lastSyncAt,omitempty"`
	LastError  string `json:"lastError,omitempty"`
	ExecCount  int    `json:"execCount,omitempty"`
	ErrorCount int    `json:"errorCount,omitempty"`
	VMMemory   int64  `json:"vmMemoryBytes,omitempty"`
}

// AddonLogEntry represents a single log entry from an addon.
type AddonLogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// MarketplaceItem represents an item in the marketplace catalog (for install).
type MarketplaceItem struct {
	Slug        string `json:"slug"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	AuthorName  string `json:"authorName"`
	Tier        string `json:"tier"`
	Category    string `json:"category"`
	Status      string `json:"status"`
	Verified    bool   `json:"verified"`
	Featured    bool   `json:"featured"`
}

// ─── API Methods ──────────────────────────────────────────────────────────────

// ListInstalledAddons returns all installed marketplace items for the current tenant.
func (c *Client) ListInstalledAddons(ctx context.Context) ([]*Addon, error) {
	var resp struct {
		Installations []Addon `json:"installations"`
	}
	if err := c.Get(ctx, "/api/v1/marketplace/installed", &resp); err != nil {
		return nil, fmt.Errorf("list installed addons: %w", err)
	}

	var addons []*Addon
	for i := range resp.Installations {
		addons = append(addons, &resp.Installations[i])
	}
	return addons, nil
}

// GetAddonStatus returns the runtime status of a specific addon.
func (c *Client) GetAddonStatus(ctx context.Context, slug string) (*AddonStatus, error) {
	var status AddonStatus
	if err := c.Get(ctx, fmt.Sprintf("/api/v1/addons/%s/status", url.PathEscape(slug)), &status); err != nil {
		return nil, fmt.Errorf("get addon status: %w", err)
	}
	return &status, nil
}

// InstallAddon installs a marketplace item by slug.
func (c *Client) InstallAddon(ctx context.Context, slug string) (*Addon, error) {
	body := map[string]string{"slug": slug}
	var addon Addon
	if err := c.Post(ctx, "/api/v1/marketplace/install", body, &addon); err != nil {
		return nil, fmt.Errorf("install addon: %w", err)
	}
	return &addon, nil
}

// UninstallAddon removes an installed addon.
func (c *Client) UninstallAddon(ctx context.Context, slug string) error {
	if err := c.Delete(ctx, fmt.Sprintf("/api/v1/marketplace/%s/uninstall", url.PathEscape(slug))); err != nil {
		return fmt.Errorf("uninstall addon: %w", err)
	}
	return nil
}

// EnableAddon enables a disabled addon.
func (c *Client) EnableAddon(ctx context.Context, slug string) error {
	if err := c.Post(ctx, fmt.Sprintf("/api/v1/marketplace/%s/enable", url.PathEscape(slug)), nil, nil); err != nil {
		return fmt.Errorf("enable addon: %w", err)
	}
	return nil
}

// DisableAddon disables an addon without uninstalling.
func (c *Client) DisableAddon(ctx context.Context, slug string) error {
	if err := c.Post(ctx, fmt.Sprintf("/api/v1/marketplace/%s/disable", url.PathEscape(slug)), nil, nil); err != nil {
		return fmt.Errorf("disable addon: %w", err)
	}
	return nil
}

// GetAddonLogs returns recent log entries for an addon.
func (c *Client) GetAddonLogs(ctx context.Context, slug string, limit int) ([]*AddonLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	var resp struct {
		Entries []AddonLogEntry `json:"entries"`
	}
	path := fmt.Sprintf("/api/v1/addons/%s/logs?limit=%d", url.PathEscape(slug), limit)
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("get addon logs: %w", err)
	}

	var entries []*AddonLogEntry
	for i := range resp.Entries {
		entries = append(entries, &resp.Entries[i])
	}
	return entries, nil
}

// ListMarketplaceItems lists available marketplace items (for browsing before install).
func (c *Client) ListMarketplaceItems(ctx context.Context, kind string) ([]*MarketplaceItem, error) {
	path := "/api/v1/marketplace"
	if kind != "" {
		q := url.Values{}
		q.Set("kind", kind)
		path += "?" + q.Encode()
	}
	var resp struct {
		Items []MarketplaceItem `json:"items"`
	}
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list marketplace items: %w", err)
	}

	var items []*MarketplaceItem
	for i := range resp.Items {
		items = append(items, &resp.Items[i])
	}
	return items, nil
}

// PublishAddonManifest publishes an addon manifest to the marketplace.
func (c *Client) PublishAddonManifest(ctx context.Context, manifest map[string]any) (*PublishResult, error) {
	var result PublishResult
	if err := c.Post(ctx, "/api/v1/marketplace/import-manifest", manifest, &result); err != nil {
		return nil, fmt.Errorf("publish addon: %w", err)
	}
	return &result, nil
}

// UploadAddonBundle uploads backend and/or frontend bundles for an addon.
func (c *Client) UploadAddonBundle(ctx context.Context, slug string, bundles map[string][]byte) (*BundleUploadResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for fieldName, data := range bundles {
		part, err := writer.CreateFormFile(fieldName, fieldName+".js")
		if err != nil {
			return nil, fmt.Errorf("create form file %s: %w", fieldName, err)
		}
		if _, err := part.Write(data); err != nil {
			return nil, fmt.Errorf("write form file %s: %w", fieldName, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+fmt.Sprintf("/api/v1/marketplace/%s/bundle", url.PathEscape(slug)), &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if tok := c.GetToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload bundle: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if int64(len(respBody)) > maxResponseSize {
		return nil, fmt.Errorf("response too large (>%d bytes)", maxResponseSize)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Attempt structured error parsing, fall back to truncated raw body
		// to avoid leaking server internals (matches do() behavior).
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
			return nil, NewAPIError(resp.StatusCode, errResp.Error.Message, errResp.Error.Code)
		}
		body := string(respBody)
		if len(body) > 200 {
			body = body[:200] + "... (truncated)"
		}
		return nil, NewAPIError(resp.StatusCode, body, "")
	}

	var result BundleUploadResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// PublishResult represents the response from publishing an addon manifest.
type PublishResult struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Created   bool   `json:"created"`
	Installed bool   `json:"installed"`
}

// BundleUploadResult represents the response from uploading addon bundles.
type BundleUploadResult struct {
	Slug     string         `json:"slug"`
	Uploaded map[string]int `json:"uploaded"`
}
