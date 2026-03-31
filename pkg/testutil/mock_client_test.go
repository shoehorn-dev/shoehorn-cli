package testutil

import (
	"context"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

func TestMockClient_SatisfiesInterfaces(t *testing.T) {
	// These are compile-time checks via the var block in mock_client.go,
	// but verify at runtime too for clarity.
	var m MockClient
	var _ api.CatalogReader = &m
	var _ api.ForgeClient = &m
	var _ api.AddonClient = &m
	var _ api.ManifestClient = &m
	var _ api.AuthClient = &m
}

func TestMockClient_GetMe_ReturnsConfiguredValue(t *testing.T) {
	m := &MockClient{
		GetMeFunc: func(ctx context.Context) (*api.MeResponse, error) {
			return FixtureMe(), nil
		},
	}

	me, err := m.GetMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.Email != "jane@example.com" {
		t.Errorf("Email = %q", me.Email)
	}
}

func TestMockClient_UnsetFunc_Panics(t *testing.T) {
	m := &MockClient{} // no funcs set
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unset func")
		}
	}()
	m.GetMe(context.Background()) // should panic
}

func TestFixtureEntity_Defaults(t *testing.T) {
	e := FixtureEntity()
	if e.Name != "Test Service" {
		t.Errorf("Name = %q", e.Name)
	}
	if e.Type != "service" {
		t.Errorf("Type = %q", e.Type)
	}
}

func TestFixtureEntity_Override(t *testing.T) {
	e := FixtureEntity(func(e *api.Entity) {
		e.Name = "Custom"
		e.Type = "library"
	})
	if e.Name != "Custom" {
		t.Errorf("Name = %q", e.Name)
	}
	if e.Type != "library" {
		t.Errorf("Type = %q", e.Type)
	}
}

func TestNewTestServer_Routes(t *testing.T) {
	ts := NewTestServer(RouteHandler{
		"GET /api/v1/me": JSONHandler(map[string]string{"id": "u-1"}),
	})
	defer ts.Close()

	client := api.NewClient(ts.URL)
	client.SetToken("test")
	// GetMe calls c.Get which uses GET method
	me, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if me.ID != "u-1" {
		t.Errorf("ID = %q", me.ID)
	}
}

func TestNewTestServer_404ForUnknownRoute(t *testing.T) {
	ts := NewTestServer(RouteHandler{})
	defer ts.Close()

	client := api.NewClient(ts.URL)
	client.SetToken("test")
	_, err := client.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !api.IsNotFound(err) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
