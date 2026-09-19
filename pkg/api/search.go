package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SearchHit is a single search result item
type SearchHit struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Owner       string  `json:"owner"`
	Score       float64 `json:"score"`
}

// SearchResult wraps search hits
type SearchResult struct {
	Hits       []SearchHit `json:"hits"`
	TotalCount int         `json:"total_count"`
}

// searchAPIResult matches a single result from the actual API
type searchAPIResult struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// searchAPIResponse matches the actual API response for /search: the match
// count is the top-level total; page carries only limit, offset and nextCursor.
type searchAPIResponse struct {
	Results []searchAPIResult `json:"results"`
	Total   int               `json:"total"`
}

// Search performs a full-text search. limit is sent when positive; the API
// accepts 1-100 and defaults to 20.
func (c *Client) Search(ctx context.Context, query string, limit int) (*SearchResult, error) {
	q := url.Values{}
	q.Set("q", query)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var resp searchAPIResponse
	if err := c.Get(ctx, "/api/v1/search?"+q.Encode(), &resp); err != nil {
		return nil, fmt.Errorf("search %q: %w", query, err)
	}

	hits := make([]SearchHit, len(resp.Results))
	for i, r := range resp.Results {
		hits[i] = SearchHit{
			ID:          r.ID,
			Name:        r.Title,
			Type:        r.Type,
			Description: r.Description,
			Score:       r.Score,
		}
	}

	return &SearchResult{
		Hits:       hits,
		TotalCount: resp.Total,
	}, nil
}
