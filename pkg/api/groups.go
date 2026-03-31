package api

import (
	"context"
	"fmt"
	"net/url"
)

// Group represents a directory group
type Group struct {
	Name      string `json:"name"`
	RoleCount int    `json:"role_count"`
}

// Role represents a platform role
type Role struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// groupAPIItem matches the actual API group JSON shape
type groupAPIItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"memberCount"`
	Roles       []Role `json:"roles"`
}

// groupsAPIResponse matches the actual API response for /groups
type groupsAPIResponse struct {
	Items []groupAPIItem `json:"items"`
}

// RolesResponse is the response from /groups/{name}/roles
type RolesResponse struct {
	Roles []Role `json:"roles"`
}

// ListGroups returns all groups
func (c *Client) ListGroups(ctx context.Context) ([]*Group, error) {
	var resp groupsAPIResponse
	if err := c.Get(ctx, "/api/v1/groups", &resp); err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	groups := make([]*Group, len(resp.Items))
	for i, g := range resp.Items {
		groups[i] = &Group{
			Name:      g.Name,
			RoleCount: len(g.Roles),
		}
	}
	return groups, nil
}

// GetGroupRoles fetches the roles mapped to a group
func (c *Client) GetGroupRoles(ctx context.Context, groupName string) ([]*Role, error) {
	var resp RolesResponse
	if err := c.Get(ctx, fmt.Sprintf("/api/v1/groups/%s/roles", url.PathEscape(groupName)), &resp); err != nil {
		return nil, fmt.Errorf("get group roles %s: %w", groupName, err)
	}
	roles := make([]*Role, len(resp.Roles))
	for i := range resp.Roles {
		r := resp.Roles[i]
		roles[i] = &r
	}
	return roles, nil
}
