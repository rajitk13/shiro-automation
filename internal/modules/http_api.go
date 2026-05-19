package modules

import "context"

// HTTPModuleAPI defines the contract for HTTP-based modules
type HTTPModuleAPI interface {
	// Execute runs the module with the given step data
	Execute(ctx context.Context, request ExecuteRequest) (ExecuteResponse, error)

	// Metadata returns module metadata
	Metadata() (MetadataResponse, error)

	// Health checks if the module is available
	Health() (HealthResponse, error)
}

// ExecuteRequest represents a request to execute a module
type ExecuteRequest struct {
	StepID    string                 `json:"step_id"`
	StepType  string                 `json:"step_type"`
	Operation string                 `json:"operation,omitempty"` // Operation to execute (e.g., "create_issue")
	Config    map[string]interface{} `json:"config"`
	Input     map[string]interface{} `json:"input"`
	Context   map[string]interface{} `json:"context"`
}

// ExecuteResponse represents the result of module execution
type ExecuteResponse struct {
	Success bool                   `json:"success"`
	Output  map[string]interface{} `json:"output"`
	Error   string                 `json:"error,omitempty"`
}

// MetadataResponse represents module metadata
type MetadataResponse struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	InputSchema  map[string]SchemaField `json:"input_schema"`
	OutputSchema map[string]SchemaField `json:"output_schema"`
}

// HealthResponse represents the health status of a module
type HealthResponse struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}
