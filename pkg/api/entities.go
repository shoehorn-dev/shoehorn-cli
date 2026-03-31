package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// ─── /me ────────────────────────────────────────────────────────────────────

// MeResponse represents the current user's full profile
type MeResponse struct {
	ID       string   `json:"id"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Groups   []string `json:"groups"`
	Teams    []string `json:"teams"`
}

// meAPIResponse matches the actual API JSON shape for /me
type meAPIResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	User      string   `json:"user"` // API returns username in "user" field
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Tenant    string   `json:"tenant"`
	Roles     []string `json:"roles"`
	Groups    []string `json:"groups"`
	Teams     []string `json:"teams"`
}

// GetMe fetches the current user's profile
func (c *Client) GetMe(ctx context.Context) (*MeResponse, error) {
	var raw meAPIResponse
	if err := c.Get(ctx, "/api/v1/me", &raw); err != nil {
		return nil, fmt.Errorf("get me: %w", err)
	}
	name := raw.Name
	if name == "" {
		name = strings.TrimSpace(raw.FirstName + " " + raw.LastName)
	}
	if name == "" {
		name = raw.User
	}
	email := raw.Email
	// If the API returned a numeric ID instead of a real email, clear it
	// so callers don't display meaningless IDs to the user.
	if email != "" && !strings.Contains(email, "@") {
		email = ""
	}
	if name != "" && !strings.Contains(name, " ") && !strings.Contains(name, "@") && name == raw.ID {
		// Name is just the numeric ID — not useful
		name = ""
	}

	return &MeResponse{
		ID:       raw.ID,
		Email:    email,
		Name:     name,
		TenantID: raw.Tenant,
		Roles:    raw.Roles,
		Groups:   raw.Groups,
		Teams:    raw.Teams,
	}, nil
}

// ─── Entities ────────────────────────────────────────────────────────────────

// Entity represents a catalog entity (summary form)
type Entity struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Type        string   `json:"type"`
	Owner       string   `json:"owner"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	TenantID    string   `json:"tenant_id"`
}

// EntityDetail represents full entity detail with all sub-resources
type EntityDetail struct {
	Entity
	Links     []EntityLink `json:"links"`
	Lifecycle string       `json:"lifecycle"`
	Tier      string       `json:"tier"`
}

// EntityLink represents a link on an entity
type EntityLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Icon  string `json:"icon"`
}

// Resource represents an entity resource (K8s workload, etc.)
type Resource struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Replicas  string `json:"replicas"`
}

// EntityStatus represents entity health/status
type EntityStatus struct {
	Health        string  `json:"health"`
	Uptime        float64 `json:"uptime"`
	LastDeployAt  string  `json:"last_deploy_at"`
	IncidentCount int     `json:"incident_count"`
}

// ChangelogEntry represents a single changelog item
type ChangelogEntry struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Timestamp   string `json:"timestamp"`
}

// ListEntitiesOpts holds optional filters for listing entities
type ListEntitiesOpts struct {
	Type   string
	Search string
	Owner  string
}

