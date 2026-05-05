package commands

import (
	"strings"
	"testing"
)

const validMoldYAML = `
slug: test-mold
name: Test mold
version: 1.0.0
visibility: tenant
category: data
description: a test
icon: "🔧"
tags: [test, demo]
schema:
  type: object
  required: [name]
  properties:
    name:
      type: string
defaults:
  name: hello
actions:
  - action: create
    label: Create
    primary: true
ownerTeamIds: [team-1]
`

func TestParseMoldManifest_Valid(t *testing.T) {
	req, err := parseMoldManifest([]byte(validMoldYAML))
	if err != nil {
		t.Fatalf("parseMoldManifest: %v", err)
	}
	if req.Slug != "test-mold" {
		t.Errorf("slug = %q, want test-mold", req.Slug)
	}
	if req.Visibility != "tenant" {
		t.Errorf("visibility = %q, want tenant", req.Visibility)
	}
	if len(req.Schema) == 0 {
		t.Errorf("schema empty")
	}
	if len(req.Actions) != 1 || req.Actions[0]["action"] != "create" {
		t.Errorf("actions = %+v, want one create", req.Actions)
	}
	if len(req.OwnerTeamIDs) != 1 || req.OwnerTeamIDs[0] != "team-1" {
		t.Errorf("ownerTeamIds = %+v, want [team-1]", req.OwnerTeamIDs)
	}
}

func TestParseMoldManifest_MissingFields(t *testing.T) {
	cases := []struct {
		name      string
		stripLine string
		wantErr   string
	}{
		{"slug", "slug: test-mold", "slug is required"},
		{"name", "name: Test mold", "name is required"},
		{"version", "version: 1.0.0", "version is required"},
		{"visibility", "visibility: tenant", "visibility is required"},
		{"category", "category: data", "category is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			yaml := strings.Replace(validMoldYAML, tc.stripLine, "", 1)
			_, err := parseMoldManifest([]byte(yaml))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want contains %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseMoldManifest_NoActions(t *testing.T) {
	yaml := strings.Replace(validMoldYAML, "actions:\n  - action: create\n    label: Create\n    primary: true", "actions: []", 1)
	_, err := parseMoldManifest([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "at least one action") {
		t.Errorf("err = %v, want 'at least one action'", err)
	}
}

func TestParseMoldManifest_Empty(t *testing.T) {
	_, err := parseMoldManifest(nil)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v, want 'empty'", err)
	}
}

func TestParseMoldManifest_RejectsUnknownFields(t *testing.T) {
	yaml := validMoldYAML + "\nbogusField: nope\n"
	_, err := parseMoldManifest([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "bogusField") {
		t.Errorf("err = %v, want unknown-field error mentioning bogusField", err)
	}
}
