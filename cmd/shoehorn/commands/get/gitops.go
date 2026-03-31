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

var (
	gitopsClusterID    string
	gitopsTool         string
	gitopsSyncStatus   string
	gitopsHealthStatus string
)

var gitopsCmd = &cobra.Command{
	Use:   "gitops [id]",
	Short: "List or get GitOps resources",
	Long: `Display GitOps resources (ArgoCD Applications, FluxCD Kustomizations).

Without arguments, lists all GitOps resources. With an ID argument,
shows details for a specific resource.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGetGitOps,
}

func init() {
	gitopsCmd.Flags().StringVar(&gitopsClusterID, "cluster-id", "", "filter by cluster ID")
	gitopsCmd.Flags().StringVar(&gitopsTool, "tool", "", "filter by tool (argocd, fluxcd)")
	gitopsCmd.Flags().StringVar(&gitopsSyncStatus, "sync-status", "", "filter by sync status")
	gitopsCmd.Flags().StringVar(&gitopsHealthStatus, "health-status", "", "filter by health status")

	GetCmd.AddCommand(gitopsCmd)
}

func runGetGitOps(cmd *cobra.Command, args []string) error {
	// Detail mode: get gitops <id>
	if len(args) == 1 {
		return runGetGitOpsDetail(cmd, args[0])
	}

	// List mode: get gitops
	return runGetGitOpsList(cmd)
}

func runGetGitOpsList(cmd *cobra.Command) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	opts := api.ListGitOpsResourcesOpts{
		ClusterID:    gitopsClusterID,
		Tool:         gitopsTool,
		SyncStatus:   gitopsSyncStatus,
		HealthStatus: gitopsHealthStatus,
	}

	result, spinErr := tui.RunSpinner("Loading GitOps resources...", func() (any, error) {
		return client.ListGitOpsResources(cmd.Context(), opts)
	})
	if spinErr != nil {
		return fmt.Errorf("list gitops resources: %w", spinErr)
	}

	resources := result.([]*api.GitOpsResource)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())

	rows := make([][]string, len(resources))
	for i, r := range resources {
		lastSync := r.LastSyncedAt
		if lastSync == "" {
			lastSync = "-"
		}
		rows[i] = []string{
			r.ClusterID,
			r.Name,
			r.Namespace,
			r.Tool,
			r.Kind,
			r.SyncStatus,
			r.HealthStatus,
			lastSync,
		}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Cluster", Width: 18},
			{Title: "Name", Width: 24},
			{Title: "Namespace", Width: 16},
			{Title: "Tool", Width: 10},
			{Title: "Kind", Width: 14},
			{Title: "Sync Status", Width: 14},
			{Title: "Health Status", Width: 14},
			{Title: "Last Sync", Width: 20},
		}
		tuiRows := make([]table.Row, len(resources))
		for j, r := range resources {
			lastSync := r.LastSyncedAt
			if lastSync == "" {
				lastSync = "-"
			}
			syncStatus := tui.StatusColor(r.SyncStatus).Render(r.SyncStatus)
			healthStatus := tui.StatusColor(r.HealthStatus).Render(r.HealthStatus)
			tuiRows[j] = table.Row{
				r.ClusterID, r.Name, r.Namespace, r.Tool, r.Kind,
				syncStatus, healthStatus, lastSync,
			}
		}
		title := fmt.Sprintf("GitOps Resources  (%d)", len(resources))
		if gitopsTool != "" {
			title += fmt.Sprintf("  tool=%s", gitopsTool)
		}
		if gitopsClusterID != "" {
			title += fmt.Sprintf("  cluster=%s", gitopsClusterID)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   title,
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	return ui.RenderListResult(mode, resources, ui.ListConfig{
		Columns:  []string{"Cluster", "Name", "Namespace", "Tool", "Kind", "Sync Status", "Health Status", "Last Sync"},
		Rows:     rows,
		EmptyMsg: "No GitOps resources found",
	})
}

func runGetGitOpsDetail(cmd *cobra.Command, id string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Loading GitOps resource %q...", id), func() (any, error) {
		return client.GetGitOpsResource(cmd.Context(), id)
	})
	if spinErr != nil {
		if api.IsNotFound(spinErr) {
			return fmt.Errorf("gitops resource %q not found", id)
		}
		return fmt.Errorf("get gitops resource: %w", spinErr)
	}

	resource := result.(*api.GitOpsResource)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	switch mode {
	case ui.ModeJSON:
		return ui.RenderJSON(resource)
	case ui.ModeYAML:
		return ui.RenderYAML(resource)
	default:
		syncStatus := tui.StatusColor(resource.SyncStatus).Render(resource.SyncStatus)
		healthStatus := tui.StatusColor(resource.HealthStatus).Render(resource.HealthStatus)

		sections := []tui.DetailSection{
			{
				Fields: []tui.Field{
					{Label: "ID", Value: resource.ID},
					{Label: "Name", Value: resource.Name},
					{Label: "Namespace", Value: resource.Namespace},
					{Label: "Cluster", Value: resource.ClusterID},
					{Label: "Tool", Value: resource.Tool},
					{Label: "Kind", Value: resource.Kind},
					{Label: "Sync Status", Value: syncStatus},
					{Label: "Health Status", Value: healthStatus},
				},
			},
		}

		if resource.LastSyncedAt != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Last Sync", Value: resource.LastSyncedAt})
		}
		if resource.SourceURL != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Source URL", Value: resource.SourceURL})
		}
		if resource.EntityName != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Entity", Value: fmt.Sprintf("%s (%s)", resource.EntityName, resource.EntityID)})
		}
		if resource.OwnerTeam != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Owner Team", Value: resource.OwnerTeam})
		}

		fmt.Println(tui.RenderDetail(resource.Name, sections))
		return nil
	}
}
