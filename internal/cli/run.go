package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rkuthiala/shiro-automation/internal/config"
	"github.com/rkuthiala/shiro-automation/internal/github"
	"github.com/rkuthiala/shiro-automation/internal/gitlab"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/runtime"
	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
	"github.com/rkuthiala/shiro-automation/pkg/ai"
	"github.com/rkuthiala/shiro-automation/pkg/git"
	printpkg "github.com/rkuthiala/shiro-automation/pkg/print"
	"github.com/rkuthiala/shiro-automation/pkg/slack"
)

// RunCommand handles the workflow execution command
func RunCommand(args []string) {
	// Parse flags
	flagSet := flag.NewFlagSet("run", flag.ExitOnError)
	workflowFile := flagSet.String("workflow", "", "Path to workflow JSON file")
	configFile := flagSet.String("config", "", "Path to model configuration file")
	stateStoreType := flagSet.String("state-store", "gitlab", "State store type (memory, filesystem, gitlab)")
	shiroDir := flagSet.String("shiro-dir", ".shiro", "Path to .shiro directory")
	showHelp := flagSet.Bool("help", false, "Show help information")
	flagSet.Parse(args)

	// Load configuration
	cfg, err := config.LoadConfig(*shiroDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override with CLI flags if provided
	if *workflowFile != "" {
		cfg.WorkflowFile = *workflowFile
	}
	if *configFile != "" {
		cfg.ConfigFile = *configFile
	}
	if *stateStoreType != "" {
		cfg.StateStore = *stateStoreType
	}

	if *showHelp {
		printRunHelp()
		os.Exit(0)
	}

	if cfg.WorkflowFile == "" {
		printRunHelp()
		os.Exit(1)
	}

	logger := log.New(os.Stdout, "[Shiro] ", log.LstdFlags)

	// Load workflow
	workflowData, err := os.ReadFile(cfg.WorkflowFile)
	if err != nil {
		log.Fatalf("Failed to read workflow file: %v", err)
	}

	wf, err := workflow.LoadWorkflow(workflowData)
	if err != nil {
		log.Fatalf("Failed to load workflow: %v", err)
	}

	if err := wf.Validate(); err != nil {
		log.Fatalf("Workflow validation failed: %v", err)
	}

	logger.Printf("Loaded workflow: %s", wf.Name)

	// Load model configuration
	modelConfig, err := config.LoadModelConfig(cfg.ConfigFile)
	if err != nil {
		log.Printf("Warning: Failed to load model config: %v", err)
		modelConfig = make(map[string]map[string]interface{})
	}

	// Create module registry
	registry := modules.NewRegistry()

	// Load HTTP modules from registry
	registryPath := config.GetRegistryPath(cfg.ShiroDir)
	httpClient := modules.NewHTTPModuleClient(30) // 30 second timeout
	discoverer := modules.NewDiscoverer(registryPath, httpClient)
	if err := discoverer.LoadRegistry(); err != nil {
		logger.Printf("Warning: Failed to load module registry: %v", err)
	} else {
		// Register HTTP modules
		moduleNames := discoverer.ListModules()
		for _, name := range moduleNames {
			moduleConfig, err := discoverer.GetModuleConfig(name)
			if err != nil {
				logger.Printf("Warning: Failed to get config for module %s: %v", name, err)
				continue
			}
			if moduleConfig.Type == "http" {
				// Handle both single endpoint and multiple endpoints
				endpoints := moduleConfig.Endpoints
				if len(endpoints) == 0 && moduleConfig.Endpoint != "" {
					endpoints = []string{moduleConfig.Endpoint}
				}
				if len(endpoints) > 0 {
					httpModuleConfig := &modules.HTTPModuleConfig{
						Name:     moduleConfig.Name,
						Endpoint: endpoints[0], // Use first endpoint for now
						Config:   make(map[string]interface{}),
					}
					registry.RegisterHTTPModule(name, httpModuleConfig)
				}
			}
		}
	}

	// Register built-in modules
	if err := registerBuiltInModules(registry); err != nil {
		log.Fatalf("Failed to register built-in modules: %v", err)
	}

	// Register AI module with providers
	aiModule, err := registerAIProviders(modelConfig, logger)
	if err != nil {
		log.Fatalf("Failed to register AI providers: %v", err)
	}

	if err := registry.Register("ai.generate", aiModule); err != nil {
		log.Fatalf("Failed to register AI module: %v", err)
	}

	// Create executor
	executor := runtime.NewExecutor(registry, logger)

	// Create state store
	stateFactory := state.NewStoreFactory()
	stateStore, err := stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	// Load environment variables
	env := loadEnvironment()

	// Execute workflow
	ctx := context.Background()
	execCtx, err := executor.Execute(ctx, wf, wf.Inputs, env)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Save state
	if err := stateStore.Save(ctx, wf.Name, execCtx); err != nil {
		logger.Printf("Failed to save state: %v", err)
	}

	// Output results
	outputResults(execCtx)
}

// registerBuiltInModules registers the built-in modules
func registerBuiltInModules(registry *modules.Registry) error {
	// Register Slack module
	slackModule := slack.NewSlackModule()
	if err := registry.Register("slack.notify", slackModule); err != nil {
		return err
	}

	// Register Git module
	gitModule := git.NewGitModule()
	if err := registry.Register("git.diff", gitModule); err != nil {
		return err
	}

	// Register Print module
	printModule := printpkg.NewPrintModule()
	if err := registry.Register("print", printModule); err != nil {
		return err
	}

	return nil
}

// registerAIProviders initializes AI providers from config
func registerAIProviders(modelConfig map[string]map[string]interface{}, logger *log.Logger) (*ai.AIModule, error) {
	aiModule := ai.NewAIModule()

	// Initialize AI providers from config
	for modelName, modelDef := range modelConfig {
		var provider ai.Provider
		var err error

		// Check if type field exists
		modelType, ok := modelDef["type"].(string)
		if !ok {
			logger.Printf("Skipping model %s: missing type field", modelName)
			continue
		}

		switch modelType {
		case "ollama":
			baseURL, ok := modelDef["base_url"].(string)
			if !ok {
				logger.Printf("Skipping model %s: missing base_url field", modelName)
				continue
			}
			providerConfig := &ai.ProviderConfig{
				Type:    "ollama",
				BaseURL: baseURL,
				Model:   modelName,
			}
			provider, err = ai.NewOllamaProvider(providerConfig)
		case "openai":
			baseURL, ok := modelDef["base_url"].(string)
			if !ok {
				logger.Printf("Skipping model %s: missing base_url field", modelName)
				continue
			}
			apiKey, ok := modelDef["api_key"].(string)
			if !ok {
				logger.Printf("Skipping model %s: missing api_key field", modelName)
				continue
			}
			providerConfig := &ai.ProviderConfig{
				Type:    "openai",
				BaseURL: baseURL,
				APIKey:  apiKey,
				Model:   modelName,
			}
			provider, err = ai.NewOpenAIProvider(providerConfig)
		default:
			logger.Printf("Unknown provider type: %s", modelType)
			continue
		}

		if err != nil {
			logger.Printf("Failed to create provider for %s: %v", modelName, err)
			continue
		}

		aiModule.AddProvider(modelName, provider)
		defer provider.Close()
	}

	return aiModule, nil
}

// loadEnvironment loads environment variables
func loadEnvironment() map[string]string {
	env := make(map[string]string)

	// Load all environment variables
	for _, envVar := range os.Environ() {
		parts := splitEnv(envVar)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Add GitLab-specific environment variables
	env["CI_PROJECT_ID"] = gitlab.GetProjectID()
	env["CI_MERGE_REQUEST_IID"] = gitlab.GetMRID()
	env["CI_COMMIT_SHA"] = gitlab.GetCommitSHA()
	env["CI_COMMIT_REF_NAME"] = gitlab.GetBranch()

	// Add GitHub-specific environment variables
	env["GITHUB_REPOSITORY"] = github.GetRepository()
	env["GITHUB_PR_NUMBER"] = github.GetPRNumber()
	env["GITHUB_SHA"] = github.GetCommitSHA()
	env["GITHUB_REF_NAME"] = github.GetBranch()
	env["GITHUB_REPOSITORY_OWNER"] = github.GetOwner()

	return env
}

// splitEnv splits an environment variable into key and value
func splitEnv(envVar string) []string {
	for i := 0; i < len(envVar); i++ {
		if envVar[i] == '=' {
			return []string{envVar[:i], envVar[i+1:]}
		}
	}
	return []string{envVar}
}

// outputResults outputs workflow execution results
func outputResults(execCtx *workflow.ExecutionContext) {
	fmt.Println("\n=== Workflow Results ===")

	for stepID, result := range execCtx.Steps {
		fmt.Printf("\nStep: %s\n", stepID)
		fmt.Printf("  Success: %v\n", result.Success)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		if len(result.Output) > 0 {
			fmt.Printf("  Output:\n")
			outputJSON, _ := json.MarshalIndent(result.Output, "    ", "  ")
			fmt.Printf("    %s\n", string(outputJSON))
		}
	}
}

// printRunHelp prints help for the run command
func printRunHelp() {
	fmt.Println("Usage: shiro run [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -workflow string")
	fmt.Println("        Path to workflow JSON file (auto-detected if not specified)")
	fmt.Println("  -config string")
	fmt.Println("        Path to model configuration file (auto-detected if not specified)")
	fmt.Println("  -state-store string")
	fmt.Println("        State store type (memory, filesystem, gitlab) (default \"gitlab\")")
	fmt.Println("  -shiro-dir string")
	fmt.Println("        Path to .shiro directory (default \".shiro\")")
	fmt.Println("  -help")
	fmt.Println("        Show help information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shiro run")
	fmt.Println("  shiro run -workflow examples/simple-test.json")
	fmt.Println("  shiro run -config configs/models.yaml -state-store filesystem")
}
