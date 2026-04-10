package mold

import (
	"strings"
	"testing"
)

// ─── Table-driven unit tests ────────────────────────────────────────────────

func TestValidateMold_RequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		def        MoldDefinition
		wantValid  bool
		wantErrors []string // substrings to find in error messages
	}{
		{
			name:       "empty definition",
			def:        MoldDefinition{},
			wantValid:  false,
			wantErrors: []string{"version", "metadata.name", "steps"},
		},
		{
			name: "missing version",
			def: MoldDefinition{
				Metadata: WorkflowMetadata{Name: "test"},
				Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
			},
			wantValid:  false,
			wantErrors: []string{"version"},
		},
		{
			name: "missing metadata.name",
			def: MoldDefinition{
				Version: "1.0.0",
				Steps:   []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
			},
			wantValid:  false,
			wantErrors: []string{"metadata.name"},
		},
		{
			name: "missing steps and actions",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test"},
			},
			wantValid:  false,
			wantErrors: []string{"steps"},
		},
		{
			name: "valid minimal with steps",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test"},
				Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
			},
			wantValid: true,
		},
		{
			name: "valid minimal with actions only",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test"},
				Actions:  []WorkflowAction{{Action: "github.repo.create", Label: "Create"}},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMold(&tt.def)
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v; errors: %v", result.Valid, tt.wantValid, result.Errors)
			}
			for _, want := range tt.wantErrors {
				found := false
				for _, e := range result.Errors {
					if strings.Contains(e.Field, want) || strings.Contains(e.Message, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got: %v", want, result.Errors)
				}
			}
		})
	}
}

func TestValidateMold_StepValidation(t *testing.T) {
	base := func(steps []WorkflowStep) *MoldDefinition {
		return &MoldDefinition{
			Version:  "1.0.0",
			Metadata: WorkflowMetadata{Name: "test"},
			Steps:    steps,
		}
	}

	tests := []struct {
		name       string
		def        *MoldDefinition
		wantValid  bool
		wantErrors []string
	}{
		{
			name:       "step missing id",
			def:        base([]WorkflowStep{{Name: "step", Action: "github.repo.create"}}),
			wantValid:  false,
			wantErrors: []string{"id"},
		},
		{
			name:       "step missing name",
			def:        base([]WorkflowStep{{ID: "s1", Action: "github.repo.create"}}),
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name:       "step missing action and adapter",
			def:        base([]WorkflowStep{{ID: "s1", Name: "step"}}),
			wantValid:  false,
			wantErrors: []string{"action", "adapter"},
		},
		{
			name: "step has both action and adapter",
			def: base([]WorkflowStep{{
				ID: "s1", Name: "step",
				Action:  "github.repo.create",
				Adapter: "http",
			}}),
			wantValid:  false,
			wantErrors: []string{"action", "adapter"},
		},
		{
			name: "duplicate step ids",
			def: base([]WorkflowStep{
				{ID: "s1", Name: "first", Action: "github.repo.create"},
				{ID: "s1", Name: "second", Action: "github.file.create"},
			}),
			wantValid:  false,
			wantErrors: []string{"duplicate"},
		},
		{
			name: "valid scaffolder step",
			def: base([]WorkflowStep{
				{ID: "s1", Name: "Create repo", Action: "github.repo.create"},
			}),
			wantValid: true,
		},
		{
			name: "valid adapter step",
			def: base([]WorkflowStep{
				{ID: "s1", Name: "Fetch data", Adapter: "http"},
			}),
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMold(tt.def)
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v; errors: %v", result.Valid, tt.wantValid, result.Errors)
			}
			for _, want := range tt.wantErrors {
				found := false
				for _, e := range result.Errors {
					if strings.Contains(strings.ToLower(e.Field+e.Message), strings.ToLower(want)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got: %v", want, result.Errors)
				}
			}
		})
	}
}

func TestValidateMold_ActionFormat(t *testing.T) {
	base := func(action string) *MoldDefinition {
		return &MoldDefinition{
			Version:  "1.0.0",
			Metadata: WorkflowMetadata{Name: "test"},
			Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: action}},
		}
	}

	tests := []struct {
		name      string
		action    string
		wantValid bool
	}{
		{"valid 3-part action", "github.repo.create", true},
		{"valid 2-part action", "system.echo", true},
		{"single word invalid", "create", false},
		{"empty action", "", false}, // caught by missing action/adapter check
		{"valid catalog action", "catalog.entity.register", true},
		{"valid deployment action", "deployment.service.production", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := base(tt.action)
			if tt.action == "" {
				def.Steps[0].Action = ""
				def.Steps[0].Adapter = "" // ensure both empty
			}
			result := ValidateMold(def)
			if result.Valid != tt.wantValid {
				t.Errorf("action %q: Valid = %v, want %v; errors: %v",
					tt.action, result.Valid, tt.wantValid, result.Errors)
			}
		})
	}
}

