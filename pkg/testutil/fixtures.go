package testutil

import "github.com/shoehorn-dev/shoehorn-cli/pkg/api"

// FixtureEntity returns a test entity with sensible defaults.
// Pass override functions to customize specific fields.
func FixtureEntity(overrides ...func(*api.Entity)) *api.Entity {
	e := &api.Entity{
		ID:          "test-service",
		Name:        "Test Service",
		Slug:        "test-service",
		Type:        "service",
		Owner:       "platform-team",
		Description: "A test service for unit tests",
		Tags:        []string{"test", "go"},
	}
	for _, fn := range overrides {
		fn(e)
	}
	return e
}

// FixtureEntityDetail returns a test entity detail with defaults.
func FixtureEntityDetail(overrides ...func(*api.EntityDetail)) *api.EntityDetail {
	ed := &api.EntityDetail{
		Entity: api.Entity{
			ID:          "test-service",
			Name:        "Test Service",
			Slug:        "test-service",
			Type:        "service",
			Owner:       "platform-team",
			Description: "A test service for unit tests",
			Tags:        []string{"test", "go"},
		},
		Lifecycle: "production",
		Tier:      "1",
		Links: []api.EntityLink{
			{Title: "GitHub", URL: "https://github.com/example/test-service"},
		},
	}
	for _, fn := range overrides {
		fn(ed)
	}
	return ed
}

// FixtureTeam returns a test team.
func FixtureTeam(overrides ...func(*api.Team)) *api.Team {
	t := &api.Team{
		ID:          "platform-team",
		Name:        "Platform Team",
		Slug:        "platform-team",
		Description: "Infrastructure and platform services",
		MemberCount: 5,
	}
	for _, fn := range overrides {
		fn(t)
	}
	return t
}

// FixtureMold returns a test mold.
func FixtureMold(overrides ...func(*api.Mold)) *api.Mold {
	m := &api.Mold{
		ID:          "mold-1",
		Name:        "Create Repo",
		Slug:        "create-repo",
		Description: "Creates a GitHub repository",
		Version:     "1.0.0",
	}
	for _, fn := range overrides {
		fn(m)
	}
	return m
}

// FixtureMoldDetail returns a test mold with actions and inputs.
func FixtureMoldDetail(overrides ...func(*api.MoldDetail)) *api.MoldDetail {
	md := &api.MoldDetail{
		Mold: api.Mold{
			ID:          "mold-1",
			Name:        "Create Repo",
			Slug:        "create-repo",
			Description: "Creates a GitHub repository",
			Version:     "1.0.0",
		},
		Actions: []api.MoldAction{
			{Action: "create", Label: "Create", Primary: true},
		},
		Inputs: []api.MoldInput{
			{Name: "name", Type: "string", Required: true, Description: "Repository name"},
			{Name: "owner", Type: "string", Required: false, Default: "my-org"},
		},
	}
	for _, fn := range overrides {
		fn(md)
	}
	return md
}

// FixtureForgeRun returns a test forge run.
func FixtureForgeRun(overrides ...func(*api.ForgeRun)) *api.ForgeRun {
	r := &api.ForgeRun{
		ID:       "run-123",
		MoldSlug: "create-repo",
		Action:   "create",
		Status:   "completed",
	}
	for _, fn := range overrides {
		fn(r)
	}
	return r
}

// FixtureMe returns a test user profile.
func FixtureMe(overrides ...func(*api.MeResponse)) *api.MeResponse {
	me := &api.MeResponse{
		ID:       "user-1",
		Email:    "jane@example.com",
		Name:     "Jane Smith",
		TenantID: "acme-corp",
		Roles:    []string{"admin"},
		Groups:   []string{"engineering"},
		Teams:    []string{"platform-team"},
	}
	for _, fn := range overrides {
		fn(me)
	}
	return me
}
