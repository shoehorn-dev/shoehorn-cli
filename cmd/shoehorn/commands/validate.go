package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/spf13/cobra"
)

// maxManifestSize is the maximum allowed manifest size for validation (10 MB).
// Prevents memory exhaustion when reading from stdin or large files.
const maxManifestSize int64 = 10 * 1024 * 1024

var validateFormat string

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate Shoehorn or Backstage manifest files",
	Long: `Validate manifest files and output structured validation errors.

Supports both Shoehorn and Backstage manifest formats with automatic detection.

Examples:
  # Validate a manifest file (text output)
  shoehorn validate catalog-info.yaml

  # Validate with JSON output
  shoehorn validate .shoehorn/service.yml --format json

  # Validate from stdin
  cat catalog-info.yaml | shoehorn validate -`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVar(&validateFormat, "format", "text", "output format: text or json")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Determine input source
	var content string
	var filename string

	if len(args) == 0 || args[0] == "-" {
		// Read from stdin with size limit to prevent memory exhaustion (S6)
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxManifestSize+1))
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		if int64(len(data)) > maxManifestSize {
			return fmt.Errorf("manifest exceeds maximum size of %d bytes (10MB)", maxManifestSize)
		}
		content = string(data)
		filename = "stdin"
	} else {
		// Read from file with size check before loading into memory (S6)
		filename = args[0]
		info, err := os.Stat(filename)
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}
		if info.Size() > maxManifestSize {
			return fmt.Errorf("manifest %s exceeds maximum size of %d bytes (10MB)", filename, maxManifestSize)
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		content = string(data)
	}

	// Create API client from config
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	// Call API
	ctx := cmd.Context()
	result, err := client.ValidateManifest(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to validate manifest: %w", err)
	}

	// Output based on format
	if validateFormat == "json" {
		output := map[string]any{
			"file":   filename,
			"valid":  result.Valid,
			"errors": result.Errors,
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Text output
		if result.Valid {
			fmt.Printf("✓ %s is valid\n", filename)
		} else {
			fmt.Printf("✗ %s has validation errors:\n\n", filename)
			for _, err := range result.Errors {
				if err.Field != "" {
					fmt.Printf("  - %s: %s\n", err.Field, err.Message)
				} else {
					fmt.Printf("  - %s\n", err.Message)
				}
			}
			return fmt.Errorf("validation failed")
		}
	}

	return nil
}
