package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearch_LimitFlag(t *testing.T) {
	var gotLimit string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[],"total":0,"page":{"limit":1,"offset":0,"nextCursor":null}}`)
	}))
	t.Cleanup(srv.Close)
	useTestProfile(t, srv.URL)

	if _, err := runRoot(t, "search", "checkout", "--limit", "75", "-I"); err != nil {
		t.Fatalf("search --limit 75: %v", err)
	}
	if gotLimit != "75" {
		t.Errorf("sent limit %q, want 75", gotLimit)
	}

	for _, bad := range []string{"0", "101"} {
		gotLimit = ""
		_, err := runRoot(t, "search", "checkout", "--limit", bad, "-I")
		if err == nil || !strings.Contains(err.Error(), "between 1 and 100") {
			t.Errorf("--limit %s: err = %v, want a 1-100 range error", bad, err)
		}
		if gotLimit != "" {
			t.Errorf("--limit %s still called the API", bad)
		}
	}

	if _, err := runRoot(t, "search", "checkout", "-I"); err != nil {
		t.Fatalf("search without --limit: %v", err)
	}
	if want := fmt.Sprint(defaultSearchLimit); gotLimit != want {
		t.Errorf("sent limit %q after an earlier --limit 75, want the default %s", gotLimit, want)
	}
}
