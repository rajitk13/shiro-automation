package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		initShiro(args)
	case "run":
		runWorkflow(args)
	case "add":
		runAddCommand(args)
	case "search":
		runSearchCommand(args)
	case "list":
		runListCommand(args)
	case "remove":
		runRemoveCommand(args)
	case "install":
		runInstallCommand(args)
	case "info":
		runInfoCommand(args)
	case "docs":
		runDocsCommand(args)
	case "help", "-help", "--help":
		printHelp()
	default:
		// Try to run as workflow if no recognized command
		runWorkflow(os.Args[1:])
	}
}

func runWorkflow(args []string) {
	// Parse flags
	flagSet := flag.NewFlagSet("run", flag.ExitOnError)
	workflowFile := flagSet.String("workflow", "", "Path to workflow JSON file")
	configFile := flagSet.String("config", "", "Path to model configuration file")
	stateStoreType := flagSet.String("state-store", "gitlab", "State store type (memory, filesystem, gitlab)")
	shiroDir := flagSet.String("shiro-dir", ".shiro", "Path to .shiro directory")
	showHelp := flagSet.Bool("help", false, "Show help information")
	flagSet.Parse(args)

	// Auto-detect workflow file if not specified
	if *workflowFile == "" {
		// Check for .shiro/workflow.json
		if _, err := os.Stat(fmt.Sprintf("%s/workflow.json", *shiroDir)); err == nil {
			*workflowFile = fmt.Sprintf("%s/workflow.json", *shiroDir)
		} else if _, err := os.Stat(".shiro/workflow.json"); err == nil {
			*workflowFile = ".shiro/workflow.json"
		} else if _, err := os.Stat("workflow.json"); err == nil {
			*workflowFile = "workflow.json"
		} else if flagSet.NArg() > 0 {
			*workflowFile = flagSet.Arg(0)
		} else {
			log.Fatal("No workflow file found. Please create .shiro/workflow.json or specify with -workflow flag")
		}
	}

	// Auto-detect config file if not specified
	if *configFile == "" {
		// Check for .shiro/config.yaml
		if _, err := os.Stat(fmt.Sprintf("%s/config.yaml", *shiroDir)); err == nil {
			*configFile = fmt.Sprintf("%s/config.yaml", *shiroDir)
		} else if _, err := os.Stat(".shiro/config.yaml"); err == nil {
			*configFile = ".shiro/config.yaml"
		} else if _, err := os.Stat("configs/models.yaml"); err == nil {
			*configFile = "configs/models.yaml"
		}
	}

	// Use shiroDir for module registry
	registryPath := fmt.Sprintf("%s/modules/registry.yaml", *shiroDir)
	if _, err := os.Stat(registryPath); err != nil {
		registryPath = "modules/registry.yaml" // Fallback to default
	}

	if *showHelp {
		printRunHelp()
		os.Exit(0)
	}

	if *workflowFile == "" {
		printRunHelp()
		os.Exit(1)
	}

	logger := log.New(os.Stdout, "[Shiro] ", log.LstdFlags)

	// Load workflow
	workflowData, err := os.ReadFile(*workflowFile)
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
	modelConfig := loadModelConfig(*configFile)

	// Create module registry
	registry := modules.NewRegistry()

	// Load HTTP modules from registry
	httpClient := modules.NewHTTPModuleClient(30 * time.Second)
	discoverer := modules.NewDiscoverer(registryPath, httpClient)
	if err := discoverer.LoadRegistry(); err != nil {
		logger.Printf("Warning: Failed to load module registry: %v", err)
	} else {
		// Register HTTP modules
		moduleNames := discoverer.ListModules()
		for _, name := range moduleNames {
			config, err := discoverer.GetModuleConfig(name)
			if err != nil {
				logger.Printf("Warning: Failed to get config for module %s: %v", name, err)
				continue
			}
			if config.Type == "http" {
				// Handle both single endpoint and multiple endpoints
				endpoints := config.Endpoints
				if len(endpoints) == 0 && config.Endpoint != "" {
					endpoints = []string{config.Endpoint}
				}
				if len(endpoints) > 0 {
					httpModuleConfig := &modules.HTTPModuleConfig{
						Name:     config.Name,
						Endpoint: endpoints[0], // Use first endpoint for now
						Config:   make(map[string]interface{}),
					}
					registry.RegisterHTTPModule(name, httpModuleConfig)
				}
			}
		}
	}

	// Register Slack module
	slackModule := slack.NewSlackModule()
	if err := registry.Register("slack.notify", slackModule); err != nil {
		log.Fatalf("Failed to register Slack module: %v", err)
	}

	// Register Git module
	gitModule := git.NewGitModule()
	if err := registry.Register("git.diff", gitModule); err != nil {
		log.Fatalf("Failed to register Git module: %v", err)
	}

	// Register Print module
	printModule := printpkg.NewPrintModule()
	if err := registry.Register("print", printModule); err != nil {
		log.Fatalf("Failed to register Print module: %v", err)
	}

	// Register AI module with providers
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

	if err := registry.Register("ai.generate", aiModule); err != nil {
		log.Fatalf("Failed to register AI module: %v", err)
	}

	// Create executor
	executor := runtime.NewExecutor(registry, logger)

	// Create state store
	stateFactory := state.NewStoreFactory()
	stateStore, err := stateFactory.Create(*stateStoreType, map[string]interface{}{})
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

func loadModelConfig(configFile string) map[string]map[string]interface{} {
	if configFile == "" {
		return make(map[string]map[string]interface{})
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("Failed to read config file: %v", err)
		return make(map[string]map[string]interface{})
	}

	var config struct {
		Models map[string]map[string]interface{} `json:"models" yaml:"models"`
	}

	// Detect file format by extension
	ext := strings.ToLower(filepath.Ext(configFile))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &config); err != nil {
			log.Printf("Failed to parse YAML config file: %v", err)
			return make(map[string]map[string]interface{})
		}
	} else {
		if err := json.Unmarshal(data, &config); err != nil {
			log.Printf("Failed to parse JSON config file: %v", err)
			return make(map[string]map[string]interface{})
		}
	}

	return config.Models
}

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

