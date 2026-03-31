package commands

import (
	"testing"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
)

// TestBuildInputs tests JSON and key=value input merging.
func TestBuildInputs_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		kvPairs  []string
		wantKeys []string
		wantErr  bool
	}{
		{
			name:     "empty inputs",
			jsonStr:  "",
			kvPairs:  nil,
			wantKeys: nil,
			wantErr:  false,
		},
		{
			name:     "json only",
			jsonStr:  `{"env":"staging","count":1}`,
			kvPairs:  nil,
			wantKeys: []string{"env", "count"},
			wantErr:  false,
		},
		{
			name:     "kv only",
			jsonStr:  "",
			kvPairs:  []string{"name=my-repo", "owner=acme"},
			wantKeys: []string{"name", "owner"},
			wantErr:  false,
		},
		{
			name:     "json and kv merged",
			jsonStr:  `{"env":"staging"}`,
			kvPairs:  []string{"name=my-repo"},
			wantKeys: []string{"env", "name"},
			wantErr:  false,
		},
		{
			name:     "kv overrides json",
			jsonStr:  `{"name":"old"}`,
			kvPairs:  []string{"name=new"},
			wantKeys: []string{"name"},
			wantErr:  false,
		},
		{
			name:    "invalid json",
			jsonStr: `{invalid}`,
			wantErr: true,
		},
		{
			name:    "invalid kv format",
			jsonStr: "",
			kvPairs: []string{"no-equals-sign"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildInputs(tt.jsonStr, tt.kvPairs)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildInputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			for _, key := range tt.wantKeys {
				if _, exists := got[key]; !exists {
					t.Errorf("buildInputs() missing expected key %q", key)
				}
			}
		})
	}
}

// TestBuildInputs_KVOverridesJSON verifies that --input flags override --inputs JSON.
func TestBuildInputs_KVOverridesJSON(t *testing.T) {
	got, err := buildInputs(`{"name":"old"}`, []string{"name=new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["name"] != "new" {
		t.Errorf("expected kv to override json: got %q, want %q", got["name"], "new")
	}
}

// TestResolveAction tests action resolution logic.
func TestResolveAction_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		actions []struct {
			action  string
			primary bool
		}
		want string
	}{
		{"explicit flag", "create", nil, "create"},
		{"primary action", "", []struct {
			action  string
			primary bool
		}{
			{"scaffold", false}, {"create", true},
		}, "create"},
		{"first action fallback", "", []struct {
			action  string
			primary bool
		}{
			{"scaffold", false}, {"delete", false},
		}, "scaffold"},
		{"no actions", "", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := make([]api.MoldAction, len(tt.actions))
			for i, action := range tt.actions {
				actions[i] = api.MoldAction{
					Action:  action.action,
					Primary: action.primary,
				}
			}

			got := resolveAction(tt.flag, actions)
			if got != tt.want {
				t.Errorf("resolveAction(%q, %+v) = %q, want %q", tt.flag, actions, got, tt.want)
			}
		})
	}
}

// TestFormatStatus tests status formatting.
func TestFormatStatus_TableDriven(t *testing.T) {
	tests := []struct {
		status   string
		wantIcon string
	}{
		{"pending", "? "},
		{"executing", "> "},
		{"completed", "v "},
		{"failed", "x "},
		{"cancelled", "o "},
		{"rolled_back", "< "},
		{"unknown", "  "},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := formatStatus(tt.status)
			if got != tt.wantIcon+tt.status {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, got, tt.wantIcon+tt.status)
			}
		})
	}
}

// TestTruncateID tests ID truncation.
func TestTruncateID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"short", "short"},
		{"exactly12ch", "exactly12ch"},
		{"this-is-a-very-long-uuid-string", "this-is-a-ve"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateID(tt.input)
			if got != tt.want {
				t.Errorf("truncateID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestIsTerminalStatus_TableDriven verifies terminal status detection.
func TestIsTerminalStatus_TableDriven(t *testing.T) {
	tests := []struct {
		status   string
		terminal bool
	}{
		{"completed", true},
		{"failed", true},
		{"cancelled", true},
		{"rolled_back", true},
		{"pending", false},
		{"executing", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := isTerminalStatus(tt.status)
			if got != tt.terminal {
				t.Errorf("isTerminalStatus(%q) = %v, want %v", tt.status, got, tt.terminal)
			}
		})
	}
}
