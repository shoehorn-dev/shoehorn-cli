package commands

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search entities",
	Long:  `Search across all catalog entities by name, description, or tags.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Searching for %q...", query), func() (any, error) {
		return client.Search(ctx, query)
	})
	if spinErr != nil {
		return fmt.Errorf("search: %w", spinErr)
	}

	sr := result.(*api.SearchResult)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(sr)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(sr)
	}

	if len(sr.Hits) == 0 {
		fmt.Printf("No results for %q\n", query)
		return nil
	}

	colNames := []string{"Name", "Type", "Owner", "Description"}
	rows := make([][]string, len(sr.Hits))
	for i, h := range sr.Hits {
		desc := h.Description
		if len(desc) > 60 {
			desc = desc[:60] + "…"
		}
		rows[i] = []string{h.Name, h.Type, h.Owner, desc}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Type", Width: 14},
			{Title: "Owner", Width: 20},
			{Title: "Description", Width: 50},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		selected, tErr := tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Search Results — %q  (%d hits)", query, sr.TotalCount),
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		if tErr != nil {
			return tErr
		}

		if selected != nil {
			// Find the matching hit and show entity detail
			for _, h := range sr.Hits {
				if h.Name == selected[0] {
					// Fetch full entity details for drilldown
					eResult, eErr := tui.RunSpinner("Loading entity...", func() (any, error) {
						return client.GetEntity(ctx, h.ID)
					})
					if eErr != nil {
						// Fallback to search hit data
						fmt.Println(tui.RenderDetail(h.Name, []tui.DetailSection{
							{
								Fields: []tui.Field{
									{Label: "ID", Value: h.ID},
									{Label: "Type", Value: h.Type},
									{Label: "Owner", Value: h.Owner},
									{Label: "Description", Value: h.Description},
								},
							},
						}))
					} else {
						entity := eResult.(*api.EntityDetail)
						fmt.Println(tui.RenderDetail(entity.Name, []tui.DetailSection{
							{
								Fields: []tui.Field{
									{Label: "ID", Value: entity.ID},
									{Label: "Type", Value: entity.Type},
									{Label: "Owner", Value: entity.Owner},
									{Label: "Lifecycle", Value: entity.Lifecycle},
									{Label: "Tier", Value: entity.Tier},
									{Label: "Description", Value: entity.Description},
								},
							},
						}))
					}
					break
				}
			}
		}
		return nil
	}

	ui.RenderTable(colNames, rows)
	return nil
}
