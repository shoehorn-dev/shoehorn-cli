package commands

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

// governanceCmd is the parent command group.
var governanceCmd = &cobra.Command{
	Use:   "governance",
	Short: "Governance actions and compliance",
	Long:  `Manage governance actions, compliance tracking, and view dashboard metrics.`,
}

// actionsCmd is the governance actions subgroup.
var actionsCmd = &cobra.Command{
	Use:   "actions",
	Short: "Manage governance actions",
	Long:  `List, create, update, and delete governance action items.`,
}

var actionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List governance actions",
	RunE:  runActionsList,
}

var actionsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get governance action details",
	Args:  cobra.ExactArgs(1),
	RunE:  runActionsGet,
}

var actionsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a governance action",
	RunE:  runActionsCreate,
}

var actionsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a governance action",
	Args:  cobra.ExactArgs(1),
	RunE:  runActionsUpdate,
}

var actionsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a governance action",
	Args:  cobra.ExactArgs(1),
	RunE:  runActionsDelete,
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show governance dashboard metrics",
	RunE:  runDashboard,
}

// actions create flags
var (
	govCreateEntityID    string
	govCreateTitle       string
	govCreatePriority    string
	govCreateSourceType  string
	govCreateDescription string
	govCreateAssignedTo  string
	govCreateSLADays     int
)

// actions update flags
var (
	govUpdateStatus         string
	govUpdatePriority       string
	govUpdateAssignedTo     string
	govUpdateDueDate        string
	govUpdateResolutionNote string
)

func init() {
	// actions create flags
	actionsCreateCmd.Flags().StringVar(&govCreateEntityID, "entity-id", "", "entity ID (required)")
	actionsCreateCmd.Flags().StringVar(&govCreateTitle, "title", "", "action title (required)")
	actionsCreateCmd.Flags().StringVar(&govCreatePriority, "priority", "", "priority: critical, high, medium, low (required)")
	actionsCreateCmd.Flags().StringVar(&govCreateSourceType, "source-type", "", "source type: scorecard, policy, manual (required)")
	actionsCreateCmd.Flags().StringVar(&govCreateDescription, "description", "", "action description")
	actionsCreateCmd.Flags().StringVar(&govCreateAssignedTo, "assigned-to", "", "user or team to assign")
	actionsCreateCmd.Flags().IntVar(&govCreateSLADays, "sla-days", 0, "SLA in days")
	_ = actionsCreateCmd.MarkFlagRequired("entity-id")
	_ = actionsCreateCmd.MarkFlagRequired("title")
	_ = actionsCreateCmd.MarkFlagRequired("priority")
	_ = actionsCreateCmd.MarkFlagRequired("source-type")

	// actions update flags
	actionsUpdateCmd.Flags().StringVar(&govUpdateStatus, "status", "", "new status: open, in_progress, resolved, dismissed")
	actionsUpdateCmd.Flags().StringVar(&govUpdatePriority, "priority", "", "new priority: critical, high, medium, low")
	actionsUpdateCmd.Flags().StringVar(&govUpdateAssignedTo, "assigned-to", "", "new assignee")
	actionsUpdateCmd.Flags().StringVar(&govUpdateDueDate, "due-date", "", "new due date (YYYY-MM-DD)")
	actionsUpdateCmd.Flags().StringVar(&govUpdateResolutionNote, "resolution-note", "", "resolution note")

	// Wire subcommands
	actionsCmd.AddCommand(actionsListCmd)
	actionsCmd.AddCommand(actionsGetCmd)
	actionsCmd.AddCommand(actionsCreateCmd)
	actionsCmd.AddCommand(actionsUpdateCmd)
	actionsCmd.AddCommand(actionsDeleteCmd)

	governanceCmd.AddCommand(actionsCmd)
	governanceCmd.AddCommand(dashboardCmd)

	rootCmd.AddCommand(governanceCmd)
}

// ─── Actions List ───────────────────────────────────────────────────────────

