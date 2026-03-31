package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

// forgeCmd represents the forge command group
var forgeCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge workflow commands",
	Long:  `Manage and execute Forge workflows.`,
}

// runCmd represents the forge run command group
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Manage workflow runs",
	Long:  `List, create, and inspect workflow runs.`,
}

// runListCmd represents the forge run list command
var runListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workflow runs",
	RunE:  runListRuns,
}

// runGetCmd represents the forge run get command
var runGetCmd = &cobra.Command{
	Use:   "get <run-id>",
	Short: "Get workflow run details",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetRun,
}

// runCreateCmd creates a new forge run
var runCreateCmd = &cobra.Command{
	Use:   "create <mold-slug>",
	Short: "Create a new workflow run",
	Long: `Start a new Forge workflow run from a mold slug.

Optionally pass input values as JSON or key=value pairs:
  shoehorn forge run create my-mold --action create --inputs '{"env":"staging"}'
  shoehorn forge run create my-mold --action create --input env=staging --input name=my-repo`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateRun,
}

// executeCmd combines discover + fill inputs + run
var executeCmd = &cobra.Command{
	Use:   "execute <mold-slug>",
	Short: "Execute a mold workflow",
	Long: `Fetch a mold, determine the action, and start a run in one step.

Examples:
  shoehorn forge execute my-mold --input name=my-repo --input owner=acme
  shoehorn forge execute my-mold --action scaffold --inputs '{"name":"my-repo"}'
  shoehorn forge execute my-mold --dry-run --input name=test`,
	Args: cobra.ExactArgs(1),
	RunE: runExecute,
}

// moldsCmd is the forge molds subcommand group
var moldsCmd = &cobra.Command{
	Use:   "molds",
	Short: "Manage workflow molds (templates)",
}

// moldsListCmd lists all molds
var moldsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all molds",
	RunE:  runMoldsList,
}

// moldsGetCmd gets a single mold
var moldsGetCmd = &cobra.Command{
	Use:   "get <slug>",
	Short: "Get details for a specific mold",
	Args:  cobra.ExactArgs(1),
	RunE:  runMoldsGet,
}

// Flags for "forge run create"
var (
	runInputsJSON   string
	runInputKVPairs []string
	runActionFlag   string
	runDryRunFlag   bool
)

// Flags for "forge execute" (separate variables to avoid shared state with run create)
var (
	execInputsJSON   string
	execInputKVPairs []string
	execActionFlag   string
	execDryRunFlag   bool
)

func init() {
	// run create flags
	runCreateCmd.Flags().StringVar(&runInputsJSON, "inputs", "", "Input values as JSON object")
	runCreateCmd.Flags().StringArrayVar(&runInputKVPairs, "input", nil, "Input as key=value (repeatable)")
	runCreateCmd.Flags().StringVar(&runActionFlag, "action", "", "Action name (auto-selects primary if omitted)")
	runCreateCmd.Flags().BoolVar(&runDryRunFlag, "dry-run", false, "Validate without executing")

	// execute flags (own variables -- not shared with run create)
	executeCmd.Flags().StringVar(&execInputsJSON, "inputs", "", "Input values as JSON object")
	executeCmd.Flags().StringArrayVar(&execInputKVPairs, "input", nil, "Input as key=value (repeatable)")
	executeCmd.Flags().StringVar(&execActionFlag, "action", "", "Action name (auto-selects primary if omitted)")
	executeCmd.Flags().BoolVar(&execDryRunFlag, "dry-run", false, "Validate without executing")

	runCmd.AddCommand(runListCmd)
	runCmd.AddCommand(runGetCmd)
	runCmd.AddCommand(runCreateCmd)
	runCmd.AddCommand(runWatchCmd)

	moldsCmd.AddCommand(moldsListCmd)
	moldsCmd.AddCommand(moldsGetCmd)

	forgeCmd.AddCommand(runCmd)
	forgeCmd.AddCommand(moldsCmd)
	forgeCmd.AddCommand(executeCmd)
	rootCmd.AddCommand(forgeCmd)
}

// ─── Input helpers ──────────────────────────────────────────────────────────

// buildInputs merges --inputs JSON and --input key=value flags into a single map.
func buildInputs(inputsJSON string, kvPairs []string) (map[string]any, error) {
	inputs := map[string]any{}

	if inputsJSON != "" {
		if err := json.Unmarshal([]byte(inputsJSON), &inputs); err != nil {
			return nil, fmt.Errorf("parse --inputs JSON: %w", err)
		}
	}

	for _, kv := range kvPairs {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --input format %q, expected key=value", kv)
		}
		inputs[parts[0]] = parts[1]
	}

	return inputs, nil
}

