package api

import (
	"context"
	"fmt"
	"net/url"
	"sort"
)

// Mold represents a Forge mold (workflow template)
type Mold struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// MoldAction describes a named action a mold can perform
type MoldAction struct {
	Action      string `json:"action"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Primary     bool   `json:"primary"`
}

// MoldInput describes a single input parameter for a mold
type MoldInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Default     string `json:"default"`
}

// MoldStep describes a single step in a mold
type MoldStep struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

// MoldDetail is the full mold definition
type MoldDetail struct {
	Mold
	Actions []MoldAction `json:"actions"`
	Inputs  []MoldInput  `json:"inputs"`
	Steps   []MoldStep   `json:"steps"`
}

// moldAPIResponse matches the backend mold JSON (camelCase keys)
type moldAPIResponse struct {
	ID          string         `json:"id"`
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Actions     []MoldAction   `json:"actions"`
	Schema      map[string]any `json:"schema"`
	Defaults    map[string]any `json:"defaults"`
	InputOrder  []string       `json:"inputOrder"`
}

// parseMoldInputs derives MoldInput entries from a JSON Schema object and optional input order.
func parseMoldInputs(schema map[string]any, inputOrder []string, defaults map[string]any) []MoldInput {
	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) == 0 {
		return nil
	}

	// Determine required set
	requiredSet := map[string]bool{}
	if reqArr, ok := schema["required"].([]any); ok {
		for _, v := range reqArr {
			if s, ok := v.(string); ok {
				requiredSet[s] = true
			}
		}
	}

	// Determine field order: use inputOrder if available, otherwise sorted keys
	// (Go map iteration order is random, so sort for deterministic output)
	var orderedKeys []string
	if len(inputOrder) > 0 {
		orderedKeys = inputOrder
	} else {
		for k := range props {
			orderedKeys = append(orderedKeys, k)
		}
		sort.Strings(orderedKeys)
	}

	var inputs []MoldInput
	for _, key := range orderedKeys {
		propRaw, exists := props[key]
		if !exists {
			continue
		}
		prop, ok := propRaw.(map[string]any)
		if !ok {
			continue
		}

		inp := MoldInput{
			Name:     key,
			Required: requiredSet[key],
		}

		if t, ok := prop["type"].(string); ok {
			inp.Type = t
		}
		if d, ok := prop["description"].(string); ok {
			inp.Description = d
		}
		if def, ok := prop["default"]; ok {
			inp.Default = fmt.Sprintf("%v", def)
		} else if defaults != nil {
			if def, ok := defaults[key]; ok {
				inp.Default = fmt.Sprintf("%v", def)
			}
		}

		inputs = append(inputs, inp)
	}
	return inputs
}

// MoldsResponse is the response from /forge/molds
type MoldsResponse struct {
	Molds []Mold `json:"molds"`
}

// ForgeRun represents a workflow run (canonical type for the api package)
type ForgeRun struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	MoldSlug    string `json:"mold_slug"`
	Status      string `json:"status"`
	DryRun      bool   `json:"dry_run"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ForgeRunsResponse is the response from /forge/runs
type ForgeRunsResponse struct {
	Runs       []ForgeRun `json:"runs"`
	Pagination struct {
		TotalCount int64  `json:"total_count"`
		NextCursor string `json:"next_cursor"`
		HasMore    bool   `json:"has_more"`
	} `json:"pagination"`
}

// CreateRunRequest is the body for POST /forge/runs
type CreateRunRequest struct {
	Action   string         `json:"action"`
	MoldSlug string         `json:"mold_slug,omitempty"`
	Inputs   map[string]any `json:"inputs,omitempty"`
	DryRun   bool           `json:"dry_run,omitempty"`
}

// ListMolds returns all forge molds
func (c *Client) ListMolds(ctx context.Context) ([]*Mold, error) {
	var resp MoldsResponse
	if err := c.Get(ctx, "/api/v1/forge/molds", &resp); err != nil {
		return nil, fmt.Errorf("list molds: %w", err)
	}
	molds := make([]*Mold, len(resp.Molds))
	for i := range resp.Molds {
		m := resp.Molds[i]
		molds[i] = &m
	}
	return molds, nil
}

// GetMold fetches a single mold by slug
func (c *Client) GetMold(ctx context.Context, slug string) (*MoldDetail, error) {
	var wrapper struct {
		Mold moldAPIResponse `json:"mold"`
	}
	if err := c.Get(ctx, "/api/v1/forge/molds/"+url.PathEscape(slug), &wrapper); err != nil {
		return nil, fmt.Errorf("get mold %s: %w", slug, err)
	}
	raw := wrapper.Mold
	inputs := parseMoldInputs(raw.Schema, raw.InputOrder, raw.Defaults)

	return &MoldDetail{
		Mold: Mold{
			ID:          raw.ID,
			Name:        raw.Name,
			Slug:        raw.Slug,
			Description: raw.Description,
			Version:     raw.Version,
		},
		Actions: raw.Actions,
		Inputs:  inputs,
	}, nil
}

// CreateRun starts a new forge run
func (c *Client) CreateRun(ctx context.Context, moldSlug, action string, inputs map[string]any, dryRun bool) (*ForgeRun, error) {
	req := CreateRunRequest{
		Action:   action,
		MoldSlug: moldSlug,
		Inputs:   inputs,
		DryRun:   dryRun,
	}
	var wrapper struct {
		Run ForgeRun `json:"run"`
	}
	if err := c.Post(ctx, "/api/v1/forge/runs", req, &wrapper); err != nil {
		return nil, fmt.Errorf("create run for mold %s: %w", moldSlug, err)
	}
	return &wrapper.Run, nil
}

// ListRuns returns forge workflow runs
func (c *Client) ListRuns(ctx context.Context) (*ForgeRunsResponse, error) {
	var resp ForgeRunsResponse
	if err := c.Get(ctx, "/api/v1/forge/runs", &resp); err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	return &resp, nil
}

// GetRun fetches a single forge run by ID
func (c *Client) GetRun(ctx context.Context, runID string) (*ForgeRun, error) {
	var wrapper struct {
		Run ForgeRun `json:"run"`
	}
	if err := c.Get(ctx, "/api/v1/forge/runs/"+url.PathEscape(runID), &wrapper); err != nil {
		return nil, fmt.Errorf("get run %s: %w", runID, err)
	}
	return &wrapper.Run, nil
}
