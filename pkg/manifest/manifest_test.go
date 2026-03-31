package manifest

import (
	"strings"
	"testing"
)

const singleDoc = `schemaVersion: 1
service:
  id: my-service
  name: My Service
  type: service
  description: A test service
owner:
  - id: platform-team
    type: team
tags:
  - go
  - api
`

const multiDoc = `schemaVersion: 1
service:
  id: service-a
  name: Service A
  type: service
owner:
  - id: team-a
    type: team
---
schemaVersion: 1
service:
  id: service-b
  name: Service B
  type: api
owner:
  - id: team-b
    type: team
`

func TestParse_SingleDocument(t *testing.T) {
	resources, err := Parse(strings.NewReader(singleDoc))
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("got %d resources, want 1", len(resources))
	}
	r := resources[0]
	if r.ServiceID != "my-service" {
		t.Errorf("ServiceID = %q", r.ServiceID)
	}
	if r.Name != "My Service" {
		t.Errorf("Name = %q", r.Name)
	}
	if r.Type != "service" {
		t.Errorf("Type = %q", r.Type)
	}
	if r.Description != "A test service" {
		t.Errorf("Description = %q", r.Description)
	}
	if len(r.Tags) != 2 || r.Tags[0] != "go" {
		t.Errorf("Tags = %v", r.Tags)
	}
}

func TestParse_MultiDocument(t *testing.T) {
	resources, err := Parse(strings.NewReader(multiDoc))
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 2 {
		t.Fatalf("got %d resources, want 2", len(resources))
	}
	if resources[0].ServiceID != "service-a" {
		t.Errorf("[0].ServiceID = %q", resources[0].ServiceID)
	}
	if resources[1].ServiceID != "service-b" {
		t.Errorf("[1].ServiceID = %q", resources[1].ServiceID)
	}
	if resources[1].Type != "api" {
		t.Errorf("[1].Type = %q", resources[1].Type)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	resources, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 0 {
		t.Errorf("got %d resources, want 0", len(resources))
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	_, err := Parse(strings.NewReader("{{not valid yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParse_MissingServiceID(t *testing.T) {
	_, err := Parse(strings.NewReader(`schemaVersion: 1
service:
  name: No ID
  type: service
`))
	if err == nil {
		t.Fatal("expected error for missing service.id")
	}
}

func TestResource_RawYAML(t *testing.T) {
	resources, err := Parse(strings.NewReader(singleDoc))
	if err != nil {
		t.Fatal(err)
	}
	raw := resources[0].RawYAML
	if raw == "" {
		t.Fatal("RawYAML is empty")
	}
	if !strings.Contains(raw, "my-service") {
		t.Error("RawYAML should contain the service ID")
	}
}

func TestParse_DescriptionFallback_TopLevel(t *testing.T) {
	input := `schemaVersion: 1
service:
  id: svc
  name: Svc
  type: service
description: top-level desc
owner:
  - id: team
    type: team
`
	resources, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if resources[0].Description != "top-level desc" {
		t.Errorf("Description = %q, want 'top-level desc'", resources[0].Description)
	}
}

func TestParse_DescriptionPriority_ServiceOverTopLevel(t *testing.T) {
	input := `schemaVersion: 1
service:
  id: svc
  name: Svc
  type: service
  description: service-level desc
description: top-level desc
owner:
  - id: team
    type: team
`
	resources, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if resources[0].Description != "service-level desc" {
		t.Errorf("Description = %q, want 'service-level desc' (service takes priority)", resources[0].Description)
	}
}

func TestParse_MissingServiceID_IncludesDocIndex(t *testing.T) {
	input := `schemaVersion: 1
service:
  id: good-service
  name: Good
  type: service
owner:
  - id: team
    type: team
---
schemaVersion: 1
service:
  name: Bad No ID
  type: service
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "document 2") {
		t.Errorf("error should include document index, got: %v", err)
	}
}

func TestParse_SkipsEmptyDocuments(t *testing.T) {
	input := `---
schemaVersion: 1
service:
  id: svc-a
  name: A
  type: service
owner:
  - id: team
    type: team
---
---
`
	resources, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Errorf("got %d resources, want 1 (empty docs should be skipped)", len(resources))
	}
}
