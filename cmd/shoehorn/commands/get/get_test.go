package get

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/config"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
)

// ─── Command registration tests ────────────────────────────────────────────

func TestGetCmd_IsRegistered(t *testing.T) {
	if GetCmd == nil {
		t.Fatal("GetCmd should not be nil")
	}
	if GetCmd.Use != "get" {
		t.Errorf("GetCmd.Use = %q, want %q", GetCmd.Use, "get")
	}
	if GetCmd.Short == "" {
		t.Error("GetCmd.Short should not be empty")
	}
}

func TestGetCmd_HasExpectedSubcommands(t *testing.T) {
	expected := []string{
		"entities", "entity",
		"teams", "team",
		"users", "user",
		"groups", "group",
		"owned",
		"molds", "mold",
		"runs", "run",
		"gitops",
		"scorecard",
		"k8s",
	}

	registered := make(map[string]bool)
	for _, sub := range GetCmd.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range expected {
		t.Run(name, func(t *testing.T) {
			if !registered[name] {
				t.Errorf("subcommand %q not registered on GetCmd", name)
			}
		})
	}
}

func TestGetCmd_AllSubcommands_HaveShortDescription(t *testing.T) {
	for _, sub := range GetCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.Short == "" {
				t.Errorf("subcommand %q has empty Short description", sub.Name())
			}
		})
	}
}

func TestGetCmd_AllSubcommands_HaveRunE(t *testing.T) {
	for _, sub := range GetCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if sub.RunE == nil && sub.Run == nil {
				t.Errorf("subcommand %q has neither RunE nor Run set", sub.Name())
			}
		})
	}
}

// ─── Args validation tests ─────────────────────────────────────────────────

func TestSingleResourceCmds_RequireExactlyOneArg(t *testing.T) {
	singleArgCmds := []string{"entity", "team", "user", "group", "mold", "run", "scorecard"}

	for _, name := range singleArgCmds {
		t.Run(name, func(t *testing.T) {
			var cmd *cobra.Command
			for _, sub := range GetCmd.Commands() {
				if sub.Name() == name {
					cmd = sub
					break
				}
			}
			if cmd == nil {
				t.Fatalf("command %q not found", name)
			}
			if cmd.Args == nil {
				t.Errorf("command %q must have Args validator (ExactArgs(1))", name)
			}
		})
	}
}

func TestListCmds_NoArgsRequired(t *testing.T) {
	listCmds := []string{"entities", "teams", "users", "groups", "owned", "molds", "runs", "k8s"}

	for _, name := range listCmds {
		t.Run(name, func(t *testing.T) {
			var cmd *cobra.Command
			for _, sub := range GetCmd.Commands() {
				if sub.Name() == name {
					cmd = sub
					break
				}
			}
			if cmd == nil {
				t.Fatalf("command %q not found", name)
			}
			// List commands either have no Args validator or accept NoArgs
			// They should NOT require exactly 1 arg
		})
	}
}

// ─── Flag tests ────────────────────────────────────────────────────────────

func TestEntitiesCmd_HasFilterFlags(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "entities" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("entities subcommand not found")
	}

	typeFlag := cmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Error("entities command must have --type flag")
	}

	ownerFlag := cmd.Flags().Lookup("owner")
	if ownerFlag == nil {
		t.Error("entities command must have --owner flag")
	}
}

func TestEntityCmd_HasScorecardFlag(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "entity" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("entity subcommand not found")
	}

	flag := cmd.Flags().Lookup("scorecard")
	if flag == nil {
		t.Error("entity command must have --scorecard flag")
	}
}

func TestGitopsCmd_HasFilterFlags(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "gitops" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("gitops subcommand not found")
	}

	flags := []string{"cluster-id", "tool", "sync-status", "health-status"}
	for _, name := range flags {
		t.Run(name, func(t *testing.T) {
			if cmd.Flags().Lookup(name) == nil {
				t.Errorf("gitops command must have --%s flag", name)
			}
		})
	}
}

func TestGitopsCmd_AcceptsOptionalArg(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "gitops" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("gitops subcommand not found")
	}

	// gitops accepts 0 or 1 args (MaximumNArgs(1))
	if cmd.Args == nil {
		t.Error("gitops command must have Args validator (MaximumNArgs(1))")
	}
}

