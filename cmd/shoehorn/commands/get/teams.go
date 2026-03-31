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

var teamsCmd = &cobra.Command{
	Use:   "teams",
	Short: "List all teams",
	RunE:  runGetTeams,
}

var teamCmd = &cobra.Command{
	Use:   "team <slug>",
	Short: "Get details for a specific team",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetTeam,
}

func init() {
	GetCmd.AddCommand(teamsCmd)
	GetCmd.AddCommand(teamCmd)
}

func runGetTeams(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading teams...", func() (any, error) {
		return client.ListTeams(cmd.Context())
	})
	if spinErr != nil {
		return fmt.Errorf("list teams: %w", spinErr)
	}

	teams := result.([]*api.Team)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())

	// Build rows for plain/interactive table
	rows := make([][]string, len(teams))
	for i, t := range teams {
		desc := t.Description
		if len(desc) > 50 {
			desc = desc[:50] + "…"
		}
		rows[i] = []string{t.Name, t.Slug, fmt.Sprintf("%d", t.MemberCount), desc}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Name", Width: 28},
			{Title: "Slug", Width: 24},
			{Title: "Members", Width: 10},
			{Title: "Description", Width: 40},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Teams  (%d)", len(teams)),
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	return ui.RenderListResult(mode, teams, ui.ListConfig{
		Columns:  []string{"Name", "Slug", "Members", "Description"},
		Rows:     rows,
		EmptyMsg: "No teams found",
	})
}

func runGetTeam(cmd *cobra.Command, args []string) error {
	slug := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading team...", func() (any, error) {
		return client.GetTeam(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("get team: %w", spinErr)
	}

	team := result.(*api.TeamDetail)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(team)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(team)
	}

	sections := []tui.DetailSection{
		{
			Fields: []tui.Field{
				{Label: "Name", Value: team.Name},
				{Label: "Slug", Value: team.Slug},
				{Label: "Description", Value: team.Description},
				{Label: "Members", Value: fmt.Sprintf("%d", len(team.Members))},
			},
		},
	}

	if len(team.Members) > 0 {
		memberFields := make([]tui.Field, len(team.Members))
		for i, m := range team.Members {
			name := m.Name
			if name == "" {
				name = m.Email
			}
			memberFields[i] = tui.Field{
				Label: name,
				Value: fmt.Sprintf("%s  %s", m.Email, tui.MutedStyle.Render(m.Role)),
			}
		}
		sections = append(sections, tui.DetailSection{
			Title:  "Members",
			Fields: memberFields,
		})
	}

	fmt.Println(tui.RenderDetail(team.Name, sections))
	return nil
}
