package runtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// Executor handles workflow execution
type Executor struct {
	registry *modules.Registry
	logger   *log.Logger
}

// NewExecutor creates a new workflow executor
func NewExecutor(registry *modules.Registry, logger *log.Logger) *Executor {
	return &Executor{
		registry: registry,
		logger:   logger,
	}
}

// Execute runs a workflow with the given inputs
func (e *Executor) Execute(ctx context.Context, wf *workflow.Workflow, inputs map[string]interface{}, env map[string]string) (*workflow.ExecutionContext, error) {
	// Validate workflow
	if err := wf.Validate(); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create execution context
	execCtx := workflow.NewExecutionContext()
	execCtx.Inputs = inputs
	execCtx.Env = env

	e.logger.Printf("Starting workflow: %s", wf.Name)

	// Build execution graph
	graph, err := buildExecutionGraph(wf.Steps)
	if err != nil {
		return nil, fmt.Errorf("failed to build execution graph: %w", err)
	}

	// Execute steps in topological order
	for _, stepID := range graph.topologicalOrder() {
		step := wf.GetStepByID(stepID)
		if step == nil {
			return nil, fmt.Errorf("step %s not found", stepID)
		}

		// Check if dependencies are satisfied
		if !graph.dependenciesSatisfied(stepID, execCtx.Steps) {
			return nil, fmt.Errorf("dependencies not satisfied for step %s", stepID)
		}

		// Execute the step
		result, err := e.executeStep(ctx, execCtx, *step)
		if err != nil {
			return execCtx, fmt.Errorf("step %s failed: %w", stepID, err)
		}

		execCtx.Steps[stepID] = *result
		e.logger.Printf("Step %s completed: %v", stepID, result.Success)
	}

	e.logger.Printf("Workflow %s completed successfully", wf.Name)
	return execCtx, nil
}

// executeStep executes a single step with retry logic
func (e *Executor) executeStep(ctx context.Context, execCtx *workflow.ExecutionContext, step workflow.Step) (*workflow.StepResult, error) {
	// Resolve variables in step config
	resolver := workflow.NewVariableResolver(execCtx)
	resolvedConfig, err := resolver.Resolve(step.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve variables: %w", err)
	}

	// Convert to map[string]interface{}
	config, ok := resolvedConfig.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("resolved config is not a map")
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
func (e *Executor) executeWithRetry(ctx context.Context, module modules.Module, execCtx *workflow.ExecutionContext, step workflow.Step, _ map[string]interface{}, retryConfig *workflow.RetryConfig) (map[string]interface{}, error) {
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

	return nil, fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}