func TestValidateMold_InvalidActionProvider(t *testing.T) {
	def := &MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "invalid.repo.create"}},
	}
	result := ValidateMold(def)
	if result.Valid {
		t.Error("expected invalid for unknown provider")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "provider") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about provider, got: %v", result.Errors)
	}
}

func TestValidateMold_InvalidAdapter(t *testing.T) {
	def := &MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps:    []WorkflowStep{{ID: "s1", Name: "step", Adapter: "nonexistent"}},
	}
	result := ValidateMold(def)
	if result.Valid {
		t.Error("expected invalid for unknown adapter")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "adapter") || strings.Contains(e.Message, "nonexistent") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about adapter, got: %v", result.Errors)
	}
}

func TestValidateMold_ValidAdapters(t *testing.T) {
	adapters := []string{
		"http", "postgres", "slack", "github", "docker",
		"kubernetes", "terraform", "webhook", "email",
		"s3", "gcs", "file", "git", "catalog", "log",
	}
	for _, adapter := range adapters {
		t.Run(adapter, func(t *testing.T) {
			def := &MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test"},
				Steps:    []WorkflowStep{{ID: "s1", Name: "step", Adapter: adapter}},
			}
			result := ValidateMold(def)
			if !result.Valid {
				t.Errorf("adapter %q should be valid, got errors: %v", adapter, result.Errors)
			}
		})
	}
}

func TestValidateMold_ApprovalFlow(t *testing.T) {
	base := func(af *WorkflowApprovalFlow) *MoldDefinition {
		return &MoldDefinition{
			Version:      "1.0.0",
			Metadata:     WorkflowMetadata{Name: "test"},
			Actions:      []WorkflowAction{{Action: "github.repo.create", Label: "Create"}},
			ApprovalFlow: af,
		}
	}

	tests := []struct {
		name       string
		af         *WorkflowApprovalFlow
		wantValid  bool
		wantErrors []string
	}{
		{
			name:      "nil approval flow is valid",
			af:        nil,
			wantValid: true,
		},
		{
			name: "valid approval flow",
			af: &WorkflowApprovalFlow{
				Required: true,
				Steps: []ApprovalStep{
					{Name: "Review", Approvers: []string{"admin@co.com"}, RequiredCount: 1},
				},
			},
			wantValid: true,
		},
		{
			name: "approval step missing name",
			af: &WorkflowApprovalFlow{
				Required: true,
				Steps: []ApprovalStep{
					{Approvers: []string{"admin@co.com"}},
				},
			},
			wantValid:  false,
			wantErrors: []string{"name"},
		},
		{
			name: "approval step missing approvers",
			af: &WorkflowApprovalFlow{
				Required: true,
				Steps: []ApprovalStep{
					{Name: "Review"},
				},
			},
			wantValid:  false,
			wantErrors: []string{"approver"},
		},
		{
			name: "auto_approve_after too low",
			af: &WorkflowApprovalFlow{
				Required:         true,
				AutoApproveAfter: 100,
				Steps: []ApprovalStep{
					{Name: "Review", Approvers: []string{"admin@co.com"}},
				},
			},
			wantValid:  false,
			wantErrors: []string{"auto_approve_after"},
		},
		{
			name: "auto_approve_after at minimum",
			af: &WorkflowApprovalFlow{
				Required:         true,
				AutoApproveAfter: 3600,
				Steps: []ApprovalStep{
					{Name: "Review", Approvers: []string{"admin@co.com"}},
				},
			},
			wantValid: true,
		},
		{
			name: "too many approval steps",
			af: &WorkflowApprovalFlow{
				Required: true,
				Steps: func() []ApprovalStep {
					steps := make([]ApprovalStep, 11)
					for i := range steps {
						steps[i] = ApprovalStep{Name: "step", Approvers: []string{"a@b.com"}}
					}
					return steps
				}(),
			},
			wantValid:  false,
			wantErrors: []string{"10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMold(base(tt.af))
			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v; errors: %v", result.Valid, tt.wantValid, result.Errors)
			}
			for _, want := range tt.wantErrors {
				found := false
				for _, e := range result.Errors {
					if strings.Contains(strings.ToLower(e.Field+e.Message), strings.ToLower(want)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got: %v", want, result.Errors)
				}
			}
		})
	}
}

