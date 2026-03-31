package delete

import (
	"fmt"
	"os"

	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
)

var entityCmd = &cobra.Command{
	Use:   "entity <id-or-slug>",
	Short: "Delete an entity from the catalog",
	Long: `Delete a catalog entity by its service ID or slug.

Requires confirmation unless --yes is passed.

Examples:
  shoehorn delete entity my-service
  shoehorn delete entity my-service --yes`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteEntity,
}

func init() {
	DeleteCmd.AddCommand(entityCmd)
}

func runDeleteEntity(cmd *cobra.Command, args []string) error {
	id := args[0]

	confirmed, err := tui.Confirm(
		fmt.Sprintf("Delete entity %q? This cannot be undone.", id),
		commands.YesFlag(),
		os.Stdin,
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Deleting %q...", id), func() (any, error) {
		return nil, client.DeleteEntity(ctx, id)
	})
	if spinErr != nil {
		return fmt.Errorf("delete entity: %w", spinErr)
	}

	fmt.Println(tui.SuccessBox("Entity Deleted", fmt.Sprintf("%s  %s", tui.LabelStyle.Render("ID"), id)))
	return nil
}
