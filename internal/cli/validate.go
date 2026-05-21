package cli

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rkuthiala/shiro-automation/internal/config"
	"github.com/rkuthiala/shiro-automation/internal/errors"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

func ValidateCommand(args []string) {
	flagSet := flag.NewFlagSet("validate", flag.ExitOnError)
	workflowFile := flagSet.String("workflow", "", "Path to workflow JSON file")
	configFile := flagSet.String("config", "", "Path to model configuration file")
	shiroDir := flagSet.String("shiro-dir", ".shiro", "Path to .shiro directory")
	showHelp := flagSet.Bool("help", false, "Show help information")
	flagSet.Parse(args)

	if *showHelp {
		printValidateHelp()
		os.Exit(0)
	}

	cfg, err := config.LoadConfig(*shiroDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if *workflowFile != "" {
		cfg.WorkflowFile = *workflowFile
	}
	if *configFile != "" {
		cfg.ConfigFile = *configFile
	}

	if cfg.WorkflowFile == "" {
		printValidateHelp()
		os.Exit(1)
	}

	if err := validateWorkflowFile(cfg.WorkflowFile); err != nil {
		printValidationError(err)
		os.Exit(1)
	}

	if cfg.ConfigFile != "" {
		if _, err := config.LoadModelConfig(cfg.ConfigFile); err != nil {
			log.Fatalf("Config validation failed: %v", err)
		}
	}

	fmt.Printf("Workflow validation passed: %s\n", cfg.WorkflowFile)
}

func validateWorkflowFile(path string) error {
	workflowData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	wf, err := workflow.LoadWorkflow(workflowData)
	if err != nil {
		return err
	}

	return wf.Validate()
}

func printValidationError(err error) {
	if validationErrors, ok := err.(errors.ValidationErrors); ok {
		fmt.Println("Workflow validation failed:")
		for _, validationError := range validationErrors {
			fmt.Printf("  - %s: %s\n", validationError.Field, validationError.Message)
		}
		return
	}

	log.Printf("Workflow validation failed: %v", err)
}

func printValidateHelp() {
	fmt.Println("Usage:")
	fmt.Println("  shiro validate [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -workflow <file>  Path to workflow JSON file (auto-detected if not specified)")
	fmt.Println("  -config <file>    Path to model configuration file (auto-detected if not specified)")
	fmt.Println("  -shiro-dir <path> Path to .shiro directory (default \".shiro\")")
	fmt.Println("  -help             Show this help message")
}
