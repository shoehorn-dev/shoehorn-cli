package commands

import (
	"fmt"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/api"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/config"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/ui"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user info",
	Long:  `Display full information about the currently authenticated user.`,
	RunE:  runWhoami,
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, args []string) error {
	client, err := api.NewClientFromConfig(api.WithLogger(Logger))
	if err != nil {
		return err
	}
	ctx := cmd.Context()

	result, spinErr := tui.RunSpinner("Fetching user info...", func() (any, error) {
		return client.GetMe(ctx)
	})
	if spinErr != nil {
		return fmt.Errorf("fetch user: %w", spinErr)
	}

	me := result.(*api.MeResponse)

	mode := ui.DetectMode(interactive, noInteractive, outputFormat)
	if mode == ui.ModeJSON {
		return ui.RenderJSON(me)
	}
	if mode == ui.ModeYAML {
		return ui.RenderYAML(me)
	}

	// Panel title carries context — which server and which auth kind —
	// rather than mirroring a body field. Knowing "am I pointed at prod
	// or staging" is the high-value question at a glance; the actual
	// identity is in the rows below. Matches the pattern `gh auth status`
	// uses ("Logged in to github.com as …").
	title := "Signed in"
	if cfg, err := config.Load(); err == nil {
		if profile, perr := cfg.GetCurrentProfile(); perr == nil && profile != nil && profile.Server != "" {
			authKind := "session"
			if cfg.IsPATAuth() {
				authKind = "PAT"
			}
			title = fmt.Sprintf("Signed in to %s (%s)", profile.Server, authKind)
		}
	}

	panel := tui.RenderDetail(title, []tui.DetailSection{
		{
			Fields: []tui.Field{
				{Label: "Username", Value: me.Username},
				{Label: "Email", Value: me.Email},
				{Label: "Tenant", Value: me.TenantID},
				{Label: "User ID", Value: me.ID},
			},
		},
	})

	fmt.Println(panel)
	return nil
}
