package ai

import (
	"context"
	"fmt"
)

// OpenRouterProvider implements Provider for OpenRouter API
// OpenRouter is an OpenAI-compatible API that provides access to many models
type OpenRouterProvider struct {
	*OpenAIProvider
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider(config *ProviderConfig) (*OpenRouterProvider, error) {
	// Set default base URL for OpenRouter
	if config.BaseURL == "" {
		config.BaseURL = "https://openrouter.ai/api/v1"
	}

	// OpenRouter requires an API key
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenRouter provider")
	}

	// Add OpenRouter-specific headers if not already set
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	// OpenRouter recommends setting these headers for better routing
	if _, ok := config.Headers["HTTP-Referer"]; !ok {
		config.Headers["HTTP-Referer"] = "https://github.com/rkuthiala/shiro-automation"
	}
	if _, ok := config.Headers["X-Title"]; !ok {
		config.Headers["X-Title"] = "Shiro Automation"
	}

	// Initialize the underlying OpenAI-compatible provider
	openaiProvider, err := NewOpenAIProvider(config)
	if err != nil {
		return nil, err
	}

	return &OpenRouterProvider{
		OpenAIProvider: openaiProvider,
	}, nil
}

// Generate generates a response from OpenRouter API
func (p *OpenRouterProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	return p.OpenAIProvider.Generate(ctx, req)
}

// Stream generates a streaming response from OpenRouter API
func (p *OpenRouterProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	return p.OpenAIProvider.Stream(ctx, req)
}

// Close cleans up OpenRouter provider resources
func (p *OpenRouterProvider) Close() error {
	return p.OpenAIProvider.Close()
}
