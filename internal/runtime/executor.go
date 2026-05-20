package runtime

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/errors"
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

		// Check if dependencies are satisfied
		if !graph.dependenciesSatisfied(stepID, execCtx.Steps) {
			return nil, errors.NewWorkflowError(wf.Name, stepID, "dependencies not satisfied", nil)
		}

		// Check conditional execution
		if step.When != "" {
			shouldExecute, err := e.evaluateCondition(step.When, execCtx)
			if err != nil {
				e.logger.Printf("Failed to evaluate condition for step %s: %v", stepID, err)
				return nil, errors.NewWorkflowError(wf.Name, stepID, "failed to evaluate condition", err)
			}
			if !shouldExecute {
				e.logger.Printf("Skipping step %s (condition not met)", stepID)
				continue
			}
		}

		// Check if this is an approval step
		if step.Approval != nil {
			return e.handleApprovalStep(ctx, execCtx, wf, *step, graph)
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
			e.logger.Printf("Step %s completed: %v", stepID, result.Success)
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

// evaluateCondition evaluates a conditional expression
func (e *Executor) evaluateCondition(when string, execCtx *workflow.ExecutionContext) (bool, error) {
	// Simple implementation: check if the condition is true based on approval status
	// In a real implementation, this would use a proper expression evaluator
	resolver := workflow.NewVariableResolver(execCtx)

	// Try to resolve the condition
	resolved, err := resolver.Resolve(when)
	if err != nil {
		return false, err
	}

	// Check if the resolved value is truthy
	if boolVal, ok := resolved.(bool); ok {
		return boolVal, nil
	}

	// Default to false if not a boolean
	return false, nil
}

// handleApprovalStep handles an approval step
func (e *Executor) handleApprovalStep(
	ctx context.Context,
	execCtx *workflow.ExecutionContext,
	wf *workflow.Workflow,
	step workflow.Step,
	graph *executionGraph,
) (*workflow.ExecutionContext, error) {
	approvalConfig := step.Approval

	e.logger.Printf("Approval step %s: %s", step.ID, approvalConfig.ApprovalID)

	// Check if approval already exists in context
	if existingState, exists := execCtx.Approvals[approvalConfig.ApprovalID]; exists {
		// Check if approval has expired
		if time.Now().Unix() > existingState.ExpiresAt {
			e.logger.Printf("Approval %s has expired", approvalConfig.ApprovalID)
			existingState.Status = workflow.ApprovalTimedOut
			execCtx.Approvals[approvalConfig.ApprovalID] = existingState

			// Mark step as failed due to timeout
			execCtx.Steps[step.ID] = workflow.StepResult{
				Success: false,
				Error:   "approval timed out",
			}
			return execCtx, nil
		}

		// Check approval status
		if existingState.Status == workflow.ApprovalApproved {
			e.logger.Printf("Approval %s already approved", approvalConfig.ApprovalID)
			execCtx.Steps[step.ID] = workflow.StepResult{
				Success: true,
				Output:  existingState.DecisionData,
			}
			return execCtx, nil
		} else if existingState.Status == workflow.ApprovalRejected {
			e.logger.Printf("Approval %s was rejected", approvalConfig.ApprovalID)
			execCtx.Steps[step.ID] = workflow.StepResult{
				Success: false,
				Error:   "approval was rejected",
			}
			return execCtx, nil
		}

		// Still pending, continue with pause logic
	}

	// Create approval state
	timeout := approvalConfig.Timeout
	if timeout == 0 {
		timeout = 86400 // Default 24 hours
	}

	approvalState := workflow.ApprovalState{
		ApprovalID: approvalConfig.ApprovalID,
		Status:     workflow.ApprovalPending,
		Approvals:  make(map[string]workflow.ApprovalRecord),
		CreatedAt:  time.Now().Unix(),
		ExpiresAt:  time.Now().Unix() + int64(timeout),
	}

	// Store approval state in execution context
	if execCtx.Approvals == nil {
		execCtx.Approvals = make(map[string]workflow.ApprovalState)
	}
	execCtx.Approvals[approvalConfig.ApprovalID] = approvalState

	// Mark workflow as paused
	execCtx.Paused = true
	execCtx.CurrentStepID = step.ID
	execCtx.PausedAt = time.Now().Unix()

	e.logger.Printf("Workflow paused waiting for approval: %s (expires in %d seconds)", approvalConfig.ApprovalID, timeout)

	// In a real implementation, this would:
	// 1. Send approval notification based on method (slack, webhook, etc.)
	// 2. Save state to state store
	// 3. Exit execution

	// For now, return the paused context
	return execCtx, nil
}
