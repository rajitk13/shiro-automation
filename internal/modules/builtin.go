package modules

import (
	"log"
	"sync"
)

// BuiltinModuleRegistry holds built-in module factories
var builtinRegistry struct {
	factories map[string]BuiltinModuleFactory
	mu        sync.RWMutex
}

func init() {
	builtinRegistry.factories = make(map[string]BuiltinModuleFactory)
}

// RegisterBuiltin registers a built-in module globally
// This should be called from init() functions in module packages
func RegisterBuiltin(name string, factory BuiltinModuleFactory) {
	builtinRegistry.mu.Lock()
	defer builtinRegistry.mu.Unlock()

	if _, exists := builtinRegistry.factories[name]; exists {
		log.Printf("Warning: Built-in module %s already registered, overwriting", name)
	}

	builtinRegistry.factories[name] = factory
	log.Printf("Registered built-in module: %s", name)
}

// GetBuiltinFactory returns a factory for a built-in module
func GetBuiltinFactory(name string) (BuiltinModuleFactory, bool) {
	builtinRegistry.mu.RLock()
	defer builtinRegistry.mu.RUnlock()

	factory, exists := builtinRegistry.factories[name]
	return factory, exists
}

// ListBuiltinModules returns all registered built-in module names
func ListBuiltinModules() []string {
	builtinRegistry.mu.RLock()
	defer builtinRegistry.mu.RUnlock()

	names := make([]string, 0, len(builtinRegistry.factories))
	for name := range builtinRegistry.factories {
		names = append(names, name)
	}
	return names
}
