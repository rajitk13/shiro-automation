package print

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
)

// PrintModule implements the print module
type PrintModule struct{}

// NewPrintModule creates a new print module
func NewPrintModule() *PrintModule {
	return &PrintModule{}
}

// Run executes the print operation
func (m *PrintModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	// Type assert to get the step
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	// Extract configuration
	message, ok := wfStep.Config["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message is required")
	}

	level := "info"
	if l, ok := wfStep.Config["level"].(string); ok {
		level = l
	}

	format := "text"
	if f, ok := wfStep.Config["format"].(string); ok {
		format = f
	}

	// Resolve variables in message if we have execution context
	if execCtx, ok := stepCtx.(*workflow.ExecutionContext); ok {
		resolver := workflow.NewVariableResolver(execCtx)
		resolved, err := resolver.Resolve(message)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve variables: %w", err)
		}
		if resolvedStr, ok := resolved.(string); ok {
			message = resolvedStr
		}
	}

	// Print based on level and format
	var output string
	var color string

	switch level {
	case "info":
		color = colorGreen
	case "debug":
		color = colorGray
	case "error":
		color = colorRed
	case "warning":
		color = colorYellow
	default:
		color = colorGreen
	}

	if format == "json" {
		// JSON format
		jsonOutput := map[string]interface{}{
			"level":   level,
			"message": message,
		}
		jsonData, err := json.MarshalIndent(jsonOutput, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		output = string(jsonData)
		fmt.Printf("%s[%s]%s %s\n", color, level, colorReset, output)
	} else {
		// Text format with color
		fmt.Printf("%s[%s]%s %s\n", color, level, colorReset, message)
		output = message
	}

	return map[string]interface{}{
		"printed": true,
		"message": output,
		"level":   level,
		"status":  "success",
	}, nil
}

// Metadata returns module metadata
func (m *PrintModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "print",
		Description: "Prints output to console with optional log levels and colors",
		InputSchema: map[string]modules.SchemaField{
			"message": {
				Type:        "string",
				Description: "Message to print",
				Required:    true,
			},
			"level": {
				Type:        "string",
				Description: "Log level (info, debug, error, warning)",
				Required:    false,
				Default:     "info",
			},
			"format": {
				Type:        "string",
				Description: "Output format (text, json)",
				Required:    false,
				Default:     "text",
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"printed": {
				Type:        "boolean",
				Description: "Whether the message was printed successfully",
				Required:    true,
			},
			"message": {
				Type:        "string",
				Description: "Message content that was printed",
				Required:    true,
			},
			"level": {
				Type:        "string",
				Description: "Log level used",
				Required:    true,
			},
			"status": {
				Type:        "string",
				Description: "Status of the operation",
				Required:    true,
			},
		},
	}
}
