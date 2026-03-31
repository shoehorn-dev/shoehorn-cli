package ui

import (
	"os"

	"golang.org/x/term"
)

// OutputMode represents the CLI output mode
type OutputMode string

const (
	ModeInteractive OutputMode = "interactive" // TUI with colors and interactivity
	ModePlain       OutputMode = "plain"       // Plain table without colors
	ModeJSON        OutputMode = "json"        // JSON output for scripting
	ModeYAML        OutputMode = "yaml"        // YAML output
)

// DetectMode determines the appropriate output mode based on environment and flags.
// Default is plain (kubectl-style); interactive requires explicit -i flag.
func DetectMode(interactive bool, noInteractive bool, outputFormat string) OutputMode {
	// 1. Explicit format flag takes precedence
	if outputFormat == "json" {
		return ModeJSON
	}
	if outputFormat == "yaml" {
		return ModeYAML
	}

	// 2. --no-interactive flag forces plain mode (backwards compat)
	if noInteractive {
		return ModePlain
	}

	// 3. Explicit -i flag requests interactive mode (only if TTY)
	if interactive && isTerminal() && !isCIEnvironment() && os.Getenv("NO_COLOR") == "" {
		return ModeInteractive
	}

	// 4. Default: plain column output
	return ModePlain
}

// IsInteractive returns true if the mode supports interactive features
func IsInteractive(mode OutputMode) bool {
	return mode == ModeInteractive
}

// isTerminal checks if stdout is connected to a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// isCIEnvironment detects common CI/CD environments
func isCIEnvironment() bool {
	ciEnvVars := []string{
		"CI",                     // Generic CI flag
		"GITHUB_ACTIONS",         // GitHub Actions
		"GITLAB_CI",              // GitLab CI
		"CIRCLECI",               // CircleCI
		"TRAVIS",                 // Travis CI
		"JENKINS_HOME",           // Jenkins
		"TEAMCITY_VERSION",       // TeamCity
		"BUILDKITE",              // Buildkite
		"TF_BUILD",               // Azure Pipelines
		"CONTINUOUS_INTEGRATION", // Generic
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// ShouldUseColor returns true if color output should be used
func ShouldUseColor(mode OutputMode) bool {
	// Only use color in interactive or plain mode with TTY
	if mode == ModeJSON || mode == ModeYAML {
		return false
	}

	// NO_COLOR env var disables colors
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Only use colors if we have a terminal
	return isTerminal()
}
