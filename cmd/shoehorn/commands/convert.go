package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/spf13/cobra"
)

var (
	convertOutput     string
	convertOutputType string
	convertValidate   bool
	convertRecursive  bool
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert [file]",
	Short: "Convert between Backstage and Shoehorn manifest formats",
	Long: `Convert manifest files between Backstage and Shoehorn formats.

Examples:
  # Convert a Backstage manifest to Shoehorn format
  shoehorn convert catalog-info.yaml

  # Convert and save to file
  shoehorn convert catalog-info.yaml -o .shoehorn/my-service.yml

  # Convert Shoehorn to Backstage format
  shoehorn convert .shoehorn/my-service.yml --to backstage

  # Convert Backstage Template to Shoehorn Mold (outputs JSON)
  shoehorn convert template.yaml --to mold -o mold.json

  # Convert all manifests in a directory
  shoehorn convert ./manifests -r

  # Validate during conversion
  shoehorn convert catalog-info.yaml --validate`,
	Args: cobra.ExactArgs(1),
	RunE: runConvert,
}

func init() {
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "output file (default: stdout)")
	convertCmd.Flags().StringVar(&convertOutputType, "to", "shoehorn", "output format: shoehorn, backstage, or mold")
	convertCmd.Flags().BoolVar(&convertValidate, "validate", false, "validate manifest after conversion")
	convertCmd.Flags().BoolVarP(&convertRecursive, "recursive", "r", false, "recursively process directories")
	rootCmd.AddCommand(convertCmd)
}

func runConvert(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Create API client from config
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// Handle stdin
	if inputPath == "-" {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxConvertFileSize+1))
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		if int64(len(data)) > maxConvertFileSize {
			return fmt.Errorf("stdin exceeds maximum size of %d bytes (10MB)", maxConvertFileSize)
		}
		return convertContent(ctx, client, string(data), convertOutput)
	}

	// Check if input is a directory
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat input: %w", err)
	}

	if fileInfo.IsDir() {
		if !convertRecursive {
			return fmt.Errorf("input is a directory - use -r flag for recursive processing")
		}
		return convertDirectory(ctx, client, inputPath)
	}

	// Single file conversion
	return convertFile(ctx, client, inputPath, convertOutput)
}

func convertDirectory(ctx context.Context, client *api.Client, dirPath string) error {
	var filesProcessed int
	var filesFailed int

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process YAML files
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Determine output path
		var outputPath string
		if convertOutput != "" {
			// If output is specified, use it as base directory
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			outputPath = filepath.Join(convertOutput, relPath)
			// Validate the resolved path doesn't escape the output directory (S10)
			if err := validateOutputPath(outputPath, convertOutput); err != nil {
				fmt.Fprintf(os.Stderr, "  Skipping %s: %v\n", path, err)
				return nil
			}
		}

		fmt.Printf("Converting %s...\n", path)
		if err := convertFile(ctx, client, path, outputPath); err != nil {
			// Abort early on auth errors -- every subsequent file will also fail.
			if api.IsNotAuthenticated(err) {
				return fmt.Errorf("authentication failed: %w", err)
			}
			fmt.Fprintf(os.Stderr, "  ✗ Failed: %v\n", err)
			filesFailed++
			return nil // Continue processing other files
		}

		filesProcessed++
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Printf("\nProcessed %d files, %d failed\n", filesProcessed, filesFailed)
	if filesFailed > 0 {
		return fmt.Errorf("%d files failed to convert", filesFailed)
	}

	return nil
}

// validateOutputPath checks that outputPath is under or equal to baseDir.
// Prevents path traversal when building output paths from user-supplied
// relative input paths (security finding S10).
func validateOutputPath(outputPath, baseDir string) error {
	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolve base directory: %w", err)
	}
	if absOutput != absBase && !strings.HasPrefix(absOutput, absBase+string(filepath.Separator)) {
		return fmt.Errorf("path traversal detected: %q escapes output directory %q", outputPath, baseDir)
	}
	return nil
}

// maxConvertFileSize is the maximum input file size for conversion (10 MB).
const maxConvertFileSize = 10 * 1024 * 1024

func convertFile(ctx context.Context, client *api.Client, inputPath string, outputPath string) error {
	// Read input file with size check to prevent memory exhaustion
	info, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	if info.Size() > maxConvertFileSize {
		return fmt.Errorf("file %s exceeds maximum size of %d bytes (10MB)", inputPath, maxConvertFileSize)
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return convertContent(ctx, client, string(data), outputPath)
}

func convertContent(ctx context.Context, client *api.Client, content string, outputPath string) error {
	// Call API
	result, err := client.ConvertManifest(ctx, content, convertOutputType, convertValidate)
	if err != nil {
		return fmt.Errorf("failed to convert manifest: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("conversion failed - manifest is invalid")
	}

	// Prepare output
	var outputData []byte
	if convertOutputType == "mold" {
		// Mold format is JSON
		outputData, err = json.MarshalIndent(result.Mold, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal mold: %w", err)
		}
	} else {
		// Shoehorn and Backstage formats are YAML
		outputData = []byte(result.Content)
	}

	// Write output
	if outputPath != "" {
		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(outputPath, outputData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("  ✓ Wrote to %s\n", outputPath)
	} else {
		// Write to stdout
		fmt.Println(string(outputData))
	}

	// Show validation results if requested
	if convertValidate && result.Validation != nil {
		if !result.Validation.Valid {
			fmt.Println("\nValidation errors:")
			for _, err := range result.Validation.Errors {
				if err.Field != "" {
					fmt.Printf("  - %s: %s\n", err.Field, err.Message)
				} else {
					fmt.Printf("  - %s\n", err.Message)
				}
			}
		} else {
			fmt.Println("  ✓ Validation passed")
		}
	}

	return nil
}
