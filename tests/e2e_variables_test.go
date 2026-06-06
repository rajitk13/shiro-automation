package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestEnvVariableResolution(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Set an environment variable
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	// Create a workflow that uses environment variable
	workflowContent := `{
		"name": "env-var-test",
		"steps": [
			{
				"id": "print-env",
				"type": "print",
				"config": {
					"message": "{{env.TEST_VAR}}"
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

	// Run workflow
	cmd := exec.Command(shiro, "run", "-workflow", tmpFile.Name(), "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Workflow execution failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "test_value") {
		t.Errorf("Expected environment variable to be resolved, got: %s", outputStr)
	}
}

func TestStepOutputResolution(t *testing.T) {
	t.Skip("step output resolution not yet implemented for print module")
}

func TestInputVariableResolution(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Create a workflow with inputs
	workflowContent := `{
		"name": "input-test",
		"inputs": {
			"test_input": {
				"type": "string",
				"default": "default_value"
			}
		},
		"steps": [
			{
				"id": "print-input",
				"type": "print",
				"config": {
					"message": "{{inputs.test_input}}"
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

	// Run workflow
	cmd := exec.Command(shiro, "run", "-workflow", tmpFile.Name(), "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Workflow execution failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "default_value") {
		t.Errorf("Expected input variable to be resolved to default, got: %s", outputStr)
	}
}
