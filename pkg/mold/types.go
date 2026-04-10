package mold

// MoldDefinition represents a workflow/mold definition parsed from YAML/JSON.
type MoldDefinition struct {
	Version      string                `yaml:"version" json:"version"`
	Metadata     WorkflowMetadata      `yaml:"metadata" json:"metadata"`
	Actions      []WorkflowAction      `yaml:"actions,omitempty" json:"actions,omitempty"`
	Inputs       map[string]any        `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Defaults     map[string]any        `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Steps        []WorkflowStep        `yaml:"steps,omitempty" json:"steps,omitempty"`
	Output       map[string]any        `yaml:"output,omitempty" json:"output,omitempty"`
	Outputs      map[string]any        `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Rollback     *WorkflowRollback     `yaml:"rollback,omitempty" json:"rollback,omitempty"`
	ApprovalFlow *WorkflowApprovalFlow `yaml:"approvalFlow,omitempty" json:"approvalFlow,omitempty"`
}

// WorkflowMetadata contains mold identification and categorization.
type WorkflowMetadata struct {
	Name        string   `yaml:"name" json:"name"`
	DisplayName string   `yaml:"displayName,omitempty" json:"displayName,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Author      string   `yaml:"author,omitempty" json:"author,omitempty"`
	Icon        string   `yaml:"icon,omitempty" json:"icon,omitempty"`
	Category    string   `yaml:"category,omitempty" json:"category,omitempty"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// WorkflowAction represents an action that can be triggered on a mold.
type WorkflowAction struct {
	Action      string `yaml:"action" json:"action"`
	Label       string `yaml:"label" json:"label"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Primary     bool   `yaml:"primary,omitempty" json:"primary,omitempty"`
}

// WorkflowStep represents a single step in a mold workflow.
type WorkflowStep struct {
	ID              string               `yaml:"id" json:"id"`
	Name            string               `yaml:"name" json:"name"`
	Action          string               `yaml:"action,omitempty" json:"action,omitempty"`
	Adapter         string               `yaml:"adapter,omitempty" json:"adapter,omitempty"`
	Inputs          map[string]any       `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Config          map[string]any       `yaml:"config,omitempty" json:"config,omitempty"`
	DependsOn       []string             `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Condition       string               `yaml:"condition,omitempty" json:"condition,omitempty"`
	When            string               `yaml:"when,omitempty" json:"when,omitempty"`
	Timeout         string               `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retry           *WorkflowRetryConfig `yaml:"retry,omitempty" json:"retry,omitempty"`
	ContinueOnError any                  `yaml:"continue_on_error,omitempty" json:"continue_on_error,omitempty"`
	ForEach         any                  `yaml:"for_each,omitempty" json:"for_each,omitempty"`
	ItemVar         string               `yaml:"item_var,omitempty" json:"item_var,omitempty"`
	Outputs         map[string]any       `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

// WorkflowRetryConfig configures retry behavior for a step.
type WorkflowRetryConfig struct {
	MaxAttempts int    `yaml:"max_attempts" json:"max_attempts"`
	Delay       string `yaml:"delay,omitempty" json:"delay,omitempty"`
}

// WorkflowRollback defines rollback behavior.
type WorkflowRollback struct {
	Enabled bool           `yaml:"enabled" json:"enabled"`
	Steps   []WorkflowStep `yaml:"steps,omitempty" json:"steps,omitempty"`
}

// WorkflowApprovalFlow defines multi-step approval requirements.
type WorkflowApprovalFlow struct {
	Required         bool           `yaml:"required" json:"required"`
	AutoApproveAfter int            `yaml:"auto_approve_after,omitempty" json:"auto_approve_after,omitempty"`
	Steps            []ApprovalStep `yaml:"steps,omitempty" json:"steps,omitempty"`
}

// ApprovalStep defines a single approval step.
type ApprovalStep struct {
	Name          string   `yaml:"name" json:"name"`
	Description   string   `yaml:"description,omitempty" json:"description,omitempty"`
	Approvers     []string `yaml:"approvers" json:"approvers"`
	RequiredCount int      `yaml:"required_count,omitempty" json:"required_count,omitempty"`
}

// ValidationResult holds the outcome of validating a mold definition.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// ValidationError represents a single validation finding.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
