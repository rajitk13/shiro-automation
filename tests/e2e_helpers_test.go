package tests

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	return filepath.Dir(filepath.Dir(file))
}

func buildShiroBinary(t *testing.T) string {
	t.Helper()

	binaryPath := filepath.Join(t.TempDir(), "shiro-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/runtime")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build shiro: %v\nOutput: %s", err, string(output))
	}

	return binaryPath
}
