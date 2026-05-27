package tests

import (
	"os"
	"testing"

	"github.com/rkuthiala/shiro-automation/internal/cli"
)

func TestInitCodeReviewDirectConfig(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "shiro-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Run init code-review with direct config
	args := []string{"-template", "code-review", "-d", "provider=openai", "-d", "api_key=test-key", "-d", "model=gpt-4"}
	cli.InitCommand(args)

	// Verify files were created
	workflowPath := ".shiro/workflows/code-review.json"
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Errorf("Workflow file not created: %s", workflowPath)
	}

	configPath := ".shiro/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file not created: %s", configPath)
	}

	// Verify config content
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	configStr := string(configContent)
	if !contains(configStr, "openai") || !contains(configStr, "test-key") {
		t.Errorf("Config doesn't contain expected values: %s", configStr)
	}
}

func TestInitCodeReviewDefaultConfig(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "shiro-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Run init code-review without config flags (default template)
	args := []string{"-template", "code-review"}
	cli.InitCommand(args)

	// Verify files were created
	workflowPath := ".shiro/workflows/code-review.json"
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Errorf("Workflow file not created: %s", workflowPath)
	}

	configPath := ".shiro/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file not created: %s", configPath)
	}

	// Verify config content has template placeholder
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}
	configStr := string(configContent)
	if !contains(configStr, "{{env.OPENAI_API_KEY}}") {
		t.Errorf("Config doesn't contain expected placeholder")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