// coerceInputTypes converts string values supplied via --input key=value flags
// to their schema-declared types (boolean, number, integer). Values that were
// provided through --inputs JSON are already correctly typed and are skipped.
// Unrecognized types or unparsable values are left as strings.
func coerceInputTypes(inputs map[string]any, schema []api.MoldInput) {
	typeMap := map[string]string{}
	for _, inp := range schema {
		typeMap[inp.Name] = inp.Type
	}

	for key, val := range inputs {
		s, ok := val.(string)
		if !ok {
			continue
		}
		switch typeMap[key] {
		case "boolean":
			if b, err := strconv.ParseBool(s); err == nil {
				inputs[key] = b
			}
		case "number":
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				inputs[key] = f
			}
		case "integer":
			if i, err := strconv.ParseInt(s, 10, 64); err == nil {
				inputs[key] = i
			}
		}
	}
}

// resolveAction determines which action to use. It returns, in priority order:
// the explicit --action flag value, the first action marked as primary, or the
// first action in the list. It returns an empty string when no actions exist.
func resolveAction(flag string, actions []api.MoldAction) string {
	if flag != "" {
		return flag
	}
	for _, a := range actions {
		if a.Primary {
			return a.Action
		}
	}
	if len(actions) > 0 {
		return actions[0].Action
	}
	return ""
}

// ─── Execute ────────────────────────────────────────────────────────────────

func runExecute(cmd *cobra.Command, args []string) error {
	moldSlug := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	// 1. Fetch mold detail
	moldResult, spinErr := tui.RunSpinner(fmt.Sprintf("Loading mold %q...", moldSlug), func() (any, error) {
		return client.GetMold(ctx, moldSlug)
	})
	if spinErr != nil {
		return fmt.Errorf("get mold: %w", spinErr)
	}
	mold := moldResult.(*api.MoldDetail)

	// 2. Determine action
	action := resolveAction(execActionFlag, mold.Actions)
	if action == "" {
		return fmt.Errorf("no action specified and mold %q has no actions defined; use --action flag", moldSlug)
	}

	// 3. Build inputs
	inputs, err := buildInputs(execInputsJSON, execInputKVPairs)
	if err != nil {
		return err
	}

	// 4. Fill defaults for missing non-required inputs
	for _, inp := range mold.Inputs {
		if _, exists := inputs[inp.Name]; !exists && inp.Default != "" {
			inputs[inp.Name] = inp.Default
		}
	}

	// 5. Coerce string values to schema types (boolean, number, integer)
	coerceInputTypes(inputs, mold.Inputs)

	// 6. Validate required inputs
	var missing []string
	for _, inp := range mold.Inputs {
		if inp.Required {
			if _, exists := inputs[inp.Name]; !exists {
				missing = append(missing, inp.Name)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required inputs: %s\nUse --input key=value to provide them", strings.Join(missing, ", "))
	}

	// 7. Handle JSON/YAML output
	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(map[string]any{
			"mold_slug": moldSlug,
			"action":    action,
			"inputs":    inputs,
			"dry_run":   execDryRunFlag,
		})
	}

	// 8. Create run
	result, spinErr := tui.RunSpinner(fmt.Sprintf("Executing %q action %q...", moldSlug, action), func() (any, error) {
		return client.CreateRun(ctx, moldSlug, action, inputs, execDryRunFlag)
	})
	if spinErr != nil {
		return fmt.Errorf("execution failed: %w", spinErr)
	}

	run := result.(*api.ForgeRun)

	prefix := "Run Created"
	if execDryRunFlag {
		prefix = "Dry Run Complete"
	}

	body := fmt.Sprintf(
		"%s  %s\n%s  %s\n%s  %s\n%s  %s",
		tui.LabelStyle.Render("Run ID"), run.ID,
		tui.LabelStyle.Render("Mold"), moldSlug,
		tui.LabelStyle.Render("Action"), action,
		tui.LabelStyle.Render("Status"), tui.StatusColor(run.Status).Render(run.Status),
	)
	fmt.Println(tui.SuccessBox(prefix, body))
	if !execDryRunFlag {
		fmt.Printf("\nCheck progress: shoehorn forge run get %s\n", run.ID)
	}
	return nil
}

// ─── Runs ────────────────────────────────────────────────────────────────────

func runListRuns(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	result, spinErr := tui.RunSpinner("Loading runs...", func() (any, error) {
		return client.ListRuns(ctx)
	})
	if spinErr != nil {
		return fmt.Errorf("list runs: %w", spinErr)
	}

	response := result.(*api.ForgeRunsResponse)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	switch mode {
	case ui.ModeJSON:
		return ui.RenderJSON(response)
	case ui.ModeYAML:
		return ui.RenderYAML(response)
	default:
		if len(response.Runs) == 0 {
			fmt.Println("No runs found")
			return nil
		}

		colNames := []string{"ID", "Mold", "Action", "Status", "Created By", "Created At"}
		rows := make([][]string, len(response.Runs))
		for i, run := range response.Runs {
			rows[i] = []string{
				truncateID(run.ID),
				run.MoldSlug,
				run.Action,
				formatStatus(run.Status),
				run.CreatedBy,
				run.CreatedAt,
			}
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
				Title:   fmt.Sprintf("Runs  (%d)", len(response.Runs)),
				Columns: tuiCols,
				Rows:    tuiRows,
			})
			return tErr
		}

		ui.RenderTable(colNames, rows)
		return nil
	}
}