func runActionsList(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	type listResult struct {
		actions []*api.GovernanceAction
		total   int
	}

	result, spinErr := tui.RunSpinner("Loading governance actions...", func() (any, error) {
		actions, total, err := client.ListGovernanceActions(ctx, api.ListGovernanceActionsOpts{})
		if err != nil {
			return nil, err
		}
		return &listResult{actions: actions, total: total}, nil
	})
	if spinErr != nil {
		return fmt.Errorf("list governance actions: %w", spinErr)
	}

	lr := result.(*listResult)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	switch mode {
	case ui.ModeJSON:
		return ui.RenderJSON(lr.actions)
	case ui.ModeYAML:
		return ui.RenderYAML(lr.actions)
	default:
		if len(lr.actions) == 0 {
			fmt.Println("No governance actions found.")
			return nil
		}

		colNames := []string{"Priority", "Status", "Title", "Entity", "Assigned To", "Due Date"}
		rows := make([][]string, len(lr.actions))
		for i, a := range lr.actions {
			dueDate := a.DueDate
			if dueDate == "" {
				dueDate = "-"
			}
			assignedTo := a.AssignedTo
			if assignedTo == "" {
				assignedTo = "-"
			}
			title := a.Title
			if len(title) > 40 {
				title = title[:40] + "..."
			}
			rows[i] = []string{a.Priority, a.Status, title, a.EntityName, assignedTo, dueDate}
		}

		if mode == ui.ModeInteractive {
			tuiCols := []table.Column{
				{Title: "Priority", Width: 10},
				{Title: "Status", Width: 14},
				{Title: "Title", Width: 36},
				{Title: "Entity", Width: 22},
				{Title: "Assigned To", Width: 18},
				{Title: "Due Date", Width: 12},
			}
			tuiRows := make([]table.Row, len(rows))
			for i, r := range rows {
				tuiRows[i] = table.Row(r)
			}
			_, tErr := tui.RunTable(tui.TableConfig{
				Title:   fmt.Sprintf("Governance Actions  (%d)", lr.total),
				Columns: tuiCols,
				Rows:    tuiRows,
			})
			return tErr
		}

		ui.RenderTable(colNames, rows)
		return nil
	}
}

// ─── Actions Get ────────────────────────────────────────────────────────────

func runActionsGet(cmd *cobra.Command, args []string) error {
	actionID := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Loading action %q...", actionID), func() (any, error) {
		return client.GetGovernanceAction(ctx, actionID)
	})
	if spinErr != nil {
		return fmt.Errorf("get governance action: %w", spinErr)
	}

	action := result.(*api.GovernanceAction)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	switch mode {
	case ui.ModeJSON:
		return ui.RenderJSON(action)
	case ui.ModeYAML:
		return ui.RenderYAML(action)
	default:
		sections := []tui.DetailSection{
			{
				Fields: []tui.Field{
					{Label: "ID", Value: action.ID},
					{Label: "Title", Value: action.Title},
					{Label: "Description", Value: action.Description},
					{Label: "Priority", Value: action.Priority},
					{Label: "Status", Value: tui.StatusColor(action.Status).Render(action.Status)},
					{Label: "Source Type", Value: action.SourceType},
					{Label: "Entity", Value: fmt.Sprintf("%s (%s)", action.EntityName, action.EntityID)},
					{Label: "Assigned To", Value: action.AssignedTo},
				},
			},
		}

		if action.DueDate != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Due Date", Value: action.DueDate})
		}
		if action.SLADays != nil {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "SLA Days", Value: fmt.Sprintf("%d", *action.SLADays)})
		}
		if action.ResolutionNote != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Resolution", Value: action.ResolutionNote})
		}
		if action.CreatedAt != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Created At", Value: action.CreatedAt})
		}
		if action.UpdatedAt != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Updated At", Value: action.UpdatedAt})
		}

		fmt.Println(tui.RenderDetail("Governance Action", sections))
		return nil
	}
}

// ─── Actions Create ─────────────────────────────────────────────────────────

func runActionsCreate(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	req := api.CreateGovernanceActionRequest{
		EntityID:    govCreateEntityID,
		Title:       govCreateTitle,
		Priority:    govCreatePriority,
		SourceType:  govCreateSourceType,
		Description: govCreateDescription,
	}
	if govCreateAssignedTo != "" {
		req.AssignedTo = &govCreateAssignedTo
	}
	if govCreateSLADays > 0 {
		req.SLADays = &govCreateSLADays
	}

	result, spinErr := tui.RunSpinner("Creating governance action...", func() (any, error) {
		return client.CreateGovernanceAction(ctx, req)
	})
	if spinErr != nil {
		return fmt.Errorf("create governance action: %w", spinErr)
	}

	action := result.(*api.GovernanceAction)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(action)
	}

	body := fmt.Sprintf(
		"%s  %s\n%s  %s\n%s  %s\n%s  %s",
		tui.LabelStyle.Render("ID"), action.ID,
		tui.LabelStyle.Render("Title"), action.Title,
		tui.LabelStyle.Render("Priority"), action.Priority,
		tui.LabelStyle.Render("Status"), tui.StatusColor(action.Status).Render(action.Status),
	)
	fmt.Println(tui.SuccessBox("Action Created", body))
	return nil
}

