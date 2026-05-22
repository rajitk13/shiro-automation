package modules

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ModuleConfig represents a module's configuration from the registry
type ModuleConfig struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`                // "builtin" or "http"
	Endpoint    string   `yaml:"endpoint,omitempty"`  // Deprecated, use endpoints
	Endpoints   []string `yaml:"endpoints,omitempty"` // Multiple endpoints for load balancing
	Config      string   `yaml:"config,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Description string   `yaml:"description"`
	Source      string   `yaml:"source,omitempty"` // GitHub repo URL
	Docs        string   `yaml:"docs,omitempty"`   // Documentation URL
	// Fields for builtin modules (external compiled modules)
	Package string                 `yaml:"package,omitempty"`  // Go package import path
	Factory string                 `yaml:"factory,omitempty"`  // Factory function name (default: New<Module>Module)
	AddedAt string                 `yaml:"added_at,omitempty"` // Timestamp when added
	Extra   map[string]interface{} `yaml:",inline"`
}

// ModuleReviews represents review information for a module
type ModuleReviews struct {
	Count         int     `yaml:"count"`
	AverageRating float64 `yaml:"average_rating"`
}

// RegistryConfig represents the module registry configuration
type RegistryConfig struct {
	Modules map[string]ModuleConfig `yaml:"modules"`
}

// BuiltinModuleFactory is a function that creates a built-in module instance
type BuiltinModuleFactory func() (interface{}, error)

// Discoverer handles module discovery and loading
type Discoverer struct {
	registryPath     string
	registry         *RegistryConfig
	httpClient       *HTTPModuleClient
	builtinFactories map[string]BuiltinModuleFactory
	mu               sync.RWMutex
}

// NewDiscoverer creates a new module discoverer
func NewDiscoverer(registryPath string, httpClient *HTTPModuleClient) *Discoverer {
	return &Discoverer{
		registryPath:     registryPath,
		httpClient:       httpClient,
		builtinFactories: make(map[string]BuiltinModuleFactory),
	}
}

// RegisterBuiltinFactory registers a factory function for a built-in module
func (d *Discoverer) RegisterBuiltinFactory(name string, factory BuiltinModuleFactory) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.builtinFactories[name] = factory
}

// LoadRegistry loads the module registry configuration
func (d *Discoverer) LoadRegistry() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := os.ReadFile(d.registryPath)
	if err != nil {
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	var config RegistryConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse registry file: %w", err)
	}

	d.registry = &config
	log.Printf("Loaded module registry with %d modules", len(config.Modules))
	return nil
}

// Discover discovers and validates all modules
func (d *Discoverer) Discover(ctx context.Context) ([]ModuleConfig, error) {
	d.mu.RLock()
	if d.registry == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("registry not loaded, call LoadRegistry first")
	}
	d.mu.RUnlock()

	var availableModules []ModuleConfig
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, config := range d.registry.Modules {
		wg.Add(1)
		go func(moduleName string, moduleConfig ModuleConfig) {
			defer wg.Done()

			if moduleConfig.Type == "http" {
				// Validate HTTP module is available
				healthy, err := d.validateHTTPModule(ctx, moduleConfig)
				if err != nil {
					log.Printf("Module %s health check failed: %v", moduleName, err)
					return
				}
				if !healthy {
					log.Printf("Module %s is not healthy", moduleName)
					return
				}
			} else if moduleConfig.Type == "builtin" {
				// Validate built-in module factory exists
				d.mu.RLock()
				_, exists := d.builtinFactories[moduleName]
				d.mu.RUnlock()
				if !exists {
					log.Printf("Built-in module %s factory not registered", moduleName)
					return
				}
			}

			mu.Lock()
			availableModules = append(availableModules, moduleConfig)
			mu.Unlock()
		}(name, config)
	}

	wg.Wait()
	return availableModules, nil
}

