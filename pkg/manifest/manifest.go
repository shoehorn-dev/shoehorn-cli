// Package manifest parses Shoehorn manifest YAML files into typed resources.
// Supports single and multi-document YAML (separated by ---).
package manifest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Resource represents a parsed manifest resource.
type Resource struct {
	// Extracted metadata
	ServiceID   string   `yaml:"-"`
	Name        string   `yaml:"-"`
	Type        string   `yaml:"-"`
	Description string   `yaml:"-"`
	Tags        []string `yaml:"-"`

	// RawYAML is the original YAML content for this resource,
	// suitable for sending to the manifest API.
	RawYAML string `yaml:"-"`
}

// manifestDoc matches the Shoehorn manifest YAML structure.
type manifestDoc struct {
	SchemaVersion int `yaml:"schemaVersion"`
	Service       struct {
		ID          string `yaml:"id"`
		Name        string `yaml:"name"`
		Type        string `yaml:"type"`
		Description string `yaml:"description"`
	} `yaml:"service"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
}

// Parse reads YAML from r and returns parsed resources.
// Supports multi-document YAML (documents separated by ---).
// Empty documents are silently skipped.
func Parse(r io.Reader) ([]Resource, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	// Split on --- for multi-document support
	docs := splitYAMLDocuments(string(data))

	var resources []Resource
	docNum := 0
	for _, doc := range docs {
		trimmed := strings.TrimSpace(doc)
		if trimmed == "" || trimmed == "---" {
			continue
		}
		docNum++

		var m manifestDoc
		if err := yaml.Unmarshal([]byte(trimmed), &m); err != nil {
			return nil, fmt.Errorf("document %d: parse manifest YAML: %w", docNum, err)
		}

		// Skip empty documents (parsed but no content)
		if m.Service.ID == "" && m.Service.Name == "" {
			continue
		}

		if m.Service.ID == "" {
			return nil, fmt.Errorf("document %d: manifest missing required field: service.id", docNum)
		}

		// Description can be at service level or top level
		desc := m.Service.Description
		if desc == "" {
			desc = m.Description
		}

		resources = append(resources, Resource{
			ServiceID:   m.Service.ID,
			Name:        m.Service.Name,
			Type:        m.Service.Type,
			Description: desc,
			Tags:        m.Tags,
			RawYAML:     trimmed,
		})
	}

	return resources, nil
}

// maxManifestFileSize is the maximum allowed manifest file size (10 MB).
const maxManifestFileSize = 10 * 1024 * 1024

// ReadFile reads manifest content from a file path or stdin ("-").
// Rejects files larger than 10 MB to prevent memory exhaustion.
func ReadFile(path string) (string, error) {
	if path == "-" {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, maxManifestFileSize+1))
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		if int64(len(data)) > maxManifestFileSize {
			return "", fmt.Errorf("stdin exceeds maximum manifest size (%d bytes)", maxManifestFileSize)
		}
		return string(data), nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}
	if info.Size() > maxManifestFileSize {
		return "", fmt.Errorf("%s exceeds maximum manifest size (%d bytes)", path, maxManifestFileSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", path, err)
	}
	return string(data), nil
}

// splitYAMLDocuments splits a multi-document YAML string into individual documents.
func splitYAMLDocuments(data string) []string {
	// Split on lines that are exactly "---" (YAML document separator)
	var docs []string
	var current strings.Builder

	for _, line := range strings.Split(data, "\n") {
		if strings.TrimSpace(line) == "---" {
			if current.Len() > 0 {
				docs = append(docs, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		docs = append(docs, current.String())
	}

	return docs
}
