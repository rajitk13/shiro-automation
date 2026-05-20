package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/rkuthiala/shiro-automation/internal/errors"
)

// Workflow represents a complete workflow definition
type Workflow struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Steps       []Step                 `json:"steps"`
}

// ApprovalStatus represents the status of an approval
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
	ApprovalTimedOut ApprovalStatus = "timed_out"
)

// ApprovalConfig configures an approval step
type ApprovalConfig struct {
	ApprovalID     string   `json:"approval_id"`     // Custom ID for tracking
	ApprovalMethod string   `json:"approval_method"` // slack, webhook, webui, cli
	Timeout        int      `json:"timeout"`         // Timeout in seconds (default 86400 = 24 hours)
	Approvers      []string `json:"approvers"`       // List of approver IDs
	RequiredCount  int      `json:"required_count"`  // Number of approvals needed (default 1)
	Message        string   `json:"message"`         // Approval message
}

// ApprovalState tracks the state of an approval
type ApprovalState struct {
	ApprovalID   string                    `json:"approval_id"`
	Status       ApprovalStatus            `json:"status"`
	Approvals    map[string]ApprovalRecord `json:"approvals"`     // approver_id -> record
	CreatedAt    int64                     `json:"created_at"`    // Unix timestamp
	ExpiresAt    int64                     `json:"expires_at"`    // Unix timestamp
	DecisionData map[string]interface{}    `json:"decision_data"` // Data from approval
}

// ApprovalRecord records a single approval decision
type ApprovalRecord struct {
	ApproverID string `json:"approver_id"`
	Decision   string `json:"decision"` // approved, rejected
	Reason     string `json:"reason,omitempty"`
	Timestamp  int64  `json:"timestamp"` // Unix timestamp
}

// Step represents a single step in the workflow
type Step struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config,omitempty"`
	DependsOn []string               `json:"depends_on,omitempty"`
	Retry     *RetryConfig           `json:"retry,omitempty"`
	Timeout   int                    `json:"timeout,omitempty"`  // seconds
	Approval  *ApprovalConfig        `json:"approval,omitempty"` // Approval configuration
	When      string                 `json:"when,omitempty"`     // Conditional execution
}

// RetryConfig defines retry behavior for a step
type RetryConfig struct {
	MaxAttempts int     `json:"max_attempts"`
	Delay       int     `json:"delay"` // seconds
	Backoff     float64 `json:"backoff"`
	MaxDelay    int     `json:"max_delay"` // seconds
}

// StepResult represents the output of a step execution
type StepResult struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output"`
	Error   string                 `json:"error,omitempty"`
}

// ExecutionContext holds the shared state during workflow execution
type ExecutionContext struct {
	Inputs        map[string]interface{}
	Steps         map[string]StepResult
	Memory        map[string]interface{}
	Env           map[string]string
	Paused        bool                     `json:"paused"`
	CurrentStepID string                   `json:"current_step_id,omitempty"`
	PausedAt      int64                    `json:"paused_at,omitempty"`
	Approvals     map[string]ApprovalState `json:"approvals"` // approval_id -> state
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		Inputs:    make(map[string]interface{}),
		Steps:     make(map[string]StepResult),
		Memory:    make(map[string]interface{}),
		Env:       make(map[string]string),
		Approvals: make(map[string]ApprovalState),
	}
}

// LoadWorkflow loads a workflow from JSON bytes
func LoadWorkflow(data []byte) (*Workflow, error) {
	var wf Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, errors.NewValidationError("workflow", "failed to parse workflow JSON", err)
	}
	return &wf, nil
}

// LoadWorkflowFromFile loads a workflow from a file path
func LoadWorkflowFromFile(path string) (*Workflow, error) {
	// This will be implemented when we add file I/O
	return nil, fmt.Errorf("not implemented")
}

// Validate performs basic validation on the workflow
func (w *Workflow) Validate() error {
	if w.Name == "" {
		return errors.NewValidationError("name", "workflow name is required", nil)
	}

	if len(w.Steps) == 0 {
		return errors.NewValidationError("steps", "workflow must have at least one step", nil)
	}

	stepIDs := make(map[string]bool)
	for i, step := range w.Steps {
		if step.ID == "" {
			return errors.NewValidationError(fmt.Sprintf("steps[%d].id", i), "step ID is required", nil)
		}
		if step.Type == "" {
			return errors.NewValidationError(fmt.Sprintf("steps[%s].type", step.ID), "step type is required", nil)
		}
		if stepIDs[step.ID] {
			return errors.NewValidationError(fmt.Sprintf("steps[%s].id", step.ID), "duplicate step ID", nil)
		}
		stepIDs[step.ID] = true

		// Validate dependencies
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				return errors.NewValidationError(fmt.Sprintf("steps[%s].depends_on", step.ID), fmt.Sprintf("depends on non-existent step %s", dep), nil)
			}
		}
	}

	return nil
}

// GetStepByID retrieves a step by its ID
func (w *Workflow) GetStepByID(id string) *Step {
	for i := range w.Steps {
		if w.Steps[i].ID == id {
			return &w.Steps[i]
		}
	}
	return nil
}
