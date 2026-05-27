package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestNormalLoggingWithPrintWorkflow(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run workflow with normal logging
	cmd := exec.Command(shiro, "run", "-workflow", "tests/fixtures/simple-print-workflow.json", "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Workflow execution failed: %v\nOutput: %s", err, string(output))
	}

	// Verify normal logging output
	outputStr := string(output)
	expectedOutputs := []string{
		"Hello, World!",
		"Workflow is running successfully",
		"Workflow Results",
		"Step: print-hello",
		"Step: print-status",
	}

	for _, expected := range expectedOutputs {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Normal logging should include '%s', got: %s", expected, outputStr)
		}
	}
}

func TestQuietModeSuppressesOutput(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run workflow with quiet mode
	cmd := exec.Command(shiro, "run", "-workflow", "tests/fixtures/quiet-mode-workflow.json", "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Workflow execution failed: %v\nOutput: %s", err, string(output))
	}

	// Verify quiet mode suppresses output
	outputStr := string(output)

	// Should NOT contain the print messages
	quietOutputs := []string{
		"This should not be printed",
		"This should also not be printed",
	}

	for _, quietOutput := range quietOutputs {
		if strings.Contains(outputStr, quietOutput) {
			t.Errorf("Quiet mode should suppress output, but found: %s", quietOutput)
		}
	}

	// Should NOT contain workflow results section
	if strings.Contains(outputStr, "Workflow Results") {
		t.Error("Quiet mode should not show workflow results")
	}
	if strings.Contains(outputStr, "[Shiro]") {
		t.Error("Quiet mode should not show Shiro logs")
	}
}

func TestStepLevelQuietFlag(t *testing.T) {
	t.Parallel()

	workflowContent := `{
		"name": "step-quiet-test",
		"steps": [
			{
				"id": "print-loud",
				"type": "print",
				"config": {"message": "This should be visible"}
			},
			{
				"id": "print-quiet",
				"type": "print",
				"quiet": true,
				"config": {"message": "This should be suppressed"}
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

	shiro := buildShiroBinary(t)

	cmd := exec.Command(shiro, "run", "-workflow", tmpFile.Name(), "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Workflow execution failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Loud step should be visible
	if !strings.Contains(outputStr, "This should be visible") {
		t.Error("Loud step output should be visible")
	}

	// Quiet step should not be in output (but step result should exist)
	if strings.Contains(outputStr, "This should be suppressed") {
		t.Error("Step-level quiet flag should suppress output")
	}
}

func TestWorkflowLogLevel(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run workflow and capture logs
	cmd := exec.Command(shiro, "run", "-workflow", "tests/fixtures/simple-print-workflow.json", "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Workflow execution failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)

	// Verify log format with [Shiro] prefix
	lines := strings.Split(outputStr, "\n")
	hasShiroPrefix := false
	for _, line := range lines {
		if strings.HasPrefix(line, "[Shiro]") {
			hasShiroPrefix = true
			break
		}
	}

	if !hasShiroPrefix {
		t.Error("Logs should have [Shiro] prefix")
	}
}

func TestErrorLogging(t *testing.T) {
	t.Parallel()

	workflowContent := `{
		"name": "error-logging-test",
		"steps": [
			{
				"id": "invalid-step",
				"type": "nonexistent.module",
				"config": {"message": "test"}
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

	shiro := buildShiroBinary(t)

	cmd := exec.Command(shiro, "run", "-workflow", tmpFile.Name(), "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Workflow with invalid module should fail")
	}

	outputStr := string(output)

	// Verify error is logged
	if !strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "Error") {
		t.Error("Errors should be logged")
	}
}
