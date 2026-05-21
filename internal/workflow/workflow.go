package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/errors"
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
	Condition string                 `json:"condition,omitempty"`
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
	Inputs     map[string]interface{}
	Steps      map[string]StepResult
	Memory     map[string]interface{}
	Env        map[string]string
	Completed  map[string]bool   // Track completed steps for resumption
	StepStatus map[string]string // Track individual step statuses
}

// NewExecutionContext creates a new execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		Inputs:     make(map[string]interface{}),
		Steps:      make(map[string]StepResult),
		Memory:     make(map[string]interface{}),
		Env:        make(map[string]string),
		Completed:  make(map[string]bool),
		StepStatus: make(map[string]string),
	}
}

// LoadWorkflow loads a workflow from JSON bytes
func LoadWorkflow(data []byte) (*Workflow, error) {
	var wf Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, errors.NewValidationError("workflow", "failed to parse workflow JSON", err)
	}

	// Resolve environment variables in workflow
	resolveEnvVars(&wf)

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

// resolveEnvVars resolves {{env.VARIABLE}} templates in workflow
func resolveEnvVars(wf *Workflow) {
	// Resolve in inputs
	for key, value := range wf.Inputs {
		if strValue, ok := value.(string); ok {
			wf.Inputs[key] = resolveEnvVarString(strValue)
		}
	}

	// Resolve in step configs
	for _, step := range wf.Steps {
		if step.Config != nil {
			for key, value := range step.Config {
				if strValue, ok := value.(string); ok {
					step.Config[key] = resolveEnvVarString(strValue)
				}
			}
		}
	}
}

// resolveEnvVarString resolves a single {{env.VARIABLE}} template
func resolveEnvVarString(input string) string {
	if strings.HasPrefix(input, "{{env.") && strings.HasSuffix(input, "}}") {
		envVar := strings.TrimPrefix(input, "{{env.")
		envVar = strings.TrimSuffix(envVar, "}}")
		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
	}
	return input
}