func splitEnv(envVar string) []string {
	for i := 0; i < len(envVar); i++ {
		if envVar[i] == '=' {
			return []string{envVar[:i], envVar[i+1:]}
		}
	}
	return []string{envVar}
}

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

func listModules() {
	registryPath := "modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	modules := discoverer.ListModules()
	fmt.Println("Available modules:")
	for _, name := range modules {
		config, _ := discoverer.GetModuleConfig(name)
		fmt.Printf("  - %s (%s): %s\n", name, config.Type, config.Description)
	}
}

func addModule(args []string) {
	// Check if it's a GitHub URL or a module name
	var moduleName, gitURL string
	var isGitURL bool

	if len(args) > 0 {
		input := args[0]
		if strings.Contains(input, "github.com") {
			isGitURL = true
			gitURL = input
			// Extract module name from URL
			parts := strings.Split(input, "/")
			if len(parts) > 0 {
				moduleName = parts[len(parts)-1]
			}
		} else {
			moduleName = input
			// Auto-discover from official repo
			gitURL = fmt.Sprintf("github.com/rkuthiala/%s-module", moduleName)
			fmt.Printf("Auto-discovering module '%s' from official repository...\n", moduleName)
		}
	} else {
		log.Fatal("Module name or GitHub URL is required")
	}

	// For auto-discovery, search GitHub first
	if !isGitURL {
		token := os.Getenv("GITHUB_TOKEN")
		githubClient := modules.NewGitHubClient(token)
		results, err := githubClient.SearchModules(moduleName)
		if err != nil {
			fmt.Printf("Warning: Failed to search GitHub: %v\n", err)
		} else if len(results) > 0 {
			// Use the first result
			gitURL = results[0].FullName
			fmt.Printf("Found module: %s\n", results[0].Name)
			fmt.Printf("Repository: %s\n", gitURL)
			fmt.Printf("Stars: %d\n", results[0].Stargazers)
			fmt.Printf("Description: %s\n", results[0].Description)
		} else {
			fmt.Printf("Module '%s' not found in official repository.\n", moduleName)
			fmt.Printf("Please provide a GitHub URL: shiro add module github.com/user/%s-module\n", moduleName)
			os.Exit(1)
		}
	}

	// Parse GitHub repository
	repoPath, err := modules.ParseGitHubRepo(gitURL)
	if err != nil {
		log.Fatalf("Invalid GitHub repository format: %v", err)
	}

	// Get module metadata from GitHub
	token := os.Getenv("GITHUB_TOKEN")
	githubClient := modules.NewGitHubClient(token)
	metadata, err := githubClient.GetModuleMetadata(repoPath)
	if err != nil {
		log.Fatalf("Failed to get module metadata: %v", err)
	}

	// Determine module name
	if moduleName == "" {
		parts := strings.Split(repoPath, "/")
		moduleName = parts[len(parts)-1]
	}

	// Create module config
	registryPath := "modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)
	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	config := modules.ModuleConfig{
		Name:        metadata.Name,
		Type:        "http",
		Description: metadata.Description,
		Version:     "1.0.0",
		Source:      metadata.Repository,
		Docs:        fmt.Sprintf("%s/blob/main/README.md", metadata.Repository),
	}

	// Add to registry
	if err := discoverer.AddModule(moduleName, config); err != nil {
		log.Fatalf("Failed to add module: %v", err)
	}

	fmt.Printf("Module '%s' added successfully!\n", moduleName)
	fmt.Printf("Source: %s\n", metadata.Repository)
	fmt.Printf("To use this module, configure its endpoints in modules/registry.yaml\n")
}

