package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rkuthiala/shiro-automation/internal/github"
	"github.com/rkuthiala/shiro-automation/internal/gitlab"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/runtime"
	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
	"github.com/rkuthiala/shiro-automation/pkg/ai"
	"github.com/rkuthiala/shiro-automation/pkg/git"
	"github.com/rkuthiala/shiro-automation/pkg/slack"
)

func main() {
	// Parse flags
	workflowFile := flag.String("workflow", "", "Path to workflow JSON file")
	configFile := flag.String("config", "", "Path to model configuration file")
	stateStoreType := flag.String("state-store", "gitlab", "State store type (memory, filesystem, gitlab)")
	flag.Parse()

	if *workflowFile == "" {
		log.Fatal("workflow file is required (use -workflow flag)")
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

	// Register AI module with providers
	aiModule := ai.NewAIModule()

	// Initialize AI providers from config
	for modelName, modelDef := range modelConfig {
		var provider ai.Provider
		var err error

		switch modelDef["type"].(string) {
		case "ollama":
			providerConfig := &ai.ProviderConfig{
				Type:    "ollama",
				BaseURL: modelDef["base_url"].(string),
				Model:   modelName,
			}
			provider, err = ai.NewOllamaProvider(providerConfig)
		case "openai":
			providerConfig := &ai.ProviderConfig{
				Type:    "openai",
				BaseURL: modelDef["base_url"].(string),
				APIKey:  modelDef["api_key"].(string),
				Model:   modelName,
			}
			provider, err = ai.NewOpenAIProvider(providerConfig)
		default:
			logger.Printf("Unknown provider type: %s", modelDef["type"])
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
		Models map[string]map[string]interface{} `json:"models"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Failed to parse config file: %v", err)
		return make(map[string]map[string]interface{})
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
