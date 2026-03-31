package commands

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/config"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
)

var (
	serverURL string
	patToken  string
)

// authCmd represents the auth command group
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication with the Shoehorn platform.`,
}

// loginCmd represents the auth login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Shoehorn",
	Long: `Authenticate with the Shoehorn platform using a Personal Access Token.

Create a PAT in the Shoehorn UI under Settings > API Keys, then run:
  shoehorn auth login --server http://localhost:8080 --token shp_your_token`,
	RunE: runLogin,
}

// statusCmd represents the auth status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Display current authentication status and profile information.`,
	RunE:  runStatus,
}

// logoutCmd represents the auth logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Shoehorn",
	Long:  `Clear local credentials. Note: tokens are not revoked on the server.`,
	RunE:  runLogout,
}

func init() {
	loginCmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Shoehorn API server URL")
	loginCmd.Flags().StringVar(&patToken, "token", "", "Personal Access Token (shp_xxx)")

	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(logoutCmd)

	rootCmd.AddCommand(authCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	token, source, err := resolveToken(patToken)
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("a Personal Access Token is required\n\nUsage:\n  shoehorn auth login --server <url> --token <PAT>\n\nPreferred (avoids process-list exposure):\n  export SHOEHORN_TOKEN_FILE=/path/to/token\n  export SHOEHORN_TOKEN=shp_xxx")
	}
	if source == "flag" {
		fmt.Fprintln(os.Stderr, "Warning: passing tokens via --token is visible in process lists. Prefer SHOEHORN_TOKEN_FILE or SHOEHORN_TOKEN env var instead.")
	}
	serverURL = NormalizeServerURL(serverURL)
	if err := validateServerSecurity(serverURL); err != nil {
		return err
	}
	return runLoginWithPAT(cmd.Context(), serverURL, token)
}

// maxTokenFileSize is the maximum allowed token file size (64 KiB).
const maxTokenFileSize = 1 << 16

// resolveToken determines the token to use. Priority order:
//  1. --token flag (explicit CLI argument)
//  2. SHOEHORN_TOKEN_FILE env var (read token from file — recommended for CI/CD)
//  3. SHOEHORN_TOKEN env var (token in environment)
//
// Returns the token, its source ("flag", "file", "env", or "none"), and an
// error when the user explicitly configured a source that cannot be read.
func resolveToken(flagValue string) (token, source string, err error) {
	if flagValue != "" {
		return flagValue, "flag", nil
	}
	if tokenFile := os.Getenv("SHOEHORN_TOKEN_FILE"); tokenFile != "" {
		info, statErr := os.Stat(tokenFile)
		if statErr != nil {
			return "", "file", fmt.Errorf("SHOEHORN_TOKEN_FILE: %w", statErr)
		}
		if info.IsDir() {
			return "", "file", fmt.Errorf("SHOEHORN_TOKEN_FILE %q is a directory, not a file", tokenFile)
		}
		if info.Size() > maxTokenFileSize {
			return "", "file", fmt.Errorf("SHOEHORN_TOKEN_FILE %q is %d bytes (max %d)", tokenFile, info.Size(), maxTokenFileSize)
		}
		data, readErr := os.ReadFile(tokenFile)
		if readErr != nil {
			return "", "file", fmt.Errorf("SHOEHORN_TOKEN_FILE: %w", readErr)
		}
		t := strings.TrimSpace(string(data))
		if t == "" {
			return "", "file", fmt.Errorf("SHOEHORN_TOKEN_FILE %q is empty", tokenFile)
		}
		return t, "file", nil
	}
	if envToken := os.Getenv("SHOEHORN_TOKEN"); envToken != "" {
		return envToken, "env", nil
	}
	return "", "none", nil
}

// validateServerSecurity checks that non-localhost servers use HTTPS.
// HTTP is only allowed for localhost/127.0.0.1/[::1] (development).
func validateServerSecurity(serverURL string) error {
	if serverURL == "" {
		return nil
	}
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q: use https:// for remote servers or http://localhost for local development", u.Scheme)
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil // HTTP is fine for local development
	}
	return fmt.Errorf("refusing plaintext HTTP connection to %q — your token would be sent unencrypted.\nUse HTTPS: %s",
		host, strings.Replace(serverURL, "http://", "https://", 1))
}