func removeModule(args []string) {
	if len(args) < 1 {
		log.Fatal("Module name is required")
	}

	name := args[0]
	registryPath := "modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	if err := discoverer.RemoveModule(name); err != nil {
		log.Fatalf("Failed to remove module: %v", err)
	}

	fmt.Printf("Module %s removed successfully\n", name)
}

func initShiro(_ []string) {
	fmt.Println("Initializing Shiro project...")

	// Create .shiro directory structure
	dirs := []string{
		".shiro",
		".shiro/modules",
		".shiro/workflows",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		fmt.Printf("Created directory: %s\n", dir)
	}

	// Create example workflow.json
	workflowContent := `{
  "name": "example-workflow",
  "description": "Example workflow for Shiro",
  "steps": [
    {
      "id": "step1",
      "type": "print",
      "config": {
        "level": "info",
        "message": "Hello from Shiro!"
      }
    }
  ]
}`

	if err := os.WriteFile(".shiro/workflow.json", []byte(workflowContent), 0644); err != nil {
		log.Fatalf("Failed to create workflow.json: %v", err)
	}
	fmt.Println("Created file: .shiro/workflow.json")

	// Create example config.yaml
	configContent := `# Shiro configuration
# This file configures AI models and other settings

models:
  # Example AI model configuration
  # ollama:
  #   type: ollama
  #   base_url: "http://localhost:11434"
  
  # openai:
  #   type: openai
  #   base_url: "https://api.openai.com/v1"
  #   api_key: "your-api-key"`

	if err := os.WriteFile(".shiro/config.yaml", []byte(configContent), 0644); err != nil {
		log.Fatalf("Failed to create config.yaml: %v", err)
	}
	fmt.Println("Created file: .shiro/config.yaml")

	// Create module registry
	registryContent := `modules:
  # Built-in modules
  slack:
    name: "Slack Notifications"
    type: "builtin"
    description: "Send notifications to Slack channels"
  
  git:
    name: "Git Operations"
    type: "builtin"
    description: "Perform Git operations (diff, clone, etc.)"
  
  print:
    name: "Print Module"
    type: "builtin"
    description: "Print messages to console with different log levels"
  
  ai:
    name: "AI Generation"
    type: "builtin"
    description: "Generate content using AI providers"
  
  # Example HTTP module configuration
  # jira:
  #   name: "Jira Integration"
  #   type: "http"
  #   endpoints:
  #     - http://localhost:8080
  #   config: ".shiro/modules/jira/config.yaml"
  #   version: "1.0.0"
  #   description: "Integrate with Jira for issue tracking"
  #   source: "github.com/your-org/jira-module"
  #   docs: "https://github.com/your-org/jira-module/blob/main/README.md"`

	if err := os.WriteFile(".shiro/modules/registry.yaml", []byte(registryContent), 0644); err != nil {
		log.Fatalf("Failed to create modules/registry.yaml: %v", err)
	}
	fmt.Println("Created file: .shiro/modules/registry.yaml")

	// Create .gitignore entry
	gitignorePath := ".gitignore"
	gitignoreContent := `
# Shiro configuration
.shiro/
`

	// Append to .gitignore if it exists, otherwise create it
	if _, err := os.Stat(gitignorePath); err == nil {
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			log.Fatalf("Failed to read .gitignore: %v", err)
		}
		if !strings.Contains(string(content), ".shiro/") {
			f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("Failed to open .gitignore: %v", err)
			}
			defer f.Close()
			if _, err := f.WriteString(gitignoreContent); err != nil {
				log.Fatalf("Failed to write to .gitignore: %v", err)
			}
			fmt.Println("Updated .gitignore to exclude .shiro/")
		}
	} else {
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			log.Fatalf("Failed to create .gitignore: %v", err)
		}
		fmt.Println("Created file: .gitignore")
	}

	fmt.Println()
	fmt.Println("✓ Shiro project initialized successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Customize .shiro/workflow.json with your workflow")
	fmt.Println("  2. Configure AI models in .shiro/config.yaml")
	fmt.Println("  3. Add modules: shiro add module <module-name>")
	fmt.Println("  4. Run your workflow: shiro run")
	fmt.Println()
	fmt.Println("For more information: shiro help")
}

func searchModules(args []string) {
	if len(args) < 1 {
		log.Fatal("Search query is required")
	}

	query := args[0]
	token := os.Getenv("GITHUB_TOKEN") // Optional GitHub token for higher rate limits

	githubClient := modules.NewGitHubClient(token)
	results, err := githubClient.SearchModules(query)
	if err != nil {
		log.Fatalf("Failed to search modules: %v", err)
	}

	fmt.Printf("Found %d modules for '%s':\n", len(results), query)
	for _, result := range results {
		fmt.Printf("  - %s (%s)\n", result.Name, result.FullName)
		fmt.Printf("    Description: %s\n", result.Description)
		fmt.Printf("    Stars: %d, Language: %s\n", result.Stargazers, result.Language)
		fmt.Printf("    Repository: %s\n\n", result.HTMLURL)
	}
}

