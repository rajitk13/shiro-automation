package shell

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// ShellModule implements shell script execution
type ShellModule struct{}

// NewShellModule creates a new shell module
func NewShellModule() *ShellModule {
	return &ShellModule{}
}

// Run executes the shell command or script
func (m *ShellModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	// Extract configuration
	var command string
	var args []string
	var scriptLines []string
	var shell string = "bash"
	var workingDir string
	var envVars map[string]string
	var timeoutStr string = "5m"
	var captureOutput bool = true

	// Get command or script
	if cmd, ok := wfStep.Config["command"].(string); ok {
		command = cmd
	}

	if script, ok := wfStep.Config["script"].([]interface{}); ok {
		for _, line := range script {
			if s, ok := line.(string); ok {
				scriptLines = append(scriptLines, s)
			}
		}
	}

	if argsList, ok := wfStep.Config["args"].([]interface{}); ok {
		for _, arg := range argsList {
			if a, ok := arg.(string); ok {
				args = append(args, a)
			}
		}
	}

	if s, ok := wfStep.Config["shell"].(string); ok {
		shell = s
	}

	if wd, ok := wfStep.Config["working_dir"].(string); ok {
		workingDir = wd
	}

	if env, ok := wfStep.Config["env"].(map[string]interface{}); ok {
		envVars = make(map[string]string)
		for k, v := range env {
			if str, ok := v.(string); ok {
				envVars[k] = str
			}
		}
	}

	if t, ok := wfStep.Config["timeout"].(string); ok {
		timeoutStr = t
	}

	if cap, ok := wfStep.Config["capture_output"].(bool); ok {
		captureOutput = cap
	}

	// Resolve variables if we have execution context
	if execCtx, ok := stepCtx.(*workflow.ExecutionContext); ok {
		resolver := workflow.NewVariableResolver(execCtx)

		// Resolve command
		if command != "" {
			resolved, err := resolver.Resolve(command)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve command variables: %w", err)
			}
			if str, ok := resolved.(string); ok {
				command = str
			}
		}

		// Resolve working dir
		if workingDir != "" {
			resolved, err := resolver.Resolve(workingDir)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve working_dir variables: %w", err)
			}
			if str, ok := resolved.(string); ok {
				workingDir = str
			}
		}

		// Resolve env vars
		for k, v := range envVars {
			resolved, err := resolver.Resolve(v)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve env var %s: %w", k, err)
			}
			if str, ok := resolved.(string); ok {
				envVars[k] = str
			}
		}

		// Resolve script lines
		for i, line := range scriptLines {
			resolved, err := resolver.Resolve(line)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve script line %d: %w", i, err)
			}
			if str, ok := resolved.(string); ok {
				scriptLines[i] = str
			}
		}
	}

	// Parse timeout
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout format: %w", err)
	}

	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd

	// Build command
	if len(scriptLines) > 0 {
		// Inline script mode
		script := strings.Join(scriptLines, "\n")
		shellCmd := shell
		if shell == "bash" || shell == "sh" {
			cmd = exec.CommandContext(cmdCtx, shellCmd, "-c", script)
		} else if shell == "python" || shell == "python3" {
			cmd = exec.CommandContext(cmdCtx, shellCmd, "-c", script)
		} else {
			cmd = exec.CommandContext(cmdCtx, shellCmd, script)
		}
	} else if command != "" {
		// External command mode
		cmd = exec.CommandContext(cmdCtx, command, args...)
	} else {
		return nil, fmt.Errorf("either 'command' or 'script' is required")
	}

	// Set working directory
	if workingDir != "" {
		// If relative, make it relative to CI_PROJECT_DIR or current dir
		if !filepath.IsAbs(workingDir) {
			projectDir := os.Getenv("CI_PROJECT_DIR")
			if projectDir == "" {
				projectDir = "."
			}
			workingDir = filepath.Join(projectDir, workingDir)
		}
		cmd.Dir = workingDir
	} else {
		// Default to CI_PROJECT_DIR
		projectDir := os.Getenv("CI_PROJECT_DIR")
		if projectDir != "" {
			cmd.Dir = projectDir
		}
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	if captureOutput {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Run command
	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else if cmdCtx.Err() == context.DeadlineExceeded {
			exitCode = -1 // Timeout
			err = fmt.Errorf("command timed out after %s", timeoutStr)
		} else {
			exitCode = -2 // Other error
		}
	}

	result := map[string]interface{}{
		"success":   err == nil,
		"exit_code": exitCode,
		"command":   command,
		"shell":     shell,
	}

	if captureOutput {
		result["stdout"] = stdout.String()
		result["stderr"] = stderr.String()
	}

	if err != nil {
		result["error"] = err.Error()
	}

	return result, nil
}

// Metadata returns module metadata
func (m *ShellModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "shell.exec",
		Description: "Execute shell scripts and commands in GitLab CI context",
		InputSchema: map[string]modules.SchemaField{
			"command": {
				Type:        "string",
				Description: "External command or script file to execute",
				Required:    false,
			},
			"script": {
				Type:        "array",
				Description: "Inline script lines to execute",
				Required:    false,
			},
			"args": {
				Type:        "array",
				Description: "Arguments for external command",
				Required:    false,
			},
			"shell": {
				Type:        "string",
				Description: "Shell interpreter (bash, sh, python, python3)",
				Required:    false,
				Default:     "bash",
			},
			"working_dir": {
				Type:        "string",
				Description: "Working directory for command execution",
				Required:    false,
			},
			"env": {
				Type:        "object",
				Description: "Environment variables to set",
				Required:    false,
			},
			"timeout": {
				Type:        "string",
				Description: "Command timeout (e.g., '5m', '30s')",
				Required:    false,
				Default:     "5m",
			},
			"capture_output": {
				Type:        "boolean",
				Description: "Whether to capture stdout/stderr or print to console",
				Required:    false,
				Default:     true,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"success": {
				Type:        "boolean",
				Description: "Whether command executed successfully",
				Required:    true,
			},
			"exit_code": {
				Type:        "number",
				Description: "Exit code from command",
				Required:    true,
			},
			"stdout": {
				Type:        "string",
				Description: "Standard output (if capture_output=true)",
				Required:    false,
			},
			"stderr": {
				Type:        "string",
				Description: "Standard error (if capture_output=true)",
				Required:    false,
			},
			"command": {
				Type:        "string",
				Description: "Command that was executed",
				Required:    true,
			},
			"error": {
				Type:        "string",
				Description: "Error message if command failed",
				Required:    false,
			},
		},
	}
}
