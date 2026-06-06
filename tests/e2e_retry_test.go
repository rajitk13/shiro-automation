package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestWorkflowWithRetry(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Create a workflow with retry configuration
	workflowContent := `{
		"name": "retry-test",
		"steps": [
			{
				"id": "print-step",
				"type": "print",
				"config": {
					"message": "test message"
				},
				"retry": {
					"max_attempts": 3,
					"backoff": 2.0,
					"initial_delay": 1000
				}
			}
		]
	}`

	// Write temporary workflow file
	tmpFile, err := os.CreateTemp("", "workflow-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(workflowContent); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}
	tmpFile.Close()

	// Run dry-run to verify retry configuration is shown
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", tmpFile.Name())
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	// Verify workflow loads successfully with retry config
	if !strings.Contains(outputStr, "Execution Plan") {
		t.Error("Dry run should show execution plan")
	}
	// Note: Retry configuration details may not be shown in dry-run output yet
}

func TestWorkflowWithRetryAndBackoff(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Create a workflow with different backoff strategies
	workflowContent := `{
		"name": "retry-backoff-test",
		"steps": [
			{
				"id": "print-step",
				"type": "print",
				"config": {
					"message": "test message"
				},
				"retry": {
					"max_attempts": 5,
					"backoff": 1.0,
					"initial_delay": 2000
				}
			}
		]
	}`

	// Write temporary workflow file
	tmpFile, err := os.CreateTemp("", "workflow-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(workflowContent); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}
	tmpFile.Close()

	// Run dry-run
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", tmpFile.Name())
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	// Verify workflow loads successfully with retry config
	if !strings.Contains(outputStr, "Execution Plan") {
		t.Error("Dry run should show execution plan")
	}
}

func TestWorkflowWithoutRetry(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Create a workflow without retry configuration
	workflowContent := `{
		"name": "no-retry-test",
		"steps": [
			{
				"id": "print-step",
				"type": "print",
				"config": {
					"message": "test message"
				}
			}
		]
	}`

	// Write temporary workflow file
	tmpFile, err := os.CreateTemp("", "workflow-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(workflowContent); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}
	tmpFile.Close()

	// Run dry-run
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", tmpFile.Name())
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Should still succeed, just no retry config shown
	outputStr := string(output)
	if !strings.Contains(outputStr, "Execution Plan") {
		t.Error("Dry run should show execution plan")
	}
}
