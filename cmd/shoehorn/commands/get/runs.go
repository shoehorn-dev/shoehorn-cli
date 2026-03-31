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

var runsCmd = &cobra.Command{
	Use:   "runs",
	Short: "List Forge workflow runs",
	Long:  `Display all Forge workflow runs. Alias for "forge run list".`,
	RunE:  runGetRuns,
}

var runDetailCmd = &cobra.Command{
	Use:   "run <id>",
	Short: "Get details for a Forge workflow run",
	Long:  `Display details for a specific Forge workflow run. Alias for "forge run get".`,
	Args:  cobra.ExactArgs(1),
	RunE:  runGetRunDetail,
}

func init() {
	GetCmd.AddCommand(runsCmd)
	GetCmd.AddCommand(runDetailCmd)
}

func runGetRuns(cmd *cobra.Command, _ []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading runs...", func() (any, error) {
		return client.ListRuns(cmd.Context())
	})
	if spinErr != nil {
		return fmt.Errorf("list runs: %w", spinErr)
	}

	response := result.(*api.ForgeRunsResponse)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())

	if len(response.Runs) == 0 {
		if mode == ui.ModeJSON {
			return ui.RenderJSON(response)
		}
		if mode == ui.ModeYAML {
			return ui.RenderYAML(response)
		}
		fmt.Println("No runs found")
		return nil
	}

	colNames := []string{"ID", "Mold", "Action", "Status", "Created By", "Created At"}
	rows := make([][]string, len(response.Runs))
	for i, run := range response.Runs {
		id := run.ID
		if len(id) > 8 {
			id = id[:8]
		}
		rows[i] = []string{id, run.MoldSlug, run.Action, run.Status, run.CreatedBy, run.CreatedAt}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "ID", Width: 12},
			{Title: "Mold", Width: 20},
			{Title: "Action", Width: 14},
			{Title: "Status", Width: 14},
			{Title: "Created By", Width: 16},
			{Title: "Created At", Width: 22},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		_, tErr := tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Forge Runs (%d)", len(response.Runs)),
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return tErr
	}

	return ui.RenderListResult(mode, response, ui.ListConfig{
		Columns:  colNames,
		Rows:     rows,
		EmptyMsg: "No runs found",
	})
}

func runGetRunDetail(cmd *cobra.Command, args []string) error {
	runID := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Loading run %q...", runID), func() (any, error) {
		return client.GetRun(cmd.Context(), runID)
	})
	if spinErr != nil {
		return fmt.Errorf("get run: %w", spinErr)
	}

	run := result.(*api.ForgeRun)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	switch mode {
	case ui.ModeJSON:
		return ui.RenderJSON(run)
	case ui.ModeYAML:
		return ui.RenderYAML(run)
	default:
		sections := []tui.DetailSection{
			{
				Fields: []tui.Field{
					{Label: "Run ID", Value: run.ID},
					{Label: "Mold", Value: run.MoldSlug},
					{Label: "Action", Value: run.Action},
					{Label: "Status", Value: run.Status},
					{Label: "Created By", Value: run.CreatedBy},
					{Label: "Created At", Value: run.CreatedAt},
				},
			},
		}

		if run.Error != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Error", Value: run.Error})
		}

		fmt.Println(tui.RenderDetail("Run Details", sections))
		return nil
	}
}
