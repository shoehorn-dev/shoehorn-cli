package api

import (
	"context"
	"fmt"
	"net/url"
)

// ─── GitOps Types ────────────────────────────────────────────────────────────

// GitOpsResource represents a GitOps-managed resource (ArgoCD app, Flux kustomization, etc.).
type GitOpsResource struct {
	ID           string `json:"id"`
	ClusterID    string `json:"cluster_id"`
	Tool         string `json:"tool"`
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	SyncStatus   string `json:"sync_status"`
	HealthStatus string `json:"health_status"`
	SourceURL    string `json:"source_url,omitempty"`
	EntityID     string `json:"entity_id,omitempty"`
	EntityName   string `json:"entity_name,omitempty"`
	OwnerTeam    string `json:"owner_team,omitempty"`
	LastSyncedAt string `json:"last_synced_at,omitempty"`
}

// ListGitOpsResourcesOpts holds optional filters for listing GitOps resources.
type ListGitOpsResourcesOpts struct {
	ClusterID    string
	Tool         string
	SyncStatus   string
	HealthStatus string
}

// ─── GitOps API Methods ──────────────────────────────────────────────────────

// ListGitOpsResources returns GitOps resources matching the given filters.
func (c *Client) ListGitOpsResources(ctx context.Context, opts ListGitOpsResourcesOpts) ([]*GitOpsResource, error) {
	q := url.Values{}
	if opts.ClusterID != "" {
		q.Set("cluster_id", opts.ClusterID)
	}
	if opts.Tool != "" {
		q.Set("tool", opts.Tool)
	}
	if opts.SyncStatus != "" {
		q.Set("sync_status", opts.SyncStatus)
	}
	if opts.HealthStatus != "" {
		q.Set("health_status", opts.HealthStatus)
	}

	path := "/api/v1/operations/gitops"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp struct {
		Resources []GitOpsResource `json:"resources"`
	}
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list gitops resources: %w", err)
	}

	resources := make([]*GitOpsResource, len(resp.Resources))
	for i := range resp.Resources {
		resources[i] = &resp.Resources[i]
	}
	return resources, nil
}

// GetGitOpsResource fetches a single GitOps resource by ID.
func (c *Client) GetGitOpsResource(ctx context.Context, id string) (*GitOpsResource, error) {
	var resp struct {
		Resource GitOpsResource `json:"resource"`
	}
	if err := c.Get(ctx, "/api/v1/operations/gitops/"+url.PathEscape(id), &resp); err != nil {
		return nil, fmt.Errorf("get gitops resource %s: %w", id, err)
	}
	return &resp.Resource, nil
}