// entityOwnerRef matches the API owner array element: [{"id":"team-slug","type":"team"}]
type entityOwnerRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// entityServiceInfo matches the API service block: {"id":"...", "name":"...", "type":"..."}
type entityServiceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// entityAPIItem matches a single entity from the API response
type entityAPIItem struct {
	Service     entityServiceInfo `json:"service"`
	Owner       json.RawMessage   `json:"owner"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Lifecycle   string            `json:"lifecycle"`
	Links       []EntityLink      `json:"links"`
}

// entitiesAPIResponse matches the actual API paginated response
type entitiesAPIResponse struct {
	Entities []entityAPIItem `json:"entities"`
	Page     struct {
		Total      int    `json:"total"`
		NextCursor string `json:"nextCursor"`
	} `json:"page"`
}

// parseOwner extracts the first owner ID from the owner field.
// The API returns owner as an array of objects: [{"id":"team-slug","type":"team"}]
func parseOwner(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var owners []entityOwnerRef
	if err := json.Unmarshal(raw, &owners); err == nil && len(owners) > 0 {
		return owners[0].ID
	}
	// Fallback: try as a plain string
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

// ListEntities returns all entities matching the given filters (handles pagination)
func (c *Client) ListEntities(ctx context.Context, opts ListEntitiesOpts) ([]*Entity, error) {
	q := url.Values{}
	if opts.Type != "" {
		q.Set("type", opts.Type)
	}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if opts.Owner != "" {
		q.Set("owner", opts.Owner)
	}
	q.Set("limit", "100")

	path := "/api/v1/entities?" + q.Encode()

	var resp entitiesAPIResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}

	if resp.Page.NextCursor != "" {
		c.logger.Warn("results truncated",
			zap.Int("returned", len(resp.Entities)),
			zap.Int("total", resp.Page.Total),
			zap.String("hint", "results limited to 100; more available via pagination"),
		)
	}

	entities := make([]*Entity, len(resp.Entities))
	for i, raw := range resp.Entities {
		entities[i] = &Entity{
			ID:          raw.Service.ID,
			Name:        raw.Service.Name,
			Slug:        raw.Service.ID,
			Type:        raw.Service.Type,
			Owner:       parseOwner(raw.Owner),
			Description: raw.Description,
			Tags:        raw.Tags,
		}
	}
	return entities, nil
}

// entityDetailAPIResponse matches the single entity API response
type entityDetailAPIResponse struct {
	Service     entityServiceInfo  `json:"service"`
	Owner       json.RawMessage    `json:"owner"`
	Description string             `json:"description"`
	Tags        []string           `json:"tags"`
	Lifecycle   string             `json:"lifecycle"`
	Links       []entityDetailLink `json:"links"`
}

type entityDetailLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
}

// GetEntity fetches a single entity by ID or slug
func (c *Client) GetEntity(ctx context.Context, id string) (*EntityDetail, error) {
	var wrapper struct {
		Entity entityDetailAPIResponse `json:"entity"`
	}
	if err := c.Get(ctx, "/api/v1/entities/"+url.PathEscape(id), &wrapper); err != nil {
		return nil, fmt.Errorf("get entity %s: %w", id, err)
	}
	raw := wrapper.Entity

	links := make([]EntityLink, len(raw.Links))
	for i, l := range raw.Links {
		links[i] = EntityLink{Title: l.Name, URL: l.URL, Icon: l.Icon}
	}

	return &EntityDetail{
		Entity: Entity{
			ID:          raw.Service.ID,
			Name:        raw.Service.Name,
			Slug:        raw.Service.ID,
			Type:        raw.Service.Type,
			Owner:       parseOwner(raw.Owner),
			Description: raw.Description,
			Tags:        raw.Tags,
		},
		Links:     links,
		Lifecycle: raw.Lifecycle,
	}, nil
}

// GetEntityResources fetches an entity's associated resources
func (c *Client) GetEntityResources(ctx context.Context, id string) ([]*Resource, error) {
	var resp struct {
		Resources []Resource `json:"resources"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/api/v1/entities/%s/resources", url.PathEscape(id)), &resp); err != nil {
		return nil, fmt.Errorf("get entity resources %s: %w", id, err)
	}
	resources := make([]*Resource, len(resp.Resources))
	for i := range resp.Resources {
		resources[i] = &resp.Resources[i]
	}
	return resources, nil
}

// GetEntityStatus fetches an entity's live health/status
func (c *Client) GetEntityStatus(ctx context.Context, id string) (*EntityStatus, error) {
	var resp EntityStatus
	if err := c.Get(ctx, fmt.Sprintf("/api/v1/entities/%s/status", url.PathEscape(id)), &resp); err != nil {
		return nil, fmt.Errorf("get entity status %s: %w", id, err)
	}
	return &resp, nil
}

