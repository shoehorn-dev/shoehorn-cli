package get

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
)

// repeatedCursorServer always returns the same single entity and the same nextCursor.
func repeatedCursorServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entities": []any{map[string]any{"service": map[string]any{"id": "svc-stuck", "name": "Stuck Service"}}},
			"page":     map[string]any{"total": 99, "nextCursor": "1"},
		})
	}))
}

// captureStdout drains the pipe concurrently so large output can't deadlock fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	outCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		buf.ReadFrom(r)
		outCh <- buf.String()
	}()

	fn()

	w.Close()
	os.Stdout = old
	return <-outCh
}

func TestRunGetEntities_RepeatedCursor_ReturnsErrorWithoutDoublePrefix(t *testing.T) {
	srv := repeatedCursorServer()
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	var err error
	_ = captureStdout(t, func() {
		err = runGetEntities(testCmd(), nil)
	})

	if err == nil {
		t.Fatal("want an error when the server never advances its cursor")
	}
	if strings.Contains(err.Error(), "list entities: list entities:") {
		t.Fatalf("error has a doubled prefix: %v", err)
	}
}

func TestRunGetEntities_RepeatedCursor_StillRendersPartialResults(t *testing.T) {
	srv := repeatedCursorServer()
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	var err error
	out := captureStdout(t, func() {
		err = runGetEntities(testCmd(), nil)
	})

	if err == nil {
		t.Fatal("want the command to still report the fetch error (non-zero exit) even though partial data was rendered")
	}
	if !strings.Contains(out, "svc-stuck") {
		t.Fatalf("want the partially-fetched entity rendered to stdout, got: %q", out)
	}
}

// A first-page failure must not render an empty listing before the error.
func TestRunGetEntities_FirstPageFailure_DoesNotRenderEmptyListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	var err error
	out := captureStdout(t, func() {
		err = runGetEntities(testCmd(), nil)
	})

	if err == nil {
		t.Fatal("want an error when the first page fails")
	}
	if strings.Contains(out, "No resources found") || strings.Contains(out, "[]") {
		t.Fatalf("want nothing rendered on a pure first-page failure, got: %q", out)
	}
}

func TestRunGetEntities_NegativeLimit_Rejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the negative-limit guard should reject before any request is made")
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	if err := entitiesCmd.Flags().Set("limit", "-1"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = entitiesCmd.Flags().Set("limit", "0") }()

	err := runGetEntities(testCmd(), nil)
	if err == nil {
		t.Fatal("want an error for a negative --limit")
	}
	if !strings.Contains(err.Error(), "--limit") {
		t.Fatalf("error should mention --limit, got: %v", err)
	}
}

func TestRunGetEntities_ContextCancelledMidWalk_RendersNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			// Page 1 completes before anything is cancelled.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entities": []any{map[string]any{"service": map[string]any{"id": "svc-1"}}},
				"page":     map[string]any{"total": 2, "nextCursor": "1"},
			})
			return
		}
		// Cancel while the client is blocked on page 2, for a deterministic abort.
		cancel()
		<-r.Context().Done()
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	var err error
	out := captureStdout(t, func() {
		err = runGetEntities(cmd, nil)
	})

	if err == nil {
		t.Fatal("want an error when the context is cancelled mid-walk")
	}
	if out != "" {
		t.Fatalf("want nothing rendered when the context was cancelled, got: %q", out)
	}
}

// captureStderr mirrors captureStdout for the warning lines runGetEntity prints.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	outCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		buf.ReadFrom(r)
		outCh <- buf.String()
	}()

	fn()

	w.Close()
	os.Stderr = old
	return <-outCh
}

func TestRunGetEntity_TotalFailure_DoesNotDoublePrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	err := runGetEntity(testCmd(), []string{"svc-1"})
	if err == nil {
		t.Fatal("want an error")
	}
	if strings.Contains(err.Error(), "get entity: get entity") {
		t.Fatalf("error has a doubled prefix: %v", err)
	}
}

func TestRunGetEntity_FanOutWarnings_DoNotDoublePrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/resources"), strings.HasSuffix(r.URL.Path, "/status"):
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entity": map[string]any{"service": map[string]any{"id": "svc-1", "name": "Svc"}},
			})
		}
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	var err error
	errOut := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			err = runGetEntity(testCmd(), []string{"svc-1"})
		})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(errOut, "get entity resources: get entity resources") || strings.Contains(errOut, "get entity status: get entity status") {
		t.Fatalf("warning has a doubled prefix: %q", errOut)
	}
	if !strings.Contains(errOut, "Warning: some details could not be loaded") {
		t.Fatalf("want the warning header, got: %q", errOut)
	}
	if strings.Count(errOut, "get entity resources") != 1 {
		t.Fatalf("want exactly one resources warning line, got: %q", errOut)
	}
	if strings.Count(errOut, "get entity status") != 1 {
		t.Fatalf("want exactly one status warning line, got: %q", errOut)
	}
}

