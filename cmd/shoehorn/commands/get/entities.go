package get

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	entityType    string
	entityOwner   string
	showScorecard bool
)

var entitiesCmd = &cobra.Command{
	Use:   "entities",
	Short: "List all catalog entities",
	RunE:  runGetEntities,
}

var entityCmd = &cobra.Command{
	Use:   "entity <id-or-slug>",
	Short: "Get details for a specific entity",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetEntity,
}

func init() {
	entitiesCmd.Flags().StringVar(&entityType, "type", "", "Filter by entity type (service, library, etc.)")
	entitiesCmd.Flags().StringVar(&entityOwner, "owner", "", "Filter by owning team slug")

	entityCmd.Flags().BoolVar(&showScorecard, "scorecard", false, "Include scorecard in output")

	GetCmd.AddCommand(entitiesCmd)
	GetCmd.AddCommand(entityCmd)
}

func runGetEntities(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	opts := api.ListEntitiesOpts{
		Type:  entityType,
		Owner: entityOwner,
	}

	result, spinErr := tui.RunSpinner("Loading entities...", func() (any, error) {
		return client.ListEntities(cmd.Context(), opts)
	})
	if spinErr != nil {
		return fmt.Errorf("list entities: %w", spinErr)
	}

	entities := result.([]*api.Entity)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(entities)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(entities)
	}

	colNames := []string{"ID", "Name", "Type", "Owner", "Description"}
	rows := make([][]string, len(entities))
	for i, e := range entities {
		desc := e.Description
		if len(desc) > 60 {
			desc = desc[:60] + "…"
		}
		rows[i] = []string{e.ID, e.Name, e.Type, e.Owner, desc}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "ID", Width: 30},
			{Title: "Name", Width: 28},
			{Title: "Type", Width: 14},
			{Title: "Owner", Width: 20},
			{Title: "Description", Width: 45},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		title := fmt.Sprintf("Entities  (%d)", len(entities))
		if entityType != "" {
			title += fmt.Sprintf("  type=%s", entityType)
		}
		if entityOwner != "" {
			title += fmt.Sprintf("  owner=%s", entityOwner)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   title,
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	ui.RenderTable(colNames, rows)
	return nil
}

func runGetEntity(cmd *cobra.Command, args []string) error {
	id := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	result, spinErr := tui.RunSpinner("Loading entity...", func() (any, error) {
		e, err := client.GetEntity(ctx, id)
		if err != nil {
			return nil, err
		}
		return e, nil
	})
	if spinErr != nil {
		if api.IsNotFound(spinErr) {
			return fmt.Errorf("entity %q not found.\nHint: Use the ID column from `shoehorn get entities` to look up an entity by its service ID", id)
		}
		return fmt.Errorf("get entity: %w", spinErr)
	}

	entity := result.(*api.EntityDetail)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(entity)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(entity)
	}

	// Fetch resources, status, and scorecard concurrently.
	// Each goroutine writes to its own result variable; a shared mutex
	// protects the errs slice. A WaitGroup gates completion so we never
	// close or read before all goroutines finish (fixes the previous
	// errCh race where close could fire before a late send).
	type fetchResult struct {
		resources []*api.Resource
		status    *api.EntityStatus
		scorecard *api.Scorecard
	}

	var (
		fr   fetchResult
		mu   sync.Mutex
		errs []string
		wg   sync.WaitGroup
	)

	wg.Add(2) // resources + status (always)
	go func() {
		defer wg.Done()
		r, err := client.GetEntityResources(ctx, entity.ID)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Sprintf("get entity resources: %v", err))
			mu.Unlock()
			return
		}
		fr.resources = r
	}()
	go func() {
		defer wg.Done()
		s, err := client.GetEntityStatus(ctx, entity.ID)
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Sprintf("get entity status: %v", err))
			mu.Unlock()
			return
		}
		fr.status = s
	}()
	if showScorecard {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc, err := client.GetEntityScorecard(ctx, entity.ID)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Sprintf("get entity scorecard: %v", err))
				mu.Unlock()
				return
			}
			fr.scorecard = sc
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "Warning: some details could not be loaded:")
		for _, msg := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", msg)
		}
	}

	// Build detail panel
	mainFields := []tui.Field{
		{Label: "Type", Value: entity.Type},
		{Label: "Owner", Value: entity.Owner},
		{Label: "Lifecycle", Value: entity.Lifecycle},
		{Label: "Tier", Value: entity.Tier},
		{Label: "Description", Value: entity.Description},
		{Label: "Tags", Value: tui.RenderTagBadges(entity.Tags)},
	}

	if fr.status != nil {
		health := tui.StatusColor(fr.status.Health).Render("● " + fr.status.Health)
		mainFields = append(mainFields,
			tui.Field{Label: "Status", Value: fmt.Sprintf("%s  (%.2f%% uptime)", health, fr.status.Uptime)},
		)
	}

	// Links
	if len(entity.Links) > 0 {
		linkNames := make([]string, len(entity.Links))
		for i, l := range entity.Links {
			linkNames[i] = l.Title
		}
		mainFields = append(mainFields, tui.Field{Label: "Links", Value: tui.RenderLinkLine(linkNames)})
	}

	sections := []tui.DetailSection{
		{Fields: mainFields},
	}

	// Resources section
	if len(fr.resources) > 0 {
		resFields := make([]tui.Field, len(fr.resources))
		for i, r := range fr.resources {
			resFields[i] = tui.Field{
				Label: r.Name,
				Value: fmt.Sprintf("%s  %s/%s  %s", r.Kind, r.Cluster, r.Namespace, tui.MutedStyle.Render(r.Replicas)),
			}
		}
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Resources (%d)", len(fr.resources)),
			Fields: resFields,
		})
	}

	// Scorecard section
	if fr.scorecard != nil {
		gradeStr := tui.GradeColor(fr.scorecard.Grade).Render(fr.scorecard.Grade)
		bar := tui.RenderScoreBar(fr.scorecard.Score, fr.scorecard.MaxScore)
		sections = append(sections, tui.DetailSection{
			Title: "Scorecard",
			Fields: []tui.Field{
				{Label: "Grade", Value: fmt.Sprintf("%s  %s", gradeStr, bar)},
			},
		})
		// Failed checks
		var failed []string
		for _, ch := range fr.scorecard.Checks {
			if !ch.Passed {
				failed = append(failed, "✗ "+ch.Name)
			}
		}
		if len(failed) > 0 {
			sections[len(sections)-1].Fields = append(sections[len(sections)-1].Fields,
				tui.Field{Label: "Failed", Value: strings.Join(failed, "\n"+strings.Repeat(" ", 20))},
			)
		}
	}

	fmt.Println(tui.RenderDetail(entity.Name, sections))
	return nil
}
