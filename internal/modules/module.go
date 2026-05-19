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
	modules     map[string]Module
	httpModules map[string]*HTTPModuleConfig
	mu          sync.RWMutex
}

// HTTPModuleConfig represents configuration for an HTTP-based module
type HTTPModuleConfig struct {
	Name     string
	Endpoint string
	Config   map[string]interface{}
	Metadata MetadataResponse
}

// NewRegistry creates a new module registry
func NewRegistry() *Registry {
	return &Registry{
		modules:     make(map[string]Module),
		httpModules: make(map[string]*HTTPModuleConfig),
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

	types := make([]string, 0, len(r.modules)+len(r.httpModules))
	for t := range r.modules {
		types = append(types, t)
	}
	for t := range r.httpModules {
		types = append(types, t)
	}
	return types
}

// RegisterHTTPModule registers an HTTP-based module
func (r *Registry) RegisterHTTPModule(moduleType string, config *HTTPModuleConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.httpModules[moduleType]; exists {
		return fmt.Errorf("HTTP module type %s already registered", moduleType)
	}

	r.httpModules[moduleType] = config
	return nil
}

// GetHTTPModule retrieves an HTTP module configuration by type
func (r *Registry) GetHTTPModule(moduleType string) (*HTTPModuleConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.httpModules[moduleType]
	if !exists {
		return nil, fmt.Errorf("HTTP module type %s not found", moduleType)
	}

	return config, nil
}

// IsHTTPModule checks if a module type is HTTP-based
func (r *Registry) IsHTTPModule(moduleType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.httpModules[moduleType]
	return exists
}
