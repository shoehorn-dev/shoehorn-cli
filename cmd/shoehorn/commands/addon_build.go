package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/addon"
	"github.com/spf13/cobra"
)

var addonBuildDir string

var addonBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build addon TypeScript into a JS bundle",
	Long: `Compile the addon TypeScript source into a single JS bundle using esbuild.

Requires esbuild (installed via npm install).

Examples:
  # Build from the current directory
  shoehorn addon build

  # Build from a specific directory
  shoehorn addon build --dir ./addons/jira-sync

Output: dist/addon.js`,
	RunE: runAddonBuild,
}

func runAddonBuild(_ *cobra.Command, _ []string) error {
	workDir := addonBuildDir
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolve directory: %w", err)
	}
	workDir = absDir

	if err := addon.ValidateBuildPrereqs(workDir); err != nil {
		return err
	}

	// Run npm run build from the addon directory
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Validate and checksum the output
	bundlePath := filepath.Join(workDir, "dist", "addon.js")
	result, err := addon.ValidateBundle(bundlePath)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Build complete: %s\n", result.Path)
	fmt.Printf("  Size:   %s\n", result.SizeFormatted)
	fmt.Printf("  SHA256: %s\n", result.SHA256)

	return nil
}

func init() {
	addonBuildCmd.Flags().StringVarP(&addonBuildDir, "dir", "d", "", "addon project directory (default: current directory)")
	addonCmd.AddCommand(addonBuildCmd)
}