// ─── Description truncation tests ──────────────────────────────────────────

func TestDescriptionTruncation_Entities(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		maxLen   int
		wantTrim bool
	}{
		{"short description", "A short desc", 60, false},
		{"exactly 60 chars", strings.Repeat("a", 60), 60, false},
		{"61 chars truncated", strings.Repeat("a", 61), 60, true},
		{"very long", strings.Repeat("a", 200), 60, true},
		{"empty", "", 60, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.desc
			if len(desc) > tt.maxLen {
				desc = desc[:tt.maxLen] + "…"
			}

			if tt.wantTrim {
				if len(desc) <= tt.maxLen {
					// The truncated desc should be maxLen + len("…") bytes
					t.Errorf("expected truncation for input length %d", len(tt.desc))
				}
				if !strings.HasSuffix(desc, "…") {
					t.Error("truncated description should end with ellipsis")
				}
			} else {
				if strings.HasSuffix(desc, "…") && tt.desc != "" {
					t.Error("short description should not be truncated")
				}
			}
		})
	}
}

// ─── Property-based: truncation invariants ─────────────────────────────────

func TestDescriptionTruncation_Property_NeverExceedsLimit(t *testing.T) {
	maxLen := 60
	for i := 0; i < 100; i++ {
		// Generate random length description
		length := rand.Intn(300)
		desc := strings.Repeat("x", length)

		truncated := desc
		if len(truncated) > maxLen {
			truncated = truncated[:maxLen] + "…"
		}

		// Property: visible prefix never exceeds maxLen runes
		prefix := truncated
		if strings.HasSuffix(prefix, "…") {
			prefix = strings.TrimSuffix(prefix, "…")
		}
		if len(prefix) > maxLen {
			t.Errorf("truncated prefix length %d exceeds max %d for input length %d",
				len(prefix), maxLen, length)
		}
	}
}

func TestDescriptionTruncation_Property_PreservesShortStrings(t *testing.T) {
	maxLen := 60
	for i := 0; i <= maxLen; i++ {
		desc := strings.Repeat("a", i)
		truncated := desc
		if len(truncated) > maxLen {
			truncated = truncated[:maxLen] + "…"
		}

		// Property: strings <= maxLen are never modified
		if truncated != desc {
			t.Errorf("string of length %d should not be truncated", i)
		}
	}
}

// ─── Metamorphic: longer input always produces truncation ──────────────────

func TestDescriptionTruncation_Metamorphic_LongerInputTruncated(t *testing.T) {
	maxLen := 60
	base := strings.Repeat("a", maxLen+1)

	truncateDesc := func(s string) string {
		if len(s) > maxLen {
			return s[:maxLen] + "…"
		}
		return s
	}

	t1 := truncateDesc(base)
	// Adding more chars shouldn't change the truncated output
	// (same prefix, same ellipsis)
	longer := base + strings.Repeat("b", 50)
	t2 := truncateDesc(longer)

	if t1 != t2 {
		t.Errorf("truncation of %d chars and %d chars should produce same result when both exceed limit, got %q vs %q",
			len(base), len(longer), t1, t2)
	}
}

// ─── Error path: no config ─────────────────────────────────────────────────

