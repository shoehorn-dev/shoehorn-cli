package get

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var scorecardCmd = &cobra.Command{
	Use:   "scorecard <entity-id>",
	Short: "Show scorecard for an entity",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetScorecard,
}

func init() {
	GetCmd.AddCommand(scorecardCmd)
}

func runGetScorecard(cmd *cobra.Command, args []string) error {
	id := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading scorecard...", func() (any, error) {
		return client.GetEntityScorecard(cmd.Context(), id)
	})
	if spinErr != nil {
		return fmt.Errorf("get scorecard: %w", spinErr)
	}

	sc := result.(*api.Scorecard)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(sc)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(sc)
	}

	// Summary panel
	gradeStr := tui.GradeColor(sc.Grade).Render(sc.Grade)
	bar := tui.RenderScoreBar(sc.Score, sc.MaxScore)

	fmt.Println(tui.RenderDetail(fmt.Sprintf("Scorecard — %s", id), []tui.DetailSection{
		{
			Fields: []tui.Field{
				{Label: "Grade", Value: fmt.Sprintf("%s  %s", gradeStr, bar)},
				{Label: "Score", Value: fmt.Sprintf("%d / %d", sc.Score, sc.MaxScore)},
				{Label: "Updated", Value: sc.UpdatedAt},
			},
		},
	}))

	// Checks table
	if len(sc.Checks) > 0 {
		colNames := []string{"Check", "Status", "Weight", "Message"}
		rows := make([][]string, len(sc.Checks))
		for i, ch := range sc.Checks {
			status := "pass"
			if !ch.Passed {
				status = "FAIL"
			}
			msg := ch.Message
			if len(msg) > 60 {
				msg = msg[:60] + "…"
			}
			rows[i] = []string{ch.Name, status, fmt.Sprintf("%d", ch.Weight), msg}
		}

		if mode == ui.ModeInteractive {
			tuiCols := []table.Column{
				{Title: "Check", Width: 36},
				{Title: "Status", Width: 10},
				{Title: "Weight", Width: 8},
				{Title: "Message", Width: 40},
			}
			tuiRows := make([]table.Row, len(sc.Checks))
			for i, ch := range sc.Checks {
				status := tui.SuccessStyle.Render("✓ pass")
				if !ch.Passed {
					status = tui.ErrorStyle.Render("✗ fail")
				}
				msg := ch.Message
				if len(msg) > 38 {
					msg = msg[:38] + "…"
				}
				tuiRows[i] = table.Row{ch.Name, strings.TrimSpace(status), fmt.Sprintf("%d", ch.Weight), msg}
			}
			_, err = tui.RunTable(tui.TableConfig{
				Title:   fmt.Sprintf("Checks (%d)", len(sc.Checks)),
				Columns: tuiCols,
				Rows:    tuiRows,
			})
			return err
		}

		fmt.Println()
		ui.RenderTable(colNames, rows)
	}

	return nil
}
