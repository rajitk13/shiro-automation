package cicheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
	"gopkg.in/yaml.v3"
)

// Severity of a finding
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARN"
	SeverityInfo    Severity = "INFO"
)

// Finding represents a single validation issue
type Finding struct {
	Severity Severity
	Rule     string
	Message  string
	Hint     string
}

func (f Finding) String() string {
	if f.Hint != "" {
		return fmt.Sprintf("[%s] %s\n        Hint: %s", f.Severity, f.Message, f.Hint)
	}
	return fmt.Sprintf("[%s] %s", f.Severity, f.Message)
}

// CIChecker validates a CI file against a workflow
type CIChecker interface {
	Check(wf *workflow.Workflow, ciData []byte) ([]Finding, error)
	Platform() string
}

// AutoDetect detects the CI platform from file path and content, returns appropriate checker
func AutoDetect(ciFilePath string) (CIChecker, []byte, error) {
	data, err := os.ReadFile(ciFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CI file: %w", err)
	}

	base := filepath.Base(ciFilePath)
	dir := filepath.Dir(ciFilePath)

	// GitLab CI: .gitlab-ci.yml
	if base == ".gitlab-ci.yml" || strings.HasSuffix(base, ".gitlab-ci.yml") {
		return &GitLabChecker{}, data, nil
	}

	// GitHub Actions: .github/workflows/*.yml
	if strings.Contains(filepath.ToSlash(dir), ".github/workflows") || strings.Contains(filepath.ToSlash(ciFilePath), ".github/workflows") {
		return &GitHubChecker{}, data, nil
	}

	// Ambiguous — inspect YAML keys
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("failed to parse CI file as YAML: %w", err)
	}

	_, hasStages := raw["stages"]
	_, hasOn := raw["on"]
	_, hasJobs := raw["jobs"]

	if hasStages {
		return &GitLabChecker{}, data, nil
	}
	if hasOn && hasJobs {
		return &GitHubChecker{}, data, nil
	}

	return nil, nil, fmt.Errorf("unable to detect CI platform from file %q — rename to .gitlab-ci.yml or place under .github/workflows/", ciFilePath)
}

// RunCheck is a convenience helper that loads workflow + CI file and runs checks
func RunCheck(workflowFile, ciFile string) ([]Finding, string, error) {
	wfData, err := os.ReadFile(workflowFile)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read workflow file: %w", err)
	}

	wf, err := workflow.LoadWorkflow(wfData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse workflow: %w", err)
	}

	checker, ciData, err := AutoDetect(ciFile)
	if err != nil {
		return nil, "", err
	}

	findings, err := checker.Check(wf, ciData)
	return findings, checker.Platform(), err
}
