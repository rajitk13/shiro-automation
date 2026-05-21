package workflow

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ConditionEvaluator evaluates conditions against execution context
type ConditionEvaluator struct{}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

// Evaluate evaluates a condition string against the execution context
// Supports template syntax like: {{steps.approval.status == 'approved'}}
func (e *ConditionEvaluator) Evaluate(condition string, ctx *ExecutionContext) (bool, error) {
	if condition == "" {
		return true, nil // No condition means always execute
	}

	// Parse template syntax
	condition = strings.TrimSpace(condition)

	// Check for template syntax
	if strings.HasPrefix(condition, "{{") && strings.HasSuffix(condition, "}}") {
		// Extract the expression inside {{ }}
		expression := strings.TrimSpace(condition[2 : len(condition)-2])
		return e.evaluateExpression(expression, ctx)
	}

	// Simple boolean evaluation
	return e.evaluateBoolean(condition, ctx)
}

// evaluateExpression evaluates a template expression
func (e *ConditionEvaluator) evaluateExpression(expr string, ctx *ExecutionContext) (bool, error) {
	// Parse simple comparisons: steps.step_id.field == 'value'
	// Examples:
	// - steps.approval.status == 'approved'
	// - steps.approval.status != 'rejected'
	// - steps.ai_initial.success == true

	// Split by comparison operators
	operators := []string{"==", "!=", ">", "<", ">=", "<="}
	var op string
	var parts []string

	for _, operator := range operators {
		if strings.Contains(expr, operator) {
			op = operator
			parts = strings.SplitN(expr, operator, 2)
			break
		}
	}

	if len(parts) != 2 {
		return false, fmt.Errorf("invalid condition expression: %s", expr)
	}

	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])

	// Evaluate left side
	leftValue, err := e.evaluateValue(left, ctx)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate left side: %w", err)
	}

	// Evaluate right side
	rightValue, err := e.evaluateValue(right, ctx)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate right side: %w", err)
	}

	// Perform comparison
	return e.compareValues(leftValue, rightValue, op)
}

// evaluateValue evaluates a value reference (e.g., steps.approval.status)
func (e *ConditionEvaluator) evaluateValue(expr string, ctx *ExecutionContext) (interface{}, error) {
	// Handle string literals
	if strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") {
		return strings.Trim(expr, "'"), nil
	}

	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return strings.Trim(expr, "\""), nil
	}

	// Handle boolean literals
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	// Handle numeric literals
	if num, err := strconv.Atoi(expr); err == nil {
		return num, nil
	}

	// Handle step references: steps.step_id.field
	parts := strings.Split(expr, ".")
	if len(parts) >= 2 && parts[0] == "steps" {
		stepID := parts[1]
		field := ""
		if len(parts) > 2 {
			field = parts[2]
		}

		return e.getStepValue(stepID, field, ctx)
	}

	return nil, fmt.Errorf("cannot evaluate value: %s", expr)
}

// getStepValue retrieves a value from a step's output
func (e *ConditionEvaluator) getStepValue(stepID, field string, ctx *ExecutionContext) (interface{}, error) {
	stepResult, exists := ctx.Steps[stepID]
	if !exists {
		return nil, fmt.Errorf("step %s not found in execution context", stepID)
	}

	if field == "" || field == "success" {
		return stepResult.Success, nil
	}

	if field == "error" {
		return stepResult.Error, nil
	}

	// Get field from output
	if stepResult.Output != nil {
		if value, exists := stepResult.Output[field]; exists {
			return value, nil
		}
	}

	return nil, fmt.Errorf("field %s not found in step %s output", field, stepID)
}

// compareValues compares two values using the specified operator
func (e *ConditionEvaluator) compareValues(left, right interface{}, op string) (bool, error) {
	switch op {
	case "==":
		return reflect.DeepEqual(left, right), nil
	case "!=":
		return !reflect.DeepEqual(left, right), nil
	case ">":
		return e.compareNumbers(left, right, func(a, b float64) bool { return a > b })
	case "<":
		return e.compareNumbers(left, right, func(a, b float64) bool { return a < b })
	case ">=":
		return e.compareNumbers(left, right, func(a, b float64) bool { return a >= b })
	case "<=":
		return e.compareNumbers(left, right, func(a, b float64) bool { return a <= b })
	default:
		return false, fmt.Errorf("unsupported operator: %s", op)
	}
}

// compareNumbers compares two numeric values
func (e *ConditionEvaluator) compareNumbers(left, right interface{}, compare func(float64, float64) bool) (bool, error) {
	leftFloat, err := toFloat64(left)
	if err != nil {
		return false, fmt.Errorf("left side is not a number: %w", err)
	}

	rightFloat, err := toFloat64(right)
	if err != nil {
		return false, fmt.Errorf("right side is not a number: %w", err)
	}

	return compare(leftFloat, rightFloat), nil
}

// toFloat64 converts a value to float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

// evaluateBoolean evaluates a simple boolean condition
func (e *ConditionEvaluator) evaluateBoolean(condition string, ctx *ExecutionContext) (bool, error) {
	// This is a placeholder for more complex boolean logic
	// For now, we'll delegate to expression evaluation
	return e.evaluateExpression(condition, ctx)
}
