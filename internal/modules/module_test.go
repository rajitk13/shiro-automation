package modules

import (
	"context"
	"testing"
)

type mockModule struct {
	metadata ModuleMetadata
}

func (m *mockModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"result": "success"}, nil
}

func (m *mockModule) Metadata() ModuleMetadata {
	return m.metadata
}

func TestRegistry(t *testing.T) {
	registry := NewRegistry()
	module := &mockModule{
		metadata: ModuleMetadata{
			Name: "test.module",
		},
	}

	// Test registration
	err := registry.Register("test.module", module)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Test duplicate registration
	err = registry.Register("test.module", &mockModule{})
	if err == nil {
		t.Error("Register() should error on duplicate registration")
	}

	// Test retrieval
	retrieved, err := registry.Get("test.module")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved == nil {
		t.Error("Get() returned nil")
	}

	// Test non-existent retrieval
	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("Get() should error for non-existent module")
	}

	// Test list
	list := registry.List()
	if len(list) != 1 {
		t.Errorf("List() returned %d items, expected 1", len(list))
	}

	if list[0] != "test.module" {
		t.Errorf("List() returned '%s', expected 'test.module'", list[0])
	}
}
