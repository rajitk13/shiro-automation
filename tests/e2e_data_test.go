package tests

import (
	"os/exec"
	"strings"
	"testing"
)

func TestSetCommand(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// Set a value
	cmd := exec.Command(shiro, "set", "test_key", "test_value", "-state-store", "memory")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("set command failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Successfully set") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "test_key") {
		t.Errorf("Expected key in output, got: %s", outputStr)
	}
}

func TestGetCommand(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// First set a value
	setCmd := exec.Command(shiro, "set", "get_test_key", "get_test_value", "-state-store", "memory")
	setCmd.Dir = repoRoot(t)
	if _, err := setCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to set value for get test: %v", err)
	}

	// Get the value
	getCmd := exec.Command(shiro, "get", "get_test_key", "-state-store", "memory")
	getCmd.Dir = repoRoot(t)
	output, err := getCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get command failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "get_test_value") {
		t.Errorf("Expected value 'get_test_value', got: %s", outputStr)
	}
}

func TestGetCommandWithDefault(t *testing.T) {
	t.Skip("default value handling not yet implemented in get command")
}

func TestDeleteCommand(t *testing.T) {
	t.Parallel()

	shiro := buildShiroBinary(t)

	// First set a value
	setCmd := exec.Command(shiro, "set", "delete_test_key", "delete_test_value", "-state-store", "memory")
	setCmd.Dir = repoRoot(t)
	if _, err := setCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to set value for delete test: %v", err)
	}

	// Delete the value
	deleteCmd := exec.Command(shiro, "delete", "delete_test_key", "-state-store", "memory")
	deleteCmd.Dir = repoRoot(t)
	output, err := deleteCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("delete command failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Successfully deleted") {
		t.Errorf("Expected success message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "delete_test_key") {
		t.Errorf("Expected key in output, got: %s", outputStr)
	}
}

func TestListCommand(t *testing.T) {
	t.Skip("list operation not yet implemented in data module")
}

func TestListCommandWithPrefix(t *testing.T) {
	t.Skip("list operation not yet implemented in data module")
}

func TestSetWithNamespace(t *testing.T) {
	t.Skip("namespace output not yet implemented in set command")
}

func TestSetWithTTL(t *testing.T) {
	t.Skip("TTL output not yet implemented in set command")
}
