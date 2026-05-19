package modules

import (
	"context"
	"fmt"
	"sync"
)

// Module is the interface that all workflow modules must implement
type Module interface {
	// Run executes the module with the given context and step configuration
	Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error)

	// Metadata returns module metadata including input/output schema
	Metadata() ModuleMetadata
}

// ModuleMetadata describes a module's capabilities
type ModuleMetadata struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]SchemaField `json:"input_schema"`
	OutputSchema map[string]SchemaField `json:"output_schema"`
}

// SchemaField describes a field in the schema
type SchemaField struct {
	Type        string      `json:"type"` // string, number, boolean, array, object
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// Registry holds all registered modules
type Registry struct {
	modules map[string]Module
	mu      sync.RWMutex
}

// NewRegistry creates a new module registry
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]Module),
	}
}

// Register registers a module with the given type name
func (r *Registry) Register(moduleType string, module Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.modules[moduleType]; exists {
		return fmt.Errorf("module type %s already registered", moduleType)
	}

	r.modules[moduleType] = module
	return nil
}

// Get retrieves a module by type
func (r *Registry) Get(moduleType string) (Module, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	module, exists := r.modules[moduleType]
	if !exists {
		return nil, fmt.Errorf("module type %s not found", moduleType)
	}

	return module, nil
}

// List returns all registered module types
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.modules))
	for t := range r.modules {
		types = append(types, t)
	}
	return types
}
