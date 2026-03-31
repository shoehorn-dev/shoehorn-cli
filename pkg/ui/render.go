package ui

import "fmt"

// ListConfig defines how a list of items should be rendered in plain/interactive modes.
// For JSON/YAML output, the raw data is used directly and this config is ignored.
type ListConfig struct {
	// Columns are the plain-text table column headers (e.g. "Name", "Type")
	Columns []string
	// Rows are the plain-text table rows (each row is a slice of cell values)
	Rows [][]string
	// EmptyMsg is shown when Rows is empty. Defaults to "No results found".
	EmptyMsg string
}

// RenderListResult handles the full output dispatch for list commands.
// For JSON/YAML modes, it serializes data directly and ignores cfg.
// For Plain mode, it renders a table from cfg.Columns and cfg.Rows.
// For Interactive mode, callers should handle tui.RunTable themselves
// (interactive tables need column widths and selection callbacks that
// vary per command and don't generalize cleanly).
func RenderListResult(mode OutputMode, data any, cfg ListConfig) error {
	switch mode {
	case ModeJSON:
		return RenderJSON(data)
	case ModeYAML:
		return RenderYAML(data)
	case ModeInteractive:
		// Interactive mode needs custom column widths and selection callbacks
		// that vary per command. Callers must handle this before calling
		// RenderListResult. If we reach here, fall through to plain table.
		fallthrough
	default:
		if len(cfg.Rows) == 0 {
			msg := cfg.EmptyMsg
			if msg == "" {
				msg = "No results found"
			}
			fmt.Println(msg)
			return nil
		}
		RenderTable(cfg.Columns, cfg.Rows)
		return nil
	}
}