// ─── Actions Update ─────────────────────────────────────────────────────────

func runActionsUpdate(cmd *cobra.Command, args []string) error {
	actionID := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	req := api.UpdateGovernanceActionRequest{}
	if govUpdateStatus != "" {
		req.Status = &govUpdateStatus
	}
	if govUpdatePriority != "" {
		req.Priority = &govUpdatePriority
	}
	if govUpdateAssignedTo != "" {
		req.AssignedTo = &govUpdateAssignedTo
	}
	if govUpdateDueDate != "" {
		req.DueDate = &govUpdateDueDate
	}
	if govUpdateResolutionNote != "" {
		req.ResolutionNote = &govUpdateResolutionNote
	}

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Updating action %q...", actionID), func() (any, error) {
		return nil, client.UpdateGovernanceAction(ctx, actionID, req)
	})
	if spinErr != nil {
		return fmt.Errorf("update governance action: %w", spinErr)
	}

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(map[string]string{"id": actionID, "status": "updated"})
	}

	fmt.Printf("Governance action %q updated.\n", actionID)
	return nil
}

// ─── Actions Delete ─────────────────────────────────────────────────────────

func runActionsDelete(cmd *cobra.Command, args []string) error {
	actionID := args[0]

	if !yesFlag {
		fmt.Printf("Delete governance action %q? This cannot be undone.\n", actionID)
		fmt.Print("Use --yes to confirm: ")
		return fmt.Errorf("confirmation required (use --yes to skip)")
	}

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Deleting action %q...", actionID), func() (any, error) {
		return nil, client.DeleteGovernanceAction(ctx, actionID)
	})
	if spinErr != nil {
		return fmt.Errorf("delete governance action: %w", spinErr)
	}

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(map[string]string{"id": actionID, "status": "deleted"})
	}

	fmt.Printf("Governance action %q deleted.\n", actionID)
	return nil
}

// ─── Dashboard ──────────────────────────────────────────────────────────────

func runDashboard(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	result, spinErr := tui.RunSpinner("Loading governance dashboard...", func() (any, error) {
		return client.GetGovernanceDashboard(ctx)
	})
	if spinErr != nil {
		return fmt.Errorf("get governance dashboard: %w", spinErr)
	}

	dash := result.(*api.GovernanceDashboard)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	switch mode {
	case ui.ModeJSON:
		return ui.RenderJSON(dash)
	case ui.ModeYAML:
		return ui.RenderYAML(dash)
	default:
		sections := []tui.DetailSection{
			{
				Title: "Overview",
				Fields: []tui.Field{
					{Label: "Entities", Value: fmt.Sprintf("%d", dash.EntityCount)},
					{Label: "Scored Entities", Value: fmt.Sprintf("%d", dash.ScoredEntityCount)},
					{Label: "Overall Grade", Value: dash.OverallGrade},
					{Label: "Health Score", Value: fmt.Sprintf("%.1f", dash.OverallHealthScore)},
				},
			},
			{
				Title: "Actions",
				Fields: []tui.Field{
					{Label: "Open", Value: fmt.Sprintf("%d", dash.ActionsSummary.Open)},
					{Label: "In Progress", Value: fmt.Sprintf("%d", dash.ActionsSummary.InProgress)},
					{Label: "Overdue", Value: fmt.Sprintf("%d", dash.ActionsSummary.Overdue)},
					{Label: "Resolved (30d)", Value: fmt.Sprintf("%d", dash.ActionsSummary.ResolvedLast30d)},
				},
			},
			{
				Title: "Documentation",
				Fields: []tui.Field{
					{Label: "Coverage", Value: fmt.Sprintf("%.1f%%", dash.DocCoverage.Percentage)},
					{Label: "With README", Value: fmt.Sprintf("%d / %d", dash.DocCoverage.WithReadme, dash.DocCoverage.Total)},
				},
			},
		}

		fmt.Println(tui.RenderDetail("Governance Dashboard", sections))
		return nil
	}
}
