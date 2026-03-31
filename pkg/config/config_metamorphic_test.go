package config

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Metamorphic: IsTokenExpired with PAT auth always returns false,
// regardless of ExpiresAt value.
func TestIsTokenExpired_Metamorphic_PATNeverExpires_Property(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random expiry time (could be in the past)
		hoursDelta := rapid.IntRange(-8760, 8760).Draw(t, "hoursDelta")
		expiresAt := time.Now().Add(time.Duration(hoursDelta) * time.Hour)

		cfg := &Config{
			CurrentProfile: "default",
			Profiles: map[string]*Profile{
				"default": {
					Auth: &Auth{
						ProviderType: "pat",
						AccessToken:  "shp_test",
						ExpiresAt:    expiresAt,
					},
				},
			},
		}

		if cfg.IsTokenExpired() {
			t.Fatalf("PAT auth with ExpiresAt=%v reported as expired", expiresAt)
		}
	})
}

// Metamorphic: IsAuthenticated is true if and only if AccessToken is non-empty.
func TestIsAuthenticated_Metamorphic_TokenPresenceDeterminesAuth_Property(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		token := rapid.String().Draw(t, "token")

		cfg := &Config{
			CurrentProfile: "default",
			Profiles: map[string]*Profile{
				"default": {
					Auth: &Auth{
						AccessToken: token,
					},
				},
			},
		}

		got := cfg.IsAuthenticated()
		want := token != ""
		if got != want {
			t.Fatalf("IsAuthenticated() = %v for token %q, want %v", got, token, want)
		}
	})
}

// Metamorphic: IsPATAuth is true if and only if ProviderType is "pat" AND token exists.
func TestIsPATAuth_Metamorphic_ProviderDeterminesPAT(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		provider := rapid.SampledFrom([]string{"pat", "oauth2", "device", ""}).Draw(t, "provider")

		cfg := &Config{
			CurrentProfile: "default",
			Profiles: map[string]*Profile{
				"default": {
					Auth: &Auth{
						ProviderType: provider,
						AccessToken:  "shp_test",
					},
				},
			},
		}

		got := cfg.IsPATAuth()
		want := provider == "pat"
		if got != want {
			t.Fatalf("IsPATAuth() = %v for provider %q, want %v", got, provider, want)
		}
	})
}

// Metamorphic: GetCurrentProfile always returns the profile matching CurrentProfile key.
func TestGetCurrentProfile_Metamorphic_MatchesKey(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`^[a-z]{1,8}$`).Draw(t, "name")
		server := rapid.StringMatching(`^https?://[a-z]{1,10}\.example\.com$`).Draw(t, "server")

		cfg := &Config{
			CurrentProfile: name,
			Profiles: map[string]*Profile{
				name: {
					Name:   name,
					Server: server,
				},
			},
		}

		profile, err := cfg.GetCurrentProfile()
		if err != nil {
			t.Fatal(err)
		}
		if profile.Server != server {
			t.Fatalf("GetCurrentProfile().Server = %q, want %q", profile.Server, server)
		}
	})
}
