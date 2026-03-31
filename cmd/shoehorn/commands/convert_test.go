package commands

import (
	"path/filepath"
	"testing"
)

// TestValidateOutputPath_TableDriven tests path traversal detection.
func TestValidateOutputPath_TableDriven(t *testing.T) {
	base := filepath.Join(t.TempDir(), "output")

	tests := []struct {
		name    string
		output  string
		wantErr bool
	}{
		{"safe relative path", filepath.Join(base, "sub", "file.yml"), false},
		{"base itself", base, false},
		{"traversal attempt", filepath.Join(base, "..", "..", "etc", "shadow"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputPath(tt.output, base)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOutputPath(%q, %q) error = %v, wantErr %v",
					tt.output, base, err, tt.wantErr)
			}
		})
	}
}
