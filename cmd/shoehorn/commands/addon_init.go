package commands

import (
	"fmt"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/addon"
	"github.com/spf13/cobra"
)

var addonInitTier string

var addonInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Scaffold a new addon project",
	Long: `Create a new addon project directory with starter templates.

Examples:
  shoehorn addon init my-addon
  shoehorn addon init my-addon --tier full
  shoehorn addon init my-integration --tier declarative`,
	Args: cobra.ExactArgs(1),
	RunE: runAddonInit,
}

func runAddonInit(_ *cobra.Command, args []string) error {
	name := args[0]
	tier := addon.Tier(addonInitTier)

	if err := addon.ValidateSlug(name); err != nil {
		return err
	}

	if !addon.ValidTiers[tier] {
		return fmt.Errorf("invalid tier %q: use declarative, scripted, or full", addonInitTier)
	}

	cfg := addon.ScaffoldConfig{
		Name: name,
		Tier: tier,
	}

	if err := addon.Scaffold(cfg); err != nil {
		return fmt.Errorf("scaffold addon: %w", err)
	}

	fmt.Printf("Addon %q scaffolded in ./%s/\n", name, name)
	fmt.Println()

	switch tier {
	case addon.TierDeclarative:
		fmt.Println("Next steps:")
		fmt.Println("  1. Edit manifest.json with your integration config")
		fmt.Println("  2. shoehorn addon publish")
	case addon.TierScripted, addon.TierFull:
		fmt.Println("Next steps:")
		fmt.Printf("  1. cd %s\n", name)
		fmt.Println("  2. npm install")
		fmt.Println("  3. Edit src/index.ts")
		fmt.Println("  4. shoehorn addon build")
		fmt.Println("  5. shoehorn addon publish")
	}

	return nil
}

func init() {
	addonInitCmd.Flags().StringVar(&addonInitTier, "tier", "scripted",
		"addon tier: declarative, scripted, or full")
	addonCmd.AddCommand(addonInitCmd)
}