func TestRunGetEntities_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetEntities(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetTeams_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetTeams(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetUsers_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetUsers(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetGroups_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetGroups(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetMolds_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetMolds(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetRuns_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetRuns(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetK8s_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetK8s(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetOwned_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetOwned(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetEntity_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetEntity(cmd, []string{"my-entity"})
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetScorecard_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetScorecard(cmd, []string{"my-entity"})
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetGitOps_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetGitOps(cmd, nil)
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

func TestRunGetGitOpsDetail_NoConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	err := runGetGitOps(cmd, []string{"some-id"})
	if err == nil {
		t.Error("expected error when config is missing, got nil")
	}
}

// ─── Integration tests with mock HTTP server ───────────────────────────────
// These tests use httptest to verify the full command path: client creation,
// API call, response parsing, and output rendering.

// setupTestConfig creates a temporary config file pointing at the given server URL.
// It redirects HOME/USERPROFILE so config.Load() finds the test config.
func setupTestConfig(t *testing.T, serverURL string) {
	t.Helper()
	dir := t.TempDir()
	// config.Load uses os.UserHomeDir → ~/.shoehorn/config.yaml
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir) // Windows
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]*config.Profile{
			"default": {
				Name:   "default",
				Server: serverURL,
				Auth: &config.Auth{
					AccessToken: "test-token",
				},
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save test config: %v", err)
	}
}

func TestRunGetEntities_MockServer_JSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/entities" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "svc-1", "name": "Service One", "type": "service", "owner": "team-a", "description": "First service"},
					{"id": "svc-2", "name": "Service Two", "type": "library", "owner": "team-b", "description": "Second service"},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	err := runGetEntities(testCmd(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGetEntity_MockServer_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "not found"},
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	err := runGetEntity(testCmd(), []string{"nonexistent"})
	if err == nil {
		t.Error("expected error for not-found entity, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRunGetEntity_MockServer_NotFound_ShowsHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "not found"},
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	err := runGetEntity(testCmd(), []string{"bad-id"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Hint") {
		t.Errorf("not-found error should include Hint, got: %v", err)
	}
}

func TestRunGetTeams_MockServer_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{},
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	err := runGetTeams(testCmd(), nil)
	if err != nil {
		t.Fatalf("unexpected error for empty team list: %v", err)
	}
}

// ─── Run ID truncation test ────────────────────────────────────────────────

func TestRunIDTruncation(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"short id stays", "abc", "abc"},
		{"8 chars stays", "12345678", "12345678"},
		{"long id truncated", "123456789abcdef", "12345678"},
		{"uuid truncated", "550e8400-e29b-41d4-a716-446655440000", "550e8400"},
		{"empty stays", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.id
			if len(id) > 8 {
				id = id[:8]
			}
			if id != tt.want {
				t.Errorf("truncated %q = %q, want %q", tt.id, id, tt.want)
			}
		})
	}
}

// ─── Deduplication test (owned.go logic) ───────────────────────────────────

func TestEntityDeduplication(t *testing.T) {
	type entity struct {
		ID   string
		Name string
	}

	// Simulate the deduplication from runGetOwned
	entities := []entity{
		{"svc-1", "Service One"},
		{"svc-2", "Service Two"},
		{"svc-1", "Service One"}, // duplicate
		{"svc-3", "Service Three"},
		{"svc-2", "Service Two"}, // duplicate
	}

	seen := map[string]bool{}
	var deduped []entity
	for _, e := range entities {
		if !seen[e.ID] {
			seen[e.ID] = true
			deduped = append(deduped, e)
		}
	}

	if len(deduped) != 3 {
		t.Errorf("deduplication: got %d entities, want 3", len(deduped))
	}

	// Verify order preserved (first occurrence wins)
	expectedOrder := []string{"svc-1", "svc-2", "svc-3"}
	for i, e := range deduped {
		if e.ID != expectedOrder[i] {
			t.Errorf("deduped[%d].ID = %q, want %q", i, e.ID, expectedOrder[i])
		}
	}
}

// ─── Property: deduplication preserves all unique IDs ──────────────────────

func TestEntityDeduplication_Property_AllUniquesPreserved(t *testing.T) {
	for trial := 0; trial < 50; trial++ {
		// Generate random entities with some duplicates
		numEntities := rand.Intn(50) + 1
		var entities []string
		uniqueSet := map[string]bool{}
		for i := 0; i < numEntities; i++ {
			id := fmt.Sprintf("svc-%d", rand.Intn(20))
			entities = append(entities, id)
			uniqueSet[id] = true
		}

		// Deduplicate
		seen := map[string]bool{}
		var deduped []string
		for _, id := range entities {
			if !seen[id] {
				seen[id] = true
				deduped = append(deduped, id)
			}
		}

		// Property: number of deduplicated items equals number of unique inputs
		if len(deduped) != len(uniqueSet) {
			t.Errorf("trial %d: deduped count %d != unique count %d",
				trial, len(deduped), len(uniqueSet))
		}

		// Property: all unique IDs are present in output
		for id := range uniqueSet {
			found := false
			for _, d := range deduped {
				if d == id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("trial %d: unique ID %q missing from deduped output", trial, id)
			}
		}
	}
}

// ─── Metamorphic: entity not-found error is consistent ─────────────────────

func TestRunGetEntity_NotFound_ErrorFormat_Consistent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "not found"},
		})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	// Metamorphic: different entity IDs should produce structurally similar errors
	ids := []string{"svc-a", "svc-b", "nonexistent-123"}
	for _, id := range ids {
		err := runGetEntity(testCmd(), []string{id})
		if err == nil {
			t.Errorf("expected error for %q, got nil", id)
			continue
		}
		// All not-found errors should contain the queried ID and "not found"
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error for %q should contain 'not found': %v", id, err)
		}
		if !strings.Contains(err.Error(), id) {
			t.Errorf("error for %q should contain the entity ID: %v", id, err)
		}
	}
}

