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
		"Dry Run Complete",
	}

	for _, section := range expectedSections {
		if !strings.Contains(outputStr, section) {
			t.Errorf("Dry run output missing section '%s'", section)
		}
	}

	// State Store section should NOT appear for simple workflows (default gitlab/memory)
	if strings.Contains(outputStr, "State Store") {
		t.Error("Dry run should not show State Store for default configuration")
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

func TestDryRunWithGitHubWorkflow(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with GitHub Actions workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/valid-github-actions-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify GitHub environment variables are shown
	outputStr := string(output)
	expectedVars := []string{
		"GITHUB_TOKEN",
		"GITHUB_REPOSITORY",
		"GITHUB_PR_NUMBER",
		"GITHUB_SHA",
	}

	for _, varName := range expectedVars {
		if !strings.Contains(outputStr, varName) {
			t.Errorf("Dry run should show GitHub env var '%s'", varName)
		}
	}

	// Verify GitLab variables are NOT shown
	gitlabVars := []string{
		"CI_PROJECT_ID",
		"CI_MERGE_REQUEST_IID",
		"CI_COMMIT_SHA",
	}

	for _, varName := range gitlabVars {
		if strings.Contains(outputStr, varName) {
			t.Errorf("Dry run should NOT show GitLab env var '%s' for GitHub workflow", varName)
		}
	}
}

func TestDryRunWithGitLabWorkflow(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with GitLab CI workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/valid-gitlab-ci-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify GitLab environment variables are shown
	outputStr := string(output)
	expectedVars := []string{
		"CI_PROJECT_ID",
		"CI_MERGE_REQUEST_IID",
		"CI_COMMIT_SHA",
		"CI_COMMIT_REF_NAME",
		"CI_SERVER_URL",
	}

	for _, varName := range expectedVars {
		if !strings.Contains(outputStr, varName) {
			t.Errorf("Dry run should show GitLab env var '%s'", varName)
		}
	}

	// Verify GitHub variables are NOT shown
	ghVars := []string{
		"GITHUB_TOKEN",
		"GITHUB_REPOSITORY",
		"GITHUB_PR_NUMBER",
	}

	for _, varName := range ghVars {
		if strings.Contains(outputStr, varName) {
			t.Errorf("Dry run should NOT show GitHub env var '%s' for GitLab workflow", varName)
		}
	}
}

func TestDryRunWithNoCIModules(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with simple print workflow (no CI modules)
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/simple-print-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify Environment Variables section is NOT shown
	outputStr := string(output)
	if strings.Contains(outputStr, "Environment Variables") {
		t.Error("Dry run should NOT show Environment Variables section for workflows without CI modules")
	}

	// Verify State Store section is NOT shown
	if strings.Contains(outputStr, "State Store") {
		t.Error("Dry run should NOT show State Store for default configuration")
	}
}
