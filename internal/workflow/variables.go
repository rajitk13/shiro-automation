package workflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// VariableResolver handles template variable substitution
type VariableResolver struct {
	ctx *ExecutionContext
}

// NewVariableResolver creates a new variable resolver
func NewVariableResolver(ctx *ExecutionContext) *VariableResolver {
	return &VariableResolver{ctx: ctx}
}

// Resolve resolves all variables in a value
func (r *VariableResolver) Resolve(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return r.resolveString(v)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			resolved, err := r.Resolve(val)
			if err != nil {
				return nil, err
			}
			result[key] = resolved
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			resolved, err := r.Resolve(val)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil
	default:
		return v, nil
	}
}

// resolveString resolves variables in a string
func (r *VariableResolver) resolveString(s string) (string, error) {
	// Pattern: {{variable.path}}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	
	var err error
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		if err != nil {
			return match
		}
		
		// Extract the variable path without {{ }}
		path := strings.TrimSpace(match[2 : len(match)-2])
		
		value, resolveErr := r.resolvePath(path)
		if resolveErr != nil {
			err = resolveErr
			return match
		}
		
		return fmt.Sprintf("%v", value)
	})
	
	return result, err
}

// resolvePath resolves a variable path like "inputs.foo" or "steps.review.output"
func (r *VariableResolver) resolvePath(path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty variable path")
	}
	
	switch parts[0] {
	case "inputs":
		return r.getMapValue(r.ctx.Inputs, parts[1:])
	case "steps":
		if len(parts) < 2 {
			return nil, fmt.Errorf("steps path requires step ID")
		}
		stepResult, ok := r.ctx.Steps[parts[1]]
		if !ok {
			return nil, fmt.Errorf("step %s not found", parts[1])
		}
		if len(parts) == 2 {
			return stepResult.Output, nil
		}
		return r.getMapValue(stepResult.Output, parts[2:])
	case "env":
		if len(parts) < 2 {
			return nil, fmt.Errorf("env path requires variable name")
		}
		// Try context env first, then OS env
		if val, ok := r.ctx.Env[parts[1]]; ok {
			return val, nil
		}
		val := os.Getenv(parts[1])
		if val == "" {
			return nil, fmt.Errorf("environment variable %s not set", parts[1])
		}
		return val, nil
	case "memory":
		return r.getMapValue(r.ctx.Memory, parts[1:])
	default:
		return nil, fmt.Errorf("unknown variable namespace: %s", parts[0])
	}
}

// getMapValue retrieves a value from a map using a dot-notation path
func (r *VariableResolver) getMapValue(m map[string]interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return m, nil
	}
	
	val, ok := m[parts[0]]
	if !ok {
		return nil, fmt.Errorf("key %s not found", parts[0])
	}
	
	if len(parts) == 1 {
		return val, nil
	}
	
	// Handle nested maps
	nextMap, ok := val.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s is not a map", parts[0])
	}
	
	return r.getMapValue(nextMap, parts[1:])
}
