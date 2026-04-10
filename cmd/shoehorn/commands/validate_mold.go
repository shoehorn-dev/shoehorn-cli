package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/mold"
	"github.com/spf13/cobra"
)

var (
	validateMoldFormat string
	validateMoldStrict bool
)

var validateMoldCmd = &cobra.Command{
	Use:   "mold [file]",
	Short: "Validate a mold YAML file",
	Long: `Validate a Shoehorn mold definition file offline without needing a running server.

Checks YAML syntax, required fields, action IDs, adapter names, step structure,
and approval flow constraints.

Examples:
  shoehorn validate mold .shoehorn/molds/create-repo.yaml
  shoehorn validate mold my-mold.yaml --format json
  shoehorn validate mold my-mold.yaml --strict
  cat my-mold.yaml | shoehorn validate mold -`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidateMold,
}

func init() {
	validateMoldCmd.Flags().StringVar(&validateMoldFormat, "format", "text", "output format: text or json")
	validateMoldCmd.Flags().BoolVar(&validateMoldStrict, "strict", false, "fail on warnings too")
	validateCmd.AddCommand(validateMoldCmd)
}

func runValidateMold(cmd *cobra.Command, args []string) error {
	content, filename, err := readMoldInput(args)
	if err != nil {
		return err
	}

	result, err := mold.ValidateMoldYAML(content)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	hasFailure := !result.Valid || (validateMoldStrict && len(result.Warnings) > 0)

	if validateMoldFormat == "json" {
		return outputJSON(filename, result, hasFailure)
	}
	return outputText(filename, result, hasFailure)
}

func readMoldInput(args []string) ([]byte, string, error) {
	if len(args) == 0 || args[0] == "-" {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxManifestSize+1))
		if err != nil {
			return nil, "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		if int64(len(data)) > maxManifestSize {
			return nil, "", fmt.Errorf("input exceeds maximum size of %d bytes", maxManifestSize)
		}
		return data, "stdin", nil
	}

	filename := args[0]
	info, err := os.Stat(filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	if info.Size() > maxManifestSize {
		return nil, "", fmt.Errorf("file %s exceeds maximum size of %d bytes", filename, maxManifestSize)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return data, filename, nil
}

func outputJSON(filename string, result *mold.ValidationResult, hasFailure bool) error {
	output := map[string]any{
		"file":     filename,
		"valid":    result.Valid && !hasFailure,
		"errors":   result.Errors,
		"warnings": result.Warnings,
	}
	// Ensure arrays are never null in JSON
	if result.Errors == nil {
		output["errors"] = []mold.ValidationError{}
	}
	if result.Warnings == nil {
		output["warnings"] = []mold.ValidationError{}
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))

	if hasFailure {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func outputText(filename string, result *mold.ValidationResult, hasFailure bool) error {
	if !hasFailure {
		fmt.Printf("v %s is valid\n", filename)
		if len(result.Warnings) > 0 {
			fmt.Println()
			fmt.Println("  warnings:")
			for _, w := range result.Warnings {
				if w.Field != "" {
					fmt.Printf("    - %s: %s\n", w.Field, w.Message)
				} else {
					fmt.Printf("    - %s\n", w.Message)
				}
			}
		}
		return nil
	}

	fmt.Printf("x %s has errors:\n", filename)

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("  errors:")
		for _, e := range result.Errors {
			if e.Field != "" {
				fmt.Printf("    - %s: %s\n", e.Field, e.Message)
			} else {
				fmt.Printf("    - %s\n", e.Message)
			}
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println()
		fmt.Println("  warnings:")
		for _, w := range result.Warnings {
			if w.Field != "" {
				fmt.Printf("    - %s: %s\n", w.Field, w.Message)
			} else {
				fmt.Printf("    - %s\n", w.Message)
			}
		}
	}

	return fmt.Errorf("validation failed")
}