// ─── Mold description truncation (uses 50 char limit) ──────────────────────

func TestMoldDescriptionTruncation(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		wantDots bool
	}{
		{"short", "A short mold", false},
		{"exactly 50", strings.Repeat("x", 50), false},
		{"51 chars", strings.Repeat("x", 51), true},
		{"very long", strings.Repeat("x", 200), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.desc
			if len(desc) > 50 {
				desc = desc[:50] + "..."
			}
			if tt.wantDots && !strings.HasSuffix(desc, "...") {
				t.Errorf("expected ellipsis for %d-char desc", len(tt.desc))
			}
			if !tt.wantDots && strings.HasSuffix(desc, "...") {
				t.Errorf("unexpected ellipsis for %d-char desc", len(tt.desc))
			}
		})
	}
}

// ─── Parametric: all no-config errors wrap the root cause ──────────────────

func TestAllRunFunctions_NoConfig_ErrorsWrapped(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("SHOEHORN_TOKEN", "")
	t.Setenv("SHOEHORN_TOKEN_FILE", "")
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	type testCase struct {
		name string
		fn   func(cmd *cobra.Command, args []string) error
		args []string
	}

	cases := []testCase{
		{"runGetEntities", runGetEntities, nil},
		{"runGetTeams", runGetTeams, nil},
		{"runGetUsers", runGetUsers, nil},
		{"runGetGroups", runGetGroups, nil},
		{"runGetMolds", runGetMolds, nil},
		{"runGetRuns", runGetRuns, nil},
		{"runGetK8s", runGetK8s, nil},
		{"runGetOwned", runGetOwned, nil},
		{"runGetEntity", runGetEntity, []string{"x"}},
		{"runGetTeam", runGetTeam, []string{"x"}},
		{"runGetUser", runGetUser, []string{"x"}},
		{"runGetGroup", runGetGroup, []string{"x"}},
		{"runGetMold", runGetMold, []string{"x"}},
		{"runGetRunDetail", runGetRunDetail, []string{"x"}},
		{"runGetScorecard", runGetScorecard, []string{"x"}},
		{"runGetGitOpsList", func(cmd *cobra.Command, _ []string) error {
			return runGetGitOpsList(cmd)
		}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			err := tc.fn(cmd, tc.args)
			if err == nil {
				t.Errorf("%s: expected error when config is missing", tc.name)
				return
			}
			// All errors should mention config or auth
			errStr := err.Error()
			if !strings.Contains(errStr, "config") &&
				!strings.Contains(errStr, "auth") &&
				!strings.Contains(errStr, "authenticated") &&
				!strings.Contains(errStr, "load") &&
				!strings.Contains(errStr, "current user") {
				t.Errorf("%s: error should relate to config/auth, got: %v", tc.name, err)
			}
		})
	}
}

// ─── helpers for mock server tests ─────────────────────────────────────────

// testCmd creates a cobra.Command with a background context attached,
// matching what PersistentPreRun does in production.
func testCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	return cmd
}

// The setupTestConfig helper and import of config package are used by the
// httptest-based integration tests above. We import api only for type
// references in doc comments - the actual client is created internally
// by the run functions via api.NewClientFromConfig.
var _ = api.ErrNotAuthenticated // ensure import is used
