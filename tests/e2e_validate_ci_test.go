package tests

import (
	"os/exec"
	"strings"
	"testing"
)

func TestValidateCommandWithValidGitLabCI(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)
	cmd := exec.Command(shiro, "validate", "-workflow", "tests/fixtures/pause-workflow.json", "-ci", "tests/fixtures/valid-gitlab-ci.yml")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "[VALIDATE] Workflow") {
		t.Errorf("expected workflow validation output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "GitLab CI") {
		t.Errorf("expected GitLab CI platform output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "All checks passed") {
		t.Errorf("expected CI check pass output, got: %s", outputStr)
	}
}

func TestValidateCommandWithInvalidGitLabCI(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)
	cmd := exec.Command(shiro, "validate", "-workflow", "tests/fixtures/pause-workflow.json", "-ci", "tests/fixtures/invalid-gitlab-ci.yml")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("validate should fail for invalid GitLab CI config\nOutput: %s", string(output))
	}

	outputStr := string(output)
	expected := []string{
		"GitLab CI",
		"pause:true",
		"manual resume job",
		"state-store gitlab",
		"artifact",
	}
	for _, item := range expected {
		if !strings.Contains(outputStr, item) {
			t.Errorf("expected output to contain %q, got: %s", item, outputStr)
		}
	}
}

func TestValidateCommandWithValidGitHubActions(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)
	cmd := exec.Command(shiro, "validate", "-workflow", "tests/fixtures/pause-workflow.json", "-ci", "tests/fixtures/valid-github-actions.yml")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "GitHub Actions") {
		t.Errorf("expected GitHub Actions platform output, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "environment protection rule found") {
		t.Errorf("expected environment protection info, got: %s", outputStr)
	}
}

func TestValidateCommandWithInvalidGitHubActions(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)
	cmd := exec.Command(shiro, "validate", "-workflow", "tests/fixtures/pause-workflow.json", "-ci", "tests/fixtures/invalid-github-actions.yml")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("validate should fail for invalid GitHub Actions config\nOutput: %s", string(output))
	}

	outputStr := string(output)
	expected := []string{
		"GitHub Actions",
		"pause:true",
		"environment protection",
		"state-store gitlab",
		"filesystem",
	}
	for _, item := range expected {
		if !strings.Contains(outputStr, item) {
			t.Errorf("expected output to contain %q, got: %s", item, outputStr)
		}
	}
}
