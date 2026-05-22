package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/rkuthiala/shiro-automation/internal/config"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/runtime"
	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
	"github.com/rkuthiala/shiro-automation/pkg/ai"
	"github.com/rkuthiala/shiro-automation/pkg/data"
	"github.com/rkuthiala/shiro-automation/pkg/git"
	printpkg "github.com/rkuthiala/shiro-automation/pkg/print"
	"github.com/rkuthiala/shiro-automation/pkg/shell"
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
	fresh := flagSet.Bool("fresh", false, "Delete existing workflow state before running")
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
		log.Fatalf("Failed to register modules: %v", err)
	}

	// Register AI providers
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

	// Always set state store for pause/resume support
	executor.SetStateStore(stateStore)

	// Register data module with state store
	dataModule := data.NewDataModule(stateStore)
	if err := registry.Register("data.store", dataModule); err != nil {
		log.Fatalf("Failed to register data module: %v", err)
	}
	if err := registry.Register("data.load", dataModule); err != nil {
		log.Fatalf("Failed to register data.load module: %v", err)
	}
	if err := registry.Register("data.delete", dataModule); err != nil {
		log.Fatalf("Failed to register data.delete module: %v", err)
	}
	if err := registry.Register("data.exists", dataModule); err != nil {
		log.Fatalf("Failed to register data.exists module: %v", err)
	}

	// Load environment variables
	env := loadEnvironment()

	// Execute workflow
	ctx := context.Background()
	if *fresh {
		exists, err := stateStore.Exists(ctx, wf.Name)
		if err != nil {
			log.Fatalf("Failed to check existing state: %v", err)
		}
		if exists {
			if err := stateStore.Delete(ctx, wf.Name); err != nil {
				log.Fatalf("Failed to delete existing state: %v", err)
			}
			logger.Printf("Deleted existing workflow state for fresh run: %s", wf.Name)
		}
	}

	execCtx, err := executor.Execute(ctx, wf, wf.Inputs, env)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	// Save state
	if err := stateStore.Save(ctx, wf.Name, execCtx); err != nil {
		logger.Printf("Failed to save state: %v", err)
	}

	// Output results (respect quiet mode)
	outputResults(execCtx, wf)
}

// registerBuiltInModules registers the built-in modules
func registerBuiltInModules(registry *modules.Registry) error {
	// Register Slack module
	skipTLSVerify := os.Getenv("SHIRO_SKIP_TLS_VERIFY") == "true"
	slackModule := slack.NewSlackModule(skipTLSVerify)
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

	// Register Shell module
	shellModule := shell.NewShellModule()
	if err := registry.Register("shell.exec", shellModule); err != nil {
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
		defaultModel, _ := modelDef["model"].(string)

		modelType, ok := modelDef["type"].(string)
		if !ok {
			modelType, ok = modelDef["provider"].(string)
			if !ok {
				logger.Printf("Skipping model %s: missing type/provider field", modelName)
				continue
			}
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
				Model:   defaultModel,
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
			skipTLSVerify := false
			if skipTLS, ok := modelDef["skip_tls_verify"].(bool); ok {
				skipTLSVerify = skipTLS
			}
			providerConfig := &ai.ProviderConfig{
				Type:          "openai",
				BaseURL:       baseURL,
				APIKey:        apiKey,
				Model:         defaultModel,
				SkipTLSVerify: skipTLSVerify,
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

		aiModule.AddProviderWithDefaultModel(modelName, provider, defaultModel)
	}

	return aiModule, nil
}

// loadEnvironment loads environment variables from the system and CI context
func loadEnvironment() map[string]string {
	env := make(map[string]string)

	// Load system environment variables
	for _, e := range os.Environ() {
		if k, v := splitEnv(e); k != "" {
			env[k] = v
		}
	}

	return env
}

// splitEnv splits an environment variable into key and value
func splitEnv(envVar string) (string, string) {
	for i := 0; i < len(envVar); i++ {
		if envVar[i] == '=' {
			return envVar[:i], envVar[i+1:]
		}
	}
	return envVar, ""
}

// outputResults outputs workflow execution results (respects quiet mode)
func outputResults(execCtx *workflow.ExecutionContext, wf *workflow.Workflow) {
	// Check if workflow is in quiet mode
	if wf.Settings.QuietMode {
		return
	}

	fmt.Println("\n=== Workflow Results ===")

	stepIDs := make([]string, 0, len(execCtx.Steps))
	for stepID := range execCtx.Steps {
		stepIDs = append(stepIDs, stepID)
	}
	sort.Strings(stepIDs)

	for _, stepID := range stepIDs {
		result := execCtx.Steps[stepID]
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
	fmt.Println(`_____/\\\\\\\\\\\____/\\\________/\\\__/\\\\\\\\\\\____/\\\\\\\\\___________/\\\\\______        
 ___/\\\/////////\\\_\/\\\_______\/\\\_\/////\\\///___/\\\///////\\\_______/\\\///\\\____       
  __\//\\\______\///__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\/\\\_____/\\\/__\///\\\__      
   ___\////\\\_________\/\\\\\\\\\\\\\\\_____\/\\\_____\/\\\\\\\\\\\/_____/\\\______\//\\\_     
    ______\////\\\______\/\\\/////////\\\_____\/\\\_____\/\\\//////\\\____\/\\\_______\/\\\_    
     _________\////\\\___\/\\\_______\/\\\_____\/\\\_____\/\\\____\//\\\___\//\\\______/\\\__   
      __/\\\______\//\\\__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\//\\\___\///\\\__/\\\____  
       _\///\\\\\\\\\\\/___\/\\\_______\/\\\__/\\\\\\\\\\\_\/\\\______\//\\\____\///\\\\\/_____ 
        ___\///////////_____\///________\///__\///////////__\///________\///_______\/////_______`)
	fmt.Println("Shiro - AI-Native CI Workflow Runtime")
	fmt.Println()
	fmt.Println("Created by: Rajit Kuthiala")
	fmt.Println("https://www.linkedin.com/in/rajitkuthiala/")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  shiro run [options]")
	fmt.Println("  shiro [options] (shorthand)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -workflow <file>   Path to workflow JSON file (auto-detected if not specified)")
	fmt.Println("  -config <file>     Path to model configuration file (auto-detected if not specified)")
	fmt.Println("  -state-store <type> State store type: memory, filesystem, gitlab (default \"gitlab\")")
	fmt.Println("  -shiro-dir <path>  Path to .shiro directory (default \".shiro\")")
	fmt.Println("  -fresh             Delete existing workflow state before running")
	fmt.Println("  -help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shiro run examples/print-example.json")
	fmt.Println("  shiro run examples/mr-review.json -config configs/models.yaml")
	fmt.Println("  shiro examples/github-mr-review.json -state-store filesystem")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/rajitk13/shiro-automation")
}
