// Package delete implements the "shoehorn delete" command group.
package delete

import (
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/spf13/cobra"
)

// DeleteCmd is the parent command for all delete subcommands.
var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
	Long:  `Delete resources from the Shoehorn catalog.`,
}

func init() {
	commands.RootCmd().AddCommand(DeleteCmd)
}
