package commands

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/shoehorn-dev/shoehorn-cli/pkg/logging"
	"github.com/shoehorn-dev/shoehorn-cli/pkg/tui"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Version is set at build time via ldflags. Defaults to "dev" for local builds.
var Version = "dev"

var (
	cfgFile       string
	profile       string
	noInteractive bool
	interactive   bool
	outputFormat  string
	debug         bool
	yesFlag       bool
)

// YesFlag returns the --yes flag value (for use by sub-packages)
func YesFlag() bool { return yesFlag }

// Logger is the global structured logger, initialized in PersistentPreRun.
// Silent by default; enabled with --debug or SHOEHORN_DEBUG=1.
var Logger *zap.Logger = zap.NewNop()

// NoInteractive returns the --no-interactive flag value (for use by sub-packages)
func NoInteractive() bool { return noInteractive }

// Interactive returns true when the user explicitly requested interactive mode via -i
func Interactive() bool { return interactive }

// OutputFormat returns the --output flag value (for use by sub-packages)
func OutputFormat() string { return outputFormat }

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "shoehorn",
	Short: "Shoehorn CLI - Internal Developer Portal",
	Long: `Shoehorn CLI provides command-line access to the Shoehorn platform.

Use it to authenticate, manage workflows, and interact with the Forge service.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noInteractive {
			tui.SetPlainMode(true)
		}
		// Initialize structured logger (writes to stderr, silent unless --debug)
		Logger = logging.New(debug || logging.IsDebug())

		// Create a signal-aware context so Ctrl+C cancels in-flight API calls
		// instead of leaving them running until the HTTP timeout expires.
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		cmd.SetContext(ctx)
		cobra.OnFinalize(cancel)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

// RootCmd returns the root cobra command (used by sub-packages)
func RootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.shoehorn/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "authentication profile to use")
	rootCmd.PersistentFlags().BoolVarP(&noInteractive, "no-interactive", "I", false, "disable interactive mode (force plain output)")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "enable interactive table mode")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "", "output format (table|json|yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging to stderr (SHOEHORN_DEBUG=1)")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "skip confirmation prompts")

	// Set version for cobra's built-in --version flag
	rootCmd.Version = Version

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Shoehorn CLI " + Version)
		},
	})

	// Add shell completion command (bash, zsh, fish, powershell)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for Shoehorn CLI.

Examples:
  # Bash
  shoehorn completion bash > /etc/bash_completion.d/shoehorn

  # Zsh
  shoehorn completion zsh > "${fpath[1]}/_shoehorn"

  # Fish
  shoehorn completion fish > ~/.config/fish/completions/shoehorn.fish

  # PowerShell
  shoehorn completion powershell > shoehorn.ps1`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	})
}
