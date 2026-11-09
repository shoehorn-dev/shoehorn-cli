package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

// ─── addon config ───────────────────────────────────────────────────────────
// Pushes addon config from the command line. Until now this was only
// possible through the admin UI; scripting an addon's setup (CI, infra as
// code, a fresh environment) meant clicking through forms by hand.

var (
	addonConfigSet  []string
	addonConfigFile string
)

var addonConfigCmd = &cobra.Command{
	Use:   "config <slug>",
	Short: "View or update an addon's configuration",
	Long: `View or update an addon's non-secret configuration.

With no flags, prints the addon's current config.
With --set or --file, merges the given keys into the current config and
pushes the result -- existing keys not mentioned are left untouched.
Use "shoehorn addon secret" for credentials and other secret values.

Examples:
  # View current config
  shoehorn addon config jira-sync

  # Set one or more keys
  shoehorn addon config jira-sync --set jira_base_url=https://acme.atlassian.net --set jira_project=PROJ

  # Set keys from a JSON file
  shoehorn addon config jira-sync --file config.json`,
	Args: cobra.ExactArgs(1),
	RunE: runAddonConfig,
}

func runAddonConfig(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	if len(addonConfigSet) == 0 && addonConfigFile == "" {
		return showAddonConfig(cmd, client, slug)
	}
	return pushAddonConfig(cmd, client, slug)
}

func showAddonConfig(cmd *cobra.Command, client *api.Client, slug string) error {
	result, spinErr := tui.RunSpinner("Loading config...", func() (any, error) {
		return client.GetAddonInstallation(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("get addon config: %w", spinErr)
	}
	inst := result.(*api.InstallationDetail)

	var config map[string]any
	if len(inst.Config) > 0 {
		if err := json.Unmarshal(inst.Config, &config); err != nil {
			return fmt.Errorf("parse current config: %w", err)
		}
	}

	mode := ui.DetectMode(Interactive(), NoInteractive(), OutputFormat())
	if mode == ui.ModeYAML {
		return ui.RenderYAML(config)
	}
	return ui.RenderJSON(config)
}

func pushAddonConfig(cmd *cobra.Command, client *api.Client, slug string) error {
	result, spinErr := tui.RunSpinner("Loading current config...", func() (any, error) {
		return client.GetAddonInstallation(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("get addon config: %w", spinErr)
	}
	inst := result.(*api.InstallationDetail)

	merged := map[string]any{}
	if len(inst.Config) > 0 {
		if err := json.Unmarshal(inst.Config, &merged); err != nil {
			return fmt.Errorf("parse current config: %w", err)
		}
	}

	if addonConfigFile != "" {
		data, err := os.ReadFile(addonConfigFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", addonConfigFile, err)
		}
		var fromFile map[string]any
		if err := json.Unmarshal(data, &fromFile); err != nil {
			return fmt.Errorf("parse %s: %w", addonConfigFile, err)
		}
		for k, v := range fromFile {
			merged[k] = v
		}
	}

	for _, kv := range addonConfigSet {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return fmt.Errorf("invalid --set value %q, expected key=value", kv)
		}
		merged[key] = value
	}

	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return fmt.Errorf("marshal merged config: %w", err)
	}

	_, spinErr = tui.RunSpinner("Updating config...", func() (any, error) {
		return nil, client.UpdateAddonConfig(cmd.Context(), slug, mergedJSON)
	})
	if spinErr != nil {
		return fmt.Errorf("update addon config: %w", spinErr)
	}

	fmt.Printf("Addon %q config updated.\n", slug)
	return nil
}

// ─── addon secret ───────────────────────────────────────────────────────────

var addonSecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage an addon's secret values",
	Long:  `Set, list, or delete an addon's secret values (API tokens, credentials). Secret values are never returned once set -- only their key names can be listed.`,
}

var addonSecretSetCmd = &cobra.Command{
	Use:   "set <slug> <key> <value>",
	Short: "Set or update one secret value",
	Args:  cobra.ExactArgs(3),
	RunE:  runAddonSecretSet,
}

func runAddonSecretSet(cmd *cobra.Command, args []string) error {
	slug, key, value := args[0], args[1], args[2]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Setting secret %q...", key), func() (any, error) {
		return nil, client.SetAddonSecret(cmd.Context(), slug, key, value)
	})
	if spinErr != nil {
		return fmt.Errorf("set addon secret: %w", spinErr)
	}

	fmt.Printf("Secret %q set for addon %q.\n", key, slug)
	return nil
}

var addonSecretListCmd = &cobra.Command{
	Use:   "list <slug>",
	Short: "List an addon's secret key names (not values)",
	Args:  cobra.ExactArgs(1),
	RunE:  runAddonSecretList,
}

func runAddonSecretList(cmd *cobra.Command, args []string) error {
	slug := args[0]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	result, spinErr := tui.RunSpinner("Loading secret keys...", func() (any, error) {
		return client.ListAddonSecretKeys(cmd.Context(), slug)
	})
	if spinErr != nil {
		return fmt.Errorf("list addon secrets: %w", spinErr)
	}
	keys := result.([]string)

	mode := ui.DetectMode(Interactive(), NoInteractive(), OutputFormat())
	if mode == ui.ModeJSON {
		return ui.RenderJSON(keys)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(keys)
	}

	if len(keys) == 0 {
		fmt.Println("No secrets set for this addon.")
		return nil
	}
	for _, k := range keys {
		fmt.Println(k)
	}
	return nil
}

var addonSecretDeleteCmd = &cobra.Command{
	Use:   "delete <slug> <key>",
	Short: "Delete one secret value",
	Args:  cobra.ExactArgs(2),
	RunE:  runAddonSecretDelete,
}

func runAddonSecretDelete(cmd *cobra.Command, args []string) error {
	slug, key := args[0], args[1]
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}

	_, spinErr := tui.RunSpinner(fmt.Sprintf("Deleting secret %q...", key), func() (any, error) {
		return nil, client.DeleteAddonSecret(cmd.Context(), slug, key)
	})
	if spinErr != nil {
		return fmt.Errorf("delete addon secret: %w", spinErr)
	}

	fmt.Printf("Secret %q deleted for addon %q.\n", key, slug)
	return nil
}

// ─── Registration ───────────────────────────────────────────────────────────

func init() {
	addonConfigCmd.Flags().StringArrayVar(&addonConfigSet, "set", nil, "set a config key (key=value), repeatable")
	addonConfigCmd.Flags().StringVar(&addonConfigFile, "file", "", "merge config keys from a JSON file")

	addonSecretCmd.AddCommand(addonSecretSetCmd)
	addonSecretCmd.AddCommand(addonSecretListCmd)
	addonSecretCmd.AddCommand(addonSecretDeleteCmd)

	addonCmd.AddCommand(addonConfigCmd)
	addonCmd.AddCommand(addonSecretCmd)
}