func TestValidateMold_Warnings(t *testing.T) {
	tests := []struct {
		name         string
		def          MoldDefinition
		wantWarnings []string
	}{
		{
			name: "missing displayName warns",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test"},
				Actions:  []WorkflowAction{{Action: "github.repo.create", Label: "Create"}},
			},
			wantWarnings: []string{"displayName"},
		},
		{
			name: "missing description warns",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test", DisplayName: "Test"},
				Actions:  []WorkflowAction{{Action: "github.repo.create", Label: "Create"}},
			},
			wantWarnings: []string{"description"},
		},
		{
			name: "missing category warns",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test", DisplayName: "Test", Description: "Desc"},
				Actions:  []WorkflowAction{{Action: "github.repo.create", Label: "Create"}},
			},
			wantWarnings: []string{"category"},
		},
		{
			name: "unknown action ID warns",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test", DisplayName: "T", Description: "D", Category: "C"},
				Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.issue.create"}},
			},
			wantWarnings: []string{"github.issue.create"},
		},
		{
			name: "known action ID no warning",
			def: MoldDefinition{
				Version:  "1.0.0",
				Metadata: WorkflowMetadata{Name: "test", DisplayName: "T", Description: "D", Category: "C"},
				Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
			},
			wantWarnings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMold(&tt.def)
			for _, want := range tt.wantWarnings {
				found := false
				for _, w := range result.Warnings {
					if strings.Contains(w.Field+w.Message, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning containing %q, got: %v", want, result.Warnings)
				}
			}
		})
	}
}

// ─── YAML parsing tests ────────────────────────────────────────────────────

func TestValidateMoldYAML_ValidMinimal(t *testing.T) {
	yaml := `
version: "1.0.0"
metadata:
  name: test-mold
  displayName: Test
  description: A test mold
  category: repository
steps:
  - id: s1
    name: Create repo
    action: github.repo.create
    inputs:
      name: "${{ inputs.name }}"
`
	result, err := ValidateMoldYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateMoldYAML_InvalidYAML(t *testing.T) {
	yaml := `
version: "1.0.0"
metadata:
  name: [broken
  not: valid: yaml
`
	_, err := ValidateMoldYAML([]byte(yaml))
	if err == nil {
		t.Error("expected parse error for invalid YAML")
	}
}

func TestValidateMoldYAML_EmptyContent(t *testing.T) {
	result, err := ValidateMoldYAML([]byte(""))
	if err != nil {
		// Either an error or invalid result is acceptable
		return
	}
	if result.Valid {
		t.Error("expected invalid for empty content")
	}
}

func TestValidateMoldYAML_FullMold(t *testing.T) {
	yaml := `
version: "1.0.0"

metadata:
  name: scaffold-go-service
  displayName: Scaffold Go HTTP Service
  description: Create a GitHub repository with a Go HTTP service scaffold
  author: platform-team
  icon: server
  category: service
  tags:
    - golang
    - microservice

inputs:
  type: object
  required:
    - name
    - owner
  properties:
    name:
      type: string
      title: Service Name
      pattern: "^[a-zA-Z0-9._-]+$"
    owner:
      type: string
      title: GitHub Organization
    goVersion:
      type: string
      title: Go Version
      enum: ["1.23", "1.24", "1.25"]
    private:
      type: boolean
      title: Private Repository

defaults:
  goVersion: "1.24"
  private: true

actions:
  - action: scaffold.go.service
    label: Scaffold Go Service
    primary: true

steps:
  - id: create-repo
    name: Create Repository
    action: github.repo.create
    inputs:
      owner: "${{ parameters.owner }}"
      name: "${{ parameters.name }}"
      private: "${{ parameters.private }}"

  - id: add-main
    name: Add main.go
    action: github.file.create
    depends_on: [create-repo]
    inputs:
      owner: "${{ parameters.owner }}"
      repo: "${{ parameters.name }}"
      path: main.go
      content: |
        package main

        import "fmt"

        func main() {
            fmt.Println("Hello")
        }

  - id: set-topics
    name: Set topics
    action: github.topics.set
    depends_on: [create-repo]
    inputs:
      owner: "${{ parameters.owner }}"
      repo: "${{ parameters.name }}"
      topics: [golang, http-service, shoehorn]

output:
  links:
    - title: View Repository
      url: "${{ steps.create-repo.output.html_url }}"
`
	result, err := ValidateMoldYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
	if len(result.Warnings) > 0 {
		t.Errorf("expected no warnings for complete mold, got: %v", result.Warnings)
	}
}

func TestValidateMoldYAML_WorkflowAdapterMold(t *testing.T) {
	yaml := `
version: "2.0.0"

metadata:
  name: create-repo-for-team
  displayName: Create Repository for Team
  description: Fetch team details and create a repo
  category: repository

actions:
  - action: repo.create.team
    label: Create Team Repository
    primary: true

steps:
  - id: fetch-team
    name: Fetch Team Details
    adapter: catalog
    config:
      operation: get_team

  - id: create-repo
    name: Create GitHub Repository
    adapter: http
    depends_on: [fetch-team]
    config:
      url: "https://api.github.com/orgs/test/repos"
      method: POST
`
	result, err := ValidateMoldYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

// ─── Metamorphic tests ──────────────────────────────────────────────────────

func TestValidateMold_Metamorphic_AddingOptionalFieldsPreservesValidity(t *testing.T) {
	// Metamorphic relation: adding optional metadata fields to a valid mold
	// should not make it invalid.
	minimal := &MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
	}
	minResult := ValidateMold(minimal)
	if !minResult.Valid {
		t.Fatal("minimal mold should be valid")
	}

	enriched := &MoldDefinition{
		Version: "1.0.0",
		Metadata: WorkflowMetadata{
			Name:        "test",
			DisplayName: "Test Mold",
			Description: "A test",
			Author:      "team",
			Icon:        "server",
			Category:    "service",
			Tags:        []string{"go", "http"},
		},
		Inputs:   map[string]any{"type": "object"},
		Defaults: map[string]any{"port": "8080"},
		Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
	}
	enrichedResult := ValidateMold(enriched)
	if !enrichedResult.Valid {
		t.Errorf("enriched mold should still be valid, got errors: %v", enrichedResult.Errors)
	}
}

func TestValidateMold_Metamorphic_AddingStepsPreservesValidity(t *testing.T) {
	// Adding more valid steps to a valid mold should keep it valid.
	one := &MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
	}
	if r := ValidateMold(one); !r.Valid {
		t.Fatal("one-step mold should be valid")
	}

	two := &MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps: []WorkflowStep{
			{ID: "s1", Name: "step1", Action: "github.repo.create"},
			{ID: "s2", Name: "step2", Action: "github.file.create"},
		},
	}
	if r := ValidateMold(two); !r.Valid {
		t.Errorf("two-step mold should be valid, got errors: %v", r.Errors)
	}
}

