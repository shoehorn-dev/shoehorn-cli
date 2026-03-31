package create

import (
	"fmt"

	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/manifest"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var createEntityFile string

var entityCmd = &cobra.Command{
	Use:   "entity -f <manifest.yaml>",
	Short: "Create an entity from a manifest file",
	Long: `Create a new catalog entity from a Shoehorn manifest YAML file.

The manifest must include schemaVersion, service (id, name, type), and owner fields.

Examples:
  shoehorn create entity -f service.yaml
  cat service.yaml | shoehorn create entity -f -`,
	RunE: runCreateEntity,
}

func init() {
	entityCmd.Flags().StringVarP(&createEntityFile, "file", "f", "", "manifest file (use - for stdin)")
	entityCmd.MarkFlagRequired("file")
	CreateCmd.AddCommand(entityCmd)
}

func runCreateEntity(cmd *cobra.Command, _ []string) error {
	content, err := manifest.ReadFile(createEntityFile)
	if err != nil {
		return err
	}

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())

	result, spinErr := tui.RunSpinner("Creating entity...", func() (any, error) {
		return client.CreateEntityFromManifest(ctx, content)
	})
	if spinErr != nil {
		return fmt.Errorf("create entity: %w", spinErr)
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
	fmt.Println(tui.SuccessBox("Entity Created", body))
	return nil
}
