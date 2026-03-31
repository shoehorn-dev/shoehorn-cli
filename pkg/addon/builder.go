package addon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// MaxBundleSize is the maximum allowed addon bundle size (2MB).
const MaxBundleSize = 2 * 1024 * 1024

// BuildResult holds the result of validating a built addon bundle.
type BuildResult struct {
	Path          string // Absolute path to the bundle file
	Size          int64  // Size in bytes
	SizeFormatted string // Human-readable size
	SHA256        string // Hex-encoded SHA256 checksum
}

// ValidateBuildPrereqs checks that the working directory contains
// manifest.json and package.json (required for scripted/full addons).
func ValidateBuildPrereqs(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); os.IsNotExist(err) {
		return fmt.Errorf("no manifest.json found in %s - run this from an addon project directory", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "package.json")); os.IsNotExist(err) {
		return fmt.Errorf("no package.json found in %s - declarative addons don't need building", dir)
	}
	return nil
}

// ValidateBundle checks the bundle file exists, is within size limits,
// and computes its SHA256 checksum.
func ValidateBundle(bundlePath string) (*BuildResult, error) {
	info, err := os.Stat(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("build output not found at %s", bundlePath)
	}

	if info.Size() > MaxBundleSize {
		return nil, fmt.Errorf("bundle size %d bytes exceeds maximum %d bytes (2MB)", info.Size(), MaxBundleSize)
	}

	content, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("read bundle: %w", err)
	}

	hash := sha256.Sum256(content)

	return &BuildResult{
		Path:          bundlePath,
		Size:          info.Size(),
		SizeFormatted: FormatBuildSize(info.Size()),
		SHA256:        hex.EncodeToString(hash[:]),
	}, nil
}

// FormatBuildSize formats bytes into a human-readable string (B, KB, or MB).
func FormatBuildSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
	)
	switch {
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