// runLoginWithPAT authenticates using a Personal Access Token
func runLoginWithPAT(ctx context.Context, server, token string) error {
	client := api.NewClient(server)
	client.SetToken(token)

	// Verify token by calling /me
	result, err := tui.RunSpinner("Verifying token...", func() (any, error) {
		return client.GetMe(ctx)
	})
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	me := result.(*api.MeResponse)

	// Save config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	currentProfile := cfg.Profiles[cfg.CurrentProfile]
	if currentProfile == nil {
		currentProfile = &config.Profile{Name: cfg.CurrentProfile}
		cfg.Profiles[cfg.CurrentProfile] = currentProfile
	}

	currentProfile.Server = server
	currentProfile.Auth = &config.Auth{
		ProviderType: "pat",
		Issuer:       server,
		AccessToken:  token,
		User: &config.User{
			Email:    me.Email,
			Name:     me.Name,
			TenantID: me.TenantID,
		},
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Success panel
	var lines []string
	if me.Name != "" {
		lines = append(lines, fmt.Sprintf("%s  %s", tui.LabelStyle.Render("Name"), me.Name))
	}
	if me.Email != "" {
		lines = append(lines, fmt.Sprintf("%s  %s", tui.LabelStyle.Render("Email"), me.Email))
	}
	if me.Name == "" && me.Email == "" {
		lines = append(lines, fmt.Sprintf("%s  %s", tui.LabelStyle.Render("Account"), "Service Account (PAT)"))
	}
	lines = append(lines, fmt.Sprintf("%s  %s", tui.LabelStyle.Render("Tenant"), me.TenantID))
	lines = append(lines, fmt.Sprintf("%s  %s", tui.LabelStyle.Render("Server"), server))
	if len(me.Roles) > 0 {
		lines = append(lines, fmt.Sprintf("%s  %s", tui.LabelStyle.Render("Roles"), strings.Join(me.Roles, ", ")))
	}
	fmt.Println(tui.SuccessBox("Authenticated with PAT", strings.Join(lines, "\n")))
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	currentProfile, err := cfg.GetCurrentProfile()
	if err != nil {
		return err
	}

	fmt.Printf("Profile: %s\n", cfg.CurrentProfile)
	fmt.Printf("Server:  %s\n", currentProfile.Server)

	if !cfg.IsAuthenticated() {
		fmt.Println("Status:  Not authenticated")
		fmt.Println()
		fmt.Println("Run 'shoehorn auth login --token <PAT>' to authenticate")
		return nil
	}

	if cfg.IsPATAuth() {
		fmt.Println("Status:  Authenticated (PAT)")
	} else {
		fmt.Println("Status:  Authenticated")
	}

	if currentProfile.Auth.User != nil {
		if currentProfile.Auth.User.Email != "" {
			fmt.Printf("Email:   %s\n", currentProfile.Auth.User.Email)
		}
		if currentProfile.Auth.User.Name != "" {
			fmt.Printf("Name:    %s\n", currentProfile.Auth.User.Name)
		}
		if currentProfile.Auth.User.Email == "" && currentProfile.Auth.User.Name == "" {
			fmt.Println("Account: Service Account (PAT)")
		}
		if currentProfile.Auth.User.TenantID != "" {
			fmt.Printf("Tenant:  %s\n", currentProfile.Auth.User.TenantID)
		}
	}

	if cfg.IsTokenExpired() {
		fmt.Println("Token:   Expired (use 'shoehorn auth login' to refresh)")
	} else if cfg.IsPATAuth() {
		fmt.Println("Token:   Valid (PAT, no expiry)")
	} else {
		timeUntilExpiry := time.Until(currentProfile.Auth.ExpiresAt)
		fmt.Printf("Token:   Valid (expires in %s)\n", formatDuration(timeUntilExpiry))
	}

	// Verify with server
	if currentProfile.Auth.AccessToken != "" && !cfg.IsTokenExpired() {
		client := api.NewClient(currentProfile.Server)
		client.SetToken(currentProfile.Auth.AccessToken)
		ctx := cmd.Context()
		serverStatus, err := client.GetAuthStatus(ctx)
		if err != nil {
			fmt.Println("Server:  Unable to verify (offline or token invalid)")
		} else if serverStatus.Authenticated {
			fmt.Println("Server:  Token verified with server")
		} else {
			fmt.Println("Server:  Token rejected by server")
		}
	}

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	currentProfile, err := cfg.GetCurrentProfile()
	if err != nil {
		return err
	}

	currentProfile.Auth = nil

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Logged out from profile: %s\n", cfg.CurrentProfile)
	fmt.Println("Note: Tokens are not revoked on the server. They will expire naturally.")
	return nil
}

// NormalizeServerURL ensures the URL has an HTTP(S) scheme and no trailing
// slashes. A bare hostname such as "api.example.com" is prefixed with "https://".
// The function returns the cleaned URL suitable for use as a Client base URL.
func NormalizeServerURL(url string) string {
	if url != "" && !hasScheme(url) {
		url = "https://" + url
	}
	for len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	return url
}

func hasScheme(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}
