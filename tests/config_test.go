package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/config"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a .shiro directory
	shiroDir := filepath.Join(tmpDir, ".shiro")
	if err := os.Mkdir(shiroDir, 0755); err != nil {
		t.Fatalf("Failed to create .shiro directory: %v", err)
	}

	// Test loading config from .shiro directory
	cfg, err := config.LoadConfig(shiroDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ShiroDir != shiroDir {
		t.Errorf("Expected ShiroDir to be '%s', got '%s'", shiroDir, cfg.ShiroDir)
	}
}

func TestLoadConfigDefault(t *testing.T) {
	cfg, err := config.LoadConfig(".shiro")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if cfg.ShiroDir != ".shiro" {
		t.Errorf("Expected default ShiroDir to be '.shiro', got '%s'", cfg.ShiroDir)
	}

	// StateStore is empty by default, set to gitlab in run.go if not configured
	if cfg.StateStore != "" {
		t.Errorf("Expected default StateStore to be empty, got '%s'", cfg.StateStore)
	}
}

func TestAutoDetectWorkflowFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a workflow file
	workflowPath := filepath.Join(tmpDir, "workflow.json")
	workflowContent := []byte(`{"name": "test", "steps": []}`)
	if err := os.WriteFile(workflowPath, workflowContent, 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Load config which will auto-detect the workflow file
	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.WorkflowFile == "" {
		t.Error("Expected workflow file to be detected, got empty string")
	}
}

func TestAutoDetectConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := []byte("models:\n  openai:\n    model: gpt-4")
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Load config which will auto-detect the config file
	cfg, err := config.LoadConfig(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ConfigFile == "" {
		t.Error("Expected config file to be detected, got empty string")
	}
}

func TestGetRegistryPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .shiro directory structure
	shiroDir := filepath.Join(tmpDir, ".shiro")
	modulesDir := filepath.Join(shiroDir, "modules")
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		t.Fatalf("Failed to create modules directory: %v", err)
	}

	// Create registry file
	registryPath := filepath.Join(modulesDir, "registry.yaml")
	registryContent := []byte("modules: []")
	if err := os.WriteFile(registryPath, registryContent, 0644); err != nil {
		t.Fatalf("Failed to create registry file: %v", err)
	}

	path := config.GetRegistryPath(shiroDir)
	expectedPath := filepath.Join(shiroDir, "modules", "registry.yaml")
	if path != expectedPath {
		t.Errorf("Expected '%s', got '%s'", expectedPath, path)
	}
}

func TestLoadModelConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid model config file
	configPath := filepath.Join(tmpDir, "models.yaml")
	configContent := []byte(`models:
  openai:
    model: gpt-4
    api_key: test-key`)
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	models, err := config.LoadModelConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load model config: %v", err)
	}

	if models == nil {
		t.Error("Expected models map, got nil")
	}

	if _, ok := models["openai"]; !ok {
		t.Error("Expected openai model in models")
	}
}

func TestLoadModelConfigEmpty(t *testing.T) {
	models, err := config.LoadModelConfig("")
	if err != nil {
		t.Fatalf("Failed to load empty config: %v", err)
	}

	if models == nil {
		t.Error("Expected empty models map, got nil")
	}

	if len(models) != 0 {
		t.Errorf("Expected empty models map, got %d entries", len(models))
	}
}

func TestLoadModelConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid YAML file
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	configContent := []byte("invalid: yaml: content:")
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	_, err := config.LoadModelConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadModelConfigNonExistent(t *testing.T) {
	_, err := config.LoadModelConfig("/nonexistent/file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
