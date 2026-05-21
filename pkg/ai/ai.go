package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"

	aiprovider "github.com/rkuthiala/shiro-automation/internal/ai"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// Provider is an alias for the internal AI provider interface
type Provider = aiprovider.Provider

// ProviderConfig is an alias for the internal AI provider config
type ProviderConfig = aiprovider.ProviderConfig

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config *ProviderConfig) (Provider, error) {
	return aiprovider.NewOllamaProvider(config)
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config *ProviderConfig) (Provider, error) {
	return aiprovider.NewOpenAIProvider(config)
}

// AIModule implements the ai.generate module
type AIModule struct {
	providers     map[string]Provider
	defaultModels map[string]string
}

// NewAIModule creates a new AI module
func NewAIModule() *AIModule {
	return &AIModule{
		providers:     make(map[string]Provider),
		defaultModels: make(map[string]string),
	}
}

// AddProvider adds an AI provider
func (m *AIModule) AddProvider(name string, provider Provider) {
	m.providers[name] = provider
}

func (m *AIModule) AddProviderWithDefaultModel(name string, provider Provider, model string) {
	m.providers[name] = provider
	m.defaultModels[name] = model
}

func (m *AIModule) HasProvider(name string) bool {
	_, ok := m.providers[name]
	return ok
}

func (m *AIModule) DefaultModel(name string) string {
	return m.defaultModels[name]
}

// Run executes the AI generation
func (m *AIModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	// Get provider name (default to "default")
	providerName, _ := wfStep.Config["provider"].(string)
	if providerName == "" {
		providerName = m.resolveDefaultProvider()
	}

	provider, ok := m.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %s not found; configure a provider named %q or set config.provider to one of: %s", providerName, "default", strings.Join(m.providerNames(), ", "))
	}

	// Build request
	model, ok := wfStep.Config["model"].(string)
	if !ok || model == "" {
		model = m.defaultModels[providerName]
	}
	if model == "" {
		return nil, fmt.Errorf("model is required for provider %s: set config.model in the workflow step or model in .shiro/config.yaml", providerName)
	}

	prompt, ok := wfStep.Config["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	system, _ := wfStep.Config["system"].(string)

	req := &aiprovider.GenerateRequest{
		Model: model,
		Messages: []aiprovider.Message{
			{Role: "user", Content: prompt},
		},
		System: system,
	}

	// Optional parameters
	if temp, ok := wfStep.Config["temperature"].(float64); ok {
		req.Temperature = temp
	}
	if maxTokens, ok := wfStep.Config["max_tokens"].(float64); ok {
		req.MaxTokens = int(maxTokens)
	}

	// Generate response
	resp, err := provider.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	output := map[string]interface{}{
		"content": resp.Content,
		"success": true,
	}

	if resp.Usage != nil {
		output["usage"] = map[string]interface{}{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		}
	}

	return output, nil
}

func (m *AIModule) resolveDefaultProvider() string {
	if _, ok := m.providers["default"]; ok {
		return "default"
	}
	if len(m.providers) == 1 {
		for name := range m.providers {
			return name
		}
	}
	return "default"
}

func (m *AIModule) providerNames() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Metadata returns module metadata
func (m *AIModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "ai.generate",
		Description: "Generates content using AI models",
		InputSchema: map[string]modules.SchemaField{
			"provider": {
				Type:        "string",
				Description: "AI provider name (default: default)",
				Required:    false,
				Default:     "default",
			},
			"model": {
				Type:        "string",
				Description: "Model name to use",
				Required:    true,
			},
			"prompt": {
				Type:        "string",
				Description: "Prompt for the AI",
				Required:    true,
			},
			"system": {
				Type:        "string",
				Description: "System prompt",
				Required:    false,
			},
			"temperature": {
				Type:        "number",
				Description: "Temperature for generation",
				Required:    false,
				Default:     0.7,
			},
			"max_tokens": {
				Type:        "number",
				Description: "Maximum tokens to generate",
				Required:    false,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"content": {
				Type:        "string",
				Description: "Generated content",
				Required:    true,
			},
			"success": {
				Type:        "boolean",
				Description: "Whether generation succeeded",
				Required:    true,
			},
			"usage": {
				Type:        "object",
				Description: "Token usage information",
				Required:    false,
			},
		},
	}
}
