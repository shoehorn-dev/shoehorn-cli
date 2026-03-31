package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// RenderJSON outputs data as formatted JSON
func RenderJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// RenderYAML outputs data as formatted YAML
func RenderYAML(v any) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(v)
}

// RenderError outputs an error message to stderr
func RenderError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// RenderTable outputs a kubectl-style plain table with CAPS headers and tab-aligned columns.
func RenderTable(cols []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println("No resources found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Header in CAPS
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = strings.ToUpper(c)
	}
	fmt.Fprintln(w, strings.Join(header, "\t"))

	// Rows
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	w.Flush()
}
