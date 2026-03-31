package addon

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================================
// FormatBuildSize - Unit Tests
// ============================================================================

func TestFormatBuildSize_Bytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"small bytes", 512, "512 B"},
		{"just under 1KB", 1023, "1023 B"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBuildSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBuildSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatBuildSize_Kilobytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"exactly 1KB", 1024, "1.0 KB"},
		{"1.5KB", 1536, "1.5 KB"},
		{"just under 1MB", 1048575, "1024.0 KB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBuildSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBuildSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatBuildSize_Megabytes(t *testing.T) {
	got := FormatBuildSize(1048576)
	if got != "1.0 MB" {
		t.Errorf("FormatBuildSize(1048576) = %q, want %q", got, "1.0 MB")
	}
}

// ============================================================================
// ValidateBuildPrereqs - Unit Tests
// ============================================================================

func TestValidateBuildPrereqs_MissingManifest(t *testing.T) {
	dir := t.TempDir()
	err := ValidateBuildPrereqs(dir)
	if err == nil {
		t.Fatal("expected error for missing manifest.json")
	}
}

func TestValidateBuildPrereqs_MissingPackageJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0644)

	err := ValidateBuildPrereqs(dir)
	if err == nil {
		t.Fatal("expected error for missing package.json")
	}
}

func TestValidateBuildPrereqs_Valid(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	err := ValidateBuildPrereqs(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// ValidateBundle - Unit Tests
// ============================================================================

func TestValidateBundle_MissingFile(t *testing.T) {
	_, err := ValidateBundle(filepath.Join(t.TempDir(), "nonexistent.js"))
	if err == nil {
		t.Fatal("expected error for missing bundle")
	}
}

func TestValidateBundle_TooLarge(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "addon.js")
	// Create file larger than MaxBundleSize
	data := make([]byte, MaxBundleSize+1)
	os.WriteFile(bundlePath, data, 0644)

	_, err := ValidateBundle(bundlePath)
	if err == nil {
		t.Fatal("expected error for oversized bundle")
	}
}

func TestValidateBundle_ValidFile(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "addon.js")
	content := []byte("console.log('hello');")
	os.WriteFile(bundlePath, content, 0644)

	result, err := ValidateBundle(bundlePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Path != bundlePath {
		t.Errorf("Path = %q, want %q", result.Path, bundlePath)
	}
	if result.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", result.Size, len(content))
	}
	if result.SHA256 == "" {
		t.Error("SHA256 should not be empty")
	}
	if result.SizeFormatted == "" {
		t.Error("SizeFormatted should not be empty")
	}
}

// ============================================================================
// Parametric: all valid sizes produce non-empty formatted output
// ============================================================================

func TestFormatBuildSize_Parametric_AllRanges(t *testing.T) {
	sizes := []int64{0, 1, 100, 1023, 1024, 2048, 500000, 1048576, 2097152}
	for _, s := range sizes {
		result := FormatBuildSize(s)
		if result == "" {
			t.Errorf("FormatBuildSize(%d) returned empty string", s)
		}
	}
}

// ============================================================================
// Metamorphic: larger input produces different formatted output
// ============================================================================

func TestFormatBuildSize_Metamorphic_Monotonic(t *testing.T) {
	small := FormatBuildSize(100)
	large := FormatBuildSize(1048576)
	if small == large {
		t.Errorf("100 bytes and 1MB should produce different output, both got %q", small)
	}
}

// ============================================================================
// Metamorphic: ValidateBundle checksum is deterministic
// ============================================================================

func TestValidateBundle_Metamorphic_DeterministicChecksum(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "addon.js")
	os.WriteFile(bundlePath, []byte("deterministic content"), 0644)

	r1, err := ValidateBundle(bundlePath)
	if err != nil {
		t.Fatalf("first ValidateBundle() error: %v", err)
	}
	r2, err := ValidateBundle(bundlePath)
	if err != nil {
		t.Fatalf("second ValidateBundle() error: %v", err)
	}
	if r1.SHA256 != r2.SHA256 {
		t.Errorf("same file produced different checksums: %q vs %q", r1.SHA256, r2.SHA256)
	}
}
