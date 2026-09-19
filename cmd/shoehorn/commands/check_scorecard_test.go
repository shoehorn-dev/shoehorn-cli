package commands

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/config"
	"github.com/spf13/cobra"
)

func scorecardServer(t *testing.T, score int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/entities/checkout-api/scorecard" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"entityId":"checkout-api","overallScore":%d,"grade":"D","rules":[],"calculatedAt":"2026-09-18T20:40:00Z"}`, score)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func useTestProfile(t *testing.T, serverURL string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]*config.Profile{
			"default": {Name: "default", Server: serverURL, Auth: &config.Auth{AccessToken: "test-token"}},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save test config: %v", err)
	}
}

func runRoot(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(args)
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	// Tests share one command tree; reset what earlier runs left behind.
	resetSilenceUsage(rootCmd)
	searchLimit = defaultSearchLimit
	err := rootCmd.Execute()
	return out.String(), err
}

func resetSilenceUsage(c *cobra.Command) {
	c.SilenceUsage = false
	for _, sub := range c.Commands() {
		resetSilenceUsage(sub)
	}
}

func TestCheckScorecard_PassesAtOrAboveTheMinimum(t *testing.T) {
	useTestProfile(t, scorecardServer(t, 64).URL)
	if _, err := runRoot(t, "check", "scorecard", "checkout-api", "--min-score", "50", "-I"); err != nil {
		t.Fatalf("score 64 with --min-score 50 failed: %v", err)
	}
}

func TestCheckScorecard_FailsBelowTheMinimumWithAPlainError(t *testing.T) {
	useTestProfile(t, scorecardServer(t, 40).URL)
	out, err := runRoot(t, "check", "scorecard", "checkout-api", "--min-score", "50", "-I")
	if err == nil || !strings.Contains(err.Error(), "40 below minimum 50") {
		t.Fatalf("err = %v, want the score and the minimum", err)
	}
	if strings.Contains(out, "Usage:") {
		t.Errorf("a failed check printed the usage block:\n%s", out)
	}
	if strings.Contains(out, "Error:") {
		t.Errorf("cobra printed the error; main prints it too, so it appeared twice:\n%s", out)
	}
}

func TestUnknownFlag_StillShowsUsage(t *testing.T) {
	out, err := runRoot(t, "check", "scorecard", "checkout-api", "--no-such-flag")
	if err == nil {
		t.Fatal("an unknown flag must fail")
	}
	if !strings.Contains(out, "Usage:") {
		t.Errorf("a flag mistake should still show how to call the command:\n%s", out)
	}
}
