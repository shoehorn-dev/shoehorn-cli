package get

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

func TestEntityTable_LifecycleColumnOnlyWithTheFlag(t *testing.T) {
	entities := []*api.Entity{
		{ID: "a", Name: "A", Lifecycle: "production"},
		{ID: "b", Name: "B", Lifecycle: "end-of-life"},
	}

	cols, rows := entityTable(entities, false)
	if strings.Join(cols, ",") != "ID,Name,Type,Owner,Description" {
		t.Fatalf("cols without the flag = %v", cols)
	}
	if len(rows[1]) != 5 {
		t.Fatalf("row width = %d, want 5", len(rows[1]))
	}

	cols, rows = entityTable(entities, true)
	if cols[len(cols)-1] != "Lifecycle" {
		t.Fatalf("cols with the flag = %v, want a trailing Lifecycle column", cols)
	}
	if got := rows[1][len(rows[1])-1]; got != "end-of-life" {
		t.Fatalf("lifecycle cell = %q, want end-of-life", got)
	}
}

func TestEntityTable_TruncatesDescriptionsByCharacter(t *testing.T) {
	desc := strings.Repeat("ö", 70)
	_, rows := entityTable([]*api.Entity{{ID: "a", Description: desc}}, false)
	got := rows[0][4]
	if !utf8.ValidString(got) {
		t.Fatalf("truncated description %q is not valid UTF-8", got)
	}
	if n := utf8.RuneCountInString(got); n != 61 {
		t.Fatalf("truncated to %d characters, want 60 plus the ellipsis", n)
	}
}
