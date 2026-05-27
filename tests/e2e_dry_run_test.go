package tests

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDryRunWithValidWorkflow(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with valid workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/simple-print-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify dry-run output contains expected sections
	outputStr := string(output)
	expectedSections := []string{
		"Dry Run Mode",
		"Workflow will be validated but not executed",
		"Execution Plan",
		"Environment Variables",
		"State Store",
		"Dry Run Complete",
	}

	for _, section := range expectedSections {
		if !strings.Contains(outputStr, section) {
			t.Errorf("Dry run output missing section '%s'", section)
		}
	}
}

func TestDryRunWithInvalidWorkflow(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with invalid workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/invalid-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Dry run should fail with invalid workflow")
	}

	// Verify validation error message
	outputStr := string(output)
	if !strings.Contains(outputStr, "validation failed") && !strings.Contains(outputStr, "validation") {
		t.Errorf("Expected validation error, got: %s", outputStr)
	}
}

func TestDryRunWithComplexWorkflow(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with GitLab CI workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/valid-gitlab-ci-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify execution order is shown
	outputStr := string(output)
	if !strings.Contains(outputStr, "get-diff") || !strings.Contains(outputStr, "ai-review") || !strings.Contains(outputStr, "post-comment") {
		t.Errorf("Dry run should show all steps in execution order")
	}

	// Verify dependencies are shown
	if !strings.Contains(outputStr, "Depends On") {
		t.Error("Dry run should show step dependencies")
	}
}

func TestDryRunWithQuietMode(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with quiet mode workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/quiet-mode-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify quiet mode is detected
	outputStr := string(output)
	if !strings.Contains(outputStr, "Quiet Mode: true") {
		t.Error("Dry run should show quiet mode setting")
	}
}

func TestDryRunWithNonExistentWorkflow(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with non-existent workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/non-existent.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Dry run should fail with non-existent workflow")
	}

	// Verify file not found error
	outputStr := string(output)
	if !strings.Contains(outputStr, "no such file") && !strings.Contains(outputStr, "not found") {
		t.Errorf("Expected file not found error, got: %s", outputStr)
	}
}
