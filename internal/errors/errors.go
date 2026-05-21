package errors

import (
	"fmt"
	"strings"
)

// Error types for Shiro

// WorkflowError represents errors related to workflow execution
type WorkflowError struct {
	WorkflowName string
	StepID       string
	Message      string
	Err          error
}

func (e *WorkflowError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("workflow '%s' step '%s' error: %s: %v", e.WorkflowName, e.StepID, e.Message, e.Err)
	}
	return fmt.Sprintf("workflow '%s' step '%s' error: %s", e.WorkflowName, e.StepID, e.Message)
}

func (e *WorkflowError) Unwrap() error {
	return e.Err
}

// NewWorkflowError creates a new WorkflowError
func NewWorkflowError(workflowName, stepID, message string, err error) *WorkflowError {
	return &WorkflowError{
		WorkflowName: workflowName,
		StepID:       stepID,
		Message:      message,
		Err:          err,
	}
}

// ModuleError represents errors related to module operations
type ModuleError struct {
	ModuleName string
	Message    string
	Err        error
}

func (e *ModuleError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("module '%s' error: %s: %v", e.ModuleName, e.Message, e.Err)
	}
	return fmt.Sprintf("module '%s' error: %s", e.ModuleName, e.Message)
}

func (e *ModuleError) Unwrap() error {
	return e.Err
}

// NewModuleError creates a new ModuleError
func NewModuleError(moduleName, message string, err error) *ModuleError {
	return &ModuleError{
		ModuleName: moduleName,
		Message:    message,
		Err:        err,
	}
}

// ConfigError represents errors related to configuration
type ConfigError struct {
	ConfigPath string
	Message    string
	Err        error
}

func (e *ConfigError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("config error at '%s': %s: %v", e.ConfigPath, e.Message, e.Err)
	}
	return fmt.Sprintf("config error at '%s': %s", e.ConfigPath, e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new ConfigError
func NewConfigError(configPath, message string, err error) *ConfigError {
	return &ConfigError{
		ConfigPath: configPath,
		Message:    message,
		Err:        err,
	}
}

// ValidationError represents errors related to validation
type ValidationError struct {
	Field   string
	Message string
	Err     error
}

func (e *ValidationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("validation error for field '%s': %s: %v", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string, err error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Err:     err,
	}
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	messages := make([]string, 0, len(e))
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n")
}

// StateError represents errors related to state storage
type StateError struct {
	StoreType string
	Key       string
	Message   string
	Err       error
}

func (e *StateError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("state error (store: %s, key: %s): %s: %v", e.StoreType, e.Key, e.Message, e.Err)
	}
	return fmt.Sprintf("state error (store: %s, key: %s): %s", e.StoreType, e.Key, e.Message)
}

func (e *StateError) Unwrap() error {
	return e.Err
}

// NewStateError creates a new StateError
func NewStateError(storeType, key, message string, err error) *StateError {
	return &StateError{
		StoreType: storeType,
		Key:       key,
		Message:   message,
		Err:       err,
	}
}
