package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConvertFile_RejectsOversizedFile tests that files exceeding the size
// limit are rejected before making an API call.
func TestConvertFile_RejectsOversizedFile(t *testing.T) {
	tmpDir := t.TempDir()
	bigFile := filepath.Join(tmpDir, "huge.yaml")
	if err := os.WriteFile(bigFile, make([]byte, maxConvertFileSize+1), 0644); err != nil {
		t.Fatal(err)
	}

	err := convertFile(context.Background(), nil, bigFile, "")
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("error = %q, want 'exceeds maximum size'", err.Error())
	}
}

// TestConvertFile_RejectsMissingFile tests that nonexistent files produce
// a clear error.
func TestConvertFile_RejectsMissingFile(t *testing.T) {
	err := convertFile(context.Background(), nil, "/nonexistent/file.yaml", "")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
