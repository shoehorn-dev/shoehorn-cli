package api

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// K8sAgent represents a connected K8s agent (display model)
type K8sAgent struct {
	ID          string `json:"id"`
	ClusterName string `json:"cluster_name"`
	Status      string `json:"status"`
	Version     string `json:"version"`
	LastSeen    string `json:"last_seen"`
}

// k8sAgentAPIItem matches the actual API JSON shape for a single agent
type k8sAgentAPIItem struct {
	ID            int        `json:"id"`
	ClusterID     string     `json:"clusterId"`
	Name          string     `json:"name"`
	Status        string     `json:"status"`
	OnlineStatus  string     `json:"onlineStatus"`
	LastHeartbeat *time.Time `json:"lastHeartbeat,omitempty"`
}

// k8sAgentsAPIResponse matches the actual API response for /k8s/agents
type k8sAgentsAPIResponse struct {
	Agents []k8sAgentAPIItem `json:"agents"`
}

// formatLastSeen formats a time pointer as a human-readable string
func formatLastSeen(t *time.Time) string {
	if t == nil {
		return "never"
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// ListK8sAgents returns all registered K8s agents
func (c *Client) ListK8sAgents(ctx context.Context) ([]*K8sAgent, error) {
	var resp k8sAgentsAPIResponse
	if err := c.Get(ctx, "/api/v1/k8s/agents", &resp); err != nil {
		return nil, fmt.Errorf("list k8s agents: %w", err)
	}
	agents := make([]*K8sAgent, len(resp.Agents))
	for i, raw := range resp.Agents {
		agents[i] = &K8sAgent{
			ID:          strconv.Itoa(raw.ID),
			ClusterName: raw.ClusterID,
			Status:      raw.OnlineStatus,
			Version:     raw.Name,
			LastSeen:    formatLastSeen(raw.LastHeartbeat),
		}
	}
	return agents, nil
}