func installModule(args []string) {
	if len(args) < 1 {
		log.Fatal("GitHub repository is required")
	}

	repo := args[0]
	token := os.Getenv("GITHUB_TOKEN")

	// Parse GitHub repository format
	repoPath, err := modules.ParseGitHubRepo(repo)
	if err != nil {
		log.Fatalf("Invalid GitHub repository format: %v", err)
	}

	githubClient := modules.NewGitHubClient(token)
	metadata, err := githubClient.GetModuleMetadata(repoPath)
	if err != nil {
		log.Fatalf("Failed to get module metadata: %v", err)
	}

	// Create module configuration
	moduleConfig := modules.ModuleConfig{
		Name:        metadata.Name,
		Type:        "http",
		Version:     "1.0.0",
		Description: metadata.Description,
		Source:      metadata.Repository,
		Docs:        fmt.Sprintf("%s/blob/main/README.md", metadata.Repository),
	}

	// Add to registry
	registryPath := "modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	if err := discoverer.AddModule(metadata.Name, moduleConfig); err != nil {
		log.Fatalf("Failed to add module: %v", err)
	}

	fmt.Printf("Module %s installed successfully\n", metadata.Name)
	fmt.Printf("Source: %s\n", metadata.Repository)
	fmt.Printf("Please configure the module endpoints in modules/registry.yaml\n")
}

func moduleInfo(args []string) {
	if len(args) < 1 {
		log.Fatal("Module name is required")
	}

	name := args[0]
	registryPath := "modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	config, err := discoverer.GetModuleConfig(name)
	if err != nil {
		log.Fatalf("Failed to get module info: %v", err)
	}

	fmt.Printf("Module: %s\n", config.Name)
	fmt.Printf("Type: %s\n", config.Type)
	fmt.Printf("Version: %s\n", config.Version)
	fmt.Printf("Description: %s\n", config.Description)

	if config.Source != "" {
		fmt.Printf("Source: %s\n", config.Source)
	}

	if config.Docs != "" {
		fmt.Printf("Documentation: %s\n", config.Docs)
	}

	if len(config.Endpoints) > 0 {
		fmt.Printf("Endpoints: %v\n", config.Endpoints)
	} else if config.Endpoint != "" {
		fmt.Printf("Endpoint: %s\n", config.Endpoint)
	}
}

func moduleDocs(args []string) {
	if len(args) < 1 {
		log.Fatal("Module name is required")
	}

	name := args[0]
	registryPath := "modules/registry.yaml"
	discoverer := modules.NewDiscoverer(registryPath, nil)

	if err := discoverer.LoadRegistry(); err != nil {
		log.Fatalf("Failed to load registry: %v", err)
	}

	config, err := discoverer.GetModuleConfig(name)
	if err != nil {
		log.Fatalf("Failed to get module info: %v", err)
	}

	if config.Docs != "" {
		fmt.Printf("Opening documentation: %s\n", config.Docs)
		// In a real implementation, this would open the URL in a browser
		// For now, just print the URL
	} else if config.Source != "" {
		docsURL := fmt.Sprintf("%s/blob/main/README.md", config.Source)
		fmt.Printf("Documentation: %s\n", docsURL)
	} else {
		fmt.Printf("No documentation available for module %s\n", name)
	}
}

// Simplified CLI wrapper functions
func runAddCommand(args []string) {
	if len(args) > 0 && args[0] == "module" {
		addModule(args[1:])
	} else {
		addModule(args)
	}
}

func runSearchCommand(args []string) {
	if len(args) > 0 && args[0] == "module" {
		searchModules(args[1:])
	} else {
		searchModules(args)
	}
}

func runListCommand(args []string) {
	if len(args) > 0 && args[0] == "modules" {
		listModules()
	} else {
		listModules()
	}
}

func runRemoveCommand(args []string) {
	if len(args) > 0 && args[0] == "module" {
		removeModule(args[1:])
	} else {
		removeModule(args)
	}
}

func runInstallCommand(args []string) {
	if len(args) > 0 && args[0] == "module" {
		installModule(args[1:])
	} else {
		installModule(args)
	}
}

func runInfoCommand(args []string) {
	if len(args) > 0 && args[0] == "module" {
		moduleInfo(args[1:])
	} else {
		moduleInfo(args)
	}
}

func runDocsCommand(args []string) {
	if len(args) > 0 && args[0] == "module" {
		moduleDocs(args[1:])
	} else {
		moduleDocs(args)
	}
}
