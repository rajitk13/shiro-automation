package tests

import (
	"os/exec"
	"strings"
	"testing"
)

func TestModuleList(t *testing.T) {
	t.Skip("module list requires registry file - should be tested with proper setup")
}

func TestModuleInfo(t *testing.T) {
	t.Skip("module info requires registry file - should be tested with proper setup")
}

func TestModuleAddInvalid(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Try to add a module with invalid input
	cmd := exec.Command(shiro, "add", "module", "")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("module add with empty name should fail")
	}

	outputStr := string(output)
	// The command tries to auto-discover and fails with "not found" message
	if !strings.Contains(outputStr, "not found") {
		t.Errorf("Expected 'not found' error message, got: %s", outputStr)
	}
}

func TestModuleRemoveInvalid(t *testing.T) {
	t.Skip("module remove requires registry file - should be tested with proper setup")
}

// Note: TestModuleAdd and TestModuleSearch are skipped as they require
// GitHub API access or network calls. These should be tested with
// mocked responses or integration tests with a test GitHub repository.
