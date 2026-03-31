package api

import "context"

// CatalogWriter defines write operations on catalog entities.
type CatalogWriter interface {
	CreateEntityFromManifest(ctx context.Context, manifestContent string) (*ManifestEntityResponse, error)
	UpdateEntityFromManifest(ctx context.Context, id string, manifestContent string) (*ManifestEntityResponse, error)
	DeleteEntity(ctx context.Context, id string) error
}

// CatalogReader defines read operations on the service catalog.
// Commands that only read catalog data should depend on this interface.
type CatalogReader interface {
	GetMe(ctx context.Context) (*MeResponse, error)
	ListEntities(ctx context.Context, opts ListEntitiesOpts) ([]*Entity, error)
	GetEntity(ctx context.Context, id string) (*EntityDetail, error)
	GetEntityResources(ctx context.Context, id string) ([]*Resource, error)
	GetEntityStatus(ctx context.Context, id string) (*EntityStatus, error)
	GetEntityChangelog(ctx context.Context, id string) ([]*ChangelogEntry, error)
	GetEntityScorecard(ctx context.Context, id string) (*Scorecard, error)
	ListTeams(ctx context.Context) ([]*Team, error)
	GetTeam(ctx context.Context, idOrSlug string) (*TeamDetail, error)
	ListUsers(ctx context.Context) ([]*User, error)
	GetUser(ctx context.Context, id string) (*UserDetail, error)
	ListGroups(ctx context.Context) ([]*Group, error)
	GetGroupRoles(ctx context.Context, groupName string) ([]*Role, error)
	Search(ctx context.Context, query string) (*SearchResult, error)
	ListK8sAgents(ctx context.Context) ([]*K8sAgent, error)
}

// ForgeClient defines operations on Forge workflows.
type ForgeClient interface {
	ListMolds(ctx context.Context) ([]*Mold, error)
	GetMold(ctx context.Context, slug string) (*MoldDetail, error)
	ListRuns(ctx context.Context) (*ForgeRunsResponse, error)
	GetRun(ctx context.Context, runID string) (*ForgeRun, error)
	CreateRun(ctx context.Context, moldSlug, action string, inputs map[string]any, dryRun bool) (*ForgeRun, error)
}

// AddonClient defines operations on addon management.
type AddonClient interface {
	ListInstalledAddons(ctx context.Context) ([]*Addon, error)
	GetAddonStatus(ctx context.Context, slug string) (*AddonStatus, error)
	GetAddonLogs(ctx context.Context, slug string, limit int) ([]*AddonLogEntry, error)
	InstallAddon(ctx context.Context, slug string) (*Addon, error)
	UninstallAddon(ctx context.Context, slug string) error
	EnableAddon(ctx context.Context, slug string) error
	DisableAddon(ctx context.Context, slug string) error
	ListMarketplaceItems(ctx context.Context, kind string) ([]*MarketplaceItem, error)
	PublishAddonManifest(ctx context.Context, manifest map[string]any) (*PublishResult, error)
	UploadAddonBundle(ctx context.Context, slug string, bundles map[string][]byte) (*BundleUploadResult, error)
}

// GovernanceClient defines operations on governance actions.
type GovernanceClient interface {
	ListGovernanceActions(ctx context.Context, opts ListGovernanceActionsOpts) ([]*GovernanceAction, int, error)
	GetGovernanceAction(ctx context.Context, id string) (*GovernanceAction, error)
	CreateGovernanceAction(ctx context.Context, req CreateGovernanceActionRequest) (*GovernanceAction, error)
	UpdateGovernanceAction(ctx context.Context, id string, req UpdateGovernanceActionRequest) error
	DeleteGovernanceAction(ctx context.Context, id string) error
	GetGovernanceDashboard(ctx context.Context) (*GovernanceDashboard, error)
}

// GitOpsClient defines operations on GitOps resources.
type GitOpsClient interface {
	ListGitOpsResources(ctx context.Context, opts ListGitOpsResourcesOpts) ([]*GitOpsResource, error)
	GetGitOpsResource(ctx context.Context, id string) (*GitOpsResource, error)
}

// ManifestClient defines operations on manifest validation and conversion.
type ManifestClient interface {
	ValidateManifest(ctx context.Context, content string) (*ValidateManifestResponse, error)
	ConvertManifest(ctx context.Context, content string, targetType string, validate bool) (*ManifestConversionResponse, error)
}

// AuthClient defines authentication operations.
type AuthClient interface {
	GetAuthStatus(ctx context.Context) (*AuthStatusResponse, error)
}

// Compile-time interface satisfaction checks.
var (
	_ CatalogReader    = (*Client)(nil)
	_ CatalogWriter    = (*Client)(nil)
	_ ForgeClient      = (*Client)(nil)
	_ AddonClient      = (*Client)(nil)
	_ GovernanceClient = (*Client)(nil)
	_ GitOpsClient     = (*Client)(nil)
	_ ManifestClient   = (*Client)(nil)
	_ AuthClient       = (*Client)(nil)
)
