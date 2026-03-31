package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Property: validateOutputPath accepts paths that are under or equal to baseDir.
func TestValidateOutputPath_Property_ChildPathsAccepted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := filepath.Join("testdata", "output")
		child := rapid.StringMatching(`^[a-z0-9]{1,10}$`).Draw(t, "child")
		outputPath := filepath.Join(base, child)

		err := validateOutputPath(outputPath, base)
		if err != nil {
			t.Fatalf("validateOutputPath(%q, %q) returned error for child path: %v", outputPath, base, err)
		}
	})
}

// Property: validateOutputPath rejects paths that escape baseDir via ..
func TestValidateOutputPath_Property_TraversalRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := filepath.Join("testdata", "output")
		// Generate paths that traverse up
		nUp := rapid.IntRange(1, 3).Draw(t, "nUp")
		traversal := ""
		for range nUp {
			traversal = filepath.Join(traversal, "..")
		}
		sibling := rapid.StringMatching(`^[a-z]{1,5}$`).Draw(t, "sibling")
		outputPath := filepath.Join(base, traversal, sibling)

		// The resolved path should be outside base (unless nUp < depth of base)
		absOutput, _ := filepath.Abs(outputPath)
		absBase, _ := filepath.Abs(base)

		// Only check when the path actually escapes
		if absOutput != absBase && len(absOutput) > 0 && !strings.HasPrefix(absOutput, absBase+string(filepath.Separator)) {
			err := validateOutputPath(outputPath, base)
			if err == nil {
				t.Fatalf("validateOutputPath(%q, %q) accepted escaping path (abs: %q)", outputPath, base, absOutput)
			}
		}
	})
}
