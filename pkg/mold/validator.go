package mold

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Valid action providers (first segment of dot-separated action ID).
var validProviders = map[string]bool{
	"github":     true,
	"deployment": true,
	"system":     true,
	"catalog":    true,
	"repo":       true,
}

// Valid adapter names for workflow steps.
var validAdapters = map[string]bool{
	"http": true, "postgres": true, "slack": true, "github": true,
	"docker": true, "kubernetes": true, "terraform": true, "webhook": true,
	"email": true, "s3": true, "gcs": true, "file": true,
	"git": true, "catalog": true, "log": true,
}

// Known built-in action IDs. Unknown actions produce warnings, not errors.
var knownActions = map[string]bool{
	"github.repo.create":      true,
	"github.repo.update":      true,
	"github.file.create":      true,
	"github.template.apply":   true,
	"github.topics.set":       true,
	"github.pr.create":        true,
	"github.team.add":         true,
	"github.collaborator.add": true,
	"catalog.entity.register": true,
}

const (
	maxApprovalSteps    = 10
	maxApproversPerStep = 50
	minAutoApprove      = 3600
)

// ValidateMold validates a parsed MoldDefinition and returns structured results.
func ValidateMold(def *MoldDefinition) *ValidationResult {
	r := &ValidationResult{Valid: true}

	if def == nil {
		r.addError("", "mold definition is nil")
		return r
	}

	// Required top-level fields
	if def.Version == "" {
		r.addError("version", "is required")
	}
	if def.Metadata.Name == "" {
		r.addError("metadata.name", "is required")
	}
	if len(def.Steps) == 0 && len(def.Actions) == 0 {
		r.addError("steps", "at least one of steps or actions is required")
	}

	// Validate steps
	seenIDs := map[string]bool{}
	for i, step := range def.Steps {
		prefix := fmt.Sprintf("steps[%d]", i)
		validateStep(r, step, prefix, seenIDs)
	}

	// Validate rollback steps
	if def.Rollback != nil && def.Rollback.Enabled {
		for i, step := range def.Rollback.Steps {
			prefix := fmt.Sprintf("rollback.steps[%d]", i)
			validateStep(r, step, prefix, seenIDs)
		}
	}

	// Validate approval flow
	if def.ApprovalFlow != nil {
		validateApprovalFlow(r, def.ApprovalFlow)
	}

	// Warnings for recommended fields
	if def.Metadata.DisplayName == "" {
		r.addWarning("metadata.displayName", "is recommended for UI presentation")
	}
	if def.Metadata.Description == "" {
		r.addWarning("metadata.description", "is recommended for discoverability")
	}
	if def.Metadata.Category == "" {
		r.addWarning("metadata.category", "is recommended for categorization")
	}

	// Warn on unknown action IDs in steps
	for _, step := range def.Steps {
		if step.Action != "" && !knownActions[step.Action] {
			parts := strings.SplitN(step.Action, ".", 2)
			if len(parts) >= 2 && validProviders[parts[0]] {
				r.addWarning("steps."+step.ID+".action",
					fmt.Sprintf("unknown action %q (may be a custom action)", step.Action))
			}
		}
	}

	return r
}

func validateStep(r *ValidationResult, step WorkflowStep, prefix string, seenIDs map[string]bool) {
	if step.ID == "" {
		r.addError(prefix+".id", "is required")
	} else {
		if seenIDs[step.ID] {
			r.addError(prefix+".id", fmt.Sprintf("duplicate step id %q", step.ID))
		}
		seenIDs[step.ID] = true
	}

	if step.Name == "" {
		r.addError(prefix+".name", "is required")
	}

	hasAction := step.Action != ""
	hasAdapter := step.Adapter != ""

	if !hasAction && !hasAdapter {
		r.addError(prefix, "must have either action or adapter field")
	}
	if hasAction && hasAdapter {
		r.addError(prefix, "must have either action or adapter, not both")
	}

	if hasAction {
		validateActionID(r, step.Action, prefix+".action")
	}
	if hasAdapter {
		if !validAdapters[step.Adapter] {
			r.addError(prefix+".adapter",
				fmt.Sprintf("unknown adapter %q", step.Adapter))
		}
	}
}

func validateActionID(r *ValidationResult, action string, field string) {
	parts := strings.Split(action, ".")
	if len(parts) < 2 {
		r.addError(field, fmt.Sprintf("action %q must have at least 2 dot-separated parts (e.g. github.repo.create)", action))
		return
	}
	provider := parts[0]
	if !validProviders[provider] {
		r.addError(field, fmt.Sprintf("unknown provider %q in action %q (valid: %s)", provider, action, validProviderList()))
		return
	}
}

func validateApprovalFlow(r *ValidationResult, af *WorkflowApprovalFlow) {
	if af.AutoApproveAfter > 0 && af.AutoApproveAfter < minAutoApprove {
		r.addError("approvalFlow.auto_approve_after",
			fmt.Sprintf("must be at least %d seconds (1 hour), got %d", minAutoApprove, af.AutoApproveAfter))
	}
	if len(af.Steps) > maxApprovalSteps {
		r.addError("approvalFlow.steps",
			fmt.Sprintf("maximum %d approval steps allowed, got %d", maxApprovalSteps, len(af.Steps)))
	}
	for i, step := range af.Steps {
		prefix := fmt.Sprintf("approvalFlow.steps[%d]", i)
		if step.Name == "" {
			r.addError(prefix+".name", "is required")
		}
		if len(step.Approvers) == 0 {
			r.addError(prefix+".approvers", "at least one approver is required")
		}
		if len(step.Approvers) > maxApproversPerStep {
			r.addError(prefix+".approvers",
				fmt.Sprintf("maximum %d approvers per step, got %d", maxApproversPerStep, len(step.Approvers)))
		}
	}
}

func validProviderList() string {
	providers := make([]string, 0, len(validProviders))
	for p := range validProviders {
		providers = append(providers, p)
	}
	return strings.Join(providers, ", ")
}

// ValidateMoldYAML parses YAML content and validates the mold definition.
func ValidateMoldYAML(content []byte) (*ValidationResult, error) {
	if len(content) == 0 {
		r := &ValidationResult{Valid: false}
		r.addError("", "empty content")
		return r, nil
	}

	var def MoldDefinition
	if err := yaml.Unmarshal(content, &def); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	return ValidateMold(&def), nil
}

func (r *ValidationResult) addError(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{Field: field, Message: message})
}

func (r *ValidationResult) addWarning(field, message string) {
	r.Warnings = append(r.Warnings, ValidationError{Field: field, Message: message})
}