func runGetRun(cmd *cobra.Command, args []string) error {
	runID := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Loading run %q...", runID), func() (any, error) {
		return client.GetRun(ctx, runID)
	})
	if spinErr != nil {
		return fmt.Errorf("get run: %w", spinErr)
	}

	run := result.(*api.ForgeRun)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
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
					{Label: "Status", Value: tui.StatusColor(run.Status).Render(run.Status)},
					{Label: "Created By", Value: run.CreatedBy},
					{Label: "Created At", Value: run.CreatedAt},
				},
			},
		}

		if run.StartedAt != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Started At", Value: run.StartedAt})
		}
		if run.CompletedAt != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Completed At", Value: run.CompletedAt})
		}
		if run.DryRun {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Dry Run", Value: "true"})
		}
		if run.Error != "" {
			sections[0].Fields = append(sections[0].Fields,
				tui.Field{Label: "Error", Value: tui.ErrorStyle.Render(run.Error)})
		}

		fmt.Println(tui.RenderDetail("Run Details", sections))
		return nil
	}
}

func runCreateRun(cmd *cobra.Command, args []string) error {
	moldSlug := args[0]

	inputs, err := buildInputs(runInputsJSON, runInputKVPairs)
	if err != nil {
		return err
	}

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	// Auto-detect action from mold if not specified
	action := runActionFlag
	if action == "" {
		mold, mErr := client.GetMold(ctx, moldSlug)
		if mErr != nil {
			return fmt.Errorf("--action not specified and failed to auto-detect: %w", mErr)
		}
		action = resolveAction("", mold.Actions)
		if action == "" {
			return fmt.Errorf("--action is required (mold %q has no actions defined)", moldSlug)
		}
	}

	result, spinErr := tui.RunSpinner(fmt.Sprintf("Starting run for mold %q...", moldSlug), func() (any, error) {
		return client.CreateRun(ctx, moldSlug, action, inputs, runDryRunFlag)
	})
	if spinErr != nil {
		return fmt.Errorf("run failed: %w", spinErr)
	}

	run := result.(*api.ForgeRun)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(run)
	}

	body := fmt.Sprintf(
		"%s  %s\n%s  %s\n%s  %s\n%s  %s",
		tui.LabelStyle.Render("Run ID"), run.ID,
		tui.LabelStyle.Render("Mold"), moldSlug,
		tui.LabelStyle.Render("Action"), action,
		tui.LabelStyle.Render("Status"), tui.StatusColor(run.Status).Render(run.Status),
	)
	fmt.Println(tui.SuccessBox("Run Created", body))
	return nil
}

// ─── Molds ───────────────────────────────────────────────────────────────────

func runMoldsList(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
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

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(molds)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(molds)
	}

	colNames := []string{"Name", "Slug", "Version", "Description"}
	rows := make([][]string, len(molds))
	for i, m := range molds {
		desc := m.Description
		if len(desc) > 50 {
			desc = desc[:50] + "..."
		}
		rows[i] = []string{m.Name, m.Slug, m.Version, desc}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Name", Width: 28},
			{Title: "Slug", Width: 24},
			{Title: "Version", Width: 10},
			{Title: "Description", Width: 40},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Molds  (%d)", len(molds)),
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	ui.RenderTable(colNames, rows)
	return nil
}

