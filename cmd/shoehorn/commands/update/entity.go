package update

import (
	"fmt"

	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/manifest"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var updateEntityFile string

var entityCmd = &cobra.Command{
	Use:   "entity <id-or-slug> -f <manifest.yaml>",
	Short: "Update an entity from a manifest file",
	Long: `Update an existing catalog entity from a Shoehorn manifest YAML file.

The service ID in the manifest must match the ID argument.

Examples:
  shoehorn update entity my-service -f service.yaml
  cat service.yaml | shoehorn update entity my-service -f -`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdateEntity,
}

func init() {
	entityCmd.Flags().StringVarP(&updateEntityFile, "file", "f", "", "manifest file (use - for stdin)")
	entityCmd.MarkFlagRequired("file")
	UpdateCmd.AddCommand(entityCmd)
}

func runUpdateEntity(cmd *cobra.Command, args []string) error {
	id := args[0]

	content, err := manifest.ReadFile(updateEntityFile)
	if err != nil {
		return err
	}

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Updating %q...", id), func() (any, error) {
		return client.UpdateEntityFromManifest(ctx, id, content)
	})
	if spinErr != nil {
		return fmt.Errorf("update entity: %w", spinErr)
	}

	resp := result.(*api.ManifestEntityResponse)

	if mode == ui.ModeJSON {
		return ui.RenderJSON(resp)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(resp)
	}

	body := fmt.Sprintf(
		"%s  %s\n%s  %s\n%s  %s",
		tui.LabelStyle.Render("Entity"), resp.Entity.Name,
		tui.LabelStyle.Render("ID"), resp.Entity.ServiceID,
		tui.LabelStyle.Render("Type"), resp.Entity.Type,
	)
	fmt.Println(tui.SuccessBox("Entity Updated", body))
	return nil
}
