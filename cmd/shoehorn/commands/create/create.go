// Package create implements the "shoehorn create" command group.
package create

import (
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/spf13/cobra"
)

// CreateCmd is the parent command for all create subcommands.
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create resources",
	Long:  `Create new resources in the Shoehorn catalog.`,
}

func init() {
	commands.RootCmd().AddCommand(CreateCmd)
}
