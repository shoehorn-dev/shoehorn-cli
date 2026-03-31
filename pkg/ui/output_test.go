package ui

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRenderJSON_ValidJSON(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"name": "test", "type": "service"}
	err := RenderJSON(data)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify it's valid JSON
	var parsed map[string]string
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("RenderJSON output is not valid JSON: %v\nOutput: %s", err, output)
	}
	if parsed["name"] != "test" {
		t.Errorf("parsed name = %q", parsed["name"])
	}
}

func TestRenderYAML_ValidYAML(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"name": "test"}
	err := RenderYAML(data)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var parsed map[string]string
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("RenderYAML output is not valid YAML: %v\nOutput: %s", err, output)
	}
	if parsed["name"] != "test" {
		t.Errorf("parsed name = %q", parsed["name"])
	}
}

func TestRenderError_WritesToStderr(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	RenderError(os.ErrNotExist)

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Error:") {
		t.Errorf("RenderError output = %q, want 'Error:' prefix", output)
	}
}

func TestRenderTable_PrintsHeaders(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RenderTable(
		[]string{"Name", "Type"},
		[][]string{{"payment", "service"}, {"auth", "library"}},
	)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "NAME") {
		t.Errorf("missing uppercased header NAME in output: %s", output)
	}
	if !strings.Contains(output, "TYPE") {
		t.Errorf("missing uppercased header TYPE in output: %s", output)
	}
	if !strings.Contains(output, "payment") {
		t.Errorf("missing row data 'payment' in output: %s", output)
	}
}

func TestRenderTable_Empty_PrintsNoResourcesFound(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RenderTable([]string{"Name"}, [][]string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "No resources found") {
		t.Errorf("empty table should say 'No resources found', got: %s", output)
	}
}
