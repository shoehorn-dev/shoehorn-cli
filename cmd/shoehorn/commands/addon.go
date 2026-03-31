package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/addon"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

// addonCmd is the parent command for all addon subcommands.
var addonCmd = &cobra.Command{
	Use:   "addon",
	Short: "Manage addons",
	Long:  `List, install, manage, and develop addons for the Shoehorn platform.`,
}

// ─── addon list ─────────────────────────────────────────────────────────────

var addonListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed addons",
	RunE:    runAddonList,
}

func runAddonList(cmd *cobra.Command, _ []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading addons...", func() (any, error) {
		return client.ListInstalledAddons(cmd.Context())
	})
	if spinErr != nil {
		return fmt.Errorf("list addons: %w", spinErr)
	}

	addons := result.([]*api.Addon)

	mode := ui.DetectMode(Interactive(), NoInteractive(), OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(addons)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(addons)
	}

	if len(addons) == 0 {
		fmt.Println("No addons installed.")
		fmt.Println("  Install one with: shoehorn addon install <slug>")
		return nil
	}

	colNames := []string{"Slug", "Kind", "Version", "Enabled", "Status"}
	rows := make([][]string, len(addons))
	for i, a := range addons {
		enabled := "yes"
		if !a.Enabled {
			enabled = "no"
		}
		status := a.AddonStatus
		if status == "" {
			status = "-"
		}
		rows[i] = []string{a.Slug, a.Kind, a.Version, enabled, status}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Slug", Width: 24},
			{Title: "Kind", Width: 12},
			{Title: "Version", Width: 10},
			{Title: "Enabled", Width: 8},
			{Title: "Status", Width: 12},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Installed Addons (%d)", len(addons)),
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	ui.RenderTable(colNames, rows)
	return nil
}

// ─── addon status ───────────────────────────────────────────────────────────

var addonStatusCmd = &cobra.Command{
	Use:   "status <slug>",
	Short: "Show addon runtime status",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonStatus,
}

func runAddonStatus(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading status...", func() (any, error) {
		return client.GetAddonStatus(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("get addon status: %w", spinErr)
	}

	status := result.(*api.AddonStatus)

	mode := ui.DetectMode(Interactive(), NoInteractive(), OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(status)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(status)
	}

	enabled := "yes"
	if !status.Enabled {
		enabled = "no"
	}

	memStr := "-"
	if status.VMMemory > 0 {
		memStr = addon.FormatBuildSize(status.VMMemory)
	}

	sections := []tui.DetailSection{
		{
			Fields: []tui.Field{
				{Label: "Slug", Value: status.Slug},
				{Label: "Status", Value: tui.StatusColor(status.Status).Render(status.Status)},
				{Label: "Enabled", Value: enabled},
				{Label: "Executions", Value: fmt.Sprintf("%d", status.ExecCount)},
				{Label: "Errors", Value: fmt.Sprintf("%d", status.ErrorCount)},
				{Label: "VM Memory", Value: memStr},
			},
		},
	}

	if status.LastSyncAt != "" {
		sections[0].Fields = append(sections[0].Fields,
			tui.Field{Label: "Last Sync", Value: status.LastSyncAt})
	}
	if status.LastError != "" {
		sections[0].Fields = append(sections[0].Fields,
			tui.Field{Label: "Last Error", Value: tui.ErrorStyle.Render(status.LastError)})
	}

	fmt.Println(tui.RenderDetail(slug, sections))
	return nil
}

// ─── addon install ──────────────────────────────────────────────────────────

var addonInstallCmd = &cobra.Command{
	Use:   "install <slug>",
	Short: "Install an addon from the marketplace",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonInstall,
}

func runAddonInstall(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Installing %q...", slug), func() (any, error) {
		return client.InstallAddon(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("install addon: %w", spinErr)
	}

	fmt.Printf("Addon %q installed successfully.\n", slug)
	return nil
}

// ─── addon uninstall ────────────────────────────────────────────────────────

var addonUninstallCmd = &cobra.Command{
	Use:   "uninstall <slug>",
	Short: "Uninstall an addon",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonUninstall,
}

func runAddonUninstall(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Uninstalling %q...", slug), func() (any, error) {
		err := client.UninstallAddon(cmd.Context(), slug)
		return nil, err
	})
	if spinErr != nil {
		return fmt.Errorf("uninstall addon: %w", spinErr)
	}

	fmt.Printf("Addon %q uninstalled.\n", slug)
	return nil
}

// ─── addon enable ───────────────────────────────────────────────────────────

