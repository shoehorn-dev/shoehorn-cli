// Package update implements the "shoehorn update" command group.
package update

import (
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/spf13/cobra"
)

// UpdateCmd is the parent command for all update subcommands.
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update resources",
	Long:  `Update existing resources in the Shoehorn catalog.`,
}

func init() {
	commands.RootCmd().AddCommand(UpdateCmd)
}
