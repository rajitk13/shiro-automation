package tests

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGitLabCIWorkflowValidation(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with GitLab CI workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/valid-gitlab-ci-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify GitLab CI specific environment variables are checked
	outputStr := string(output)
	gitLabVars := []string{
		"CI_PROJECT_ID",
		"CI_MERGE_REQUEST_IID",
		"CI_COMMIT_SHA",
		"CI_COMMIT_REF_NAME",
		"CI_SERVER_URL",
	}

	for _, varName := range gitLabVars {
		if !strings.Contains(outputStr, varName) {
			t.Errorf("GitLab CI validation should check for %s", varName)
		}
	}
}

func TestGitHubActionsWorkflowValidation(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with GitHub Actions workflow
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/valid-github-actions-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify workflow is valid and execution plan is shown
	outputStr := string(output)
	if !strings.Contains(outputStr, "Execution Plan") {
		t.Error("GitHub Actions workflow should show execution plan")
	}

	if !strings.Contains(outputStr, "github-actions-workflow") {
		t.Error("GitHub Actions workflow name should be shown")
	}
}

func TestWorkflowModuleValidation(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with GitLab CI workflow to validate modules
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/valid-gitlab-ci-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify modules are validated
	outputStr := string(output)
	expectedModules := []string{
		"git.diff",
		"ai.generate",
		"gitlab",
	}

	for _, module := range expectedModules {
		if !strings.Contains(outputStr, module) {
			t.Errorf("Workflow validation should check module: %s", module)
		}
	}
}

func TestWorkflowDependencyValidation(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with workflow that has dependencies
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/valid-gitlab-ci-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify dependencies are shown
	outputStr := string(output)
	if !strings.Contains(outputStr, "Depends On") {
		t.Error("Workflow validation should show dependencies")
	}

	// Verify correct execution order (get-diff before ai-review before post-comment)
	lines := strings.Split(outputStr, "\n")
	stepOrder := make([]string, 0)
	for _, line := range lines {
		if strings.Contains(line, "Step:") {
			parts := strings.Split(line, "Step: ")
			if len(parts) > 1 {
				stepID := strings.TrimSpace(parts[1])
				stepOrder = append(stepOrder, stepID)
			}
		}
	}

	// Verify order: get-diff should come before ai-review
	getDiffIndex := -1
	aiReviewIndex := -1
	for i, step := range stepOrder {
		if strings.Contains(step, "get-diff") {
			getDiffIndex = i
		}
		if strings.Contains(step, "ai-review") {
			aiReviewIndex = i
		}
	}

	if getDiffIndex == -1 || aiReviewIndex == -1 {
		t.Error("Could not find expected steps in execution order")
	} else if getDiffIndex > aiReviewIndex {
		t.Error("get-diff should execute before ai-review")
	}
}

func TestWorkflowStateStoreValidation(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with default state store
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/simple-print-workflow.json")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify state store is shown
	outputStr := string(output)
	if !strings.Contains(outputStr, "State Store") {
		t.Error("Workflow validation should show state store configuration")
	}

	if !strings.Contains(outputStr, "gitlab") {
		t.Error("Default state store should be gitlab")
	}
}

func TestWorkflowWithCustomStateStore(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Run dry-run with custom state store
	cmd := exec.Command(shiro, "run", "-dry-run", "-workflow", "tests/fixtures/simple-print-workflow.json", "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Dry run failed: %v\nOutput: %s", err, string(output))
	}

	// Verify custom state store is shown
	outputStr := string(output)
	if !strings.Contains(outputStr, "State Store") {
		t.Error("Workflow validation should show state store configuration")
	}

	if !strings.Contains(outputStr, "memory") {
		t.Error("Custom state store should be memory")
	}
}
