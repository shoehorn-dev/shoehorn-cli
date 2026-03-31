// Package tui provides shared TUI styling and components for the Shoehorn CLI.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	// TitleStyle renders page/command titles in bright yellow (brand primary)
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))

	// HeaderStyle renders section headers in blue
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))

	// SuccessStyle renders success messages in green
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))

	// ErrorStyle renders error messages in red
	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// WarnStyle renders warnings in yellow
	WarnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))

	// MutedStyle renders secondary/muted text in grey
	MutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// BadgeStyle renders inline badges with a dark background
	BadgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("238")).
			Foreground(lipgloss.Color("250"))

	// BoxStyle renders a rounded-border box with padding
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	// LabelStyle renders the label column in detail panels (fixed width for alignment)
	LabelStyle = lipgloss.NewStyle().Bold(true).Width(18)

	// ValueStyle renders the value column in detail panels
	ValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// SelectedStyle renders the selected table row
	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57"))

	// StatusHealthy styles a healthy/OK status
	StatusHealthy = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)

	// StatusDegraded styles a degraded/warn status
	StatusDegraded = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)

	// StatusUnhealthy styles an unhealthy/error status
	StatusUnhealthy = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// StatusColor returns the appropriate style for a health/status string
func StatusColor(status string) lipgloss.Style {
	switch status {
	case "healthy", "active", "connected", "completed", "running":
		return StatusHealthy
	case "degraded", "warning", "pending", "executing":
		return StatusDegraded
	case "unhealthy", "error", "failed", "disconnected":
		return StatusUnhealthy
	default:
		return MutedStyle
	}
}

// GradeColor returns the appropriate style for a scorecard grade
func GradeColor(grade string) lipgloss.Style {
	switch grade {
	case "A":
		return StatusHealthy
	case "B":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("118")).Bold(true)
	case "C":
		return WarnStyle
	case "D":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	default: // F
		return StatusUnhealthy
	}
}
