// Package testutil provides test doubles and helpers for CLI command testing.
package testutil

import (
	"context"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

// MockClient implements all API client interfaces with configurable function fields.
// Set only the functions you need for your test; unset functions panic with a clear message.
type MockClient struct {
	// CatalogReader
	GetMeFunc              func(ctx context.Context) (*api.MeResponse, error)
	ListEntitiesFunc       func(ctx context.Context, opts api.ListEntitiesOpts) ([]*api.Entity, error)
	GetEntityFunc          func(ctx context.Context, id string) (*api.EntityDetail, error)
	GetEntityResourcesFunc func(ctx context.Context, id string) ([]*api.Resource, error)
	GetEntityStatusFunc    func(ctx context.Context, id string) (*api.EntityStatus, error)
	GetEntityChangelogFunc func(ctx context.Context, id string) ([]*api.ChangelogEntry, error)
	GetEntityScorecardFunc func(ctx context.Context, id string) (*api.Scorecard, error)
	ListTeamsFunc          func(ctx context.Context) ([]*api.Team, error)
	GetTeamFunc            func(ctx context.Context, idOrSlug string) (*api.TeamDetail, error)
	ListUsersFunc          func(ctx context.Context) ([]*api.User, error)
	GetUserFunc            func(ctx context.Context, id string) (*api.UserDetail, error)
	ListGroupsFunc         func(ctx context.Context) ([]*api.Group, error)
	GetGroupRolesFunc      func(ctx context.Context, groupName string) ([]*api.Role, error)
	SearchFunc             func(ctx context.Context, query string) (*api.SearchResult, error)
	ListK8sAgentsFunc      func(ctx context.Context) ([]*api.K8sAgent, error)

	// ForgeClient
	ListMoldsFunc func(ctx context.Context) ([]*api.Mold, error)
	GetMoldFunc   func(ctx context.Context, slug string) (*api.MoldDetail, error)
	ListRunsFunc  func(ctx context.Context) (*api.ForgeRunsResponse, error)
	GetRunFunc    func(ctx context.Context, runID string) (*api.ForgeRun, error)
	CreateRunFunc func(ctx context.Context, moldSlug, action string, inputs map[string]any, dryRun bool) (*api.ForgeRun, error)

	// AddonClient
	ListInstalledAddonsFunc  func(ctx context.Context) ([]*api.Addon, error)
	GetAddonStatusFunc       func(ctx context.Context, slug string) (*api.AddonStatus, error)
	GetAddonLogsFunc         func(ctx context.Context, slug string, limit int) ([]*api.AddonLogEntry, error)
	InstallAddonFunc         func(ctx context.Context, slug string) (*api.Addon, error)
	UninstallAddonFunc       func(ctx context.Context, slug string) error
	EnableAddonFunc          func(ctx context.Context, slug string) error
	DisableAddonFunc         func(ctx context.Context, slug string) error
	ListMarketplaceItemsFunc func(ctx context.Context, kind string) ([]*api.MarketplaceItem, error)
	PublishAddonManifestFunc func(ctx context.Context, manifest map[string]any) (*api.PublishResult, error)
	UploadAddonBundleFunc    func(ctx context.Context, slug string, bundles map[string][]byte) (*api.BundleUploadResult, error)

	// CatalogWriter
	CreateEntityFromManifestFunc func(ctx context.Context, manifestContent string) (*api.ManifestEntityResponse, error)
	UpdateEntityFromManifestFunc func(ctx context.Context, id string, manifestContent string) (*api.ManifestEntityResponse, error)
	DeleteEntityFunc             func(ctx context.Context, id string) error

	// ManifestClient
	ValidateManifestFunc func(ctx context.Context, content string) (*api.ValidateManifestResponse, error)
	ConvertManifestFunc  func(ctx context.Context, content string, targetType string, validate bool) (*api.ManifestConversionResponse, error)

	// AuthClient
	GetAuthStatusFunc func(ctx context.Context) (*api.AuthStatusResponse, error)
}

// --- CatalogReader ---

func (m *MockClient) GetMe(ctx context.Context) (*api.MeResponse, error) {
	if m.GetMeFunc == nil {
		panic("MockClient.GetMeFunc not set")
	}
	return m.GetMeFunc(ctx)
}

func (m *MockClient) ListEntities(ctx context.Context, opts api.ListEntitiesOpts) ([]*api.Entity, error) {
	if m.ListEntitiesFunc == nil {
		panic("MockClient.ListEntitiesFunc not set")
	}
	return m.ListEntitiesFunc(ctx, opts)
}

func (m *MockClient) GetEntity(ctx context.Context, id string) (*api.EntityDetail, error) {
	if m.GetEntityFunc == nil {
		panic("MockClient.GetEntityFunc not set")
	}
	return m.GetEntityFunc(ctx, id)
}

func (m *MockClient) GetEntityResources(ctx context.Context, id string) ([]*api.Resource, error) {
	if m.GetEntityResourcesFunc == nil {
		panic("MockClient.GetEntityResourcesFunc not set")
	}
	return m.GetEntityResourcesFunc(ctx, id)
}

func (m *MockClient) GetEntityStatus(ctx context.Context, id string) (*api.EntityStatus, error) {
	if m.GetEntityStatusFunc == nil {
		panic("MockClient.GetEntityStatusFunc not set")
	}
	return m.GetEntityStatusFunc(ctx, id)
}

func (m *MockClient) GetEntityChangelog(ctx context.Context, id string) ([]*api.ChangelogEntry, error) {
	if m.GetEntityChangelogFunc == nil {
		panic("MockClient.GetEntityChangelogFunc not set")
	}
	return m.GetEntityChangelogFunc(ctx, id)
}

func (m *MockClient) GetEntityScorecard(ctx context.Context, id string) (*api.Scorecard, error) {
	if m.GetEntityScorecardFunc == nil {
		panic("MockClient.GetEntityScorecardFunc not set")
	}
	return m.GetEntityScorecardFunc(ctx, id)
}

func (m *MockClient) ListTeams(ctx context.Context) ([]*api.Team, error) {
	if m.ListTeamsFunc == nil {
		panic("MockClient.ListTeamsFunc not set")
	}
	return m.ListTeamsFunc(ctx)
}

func (m *MockClient) GetTeam(ctx context.Context, idOrSlug string) (*api.TeamDetail, error) {
	if m.GetTeamFunc == nil {
		panic("MockClient.GetTeamFunc not set")
	}
	return m.GetTeamFunc(ctx, idOrSlug)
}

func (m *MockClient) ListUsers(ctx context.Context) ([]*api.User, error) {
	if m.ListUsersFunc == nil {
		panic("MockClient.ListUsersFunc not set")
	}
	return m.ListUsersFunc(ctx)
}

func (m *MockClient) GetUser(ctx context.Context, id string) (*api.UserDetail, error) {
	if m.GetUserFunc == nil {
		panic("MockClient.GetUserFunc not set")
	}
	return m.GetUserFunc(ctx, id)
}

func (m *MockClient) ListGroups(ctx context.Context) ([]*api.Group, error) {
	if m.ListGroupsFunc == nil {
		panic("MockClient.ListGroupsFunc not set")
	}
	return m.ListGroupsFunc(ctx)
}

func (m *MockClient) GetGroupRoles(ctx context.Context, groupName string) ([]*api.Role, error) {
	if m.GetGroupRolesFunc == nil {
		panic("MockClient.GetGroupRolesFunc not set")
	}
	return m.GetGroupRolesFunc(ctx, groupName)
}

func (m *MockClient) Search(ctx context.Context, query string) (*api.SearchResult, error) {
	if m.SearchFunc == nil {
		panic("MockClient.SearchFunc not set")
	}
	return m.SearchFunc(ctx, query)
}

func (m *MockClient) ListK8sAgents(ctx context.Context) ([]*api.K8sAgent, error) {
	if m.ListK8sAgentsFunc == nil {
		panic("MockClient.ListK8sAgentsFunc not set")
	}
	return m.ListK8sAgentsFunc(ctx)
}

// --- ForgeClient ---

func (m *MockClient) ListMolds(ctx context.Context) ([]*api.Mold, error) {
	if m.ListMoldsFunc == nil {
		panic("MockClient.ListMoldsFunc not set")
	}
	return m.ListMoldsFunc(ctx)
}

func (m *MockClient) GetMold(ctx context.Context, slug string) (*api.MoldDetail, error) {
	if m.GetMoldFunc == nil {
		panic("MockClient.GetMoldFunc not set")
	}
	return m.GetMoldFunc(ctx, slug)
}

func (m *MockClient) ListRuns(ctx context.Context) (*api.ForgeRunsResponse, error) {
	if m.ListRunsFunc == nil {
		panic("MockClient.ListRunsFunc not set")
	}
	return m.ListRunsFunc(ctx)
}

func (m *MockClient) GetRun(ctx context.Context, runID string) (*api.ForgeRun, error) {
	if m.GetRunFunc == nil {
		panic("MockClient.GetRunFunc not set")
	}
	return m.GetRunFunc(ctx, runID)
}

func (m *MockClient) CreateRun(ctx context.Context, moldSlug, action string, inputs map[string]any, dryRun bool) (*api.ForgeRun, error) {
	if m.CreateRunFunc == nil {
		panic("MockClient.CreateRunFunc not set")
	}
	return m.CreateRunFunc(ctx, moldSlug, action, inputs, dryRun)
}

// --- AddonClient ---

func (m *MockClient) ListInstalledAddons(ctx context.Context) ([]*api.Addon, error) {
	if m.ListInstalledAddonsFunc == nil {
		panic("MockClient.ListInstalledAddonsFunc not set")
	}
	return m.ListInstalledAddonsFunc(ctx)
}

func (m *MockClient) GetAddonStatus(ctx context.Context, slug string) (*api.AddonStatus, error) {
	if m.GetAddonStatusFunc == nil {
		panic("MockClient.GetAddonStatusFunc not set")
	}
	return m.GetAddonStatusFunc(ctx, slug)
}

func (m *MockClient) GetAddonLogs(ctx context.Context, slug string, limit int) ([]*api.AddonLogEntry, error) {
	if m.GetAddonLogsFunc == nil {
		panic("MockClient.GetAddonLogsFunc not set")
	}
	return m.GetAddonLogsFunc(ctx, slug, limit)
}

func (m *MockClient) InstallAddon(ctx context.Context, slug string) (*api.Addon, error) {
	if m.InstallAddonFunc == nil {
		panic("MockClient.InstallAddonFunc not set")
	}
	return m.InstallAddonFunc(ctx, slug)
}

func (m *MockClient) UninstallAddon(ctx context.Context, slug string) error {
	if m.UninstallAddonFunc == nil {
		panic("MockClient.UninstallAddonFunc not set")
	}
	return m.UninstallAddonFunc(ctx, slug)
}

func (m *MockClient) EnableAddon(ctx context.Context, slug string) error {
	if m.EnableAddonFunc == nil {
		panic("MockClient.EnableAddonFunc not set")
	}
	return m.EnableAddonFunc(ctx, slug)
}

func (m *MockClient) DisableAddon(ctx context.Context, slug string) error {
	if m.DisableAddonFunc == nil {
		panic("MockClient.DisableAddonFunc not set")
	}
	return m.DisableAddonFunc(ctx, slug)
}

func (m *MockClient) ListMarketplaceItems(ctx context.Context, kind string) ([]*api.MarketplaceItem, error) {
	if m.ListMarketplaceItemsFunc == nil {
		panic("MockClient.ListMarketplaceItemsFunc not set")
	}
	return m.ListMarketplaceItemsFunc(ctx, kind)
}

func (m *MockClient) PublishAddonManifest(ctx context.Context, manifest map[string]any) (*api.PublishResult, error) {
	if m.PublishAddonManifestFunc == nil {
		panic("MockClient.PublishAddonManifestFunc not set")
	}
	return m.PublishAddonManifestFunc(ctx, manifest)
}

func (m *MockClient) UploadAddonBundle(ctx context.Context, slug string, bundles map[string][]byte) (*api.BundleUploadResult, error) {
	if m.UploadAddonBundleFunc == nil {
		panic("MockClient.UploadAddonBundleFunc not set")
	}
	return m.UploadAddonBundleFunc(ctx, slug, bundles)
}

// --- CatalogWriter ---

func (m *MockClient) CreateEntityFromManifest(ctx context.Context, manifestContent string) (*api.ManifestEntityResponse, error) {
	if m.CreateEntityFromManifestFunc == nil {
		panic("MockClient.CreateEntityFromManifestFunc not set")
	}
	return m.CreateEntityFromManifestFunc(ctx, manifestContent)
}

func (m *MockClient) UpdateEntityFromManifest(ctx context.Context, id string, manifestContent string) (*api.ManifestEntityResponse, error) {
	if m.UpdateEntityFromManifestFunc == nil {
		panic("MockClient.UpdateEntityFromManifestFunc not set")
	}
	return m.UpdateEntityFromManifestFunc(ctx, id, manifestContent)
}

func (m *MockClient) DeleteEntity(ctx context.Context, id string) error {
	if m.DeleteEntityFunc == nil {
		panic("MockClient.DeleteEntityFunc not set")
	}
	return m.DeleteEntityFunc(ctx, id)
}

// --- ManifestClient ---

func (m *MockClient) ValidateManifest(ctx context.Context, content string) (*api.ValidateManifestResponse, error) {
	if m.ValidateManifestFunc == nil {
		panic("MockClient.ValidateManifestFunc not set")
	}
	return m.ValidateManifestFunc(ctx, content)
}

func (m *MockClient) ConvertManifest(ctx context.Context, content string, targetType string, validate bool) (*api.ManifestConversionResponse, error) {
	if m.ConvertManifestFunc == nil {
		panic("MockClient.ConvertManifestFunc not set")
	}
	return m.ConvertManifestFunc(ctx, content, targetType, validate)
}

// --- AuthClient ---

func (m *MockClient) GetAuthStatus(ctx context.Context) (*api.AuthStatusResponse, error) {
	if m.GetAuthStatusFunc == nil {
		panic("MockClient.GetAuthStatusFunc not set")
	}
	return m.GetAuthStatusFunc(ctx)
}

// Compile-time interface checks
var (
	_ api.CatalogReader  = (*MockClient)(nil)
	_ api.CatalogWriter  = (*MockClient)(nil)
	_ api.ForgeClient    = (*MockClient)(nil)
	_ api.AddonClient    = (*MockClient)(nil)
	_ api.ManifestClient = (*MockClient)(nil)
	_ api.AuthClient     = (*MockClient)(nil)
)
