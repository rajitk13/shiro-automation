package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitBasic(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Run init command
	cmd := exec.Command(shiro, "init")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init command failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Shiro project initialized successfully") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}

	// Verify directory structure
	expectedDirs := []string{
		".shiro",
		".shiro/workflows",
		"modules",
	}

	for _, dir := range expectedDirs {
		fullPath := filepath.Join(tempDir, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", dir)
		}
	}

	// Verify files exist
	expectedFiles := []string{
		".shiro/workflow.json",
		".shiro/config.yaml",
		".shiro/modules/registry.yaml",
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(tempDir, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", file)
		}
	}
}

func TestInitWithTemplate(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Run init command with template
	cmd := exec.Command(shiro, "init", "-template", "code-review")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init command with template failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "code-review") {
		t.Errorf("Expected template name in output, got: %s", outputStr)
	}

	// Verify code-review specific files exist
	codeReviewWorkflow := filepath.Join(tempDir, ".shiro/workflows/code-review.json")
	if _, err := os.Stat(codeReviewWorkflow); os.IsNotExist(err) {
		t.Error("Expected code-review workflow to exist")
	}

	// Verify config has AI model configuration
	configFile := filepath.Join(tempDir, ".shiro/config.yaml")
	configContent, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(configContent)
	if !strings.Contains(configStr, "models:") {
		t.Error("Expected models configuration in config.yaml")
	}
}

func TestInitGitignoreUpdate(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create an existing .gitignore file
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	initialContent := "# Existing content\n"
	if err := os.WriteFile(gitignorePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Run init command
	cmd := exec.Command(shiro, "init")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init command failed: %v\nOutput: %s", err, string(output))
	}

	// Verify .gitignore was updated
	gitignoreContent, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	gitignoreStr := string(gitignoreContent)
	if !strings.Contains(gitignoreStr, ".shiro/") {
		t.Error("Expected .shiro/ to be in .gitignore")
	}
	if !strings.Contains(gitignoreStr, initialContent) {
		t.Error("Expected initial content to be preserved in .gitignore")
	}
}
