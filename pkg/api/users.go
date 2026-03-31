package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// User represents a user summary
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// UserDetail includes groups, teams, and roles
type UserDetail struct {
	User
	Groups []string `json:"groups"`
	Teams  []string `json:"teams"`
	Roles  []string `json:"roles"`
}

// userAPIItem matches the actual API user JSON shape
type userAPIItem struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Username  string   `json:"username"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Groups    []string `json:"groups"`
	Teams     []string `json:"teams"`
	Roles     []string `json:"roles"`
}

// usersAPIResponse matches the actual API response for /users
type usersAPIResponse struct {
	Items []userAPIItem `json:"items"`
}

// ListUsers returns all users in the directory
func (c *Client) ListUsers(ctx context.Context) ([]*User, error) {
	var resp usersAPIResponse
	if err := c.Get(ctx, "/api/v1/users", &resp); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	users := make([]*User, len(resp.Items))
	for i, u := range resp.Items {
		name := strings.TrimSpace(u.FirstName + " " + u.LastName)
		if name == "" {
			name = u.Username
		}
		users[i] = &User{
			ID:    u.ID,
			Email: u.Email,
			Name:  name,
		}
	}
	return users, nil
}

// GetUser fetches a single user by ID
func (c *Client) GetUser(ctx context.Context, id string) (*UserDetail, error) {
	var raw userAPIItem
	if err := c.Get(ctx, "/api/v1/users/"+url.PathEscape(id), &raw); err != nil {
		return nil, fmt.Errorf("get user %s: %w", id, err)
	}
	name := strings.TrimSpace(raw.FirstName + " " + raw.LastName)
	if name == "" {
		name = raw.Username
	}
	return &UserDetail{
		User: User{
			ID:    raw.ID,
			Email: raw.Email,
			Name:  name,
		},
		Groups: raw.Groups,
		Teams:  raw.Teams,
		Roles:  raw.Roles,
	}, nil
}