func TestRunGetEntity_ContextCancelledDuringFanOut_ReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	arrived := make(chan struct{}, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/resources"), strings.HasSuffix(r.URL.Path, "/status"):
			arrived <- struct{}{}
			<-r.Context().Done()
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entity": map[string]any{"service": map[string]any{"id": "svc-1", "name": "Svc"}},
			})
		}
	}))
	defer srv.Close()

	go func() {
		<-arrived
		<-arrived
		cancel()
	}()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	var err error
	out := captureStdout(t, func() {
		err = runGetEntity(cmd, []string{"svc-1"})
	})
	if err == nil {
		t.Fatal("want an error when the context is cancelled during the resources/status fan-out")
	}
	if out != "" {
		t.Fatalf("want nothing rendered when the context was cancelled, got: %q", out)
	}
}

func TestEntitiesTableTitle(t *testing.T) {
	cases := []struct {
		name       string
		count      int
		entityType string
		owner      string
		incomplete bool
		want       string
	}{
		{"plain", 3, "", "", false, "Entities  (3)"},
		{"with type", 3, "service", "", false, "Entities  (3)  type=service"},
		{"with owner", 3, "", "team-a", false, "Entities  (3)  owner=team-a"},
		{"incomplete", 3, "", "", true, "Entities  (3)  (incomplete)"},
		{"all", 3, "service", "team-a", true, "Entities  (3)  type=service  owner=team-a  (incomplete)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := entitiesTableTitle(tc.count, tc.entityType, tc.owner, tc.incomplete); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRunGetEntities_JSONMode_RepeatedCursor_DoesNotExitZero drives the real
// cobra command tree (not runGetEntities directly) with -o json, so a revert
// of the JSON/YAML branches' error propagation would be caught: rendering
// JSON doesn't bypass the err != nil && len(entities) == 0 guard the way a
// plain ui.RenderJSON(entities) call (ignoring spinErr) would.
func TestRunGetEntities_JSONMode_RepeatedCursor_DoesNotExitZero(t *testing.T) {
	srv := repeatedCursorServer()
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	root := commands.RootCmd()
	root.SetArgs([]string{"get", "entities", "-o", "json"})
	t.Cleanup(func() {
		_ = root.PersistentFlags().Set("output", "")
		root.SetArgs(nil)
	})

	var err error
	out := captureStdout(t, func() {
		err = root.ExecuteContext(context.Background())
	})

	if err == nil {
		t.Fatal("want an error when the server never advances its cursor")
	}
	var parsed []map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &parsed); jsonErr != nil {
		t.Fatalf("stdout is not valid JSON: %v; got %q", jsonErr, out)
	}
	if len(parsed) != 2 || parsed[0]["id"] != "svc-stuck" {
		t.Fatalf("want the 2 partial entities in the JSON output (the fixture never advances, so svc-stuck is fetched twice before the repeat is caught), got %v", parsed)
	}
}

func TestEntitiesCmd_HasLimitFlag(t *testing.T) {
	found := false
	for _, sub := range GetCmd.Commands() {
		if sub.Name() == "entities" {
			found = sub.Flags().Lookup("limit") != nil
			break
		}
	}
	if !found {
		t.Error("entities command must have --limit flag")
	}
}

func TestRunGetEntities_LimitFlag_CapsResults(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		offset, _ := strconv.Atoi(r.URL.Query().Get("cursor"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		end := offset + limit
		if end > 250 {
			end = 250
		}
		items := make([]map[string]any, 0, end-offset)
		for i := offset; i < end; i++ {
			items = append(items, map[string]any{"service": map[string]any{"id": fmt.Sprintf("svc-%d", i)}})
		}
		page := map[string]any{"total": 250}
		if end < 250 {
			page["nextCursor"] = strconv.Itoa(end)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"entities": items, "page": page})
	}))
	defer srv.Close()

	setupTestConfig(t, srv.URL)
	tui.SetPlainMode(true)
	defer tui.SetPlainMode(false)

	if err := entitiesCmd.Flags().Set("limit", "5"); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = entitiesCmd.Flags().Set("limit", "0") }()

	var err error
	out := captureStdout(t, func() {
		err = runGetEntities(testCmd(), nil)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("made %d requests, want exactly 1 (limit reached within the first page)", calls)
	}
	if !strings.Contains(out, "svc-4") {
		t.Fatalf("want svc-0..svc-4 rendered, got: %q", out)
	}
	if strings.Contains(out, "svc-5") {
		t.Fatalf("output should stop at the limit of 5 entities (svc-0..svc-4), but found svc-5: %q", out)
	}
}
