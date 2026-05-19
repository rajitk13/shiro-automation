package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// GitModule implements git operations
type GitModule struct{}

// NewGitModule creates a new Git module
func NewGitModule() *GitModule {
	return &GitModule{}
}

// Run executes a git operation
func (m *GitModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	operation, ok := wfStep.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation is required")
	}

	switch operation {
	case "diff":
		return m.runDiff(ctx, wfStep.Config)
	case "get_changes":
		return m.getChanges(ctx, wfStep.Config)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// runDiff gets the git diff
func (m *GitModule) runDiff(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	// Get target branch (default to HEAD)
	target, _ := config["target"].(string)
	if target == "" {
		target = "HEAD"
	}

	// Get base branch (default to empty for working directory changes)
	base, _ := config["base"].(string)

	var args []string
	if base != "" {
		args = []string{"diff", base, target}
	} else {
		args = []string{"diff", target}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w, stderr: %s", err, stderr.String())
	}

	diff := stdout.String()

	return map[string]interface{}{
		"diff":    diff,
		"base":    base,
		"target":  target,
		"success": true,
	}, nil
}

// getChanges gets the list of changed files
func (m *GitModule) getChanges(ctx context.Context, config map[string]interface{}) (map[string]interface{}, error) {
	// Get target branch (default to HEAD)
	target, _ := config["target"].(string)
	if target == "" {
		target = "HEAD"
	}

	// Get base branch (default to empty for working directory changes)
	base, _ := config["base"].(string)

	var args []string
	if base != "" {
		args = []string{"diff", "--name-only", base, target}
	} else {
		args = []string{"diff", "--name-only", target}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only failed: %w, stderr: %s", err, stderr.String())
	}

	files := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	
	// Filter empty strings
	var changedFiles []string
	for _, f := range files {
		if f != "" {
			changedFiles = append(changedFiles, f)
		}
	}

	return map[string]interface{}{
		"files":   changedFiles,
		"count":   len(changedFiles),
		"base":    base,
		"target":  target,
		"success": true,
	}, nil
}

// Metadata returns module metadata
func (m *GitModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "git.diff",
		Description: "Performs git operations like diff and getting changed files",
		InputSchema: map[string]modules.SchemaField{
			"operation": {
				Type:        "string",
				Description: "Operation to perform: diff, get_changes",
				Required:    true,
			},
			"base": {
				Type:        "string",
				Description: "Base branch or commit",
				Required:    false,
			},
			"target": {
				Type:        "string",
				Description: "Target branch or commit (default: HEAD)",
				Required:    false,
				Default:     "HEAD",
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"success": {
				Type:        "boolean",
				Description: "Whether the operation succeeded",
				Required:    true,
			},
		},
	}
}
