package ui

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRenderListResult_JSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := []map[string]string{{"name": "test"}}
	err := RenderListResult(ModeJSON, data, ListConfig{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var parsed []map[string]string
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
}

func TestRenderListResult_YAML(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := []map[string]string{{"name": "test"}}
	err := RenderListResult(ModeYAML, data, ListConfig{})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "name: test") {
		t.Errorf("YAML output missing data: %s", buf.String())
	}
}

func TestRenderListResult_Plain_RendersTable(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := RenderListResult(ModePlain, nil, ListConfig{
		Columns:  []string{"Name", "Type"},
		Rows:     [][]string{{"svc-a", "service"}, {"svc-b", "library"}},
		EmptyMsg: "No entities found",
	})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "NAME") {
		t.Errorf("missing uppercased header NAME: %s", output)
	}
	if !strings.Contains(output, "svc-a") {
		t.Errorf("missing row data: %s", output)
	}
}

func TestRenderListResult_Plain_Empty(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := RenderListResult(ModePlain, nil, ListConfig{
		Columns:  []string{"Name"},
		Rows:     [][]string{},
		EmptyMsg: "No teams found",
	})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No teams found") {
		t.Errorf("missing empty message: %s", buf.String())
	}
}

func TestRenderListResult_Plain_DefaultEmptyMsg(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := RenderListResult(ModePlain, nil, ListConfig{
		Columns: []string{"Name"},
		Rows:    [][]string{},
	})

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "No results found") {
		t.Errorf("missing default empty message: %s", buf.String())
	}
}
