package api

import (
	"context"
	"fmt"
	"net/url"
)

// ─── Governance Types ────────────────────────────────────────────────────────

// GovernanceAction represents a governance action item (e.g. compliance fix, ownership gap).
type GovernanceAction struct {
	ID             string `json:"id"`
	EntityID       string `json:"entity_id"`
	EntityName     string `json:"entity_name"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Priority       string `json:"priority"`
	Status         string `json:"status"`
	SourceType     string `json:"source_type"`
	SourceID       string `json:"source_id"`
	AssignedTo     string `json:"assigned_to,omitempty"`
	DueDate        string `json:"due_date,omitempty"`
	SLADays        *int   `json:"sla_days,omitempty"`
	ResolutionNote string `json:"resolution_note,omitempty"`
	CreatedBy      string `json:"created_by,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

// GovernanceDashboard holds aggregate governance metrics.
type GovernanceDashboard struct {
	EntityCount        int                     `json:"entity_count"`
	ScoredEntityCount  int                     `json:"scored_entity_count"`
	OverallHealthScore float64                 `json:"overall_health_score"`
	OverallGrade       string                  `json:"overall_grade"`
	ActionsSummary     GovernanceActionsSummary `json:"actions_summary"`
	DocCoverage        GovernanceDocCoverage    `json:"doc_coverage"`
}

// GovernanceActionsSummary holds action counts by status.
type GovernanceActionsSummary struct {
	Open            int `json:"open"`
	InProgress      int `json:"in_progress"`
	Overdue         int `json:"overdue"`
	ResolvedLast30d int `json:"resolved_last_30d"`
}

// GovernanceDocCoverage holds documentation coverage metrics.
type GovernanceDocCoverage struct {
	Total      int     `json:"total"`
	WithReadme int     `json:"with_readme"`
	Percentage float64 `json:"percentage"`
}

// ListGovernanceActionsOpts holds optional filters for listing governance actions.
type ListGovernanceActionsOpts struct {
	Status     string
	Priority   string
	EntityID   string
	SourceType string
	Overdue    bool
}

// CreateGovernanceActionRequest is the body for POST /governance/actions.
type CreateGovernanceActionRequest struct {
	EntityID    string  `json:"entity_id"`
	EntityName  string  `json:"entity_name,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Priority    string  `json:"priority"`
	SourceType  string  `json:"source_type"`
	SourceID    string  `json:"source_id,omitempty"`
	AssignedTo  *string `json:"assigned_to,omitempty"`
	SLADays     *int    `json:"sla_days,omitempty"`
}

// UpdateGovernanceActionRequest is the body for PATCH /governance/actions/{id}.
type UpdateGovernanceActionRequest struct {
	Status         *string `json:"status,omitempty"`
	Priority       *string `json:"priority,omitempty"`
	AssignedTo     *string `json:"assigned_to,omitempty"`
	DueDate        *string `json:"due_date,omitempty"`
	ResolutionNote *string `json:"resolution_note,omitempty"`
}

// ─── Governance API Methods ──────────────────────────────────────────────────

// ListGovernanceActions returns governance actions matching the given filters.
// The second return value is the total count from the API response.
func (c *Client) ListGovernanceActions(ctx context.Context, opts ListGovernanceActionsOpts) ([]*GovernanceAction, int, error) {
	q := url.Values{}
	if opts.Status != "" {
		q.Set("status", opts.Status)
	}
	if opts.Priority != "" {
		q.Set("priority", opts.Priority)
	}
	if opts.EntityID != "" {
		q.Set("entity_id", opts.EntityID)
	}
	if opts.SourceType != "" {
		q.Set("source_type", opts.SourceType)
	}
	if opts.Overdue {
		q.Set("overdue", "true")
	}

	path := "/api/v1/governance/actions"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp struct {
		Actions []GovernanceAction `json:"actions"`
		Total   int                `json:"total"`
	}
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, 0, fmt.Errorf("list governance actions: %w", err)
	}

	actions := make([]*GovernanceAction, len(resp.Actions))
	for i := range resp.Actions {
		actions[i] = &resp.Actions[i]
	}
	return actions, resp.Total, nil
}

// GetGovernanceAction fetches a single governance action by ID.
// The API returns the action directly (not wrapped in an envelope).
func (c *Client) GetGovernanceAction(ctx context.Context, id string) (*GovernanceAction, error) {
	var action GovernanceAction
	if err := c.Get(ctx, "/api/v1/governance/actions/"+url.PathEscape(id), &action); err != nil {
		return nil, fmt.Errorf("get governance action %s: %w", id, err)
	}
	return &action, nil
}

// CreateGovernanceAction creates a new governance action and returns the full
// object by performing a follow-up GET after the initial POST.
func (c *Client) CreateGovernanceAction(ctx context.Context, req CreateGovernanceActionRequest) (*GovernanceAction, error) {
	var createResp struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	if err := c.Post(ctx, "/api/v1/governance/actions", req, &createResp); err != nil {
		return nil, fmt.Errorf("create governance action: %w", err)
	}
	return c.GetGovernanceAction(ctx, createResp.ID)
}

// UpdateGovernanceAction patches an existing governance action.
func (c *Client) UpdateGovernanceAction(ctx context.Context, id string, req UpdateGovernanceActionRequest) error {
	if err := c.Patch(ctx, "/api/v1/governance/actions/"+url.PathEscape(id), req, nil); err != nil {
		return fmt.Errorf("update governance action %s: %w", id, err)
	}
	return nil
}

// DeleteGovernanceAction removes a governance action by ID.
func (c *Client) DeleteGovernanceAction(ctx context.Context, id string) error {
	if err := c.Delete(ctx, "/api/v1/governance/actions/"+url.PathEscape(id)); err != nil {
		return fmt.Errorf("delete governance action %s: %w", id, err)
	}
	return nil
}

// GetGovernanceDashboard returns aggregate governance metrics.
func (c *Client) GetGovernanceDashboard(ctx context.Context) (*GovernanceDashboard, error) {
	var resp GovernanceDashboard
	if err := c.Get(ctx, "/api/v1/governance/dashboard", &resp); err != nil {
		return nil, fmt.Errorf("get governance dashboard: %w", err)
	}
	return &resp, nil
}
