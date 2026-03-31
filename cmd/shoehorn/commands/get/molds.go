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

var moldsCmd = &cobra.Command{
	Use:   "molds",
	Short: "List Forge workflow molds (templates)",
	Long:  `Display all available Forge workflow molds. Alias for "forge molds list".`,
	RunE:  runGetMolds,
}

var moldCmd = &cobra.Command{
	Use:   "mold <slug>",
	Short: "Get details for a Forge mold",
	Long:  `Display details for a specific Forge mold template. Alias for "forge molds get".`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGetMold,
}

func init() {
	GetCmd.AddCommand(moldsCmd)
	GetCmd.AddCommand(moldCmd)
}

func runGetMolds(cmd *cobra.Command, _ []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading molds...", func() (any, error) {
		return client.ListMolds(cmd.Context())
	})
	if spinErr != nil {
		return fmt.Errorf("list molds: %w", spinErr)
	}

	molds := result.([]*api.Mold)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())

	rows := make([][]string, len(molds))
	for i, m := range molds {
		desc := m.Description
		if len(desc) > 50 {
			desc = desc[:50] + "..."
		}
		rows[i] = []string{m.Name, m.Slug, m.Version, desc}
	}

	if mode == ui.ModeInteractive {
		columns := []table.Column{
			{Title: "Name", Width: 36},
			{Title: "Slug", Width: 36},
			{Title: "Version", Width: 10},
			{Title: "Description", Width: 50},
		}
		var tuiRows []table.Row
		for _, m := range molds {
			desc := m.Description
			if len(desc) > 47 {
				desc = desc[:47] + "..."
			}
			tuiRows = append(tuiRows, table.Row{m.Name, m.Slug, m.Version, desc})
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Forge Molds (%d)", len(molds)),
			Columns: columns,
			Rows:    tuiRows,
		})
		return err
	}

	return ui.RenderListResult(mode, molds, ui.ListConfig{
		Columns:  []string{"Name", "Slug", "Version", "Description"},
		Rows:     rows,
		EmptyMsg: "No molds found",
	})
}

func runGetMold(cmd *cobra.Command, args []string) error {
	slug := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Loading mold %q...", slug), func() (any, error) {
		return client.GetMold(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("get mold: %w", spinErr)
	}

	mold := result.(*api.MoldDetail)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	switch mode {
	case ui.ModeJSON:
		return ui.RenderJSON(mold)
	case ui.ModeYAML:
		return ui.RenderYAML(mold)
	default:
		sections := []tui.DetailSection{
			{
				Fields: []tui.Field{
					{Label: "Name", Value: mold.Name},
					{Label: "Slug", Value: mold.Slug},
					{Label: "Version", Value: mold.Version},
					{Label: "Description", Value: mold.Description},
				},
			},
		}

		if len(mold.Actions) > 0 {
			var actionFields []tui.Field
			for _, a := range mold.Actions {
				label := a.Label
				if a.Primary {
					label += " (primary)"
				}
				actionFields = append(actionFields, tui.Field{
					Label: a.Action,
					Value: label,
				})
			}
			sections = append(sections, tui.DetailSection{
				Title:  "Actions",
				Fields: actionFields,
			})
		}

		if len(mold.Inputs) > 0 {
			var inputFields []tui.Field
			for _, in := range mold.Inputs {
				req := ""
				if in.Required {
					req = " (required)"
				}
				inputFields = append(inputFields, tui.Field{
					Label: in.Name,
					Value: fmt.Sprintf("%s%s - %s", in.Type, req, in.Description),
				})
			}
			sections = append(sections, tui.DetailSection{
				Title:  "Inputs",
				Fields: inputFields,
			})
		}

		fmt.Println(tui.RenderDetail(mold.Name, sections))
		return nil
	}
}
