package api

import (
	"context"
	"fmt"
	"time"
)

// AuthStatusResponse contains current authentication status
type AuthStatusResponse struct {
	Authenticated bool      `json:"authenticated"`
	User          *UserInfo `json:"user,omitempty"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// UserInfo contains authenticated user information
type UserInfo struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	TenantID string `json:"tenant_id"`
}

// GetAuthStatus returns current authentication status (requires valid Bearer token)
func (c *Client) GetAuthStatus(ctx context.Context) (*AuthStatusResponse, error) {
	var resp AuthStatusResponse
	if err := c.Get(ctx, "/api/v1/auth/cli/status", &resp); err != nil {
		return nil, fmt.Errorf("get auth status: %w", err)
	}
	return &resp, nil
}
