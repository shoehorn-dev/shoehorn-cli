package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableConfig configures an interactive table
type TableConfig struct {
	Title    string
	Columns  []table.Column
	Rows     []table.Row
	OnSelect func(row table.Row) // optional callback when user presses Enter
}

type tableModel struct {
	table    table.Model
	title    string
	onSelect func(row table.Row)
	selected table.Row
	filter   string
	allRows  []table.Row
	quitting bool
}

var tableStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func newTableModel(cfg TableConfig) tableModel {
	t := table.New(
		table.WithColumns(cfg.Columns),
		table.WithRows(cfg.Rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("220"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	t.SetStyles(s)

	return tableModel{
		table:    t,
		title:    cfg.Title,
		onSelect: cfg.OnSelect,
		allRows:  cfg.Rows,
	}
}

func (m tableModel) Init() tea.Cmd {
	return nil
}

func (m tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			if m.filter != "" {
				m.filter = ""
				m.table.SetRows(m.allRows)
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if len(m.table.Rows()) > 0 {
				m.selected = m.table.SelectedRow()
				m.quitting = true
				return m, tea.Quit
			}
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "/":
			// Start filter mode — in this simple implementation, not interactive yet
			return m, nil
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
		default:
			// Accumulate filter characters
			if len(msg.String()) == 1 {
				char := msg.String()
				if char >= " " && char <= "~" {
					// Only accept if not a table navigation key
					if !strings.ContainsAny(char, "jk") {
						m.filter += char
						m.applyFilter()
						return m, nil
					}
				}
			}
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *tableModel) applyFilter() {
	if m.filter == "" {
		m.table.SetRows(m.allRows)
		return
	}
	var filtered []table.Row
	for _, row := range m.allRows {
		for _, cell := range row {
			if strings.Contains(strings.ToLower(cell), strings.ToLower(m.filter)) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	m.table.SetRows(filtered)
}

func (m tableModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	if m.title != "" {
		b.WriteString(TitleStyle.Render(m.title))
		b.WriteString("\n\n")
	}

	b.WriteString(tableStyle.Render(m.table.View()))
	b.WriteString("\n\n")

	// Status bar
	rowCount := len(m.table.Rows())
	hint := MutedStyle.Render(fmt.Sprintf("%d items  •  ↑/↓ navigate  •  Enter select  •  q quit", rowCount))
	if m.filter != "" {
		hint = MutedStyle.Render(fmt.Sprintf("filter: %q  •  %d matches  •  Backspace clear  •  q quit", m.filter, rowCount))
	}
	b.WriteString(hint)

	return b.String()
}

// RunTable displays an interactive table and returns the selected row (or nil if quit without selection).
func RunTable(cfg TableConfig) (table.Row, error) {
	if len(cfg.Rows) == 0 {
		fmt.Println(MutedStyle.Render("No items found."))
		return nil, nil
	}

	m := newTableModel(cfg)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("table: %w", err)
	}

	fm := final.(tableModel)

	if fm.selected != nil && fm.onSelect != nil {
		fm.onSelect(fm.selected)
	}

	return fm.selected, nil
}
