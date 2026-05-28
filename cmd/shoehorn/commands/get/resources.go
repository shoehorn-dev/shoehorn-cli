package get

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/cmd/shoehorn/commands"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	resourcesCluster   string
	resourcesTeam      string
	resourcesKind      string
	resourcesStatus    string
	resourcesSearch    string
	resourcesHasGitOps bool
	resourcesSignal    string
	resourcesCursor    string
	resourcesLimit     int
)

var resourcesCmd = &cobra.Command{
	Use:   "resources [id]",
	Short: "List or get Operations resources",
	Long: `Show resources from the Operations view.

A resource is one workload aggregated across every cluster it runs in. Same
image, helm release, and repo means one resource observed in N clusters.

Without arguments, lists resources. With an ID, shows one resource in detail,
including its per-cluster presence and recent events.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGetResources,
}

func init() {
	resourcesCmd.Flags().StringVar(&resourcesCluster, "cluster", "", "filter by cluster ID")
	resourcesCmd.Flags().StringVar(&resourcesTeam, "team", "", "filter by owning team")
	resourcesCmd.Flags().StringVar(&resourcesKind, "kind", "", "filter by kind (Deployment, StatefulSet, DaemonSet, CronJob, Job; comma-separated)")
	resourcesCmd.Flags().StringVar(&resourcesStatus, "status", "", "filter by status (Healthy, Degraded, Failing, Pending)")
	resourcesCmd.Flags().StringVar(&resourcesSearch, "search", "", "filter by name")
	resourcesCmd.Flags().BoolVar(&resourcesHasGitOps, "has-gitops", false, "show only resources with a GitOps source")
	resourcesCmd.Flags().StringVar(&resourcesSignal, "signal", "", "filter by signal (no-netpolicy)")
	resourcesCmd.Flags().StringVar(&resourcesCursor, "cursor", "", "page cursor from a prior result's next cursor")
	resourcesCmd.Flags().IntVar(&resourcesLimit, "limit", 0, "max resources per page (0 uses the server default)")

	GetCmd.AddCommand(resourcesCmd)
}

func runGetResources(cmd *cobra.Command, args []string) error {
	if len(args) == 1 {
		return runGetResourceDetail(cmd, args[0])
	}
	return runGetResourcesList(cmd)
}

func runGetResourcesList(cmd *cobra.Command) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	opts := api.ListOperationsResourcesOpts{
		ClusterID: resourcesCluster,
		Team:      resourcesTeam,
		Kind:      resourcesKind,
		Status:    resourcesStatus,
		Search:    resourcesSearch,
		HasGitOps: resourcesHasGitOps,
		Signal:    resourcesSignal,
		Cursor:    resourcesCursor,
		Limit:     resourcesLimit,
	}

	result, spinErr := tui.RunSpinner("Loading resources...", func() (any, error) {
		return client.ListOperationsResources(cmd.Context(), opts)
	})
	if spinErr != nil {
		return fmt.Errorf("list resources: %w", spinErr)
	}

	res := result.(*api.ListOperationsResourcesResult)

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(res)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(res)
	}

	cols := []string{"Name", "Kind", "Status", "Owner", "Clusters", "Replicas", "Last Seen"}
	rows := make([][]string, len(res.Resources))
	for i, r := range res.Resources {
		rows[i] = []string{
			resourceDisplayName(r),
			r.Subtype,
			r.Status,
			resourceOwnerName(r),
			fmt.Sprintf("%d", len(r.Clusters)),
			resourceReplicas(r),
			formatResourceLastSeen(r.LastSeenAt),
		}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Name", Width: 30},
			{Title: "Kind", Width: 14},
			{Title: "Status", Width: 12},
			{Title: "Owner", Width: 20},
			{Title: "Clusters", Width: 10},
			{Title: "Replicas", Width: 10},
			{Title: "Last Seen", Width: 16},
		}
		tuiRows := make([]table.Row, len(res.Resources))
		for j, r := range res.Resources {
			status := tui.StatusColor(r.Status).Render(r.Status)
			tuiRows[j] = table.Row{
				resourceDisplayName(r),
				r.Subtype,
				status,
				resourceOwnerName(r),
				fmt.Sprintf("%d", len(r.Clusters)),
				resourceReplicas(r),
				formatResourceLastSeen(r.LastSeenAt),
			}
		}
		title := fmt.Sprintf("Resources  (%d of %d)", len(res.Resources), res.Total)
		if resourcesKind != "" {
			title += fmt.Sprintf("  kind=%s", resourcesKind)
		}
		if resourcesStatus != "" {
			title += fmt.Sprintf("  status=%s", resourcesStatus)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   title,
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	if err := ui.RenderListResult(mode, res, ui.ListConfig{
		Columns:  cols,
		Rows:     rows,
		EmptyMsg: "No resources found",
	}); err != nil {
		return err
	}

	if res.NextCursor != "" {
		fmt.Printf("\nMore results. Next page: --cursor %s\n", res.NextCursor)
	}
	return nil
}

func runGetResourceDetail(cmd *cobra.Command, id string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(commands.Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Loading resource %q...", id), func() (any, error) {
		return client.GetOperationsResource(cmd.Context(), id)
	})
	if spinErr != nil {
		if api.IsNotFound(spinErr) {
			return fmt.Errorf("resource %q not found.\nHint: use the ID column from `shoehorn get resources` to look up a resource", id)
		}
		return fmt.Errorf("get resource: %w", spinErr)
	}

	detail := result.(*api.OperationsResourceDetail)
	r := detail.Resource

	mode := ui.DetectMode(commands.Interactive(), commands.NoInteractive(), commands.OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(detail)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(detail)
	}

	status := tui.StatusColor(r.Status).Render(r.Status)
	mainFields := []tui.Field{
		{Label: "ID", Value: r.ID},
		{Label: "Kind", Value: r.Subtype},
		{Label: "Status", Value: status},
		{Label: "Owner", Value: resourceOwnerName(r)},
		{Label: "Image", Value: orDash(r.Identity.ImageRepository)},
		{Label: "Helm Release", Value: orDash(r.Identity.HelmReleaseName)},
		{Label: "Repo URL", Value: orDash(r.Identity.RepoURL)},
		{Label: "Replicas", Value: resourceReplicas(r)},
		{Label: "Last Seen", Value: formatResourceLastSeen(r.LastSeenAt)},
	}
	sections := []tui.DetailSection{{Fields: mainFields}}

	if len(r.Clusters) > 0 {
		clusterFields := make([]tui.Field, len(r.Clusters))
		for i, c := range r.Clusters {
			clusterFields[i] = tui.Field{
				Label: c.ClusterID,
				Value: fmt.Sprintf("%s  %s  %d/%d replicas",
					c.Namespace, c.Status, c.ReplicasReady, c.ReplicasDesired),
			}
		}
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Clusters (%d)", len(r.Clusters)),
			Fields: clusterFields,
		})
	}

	if len(r.Issues) > 0 {
		issueFields := make([]tui.Field, len(r.Issues))
		for i, is := range r.Issues {
			issueFields[i] = tui.Field{Label: is.Severity, Value: is.Message}
		}
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Issues (%d)", len(r.Issues)),
			Fields: issueFields,
		})
	}

	if len(r.Sources) > 0 {
		sourceFields := make([]tui.Field, len(r.Sources))
		for i, s := range r.Sources {
			sourceFields[i] = tui.Field{Label: s.Kind, Value: strings.TrimSpace(s.Name + " " + s.URL)}
		}
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Sources (%d)", len(r.Sources)),
			Fields: sourceFields,
		})
	}

	if len(detail.Events) > 0 {
		eventFields := make([]tui.Field, len(detail.Events))
		for i, e := range detail.Events {
			eventFields[i] = tui.Field{
				Label: e.Reason,
				Value: fmt.Sprintf("%s  (x%d)", e.Message, e.Count),
			}
		}
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Events (%d)", len(detail.Events)),
			Fields: eventFields,
		})
	}

	fmt.Println(tui.RenderDetail(resourceDisplayName(r), sections))
	return nil
}

// ─── display helpers ─────────────────────────────────────────────────────────

// resourceDisplayName prefers the cross-cluster name hint, falling back to the
// resource's own name.
func resourceDisplayName(r *api.OperationsResource) string {
	if r.Name != "" {
		return r.Name
	}
	if r.Identity.NameHint != "" {
		return r.Identity.NameHint
	}
	return r.ID
}

// resourceOwnerName returns the owning team name, or an em-dash when unknown.
func resourceOwnerName(r *api.OperationsResource) string {
	if r.Owner != nil && r.Owner.TeamName != "" {
		return r.Owner.TeamName
	}
	return "—"
}

// resourceReplicas sums ready and desired replicas across every cluster.
func resourceReplicas(r *api.OperationsResource) string {
	ready, desired := 0, 0
	for _, c := range r.Clusters {
		ready += c.ReplicasReady
		desired += c.ReplicasDesired
	}
	return fmt.Sprintf("%d/%d", ready, desired)
}

// orDash returns the value, or an em-dash when empty.
func orDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

// formatResourceLastSeen renders a timestamp pointer as a compact relative
// string ("just now", "5m ago", "never").
func formatResourceLastSeen(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "never"
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
