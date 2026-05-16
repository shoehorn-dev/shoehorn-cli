package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ─── Operations Resource Types ───────────────────────────────────────────────
//
// These mirror the platform's v2 Operations Resources API at
// /api/v1/operations/resources. A Resource is one workload aggregated across
// every cluster it runs in — see the platform's design.md §5.1 for the wire
// shape. The type is named OperationsResource to avoid colliding with the
// catalog-entity Resource already defined in this package.

// OperationsResource is one cross-cluster workload.
type OperationsResource struct {
	ID             string                      `json:"id"`
	Name           string                      `json:"name"`
	Subtype        string                      `json:"subtype"`
	Identity       OperationsResourceIdentity  `json:"identity"`
	Owner          *OperationsResourceOwner    `json:"owner"`
	Status         string                      `json:"status"`
	Issues         []OperationsResourceIssue   `json:"issues"`
	Sources        []OperationsResourceSource  `json:"sources"`
	Clusters       []OperationsResourceCluster `json:"clusters"`
	LastSeenAt     *time.Time                  `json:"last_seen_at"`
	LastDeployedAt *time.Time                  `json:"last_deployed_at"`
	MetricsP95CPU  *OperationsResourceMetric   `json:"metrics_p95_cpu"`
	MetricsP95Mem  *OperationsResourceMetric   `json:"metrics_p95_mem"`
}

// OperationsResourceIdentity is the quadruple-key plus a display name hint.
type OperationsResourceIdentity struct {
	ImageRepository string `json:"image_repository"`
	HelmReleaseName string `json:"helm_release_name"`
	RepoURL         string `json:"repo_url"`
	Kind            string `json:"kind"`
	NameHint        string `json:"name_hint"`
}

// OperationsResourceOwner is the owning team, when known.
type OperationsResourceOwner struct {
	TeamID   string `json:"team_id"`
	TeamName string `json:"team_name"`
	Source   string `json:"source"`
}

// OperationsResourceIssue is a single problem rolled into the status.
type OperationsResourceIssue struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// OperationsResourceSource is a source of intent (ArgoCD, Helm, Flux, repo).
type OperationsResourceSource struct {
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	URL        string     `json:"url"`
	LastSyncAt *time.Time `json:"last_sync_at,omitempty"`
	Declares   []string   `json:"declares,omitempty"`
}

// OperationsResourceCluster is the Resource's presence in one cluster.
type OperationsResourceCluster struct {
	ClusterID       string    `json:"cluster_id"`
	Namespace       string    `json:"namespace"`
	InstanceName    string    `json:"instance_name"`
	Status          string    `json:"status"`
	ReplicasReady   int       `json:"replicas_ready"`
	ReplicasDesired int       `json:"replicas_desired"`
	LastSyncAt      time.Time `json:"last_sync_at"`
}

// OperationsResourceMetric is a single P95 metric sample.
type OperationsResourceMetric struct {
	ValueMillicores int `json:"value_millicores"`
	ValueBytes      int `json:"value_bytes"`
	WindowHours     int `json:"window_hours"`
}

// OperationsResourceEvent is a Kubernetes event on the Resource. The detail
// endpoint returns an empty list until the events feed lands on the platform.
type OperationsResourceEvent struct {
	ClusterID string    `json:"cluster_id"`
	Kind      string    `json:"kind"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Count     int       `json:"count"`
}

// ListOperationsResourcesOpts holds optional filters for listing Resources.
// Cursor is opaque — pass back the NextCursor from a prior result verbatim,
// never construct or inspect it.
type ListOperationsResourcesOpts struct {
	ClusterID string
	Team      string
	Kind      string // comma-list accepted by the API (e.g. "Deployment,Job")
	Status    string
	Search    string
	HasGitOps bool
	Cursor    string
	Limit     int
}

// ListOperationsResourcesResult is one page of Resources plus pagination state.
type ListOperationsResourcesResult struct {
	Resources  []*OperationsResource
	NextCursor string
	Total      int
}

// OperationsResourceDetail is a single Resource with its event list.
type OperationsResourceDetail struct {
	Resource *OperationsResource
	Events   []OperationsResourceEvent
}

// ─── Operations Resource API Methods ─────────────────────────────────────────

// listOperationsResourcesResponse matches the wire envelope for GET /resources.
type listOperationsResourcesResponse struct {
	Resources  []OperationsResource `json:"resources"`
	NextCursor string               `json:"next_cursor"`
	Total      int                  `json:"total"`
}

// getOperationsResourceResponse matches the wire envelope for GET /resources/{id}.
type getOperationsResourceResponse struct {
	Resource OperationsResource        `json:"resource"`
	Events   []OperationsResourceEvent `json:"events"`
}

// ListOperationsResources returns one page of Resources matching the filters.
func (c *Client) ListOperationsResources(ctx context.Context, opts ListOperationsResourcesOpts) (*ListOperationsResourcesResult, error) {
	q := url.Values{}
	if opts.ClusterID != "" {
		q.Set("cluster", opts.ClusterID)
	}
	if opts.Team != "" {
		q.Set("team", opts.Team)
	}
	if kind := strings.TrimSpace(opts.Kind); kind != "" {
		q.Set("kind", kind)
	}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if opts.HasGitOps {
		q.Set("has_gitops", "true")
	}
	if opts.Cursor != "" {
		q.Set("cursor", opts.Cursor)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}

	path := "/api/v1/operations/resources"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp listOperationsResourcesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("list operations resources: %w", err)
	}

	resources := make([]*OperationsResource, len(resp.Resources))
	for i := range resp.Resources {
		resources[i] = &resp.Resources[i]
	}
	return &ListOperationsResourcesResult{
		Resources:  resources,
		NextCursor: resp.NextCursor,
		Total:      resp.Total,
	}, nil
}

// GetOperationsResource fetches a single Resource by ID, with its event list.
func (c *Client) GetOperationsResource(ctx context.Context, id string) (*OperationsResourceDetail, error) {
	var resp getOperationsResourceResponse
	if err := c.Get(ctx, "/api/v1/operations/resources/"+url.PathEscape(id), &resp); err != nil {
		return nil, fmt.Errorf("get operations resource %s: %w", id, err)
	}
	resource := resp.Resource
	return &OperationsResourceDetail{
		Resource: &resource,
		Events:   resp.Events,
	}, nil
}