func runMoldsGet(cmd *cobra.Command, args []string) error {
	slug := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
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

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(mold)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(mold)
	}

	// Build actions section
	actionFields := make([]tui.Field, len(mold.Actions))
	for i, a := range mold.Actions {
		label := a.Label
		if label == "" {
			label = a.Action
		}
		primary := ""
		if a.Primary {
			primary = " (primary)"
		}
		desc := ""
		if a.Description != "" {
			desc = "  " + tui.MutedStyle.Render(a.Description)
		}
		actionFields[i] = tui.Field{
			Label: a.Action,
			Value: fmt.Sprintf("%s%s%s", label, primary, desc),
		}
	}

	// Build inputs section
	inputFields := make([]tui.Field, len(mold.Inputs))
	for i, inp := range mold.Inputs {
		req := ""
		if inp.Required {
			req = " (required)"
		}
		def := ""
		if inp.Default != "" {
			def = fmt.Sprintf("  default: %s", tui.MutedStyle.Render(inp.Default))
		}
		inputFields[i] = tui.Field{
			Label: inp.Name,
			Value: fmt.Sprintf("%s%s%s  %s", inp.Type, req, def, tui.MutedStyle.Render(inp.Description)),
		}
	}

	// Build steps section
	stepFields := make([]tui.Field, len(mold.Steps))
	for i, s := range mold.Steps {
		stepFields[i] = tui.Field{
			Label: fmt.Sprintf("Step %d", i+1),
			Value: fmt.Sprintf("%s  %s", s.Name, tui.MutedStyle.Render(s.Action)),
		}
	}

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

	if len(actionFields) > 0 {
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Actions (%d)", len(mold.Actions)),
			Fields: actionFields,
		})
	}

	if len(inputFields) > 0 {
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Inputs (%d)", len(mold.Inputs)),
			Fields: inputFields,
		})
	}

	if len(stepFields) > 0 {
		sections = append(sections, tui.DetailSection{
			Title:  fmt.Sprintf("Steps (%d)", len(mold.Steps)),
			Fields: stepFields,
		})
	}

	fmt.Println(tui.RenderDetail(mold.Name, sections))
	return nil
}

// ─── Watch ──────────────────────────────────────────────────────────────────

var watchInterval int

var runWatchCmd = &cobra.Command{
	Use:   "watch <run-id>",
	Short: "Watch a workflow run until it completes",
	Long: `Poll a workflow run's status until it reaches a terminal state
(completed, failed, cancelled, rolled_back).

Examples:
  shoehorn forge run watch abc-123
  shoehorn forge run watch abc-123 --interval 5`,
	Args: cobra.ExactArgs(1),
	RunE: runWatch,
}

func init() {
	runWatchCmd.Flags().IntVar(&watchInterval, "interval", 2, "polling interval in seconds")
}

// isTerminalStatus returns true for statuses that indicate a run is done.
func isTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "cancelled", "rolled_back":
		return true
	default:
		return false
	}
}

func runWatch(cmd *cobra.Command, args []string) error {
	runID := args[0]

	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	mode := ui.DetectMode(interactive, noInteractive, outputFormat)

	interval := time.Duration(watchInterval) * time.Second
	if interval < time.Second {
		interval = time.Second
	}

	fmt.Fprintf(os.Stderr, "Watching run %s (every %ds, Ctrl+C to stop)...\n", runID, watchInterval)

	var lastStatus string
	for {
		run, err := client.GetRun(ctx, runID)
		if err != nil {
			return fmt.Errorf("get run: %w", err)
		}

		if run.Status != lastStatus {
			lastStatus = run.Status
			fmt.Fprintf(os.Stderr, "  %s %s\n", formatStatus(run.Status), run.Status)
		}

		if isTerminalStatus(run.Status) {
			// Final output
			if mode == ui.ModeJSON {
				return ui.RenderJSON(run)
			}
			if mode == ui.ModeYAML {
				return ui.RenderYAML(run)
			}

			sections := []tui.DetailSection{
				{
					Fields: []tui.Field{
						{Label: "Run ID", Value: run.ID},
						{Label: "Mold", Value: run.MoldSlug},
						{Label: "Action", Value: run.Action},
						{Label: "Status", Value: tui.StatusColor(run.Status).Render(run.Status)},
					},
				},
			}
			if run.Error != "" {
				sections[0].Fields = append(sections[0].Fields,
					tui.Field{Label: "Error", Value: tui.ErrorStyle.Render(run.Error)})
			}
			fmt.Println(tui.RenderDetail("Run Complete", sections))

			if run.Status == "failed" || run.Status == "cancelled" {
				return fmt.Errorf("run %s: %s", runID, run.Status)
			}
			return nil
		}

		// Wait for next poll or context cancellation
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			fmt.Fprintf(os.Stderr, "\nStopped watching (run is still %s)\n", lastStatus)
			return nil
		}
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func formatStatus(status string) string {
	statusIcons := map[string]string{
		"pending":     "? ",
		"executing":   "> ",
		"completed":   "v ",
		"failed":      "x ",
		"cancelled":   "o ",
		"rolled_back": "< ",
	}

	icon, ok := statusIcons[status]
	if !ok {
		icon = "  "
	}

	return fmt.Sprintf("%s%s", icon, status)
}

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
