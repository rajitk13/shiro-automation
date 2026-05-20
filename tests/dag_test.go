package tests

import (
	"context"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/modules"
)

func TestModuleRegistry(t *testing.T) {
	registry := modules.NewRegistry()

	// Test registering a module
	mockModule := &MockModule{}
	err := registry.Register("test", mockModule)
	if err != nil {
		t.Fatalf("Failed to register module: %v", err)
	}

	// Test getting a module
	module, err := registry.Get("test")
	if err != nil {
		t.Fatalf("Failed to get module: %v", err)
	}

	if module == nil {
		t.Error("Expected module, got nil")
	}

	// Test getting non-existent module
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent module, got nil")
	}

	// Test listing modules
	modulesList := registry.List()
	if len(modulesList) == 0 {
		t.Error("Expected at least one module in registry")
	}
}

func TestModuleRegistryDuplicate(t *testing.T) {
	registry := modules.NewRegistry()

	mockModule := &MockModule{}
	err := registry.Register("test", mockModule)
	if err != nil {
		t.Fatalf("Failed to register module: %v", err)
	}

	// Test registering duplicate
	err = registry.Register("test", mockModule)
	if err == nil {
		t.Error("Expected error for duplicate module, got nil")
	}
}

// MockModule for testing
type MockModule struct{}

func (m *MockModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"result": "success"}, nil
}

func (m *MockModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name: "mock",
	}
}
