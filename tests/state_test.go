package tests

import (
	"context"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

func TestMemoryStateStore(t *testing.T) {
	store := state.NewMemoryStore()
	ctx := context.Background()

	// Test Save and Load
	testData := map[string]interface{}{
		"key":    "value",
		"number": 42,
	}

	err := store.Save(ctx, "test-key", testData)
	if err != nil {
		t.Fatalf("Failed to save to memory store: %v", err)
	}

	var loadedData map[string]interface{}
	err = store.Load(ctx, "test-key", &loadedData)
	if err != nil {
		t.Fatalf("Failed to load from memory store: %v", err)
	}

	if loadedData["key"] != "value" {
		t.Errorf("Expected 'value', got '%v'", loadedData["key"])
	}

	// Test Exists
	exists, err := store.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}

	// Test Delete
	err = store.Delete(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	exists, err = store.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence after delete: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist after delete")
	}
}

func TestFilesystemStateStore(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"base_dir": tmpDir,
	}
	store, err := state.NewFilesystemStore(config)
	if err != nil {
		t.Fatalf("Failed to create filesystem store: %v", err)
	}
	ctx := context.Background()

	// Test Save and Load
	testData := map[string]interface{}{
		"key":    "value",
		"number": 42,
	}

	err = store.Save(ctx, "test-key", testData)
	if err != nil {
		t.Fatalf("Failed to save to filesystem store: %v", err)
	}

	var loadedData map[string]interface{}
	err = store.Load(ctx, "test-key", &loadedData)
	if err != nil {
		t.Fatalf("Failed to load from filesystem store: %v", err)
	}

	if loadedData["key"] != "value" {
		t.Errorf("Expected 'value', got '%v'", loadedData["key"])
	}

	// Test Exists
	exists, err := store.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}

	// Test Delete
	err = store.Delete(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	exists, err = store.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence after delete: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist after delete")
	}
}

func TestFilesystemStateStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	testData := map[string]interface{}{
		"key": "value",
	}

	config := map[string]interface{}{
		"base_dir": tmpDir,
	}
	// Save with first store instance
	store1, err := state.NewFilesystemStore(config)
	if err != nil {
		t.Fatalf("Failed to create filesystem store: %v", err)
	}
	ctx := context.Background()

	err = store1.Save(ctx, "test-key", testData)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load with second store instance to test persistence
	store2, err := state.NewFilesystemStore(config)
	if err != nil {
		t.Fatalf("Failed to create second filesystem store: %v", err)
	}

	var loadedData map[string]interface{}
	err = store2.Load(ctx, "test-key", &loadedData)
	if err != nil {
		t.Fatalf("Failed to load from second store: %v", err)
	}

	if loadedData["key"] != "value" {
		t.Errorf("Expected 'value', got '%v'", loadedData["key"])
	}
}

func TestGitLabStateStore(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"artifacts_dir": tmpDir,
	}

	store, err := state.NewGitLabStore(config)
	if err != nil {
		t.Fatalf("Failed to create GitLab store: %v", err)
	}
	ctx := context.Background()

	// Test Save and Load
	testData := map[string]interface{}{
		"key":    "value",
		"number": 42,
	}

	err = store.Save(ctx, "test-key", testData)
	if err != nil {
		t.Fatalf("Failed to save to GitLab store: %v", err)
	}

	var loadedData map[string]interface{}
	err = store.Load(ctx, "test-key", &loadedData)
	if err != nil {
		t.Fatalf("Failed to load from GitLab store: %v", err)
	}

	if loadedData["key"] != "value" {
		t.Errorf("Expected 'value', got '%v'", loadedData["key"])
	}

	// Test Exists
	exists, err := store.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}

	// Test Delete
	err = store.Delete(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}

	exists, err = store.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence after delete: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist after delete")
	}
}

func TestStoreFactory(t *testing.T) {
	factory := state.NewStoreFactory()

	// Test memory store
	memoryStore, err := factory.Create("memory", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}
	if memoryStore == nil {
		t.Error("Expected memory store, got nil")
	}

	// Test filesystem store
	tmpDir := t.TempDir()
	fsConfig := map[string]interface{}{
		"path": tmpDir,
	}
	fsStore, err := factory.Create("filesystem", fsConfig)
	if err != nil {
		t.Fatalf("Failed to create filesystem store: %v", err)
	}
	if fsStore == nil {
		t.Error("Expected filesystem store, got nil")
	}

	// Test GitLab store
	glConfig := map[string]interface{}{
		"artifacts_dir": tmpDir,
	}
	glStore, err := factory.Create("gitlab", glConfig)
	if err != nil {
		t.Fatalf("Failed to create GitLab store: %v", err)
	}
	if glStore == nil {
		t.Error("Expected GitLab store, got nil")
	}

	// Test invalid store type
	_, err = factory.Create("invalid", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for invalid store type, got nil")
	}
}

func TestStateStoreWithExecutionContext(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"base_dir": tmpDir,
	}
	store, err := state.NewFilesystemStore(config)
	if err != nil {
		t.Fatalf("Failed to create filesystem store: %v", err)
	}
	ctx := context.Background()

	// Create an execution context
	execCtx := workflow.NewExecutionContext()
	execCtx.Inputs = map[string]interface{}{
		"user": "test",
		"repo": "test-repo",
	}
	execCtx.Steps = map[string]workflow.StepResult{
		"step1": {Success: true, Output: map[string]interface{}{"result": "success"}},
	}

	// Save execution context
	err = store.Save(ctx, "workflow-execution", execCtx)
	if err != nil {
		t.Fatalf("Failed to save execution context: %v", err)
	}

	// Load execution context
	var loadedCtx workflow.ExecutionContext
	err = store.Load(ctx, "workflow-execution", &loadedCtx)
	if err != nil {
		t.Fatalf("Failed to load execution context: %v", err)
	}

	if loadedCtx.Inputs["user"] != "test" {
		t.Errorf("Expected user 'test', got '%v'", loadedCtx.Inputs["user"])
	}

	if len(loadedCtx.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(loadedCtx.Steps))
	}
}

func TestStateStoreNonExistentKey(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"base_dir": tmpDir,
	}
	store, err := state.NewFilesystemStore(config)
	if err != nil {
		t.Fatalf("Failed to create filesystem store: %v", err)
	}
	ctx := context.Background()

	var data map[string]interface{}
	err = store.Load(ctx, "nonexistent-key", &data)
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
}

func TestStateStoreOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	config := map[string]interface{}{
		"base_dir": tmpDir,
	}
	store, err := state.NewFilesystemStore(config)
	if err != nil {
		t.Fatalf("Failed to create filesystem store: %v", err)
	}
	ctx := context.Background()

	// Save initial data
	initialData := map[string]interface{}{"key": "initial"}
	err = store.Save(ctx, "test-key", initialData)
	if err != nil {
		t.Fatalf("Failed to save initial data: %v", err)
	}

	// Overwrite with new data
	newData := map[string]interface{}{"key": "new"}
	err = store.Save(ctx, "test-key", newData)
	if err != nil {
		t.Fatalf("Failed to overwrite data: %v", err)
	}

	// Load and verify
	var loadedData map[string]interface{}
	err = store.Load(ctx, "test-key", &loadedData)
	if err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	if loadedData["key"] != "new" {
		t.Errorf("Expected 'new', got '%v'", loadedData["key"])
	}
}
