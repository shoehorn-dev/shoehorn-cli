package tui

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// plainMode disables TUI spinners and renders plain output instead.
// Uses atomic.Bool for safe concurrent access.
var plainMode atomic.Bool

// SetPlainMode forces all TUI spinners to run without animation.
// Call this when --no-interactive is set or stdout is not a TTY.
func SetPlainMode(v bool) {
	plainMode.Store(v)
}

// isTTY returns true if stdout is an interactive terminal.
func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// resultMsg carries the result back from the async function
type resultMsg struct {
	value any
	err   error
}

// spinnerModel is a bubbletea model for displaying a spinner while work happens
type spinnerModel struct {
	spinner spinner.Model
	message string
	done    bool
	err     error
	result  any
	fn      func() (any, error)
}

func newSpinnerModel(message string, fn func() (any, error)) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	return spinnerModel{
		spinner: s,
		message: message,
		fn:      fn,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			val, err := m.fn()
			return resultMsg{value: val, err: err}
		},
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resultMsg:
		m.done = true
		m.result = msg.value
		m.err = msg.err
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("  %s %s\n", m.spinner.View(), MutedStyle.Render(m.message))
}

// RunSpinner runs fn in the background while displaying a spinner.
// Falls back to plain execution (no animation) when stdout is not a TTY
// or plain mode is enabled via SetPlainMode.
func RunSpinner(message string, fn func() (any, error)) (any, error) {
	if plainMode.Load() || !isTTY() {
		fmt.Fprintf(os.Stderr, "%s\n", message)
		return fn()
	}
	m := newSpinnerModel(message, fn)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("spinner: %w", err)
	}
	fm, ok := final.(spinnerModel)
	if !ok {
		return nil, fmt.Errorf("spinner: unexpected final model type %T", final)
	}
	return fm.result, fm.err
}
