package runtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/errors"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// Executor handles workflow execution
type Executor struct {
	registry   *modules.Registry
	logger     *log.Logger
	stateStore state.StateStore
}

// NewExecutor creates a new workflow executor
func NewExecutor(registry *modules.Registry, logger *log.Logger) *Executor {
	return &Executor{
		registry: registry,
		logger:   logger,
	}
}

// SetStateStore sets the state store for resumption
func (e *Executor) SetStateStore(store state.StateStore) {
	e.stateStore = store
}

// Execute runs a workflow with the given inputs
func (e *Executor) Execute(
	ctx context.Context,
	wf *workflow.Workflow,
	inputs map[string]interface{},
	env map[string]string,
) (*workflow.ExecutionContext, error) {
	// Validate workflow
	if err := wf.Validate(); err != nil {
		return nil, errors.NewWorkflowError(wf.Name, "", "workflow validation failed", err)
	}

	// Create execution context
	execCtx := workflow.NewExecutionContext()
	execCtx.Inputs = inputs
	execCtx.Env = env

	// Try to load previous state for resumption
	if e.stateStore != nil {
		exists, err := e.stateStore.Exists(ctx, wf.Name)
		if err == nil && exists {
			previousCtx := workflow.NewExecutionContext()
			if err := e.stateStore.Load(ctx, wf.Name, previousCtx); err == nil {
				e.logger.Printf("Resuming workflow: %s from previous state", wf.Name)
				execCtx = previousCtx
				execCtx.Inputs = inputs // Update inputs with current values
				execCtx.Env = env       // Update env with current values
			}
		}
	}

	e.logger.Printf("Starting workflow: %s", wf.Name)

	// Build execution graph
	graph, err := buildExecutionGraph(wf.Steps)
	if err != nil {
		return nil, errors.NewWorkflowError(wf.Name, "", "failed to build execution graph", err)
	}

	// Execute steps in topological order
	for _, stepID := range graph.topologicalOrder() {
		step := wf.GetStepByID(stepID)
		if step == nil {
			return nil, errors.NewWorkflowError(wf.Name, stepID, "step not found", nil)
		}

		// Skip if already executed (based on state)
		if _, exists := execCtx.Steps[stepID]; exists {
			e.logger.Printf("Step %s already executed, skipping", stepID)
			continue
		}

		// Check if dependencies are satisfied
		if !graph.dependenciesSatisfied(stepID, execCtx.Steps) {
			return nil, errors.NewWorkflowError(wf.Name, stepID, "dependencies not satisfied", nil)
		}

		// Execute the step
		result, err := e.executeStep(ctx, execCtx, *step)
		if err != nil {
			e.logger.Printf("Step %s failed: %v", stepID, err)
			return execCtx, errors.NewWorkflowError(wf.Name, stepID, "step execution failed", err)
		}

		execCtx.Steps[stepID] = *result
		if !result.Success {
			e.logger.Printf("Step %s failed: %s", stepID, result.Error)
		} else {
			e.logger.Printf("Step %s completed", stepID)
		}

		// Save state after each step
		if e.stateStore != nil {
			if err := e.stateStore.Save(ctx, wf.Name, execCtx); err != nil {
				e.logger.Printf("Failed to save state after step %s: %v", stepID, err)
			}
		}

		// Pause after this step if specified
		if step.Pause {
			e.logger.Printf("Workflow paused after step %s", stepID)
			return execCtx, nil
		}
	}

	e.logger.Printf("Workflow %s completed successfully", wf.Name)
	return execCtx, nil
}

// executeStep executes a single step with retry logic
func (e *Executor) executeStep(
	ctx context.Context,
	execCtx *workflow.ExecutionContext,
	step workflow.Step,
) (*workflow.StepResult, error) {
	// Resolve variables in step config
	resolver := workflow.NewVariableResolver(execCtx)
	resolvedConfig, err := resolver.Resolve(step.Config)
	if err != nil {
		return nil, errors.NewWorkflowError("", step.ID, "failed to resolve variables", err)
	}

	// Convert to map[string]interface{}
	config, ok := resolvedConfig.(map[string]interface{})
	if !ok {
		return nil, errors.NewWorkflowError("", step.ID, "resolved config is not a map", nil)
	}

	// Get module
	module, err := e.registry.Get(step.Type)
	if err != nil {
		return &workflow.StepResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Set timeout if specified
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(step.Timeout)*time.Second)
		defer cancel()
	}

	// Execute with retry logic
	var output map[string]interface{}
	var execErr error

	if step.Retry != nil {
		output, execErr = e.executeWithRetry(ctx, module, execCtx, step, config, step.Retry)
	} else {
		output, execErr = module.Run(ctx, execCtx, step)
	}

	result := &workflow.StepResult{
		Success: execErr == nil,
		Output:  output,
	}

	if execErr != nil {
		result.Error = execErr.Error()
	}

	return result, nil
}

// executeWithRetry executes a step with retry logic
func (e *Executor) executeWithRetry(
	ctx context.Context,
	module modules.Module,
	execCtx *workflow.ExecutionContext,
	step workflow.Step,
	_ map[string]interface{},
	retryConfig *workflow.RetryConfig,
) (map[string]interface{}, error) {
	maxAttempts := retryConfig.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	delay := time.Duration(retryConfig.Delay) * time.Second
	backoff := retryConfig.Backoff
	if backoff <= 0 {
		backoff = 2.0
	}

	maxDelay := time.Duration(retryConfig.MaxDelay) * time.Second
	if maxDelay <= 0 {
		maxDelay = 5 * time.Minute
	}

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		output, err := module.Run(ctx, execCtx, step)
		if err == nil {
			return output, nil
		}

		lastErr = err
		e.logger.Printf("Step %s attempt %d failed: %v", step.ID, attempt, err)

		if attempt < maxAttempts {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * backoff)
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}

	return nil, errors.NewWorkflowError("", step.ID, fmt.Sprintf("after %d attempts", maxAttempts), lastErr)
}