// validateHTTPModule checks if an HTTP module is available
func (d *Discoverer) validateHTTPModule(ctx context.Context, config ModuleConfig) (bool, error) {
	if d.httpClient == nil {
		return false, fmt.Errorf("HTTP client not configured")
	}

	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Handle both single endpoint and multiple endpoints
	endpoints := config.Endpoints
	if len(endpoints) == 0 && config.Endpoint != "" {
		endpoints = []string{config.Endpoint}
	}

	if len(endpoints) == 0 {
		return false, fmt.Errorf("no endpoints configured")
	}

	// Try each endpoint
	for _, endpoint := range endpoints {
		resp, err := d.httpClient.Health(healthCtx, endpoint)
		if err != nil {
			continue
		}
		if resp.Healthy {
			return true, nil
		}
	}

	return false, nil
}

// GetModuleConfig returns configuration for a specific module
func (d *Discoverer) GetModuleConfig(name string) (ModuleConfig, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.registry == nil {
		return ModuleConfig{}, fmt.Errorf("registry not loaded")
	}

	config, exists := d.registry.Modules[name]
	if !exists {
		return ModuleConfig{}, fmt.Errorf("module %s not found in registry", name)
	}

	return config, nil
}

// ListModules returns all registered module names
func (d *Discoverer) ListModules() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.registry == nil {
		return []string{}
	}

	names := make([]string, 0, len(d.registry.Modules))
	for name := range d.registry.Modules {
		names = append(names, name)
	}
	return names
}

// ListBuiltinModules returns only built-in module names
func (d *Discoverer) ListBuiltinModules() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.registry == nil {
		return []string{}
	}

	var names []string
	for name, config := range d.registry.Modules {
		if config.Type == "builtin" {
			names = append(names, name)
		}
	}
	return names
}

// CreateBuiltinModule creates a built-in module instance using its registered factory
func (d *Discoverer) CreateBuiltinModule(name string) (interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	factory, exists := d.builtinFactories[name]
	if !exists {
		return nil, fmt.Errorf("no factory registered for built-in module %s", name)
	}

	return factory()
}

// AddModule adds a module to the registry
func (d *Discoverer) AddModule(name string, config ModuleConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.registry == nil {
		return fmt.Errorf("registry not loaded")
	}

	if _, exists := d.registry.Modules[name]; exists {
		return fmt.Errorf("module %s already exists in registry", name)
	}

	d.registry.Modules[name] = config
	return d.saveRegistry()
}

// RemoveModule removes a module from the registry
func (d *Discoverer) RemoveModule(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.registry == nil {
		return fmt.Errorf("registry not loaded")
	}

	if _, exists := d.registry.Modules[name]; !exists {
		return fmt.Errorf("module %s not found in registry", name)
	}

	delete(d.registry.Modules, name)
	return d.saveRegistry()
}

// saveRegistry saves the registry to disk
func (d *Discoverer) saveRegistry() error {
	data, err := yaml.Marshal(d.registry)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(d.registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

// LoadModuleConfig loads a module's specific configuration file
func (d *Discoverer) LoadModuleConfig(configPath string) (map[string]interface{}, error) {
	// Resolve relative paths relative to registry directory
	if !filepath.IsAbs(configPath) {
		registryDir := filepath.Dir(d.registryPath)
		configPath = filepath.Join(registryDir, configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read module config: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse module config: %w", err)
	}

	// Resolve environment variables in config
	resolveEnvVarsInMap(config)

	return config, nil
}

// resolveEnvVarsInMap resolves {{env.VARIABLE}} templates in a map
func resolveEnvVarsInMap(config map[string]interface{}) {
	for key, value := range config {
		if strValue, ok := value.(string); ok {
			config[key] = resolveEnvVarStringInModules(strValue)
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			resolveEnvVarsInMap(nestedMap)
		}
	}
}

// resolveEnvVarStringInModules resolves a single {{env.VARIABLE}} template
func resolveEnvVarStringInModules(input string) string {
	if strings.HasPrefix(input, "{{env.") && strings.HasSuffix(input, "}}") {
		envVar := strings.TrimPrefix(input, "{{env.")
		envVar = strings.TrimSuffix(envVar, "}}")
		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
	}
	return input
}