var addonEnableCmd = &cobra.Command{
	Use:   "enable <slug>",
	Short: "Enable a disabled addon",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonEnable,
}

func runAddonEnable(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Enabling %q...", slug), func() (any, error) {
		return nil, client.EnableAddon(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("enable addon: %w", spinErr)
	}

	fmt.Printf("Addon %q enabled.\n", slug)
	return nil
}

// ─── addon disable ──────────────────────────────────────────────────────────

var addonDisableCmd = &cobra.Command{
	Use:   "disable <slug>",
	Short: "Disable an addon without uninstalling",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonDisable,
}

func runAddonDisable(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Disabling %q...", slug), func() (any, error) {
		return nil, client.DisableAddon(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("disable addon: %w", spinErr)
	}

	fmt.Printf("Addon %q disabled.\n", slug)
	return nil
}

// ─── addon logs ─────────────────────────────────────────────────────────────

var addonLogsLimit int

var addonLogsCmd = &cobra.Command{
	Use:   "logs <slug>",
	Short: "View addon logs",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonLogs,
}

func runAddonLogs(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading logs...", func() (any, error) {
		return client.GetAddonLogs(cmd.Context(), slug, addonLogsLimit)
	})
	if spinErr != nil {
		return fmt.Errorf("get addon logs: %w", spinErr)
	}

	entries := result.([]*api.AddonLogEntry)

	mode := ui.DetectMode(Interactive(), NoInteractive(), OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(entries)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(entries)
	}

	if len(entries) == 0 {
		fmt.Println("No log entries found.")
		return nil
	}

	for _, e := range entries {
		levelStr := formatLogLevel(e.Level)
		fmt.Printf("%s  %s  %s\n", e.Timestamp, levelStr, e.Message)
	}
	return nil
}

// ─── addon browse ───────────────────────────────────────────────────────────

var addonBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse available addons in the marketplace",
	RunE:  runAddonBrowse,
}

func runAddonBrowse(cmd *cobra.Command, _ []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading marketplace...", func() (any, error) {
		return client.ListMarketplaceItems(cmd.Context(), "addon")
	})
	if spinErr != nil {
		return fmt.Errorf("browse addons: %w", spinErr)
	}

	items := result.([]*api.MarketplaceItem)

	mode := ui.DetectMode(Interactive(), NoInteractive(), OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(items)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(items)
	}

	if len(items) == 0 {
		fmt.Println("No addons available in the marketplace.")
		return nil
	}

	colNames := []string{"Slug", "Name", "Version", "Category", "Tier"}
	rows := make([][]string, len(items))
	for i, item := range items {
		rows[i] = []string{item.Slug, item.Name, item.Version, item.Category, item.Tier}
	}

	if mode == ui.ModeInteractive {
		tuiCols := []table.Column{
			{Title: "Slug", Width: 24},
			{Title: "Name", Width: 28},
			{Title: "Version", Width: 10},
			{Title: "Category", Width: 16},
			{Title: "Tier", Width: 10},
		}
		tuiRows := make([]table.Row, len(rows))
		for i, r := range rows {
			tuiRows[i] = table.Row(r)
		}
		_, err = tui.RunTable(tui.TableConfig{
			Title:   fmt.Sprintf("Available Addons (%d)", len(items)),
			Columns: tuiCols,
			Rows:    tuiRows,
		})
		return err
	}

	ui.RenderTable(colNames, rows)
	return nil
}

// ─── Registration ───────────────────────────────────────────────────────────

func init() {
	addonLogsCmd.Flags().IntVar(&addonLogsLimit, "limit", 100, "number of log entries to fetch")

	addonCmd.AddCommand(addonListCmd)
	addonCmd.AddCommand(addonStatusCmd)
	addonCmd.AddCommand(addonInstallCmd)
	addonCmd.AddCommand(addonUninstallCmd)
	addonCmd.AddCommand(addonEnableCmd)
	addonCmd.AddCommand(addonDisableCmd)
	addonCmd.AddCommand(addonLogsCmd)
	addonCmd.AddCommand(addonBrowseCmd)

	rootCmd.AddCommand(addonCmd)
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func formatLogLevel(level string) string {
	switch strings.ToLower(level) {
	case "error":
		return tui.ErrorStyle.Render("ERR")
	case "warn":
		return tui.WarnStyle.Render("WRN")
	case "info":
		return tui.SuccessStyle.Render("INF")
	case "debug":
		return tui.MutedStyle.Render("DBG")
	default:
		return level
	}
}
