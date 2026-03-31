package config

import (
	"testing"
	"time"
)

// TestGetCurrentProfile_Found tests successful profile lookup.
func TestGetCurrentProfile_Found(t *testing.T) {
	cfg := &Config{
		CurrentProfile: "default",
		Profiles: map[string]*Profile{
			"default": {Name: "Default", Server: "http://localhost:8080"},
		},
	}

	profile, err := cfg.GetCurrentProfile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Server != "http://localhost:8080" {
		t.Errorf("Server = %q, want %q", profile.Server, "http://localhost:8080")
	}
}

// TestGetCurrentProfile_NotFound tests error on missing profile.
func TestGetCurrentProfile_NotFound(t *testing.T) {
	cfg := &Config{
		CurrentProfile: "nonexistent",
		Profiles:       map[string]*Profile{},
	}

	_, err := cfg.GetCurrentProfile()
	if err == nil {
		t.Error("expected error for missing profile, got nil")
	}
}

// TestSetProfile_CreatesMap tests that SetProfile initializes nil map.
func TestSetProfile_CreatesMap(t *testing.T) {
	cfg := &Config{Profiles: nil}
	cfg.SetProfile("test", &Profile{Name: "Test"})

	if cfg.Profiles == nil {
		t.Fatal("Profiles map is still nil after SetProfile")
	}
	if cfg.Profiles["test"] == nil {
		t.Error("profile not set")
	}
}

// TestSetProfile_OverwritesExisting tests that SetProfile replaces profiles.
func TestSetProfile_OverwritesExisting(t *testing.T) {
	cfg := &Config{
		Profiles: map[string]*Profile{
			"test": {Name: "Old"},
		},
	}
	cfg.SetProfile("test", &Profile{Name: "New"})

	if cfg.Profiles["test"].Name != "New" {
		t.Errorf("Name = %q, want %q", cfg.Profiles["test"].Name, "New")
	}
}

// TestIsAuthenticated_TableDriven tests authentication detection.
func TestIsAuthenticated_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "authenticated with token",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{AccessToken: "token123"}},
				},
			},
			want: true,
		},
		{
			name: "no auth block",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles:       map[string]*Profile{"default": {Name: "Default"}},
			},
			want: false,
		},
		{
			name: "empty token",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{AccessToken: ""}},
				},
			},
			want: false,
		},
		{
			name: "missing profile",
			cfg: &Config{
				CurrentProfile: "nonexistent",
				Profiles:       map[string]*Profile{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsAuthenticated()
			if got != tt.want {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsPATAuth_TableDriven tests PAT detection.
func TestIsPATAuth_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "PAT provider",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{ProviderType: "pat", AccessToken: "shp_xxx"}},
				},
			},
			want: true,
		},
		{
			name: "OIDC provider",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{ProviderType: "oidc", AccessToken: "jwt_xxx"}},
				},
			},
			want: false,
		},
		{
			name: "nil auth",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles:       map[string]*Profile{"default": {}},
			},
			want: false,
		},
		{
			name: "missing profile",
			cfg: &Config{
				CurrentProfile: "nonexistent",
				Profiles:       map[string]*Profile{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsPATAuth()
			if got != tt.want {
				t.Errorf("IsPATAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsTokenExpired_TableDriven tests token expiry detection.
func TestIsTokenExpired_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want bool
	}{
		{
			name: "PAT never expires",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{
						ProviderType: "pat",
						AccessToken:  "shp_xxx",
						ExpiresAt:    time.Now().Add(-1 * time.Hour), // past but PAT
					}},
				},
			},
			want: false,
		},
		{
			name: "JWT expired",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{
						ProviderType: "oidc",
						ExpiresAt:    time.Now().Add(-1 * time.Hour),
					}},
				},
			},
			want: true,
		},
		{
			name: "JWT not expired",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{
						ProviderType: "oidc",
						ExpiresAt:    time.Now().Add(1 * time.Hour),
					}},
				},
			},
			want: false,
		},
		{
			name: "zero time means no expiry",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles: map[string]*Profile{
					"default": {Auth: &Auth{
						ProviderType: "oidc",
						ExpiresAt:    time.Time{},
					}},
				},
			},
			want: false,
		},
		{
			name: "nil auth is expired",
			cfg: &Config{
				CurrentProfile: "default",
				Profiles:       map[string]*Profile{"default": {}},
			},
			want: true,
		},
		{
			name: "missing profile is expired",
			cfg: &Config{
				CurrentProfile: "nonexistent",
				Profiles:       map[string]*Profile{},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsTokenExpired()
			if got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsTokenExpired_Metamorphic_PATNeverExpires verifies the metamorphic property
// that PAT auth never expires regardless of the ExpiresAt value.
func TestIsTokenExpired_Metamorphic_PATNeverExpires(t *testing.T) {
	times := []time.Time{
		{},                                    // zero
		time.Now().Add(-24 * time.Hour),       // 1 day ago
		time.Now().Add(-365 * 24 * time.Hour), // 1 year ago
		time.Now().Add(1 * time.Hour),         // future
	}

	for _, expiresAt := range times {
		cfg := &Config{
			CurrentProfile: "default",
			Profiles: map[string]*Profile{
				"default": {Auth: &Auth{
					ProviderType: "pat",
					AccessToken:  "shp_xxx",
					ExpiresAt:    expiresAt,
				}},
			},
		}
		if cfg.IsTokenExpired() {
			t.Errorf("PAT should never expire, but IsTokenExpired()=true for ExpiresAt=%v", expiresAt)
		}
	}
}

// TestIsAuthenticated_Metamorphic_TokenPresenceDeterminesAuth verifies that
// authentication status depends only on token presence, not provider type.
func TestIsAuthenticated_Metamorphic_TokenPresenceDeterminesAuth(t *testing.T) {
	providers := []string{"pat", "oidc", "saml", ""}
	for _, provider := range providers {
		cfg := &Config{
			CurrentProfile: "default",
			Profiles: map[string]*Profile{
				"default": {Auth: &Auth{
					ProviderType: provider,
					AccessToken:  "some-token",
				}},
			},
		}
		if !cfg.IsAuthenticated() {
			t.Errorf("should be authenticated with token regardless of provider %q", provider)
		}
	}
}
