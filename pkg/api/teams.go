package api

import (
	"context"
	"fmt"
	"net/url"
)

// Team represents a team summary
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	MemberCount int    `json:"member_count"`
}

// TeamDetail includes members and other team details
type TeamDetail struct {
	Team
	Members []TeamMember `json:"members"`
}

// TeamMember represents a member in a team
type TeamMember struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// TeamsResponse is the response from /teams
type TeamsResponse struct {
	Teams []Team `json:"teams"`
}

// ListTeams returns all teams
func (c *Client) ListTeams(ctx context.Context) ([]*Team, error) {
	var resp TeamsResponse
	if err := c.Get(ctx, "/api/v1/teams", &resp); err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	teams := make([]*Team, len(resp.Teams))
	for i := range resp.Teams {
		t := resp.Teams[i]
		teams[i] = &t
	}
	return teams, nil
}

// GetTeam fetches a team by ID or slug (including members)
func (c *Client) GetTeam(ctx context.Context, idOrSlug string) (*TeamDetail, error) {
	var wrapper struct {
		Team    Team         `json:"team"`
		Members []TeamMember `json:"members"`
	}
	if err := c.Get(ctx, "/api/v1/teams/"+url.PathEscape(idOrSlug), &wrapper); err != nil {
		return nil, fmt.Errorf("get team %s: %w", idOrSlug, err)
	}
	return &TeamDetail{
		Team:    wrapper.Team,
		Members: wrapper.Members,
	}, nil
}
