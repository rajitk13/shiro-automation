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

	modelConfig := make(map[string]map[string]interface{})
	if cfg.ConfigFile != "" {
		modelConfig, err = config.LoadModelConfig(cfg.ConfigFile)
		if err != nil {
			log.Fatalf("Config validation failed: %v", err)
		}
	}

	if err := validateWorkflowFile(cfg.WorkflowFile, modelConfig); err != nil {
		printValidationError(err)
		os.Exit(1)
	}

	fmt.Printf("Workflow validation passed: %s\n", cfg.WorkflowFile)
}

func validateWorkflowFile(path string, modelConfig map[string]map[string]interface{}) error {
	workflowData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	wf, err := workflow.LoadWorkflow(workflowData)
	if err != nil {
		return err
	}

	validationErrors := errors.ValidationErrors{}
	if err := wf.Validate(); err != nil {
		if errs, ok := err.(errors.ValidationErrors); ok {
			validationErrors = append(validationErrors, errs...)
		} else {
			return err
		}
	}
	validationErrors = append(validationErrors, validateAIWorkflowConfig(wf, modelConfig)...)
	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

func validateAIWorkflowConfig(wf *workflow.Workflow, modelConfig map[string]map[string]interface{}) errors.ValidationErrors {
	validationErrors := errors.ValidationErrors{}
	for _, step := range wf.Steps {
		if step.Type != "ai.generate" {
			continue
		}

		prompt, _ := step.Config["prompt"].(string)
		if prompt == "" {
			validationErrors = append(validationErrors, errors.NewValidationError(fmt.Sprintf("steps[%s].config.prompt", step.ID), "prompt is required for ai.generate", nil))
		}

		providerName, _ := step.Config["provider"].(string)
		if providerName == "" {
			providerName = resolveDefaultProviderName(modelConfig)
		}

		providerConfig, ok := modelConfig[providerName]
		if !ok {
			validationErrors = append(validationErrors, errors.NewValidationError(fmt.Sprintf("steps[%s].config.provider", step.ID), fmt.Sprintf("provider %q not found in model config", providerName), nil))
			continue
		}

		stepModel, _ := step.Config["model"].(string)
		configModel, _ := providerConfig["model"].(string)
		if stepModel == "" && configModel == "" {
			validationErrors = append(validationErrors, errors.NewValidationError(fmt.Sprintf("steps[%s].config.model", step.ID), fmt.Sprintf("model is required for provider %q: set config.model in the workflow step or model in config", providerName), nil))
		}
	}
	return validationErrors
}

func resolveDefaultProviderName(modelConfig map[string]map[string]interface{}) string {
	if _, ok := modelConfig["default"]; ok {
		return "default"
	}
	if len(modelConfig) == 1 {
		for name := range modelConfig {
			return name
		}
	}
	return "default"
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
