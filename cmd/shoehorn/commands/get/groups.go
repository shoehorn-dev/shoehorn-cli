package get

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List all directory groups",
	RunE:  runGetGroups,
}

var groupCmd = &cobra.Command{
	Use:   "group <name>",
	Short: "Get roles mapped to a specific group",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetGroup,
}

func init() {
	GetCmd.AddCommand(groupsCmd)
	GetCmd.AddCommand(groupCmd)
}

func runGetGroups(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading groups...", func() (any, error) {
		return client.ListGroups(cmd.Context())
	})
	if spinErr != nil {
		return fmt.Errorf("list groups: %w", spinErr)
	}

	groups := result.([]*api.Group)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())

	rows := make([][]string, len(groups))
	for i, g := range groups {
		rows[i] = []string{g.Name, fmt.Sprintf("%d", g.RoleCount)}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Group Name", Width: 36},
			{Title: "Roles", Width: 10},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Groups  (%d)", len(groups)),
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	return ui.RenderListResult(mode, groups, ui.ListConfig{
		Columns:  []string{"Group Name", "Roles"},
		Rows:     rows,
		EmptyMsg: "No groups found",
	})
}

func runGetGroup(cmd *cobra.Command, args []string) error {
	groupName := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Loading group %q...", groupName), func() (any, error) {
		return client.GetGroupRoles(cmd.Context(), groupName)
	})
	if spinErr != nil {
		return fmt.Errorf("get group: %w", spinErr)
	}

	roles := result.([]*api.Role)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(roles)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(roles)
	}

	roleFields := make([]tui.Field, len(roles))
	for i, r := range roles {
		roleFields[i] = tui.Field{Label: r.Name, Value: r.Description}
	}

	if len(roleFields) == 0 {
		roleFields = []tui.Field{{Label: "Roles", Value: tui.MutedStyle.Render("none")}}
	}

	fmt.Println(tui.RenderDetail(groupName, []tui.DetailSection{
		{
			Title:  fmt.Sprintf("Roles (%d)", len(roles)),
			Fields: roleFields,
		},
	}))
	return nil
}
