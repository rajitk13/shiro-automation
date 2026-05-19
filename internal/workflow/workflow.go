package workflow

import (
	"encoding/json"
	"fmt"
)

// Workflow represents a complete workflow definition
type Workflow struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Inputs      map[string]interface{} `json:"inputs,omitempty"`
	Steps       []Step                 `json:"steps"`
}

// Step represents a single step in the workflow
type Step struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Config    map[string]interface{} `json:"config,omitempty"`
	DependsOn []string               `json:"depends_on,omitempty"`
	Retry     *RetryConfig           `json:"retry,omitempty"`
	Timeout   int                    `json:"timeout,omitempty"` // seconds
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
	Inputs map[string]interface{}
	Steps  map[string]StepResult
	Memory map[string]interface{}
	Env    map[string]string
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		Inputs: make(map[string]interface{}),
		Steps:  make(map[string]StepResult),
		Memory: make(map[string]interface{}),
		Env:    make(map[string]string),
	}
}

// LoadWorkflow loads a workflow from JSON bytes
func LoadWorkflow(data []byte) (*Workflow, error) {
	var wf Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
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
		return fmt.Errorf("workflow name is required")
	}

	if len(w.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	stepIDs := make(map[string]bool)
	for i, step := range w.Steps {
		if step.ID == "" {
			return fmt.Errorf("step %d: ID is required", i)
		}
		if step.Type == "" {
			return fmt.Errorf("step %s: type is required", step.ID)
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true

		// Validate dependencies
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				return fmt.Errorf("step %s: depends on non-existent step %s", step.ID, dep)
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
