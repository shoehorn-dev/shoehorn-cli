// Package get contains all "shoehorn get" subcommands.
package get

import (
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/spf13/cobra"
)

// GetCmd is the parent "get" command registered on root
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get resources from the Shoehorn catalog",
	Long:  `Fetch and display catalog resources: entities, teams, users, groups, K8s agents, and more.`,
}

func init() {
	commands.RootCmd().AddCommand(GetCmd)
}
