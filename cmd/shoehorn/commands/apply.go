package commands

import (
	"fmt"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	applyFile   string
	applyDryRun bool
)

var applyCmd = &cobra.Command{
	Use:   "apply -f <file|directory>",
	Short: "Create or update entities from manifest files",
	Long: `Apply manifest files to the catalog. Creates new entities or updates existing ones.

Supports single files, stdin, and directories (with -r implicit for directories).

Examples:
  shoehorn apply -f service.yaml
  shoehorn apply -f catalog/
  shoehorn apply -f service.yaml --dry-run
  cat service.yaml | shoehorn apply -f -`,
	RunE: runApply,
}

func init() {
	applyCmd.Flags().StringVarP(&applyFile, "file", "f", "", "manifest file or directory")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "show what would be applied without making changes")
	applyCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, _ []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	mode := ui.DetectMode(interactive, noInteractive, outputFormat)

	// Collect and parse manifests
	resources, err := collectResources(applyFile)
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		fmt.Println("No resources found in manifest(s)")
		return nil
	}

	// Dry run -- just show what would happen
	if applyDryRun {
		fmt.Printf("Dry run: %d resource(s) would be applied:\n\n", len(resources))
		for _, r := range resources {
			fmt.Printf("  %s  %s (%s)\n", r.ServiceID, r.Name, r.Type)
		}
		return nil
	}

	// Apply each resource
	type applyResult struct {
		ServiceID string
		Name      string
		Action    string // "created" or "updated"
		Err       error
	}

	var results []applyResult
	for i, r := range resources {
		// Check if entity exists (only 404 means "does not exist" -- other errors
		// like 401/403/network failures should not be misinterpreted as "create")
		_, getErr := client.GetEntity(ctx, r.ServiceID)
		if getErr != nil && !api.IsNotFound(getErr) {
			results = append(results, applyResult{ServiceID: r.ServiceID, Name: r.Name, Err: getErr})
			fmt.Fprintf(os.Stderr, "  [%d/%d] error checking %s: %v\n", i+1, len(resources), r.ServiceID, getErr)
			if api.IsNotAuthenticated(getErr) {
				return fmt.Errorf("authentication failed: %w", getErr)
			}
			continue
		}
		exists := getErr == nil

		var action string
		var applyErr error

		if exists {
			action = "updated"
			_, applyErr = client.UpdateEntityFromManifest(ctx, r.ServiceID, r.RawYAML)
		} else {
			action = "created"
			_, applyErr = client.CreateEntityFromManifest(ctx, r.RawYAML)
		}

		results = append(results, applyResult{
			ServiceID: r.ServiceID,
			Name:      r.Name,
			Action:    action,
			Err:       applyErr,
		})

		if applyErr != nil {
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s %s: %v\n", i+1, len(resources), action, r.ServiceID, applyErr)
		} else {
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s %s\n", i+1, len(resources), action, r.ServiceID)
		}
	}

	// JSON/YAML output
	if mode == ui.ModeJSON {
		return ui.RenderJSON(results)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(results)
	}

	// Summary
	var created, updated, failed int
	for _, r := range results {
		if r.Err != nil {
			failed++
		} else if r.Action == "created" {
			created++
		} else {
			updated++
		}
	}

	body := fmt.Sprintf("%s  %d\n%s  %d",
		tui.LabelStyle.Render("Created"), created,
		tui.LabelStyle.Render("Updated"), updated,
	)
	if failed > 0 {
		body += fmt.Sprintf("\n%s  %d", tui.LabelStyle.Render("Failed"), failed)
	}
	fmt.Println(tui.SuccessBox("Apply Complete", body))

	if failed > 0 {
		return fmt.Errorf("%d resource(s) failed to apply", failed)
	}
	return nil
}