func TestValidateMold_Metamorphic_RemovingRequiredFieldsBreaksValidity(t *testing.T) {
	// Removing any required field from a valid mold should make it invalid.
	valid := MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps:    []WorkflowStep{{ID: "s1", Name: "step", Action: "github.repo.create"}},
	}
	if r := ValidateMold(&valid); !r.Valid {
		t.Fatal("base mold should be valid")
	}

	// Remove version
	noVersion := valid
	noVersion.Version = ""
	if r := ValidateMold(&noVersion); r.Valid {
		t.Error("removing version should make mold invalid")
	}

	// Remove metadata.name
	noName := valid
	noName.Metadata.Name = ""
	if r := ValidateMold(&noName); r.Valid {
		t.Error("removing metadata.name should make mold invalid")
	}

	// Remove steps
	noSteps := valid
	noSteps.Steps = nil
	if r := ValidateMold(&noSteps); r.Valid {
		t.Error("removing steps should make mold invalid")
	}
}

// ─── Edge cases ─────────────────────────────────────────────────────────────

func TestValidateMold_NilDefinition(t *testing.T) {
	result := ValidateMold(nil)
	if result.Valid {
		t.Error("nil definition should not be valid")
	}
}

func TestValidateMold_EmptySteps(t *testing.T) {
	def := &MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps:    []WorkflowStep{},
	}
	result := ValidateMold(def)
	if result.Valid {
		t.Error("empty steps array should be invalid")
	}
}

func TestValidateMold_ManySteps(t *testing.T) {
	steps := make([]WorkflowStep, 50)
	for i := range steps {
		steps[i] = WorkflowStep{
			ID:     strings.Replace("step-NN", "NN", strings.Repeat("x", i+1), 1),
			Name:   "Step",
			Action: "github.file.create",
		}
	}
	def := &MoldDefinition{
		Version:  "1.0.0",
		Metadata: WorkflowMetadata{Name: "test"},
		Steps:    steps,
	}
	result := ValidateMold(def)
	if !result.Valid {
		t.Errorf("50 valid steps should be valid, got errors: %v", result.Errors)
	}
}
