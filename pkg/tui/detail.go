package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Field is a single labeled field in a detail panel
type Field struct {
	Label string
	Value string
}

// DetailSection is a named group of fields
type DetailSection struct {
	Title  string
	Fields []Field
}

// RenderDetail renders a lipgloss detail panel with the given title and sections.
func RenderDetail(title string, sections []DetailSection) string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render(title))
	content.WriteString("\n")

	for i, section := range sections {
		if i > 0 {
			content.WriteString("\n")
		}

		if section.Title != "" {
			content.WriteString("\n")
			content.WriteString(HeaderStyle.Render(section.Title))
			content.WriteString("\n")
		}

		for _, f := range section.Fields {
			label := LabelStyle.Render(f.Label)
			value := f.Value
			if value == "" {
				value = MutedStyle.Render("—")
			}
			content.WriteString(fmt.Sprintf("%s  %s\n", label, value))
		}
	}

	return BoxStyle.Render(content.String())
}

// RenderScoreBar renders a colored progress bar for a score out of max
func RenderScoreBar(score, max int) string {
	if max <= 0 {
		return MutedStyle.Render("N/A")
	}
	pct := float64(score) / float64(max)
	barWidth := 30
	filled := int(pct * float64(barWidth))
	empty := barWidth - filled

	var barColor lipgloss.Color
	switch {
	case pct >= 0.9:
		barColor = lipgloss.Color("82")
	case pct >= 0.7:
		barColor = lipgloss.Color("118")
	case pct >= 0.5:
		barColor = lipgloss.Color("220")
	case pct >= 0.3:
		barColor = lipgloss.Color("208")
	default:
		barColor = lipgloss.Color("196")
	}

	barStyle := lipgloss.NewStyle().Foreground(barColor)
	bar := barStyle.Render(strings.Repeat("█", filled)) + MutedStyle.Render(strings.Repeat("░", empty))
	return fmt.Sprintf("%s %d/%d (%.0f%%)", bar, score, max, pct*100)
}

// RenderTagBadges renders a list of tags as inline badges
func RenderTagBadges(tags []string) string {
	if len(tags) == 0 {
		return MutedStyle.Render("none")
	}
	badges := make([]string, len(tags))
	for i, t := range tags {
		badges[i] = BadgeStyle.Render(t)
	}
	return strings.Join(badges, " ")
}

// RenderLinkLine renders a list of link strings
func RenderLinkLine(links []string) string {
	if len(links) == 0 {
		return MutedStyle.Render("none")
	}
	return strings.Join(links, "  ")
}

// SuccessBox renders a success message in a rounded green-bordered box
func SuccessBox(title, body string) string {
	content := SuccessStyle.Render("✓ "+title) + "\n\n" + body
	return BoxStyle.Copy().BorderForeground(lipgloss.Color("82")).Render(content)
}

// ErrorBox renders an error message in a rounded red-bordered box
func ErrorBox(title, body string) string {
	content := ErrorStyle.Render("✗ "+title) + "\n\n" + body
	return BoxStyle.Copy().BorderForeground(lipgloss.Color("196")).Render(content)
}