// GetEntityChangelog fetches an entity's changelog entries
func (c *Client) GetEntityChangelog(ctx context.Context, id string) ([]*ChangelogEntry, error) {
	var resp struct {
		Entries []ChangelogEntry `json:"entries"`
	}
	if err := c.Get(ctx, fmt.Sprintf("/api/v1/entities/%s/changelog", url.PathEscape(id)), &resp); err != nil {
		return nil, fmt.Errorf("get entity changelog %s: %w", id, err)
	}
	entries := make([]*ChangelogEntry, len(resp.Entries))
	for i := range resp.Entries {
		e := resp.Entries[i]
		entries[i] = &e
	}
	return entries, nil
}

// ─── Scorecard ───────────────────────────────────────────────────────────────

// Scorecard represents an entity's scorecard result
type Scorecard struct {
	Score     int              `json:"score"`
	Grade     string           `json:"grade"`
	MaxScore  int              `json:"max_score"`
	Checks    []ScorecardCheck `json:"checks"`
	UpdatedAt string           `json:"updated_at"`
}

// ScorecardCheck is a single check in a scorecard
type ScorecardCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Weight  int    `json:"weight"`
	Message string `json:"message"`
}

// GetEntityScorecard fetches an entity's scorecard
func (c *Client) GetEntityScorecard(ctx context.Context, id string) (*Scorecard, error) {
	var resp Scorecard
	if err := c.Get(ctx, fmt.Sprintf("/api/v1/entities/%s/scorecard", url.PathEscape(id)), &resp); err != nil {
		return nil, fmt.Errorf("get entity scorecard %s: %w", id, err)
	}
	return &resp, nil
}

// ─── Entity Write Operations ────────────────────────────────────────────────

// ManifestEntityResponse is the response from create/update entity via manifest.
type ManifestEntityResponse struct {
	Success bool `json:"success"`
	Entity  struct {
		ID          any    `json:"id"` // API returns int or string depending on context
		ServiceID   string `json:"serviceId"`
		Name        string `json:"name"`
		Type        string `json:"type"`
		Lifecycle   string `json:"lifecycle"`
		Description string `json:"description"`
		Source      string `json:"source"`
		CreatedAt   string `json:"createdAt"`
		UpdatedAt   string `json:"updatedAt"`
	} `json:"entity"`
}

// manifestEntityRequest is the request body for create/update entity.
type manifestEntityRequest struct {
	Content string `json:"content"`
	Source  string `json:"source"`
}

// CreateEntityFromManifest creates a new entity from YAML manifest content.
// The manifest is parsed and validated server-side.
func (c *Client) CreateEntityFromManifest(ctx context.Context, manifestContent string) (*ManifestEntityResponse, error) {
	req := manifestEntityRequest{
		Content: manifestContent,
		Source:  "cli",
	}
	var resp ManifestEntityResponse
	if err := c.Post(ctx, "/api/v1/manifests/entities", req, &resp); err != nil {
		return nil, fmt.Errorf("create entity: %w", err)
	}
	return &resp, nil
}

// UpdateEntityFromManifest updates an existing entity from YAML manifest content.
// The serviceId in the manifest must match the id parameter.
func (c *Client) UpdateEntityFromManifest(ctx context.Context, id string, manifestContent string) (*ManifestEntityResponse, error) {
	req := manifestEntityRequest{
		Content: manifestContent,
		Source:  "cli",
	}
	var resp ManifestEntityResponse
	if err := c.Put(ctx, "/api/v1/manifests/entities/"+url.PathEscape(id), req, &resp); err != nil {
		return nil, fmt.Errorf("update entity %s: %w", id, err)
	}
	return &resp, nil
}

// DeleteEntity deletes an entity by service ID.
func (c *Client) DeleteEntity(ctx context.Context, id string) error {
	if err := c.Delete(ctx, "/api/v1/manifests/entities/"+url.PathEscape(id)); err != nil {
		return fmt.Errorf("delete entity %s: %w", id, err)
	}
	return nil
}
