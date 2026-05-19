package ai

import (
	"context"
)

// Provider is the interface for AI generation providers
type Provider interface {
	// Generate generates a response from the AI model
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	
	// Stream generates a streaming response (optional)
	Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error)
	
	// Close cleans up provider resources
	Close() error
}

// GenerateRequest is a request to generate AI content
type GenerateRequest struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	System      string            `json:"system,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

// GenerateResponse is the response from AI generation
type GenerateResponse struct {
	Content   string            `json:"content"`
	FinishReason string         `json:"finish_reason"`
	Usage     *Usage            `json:"usage,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Type     string            `json:"type"`     // ollama, openai, custom
	BaseURL  string            `json:"base_url"`
	APIKey   string            `json:"api_key,omitempty"`
	Model    string            `json:"model"`
	Headers  map[string]string `json:"headers,omitempty"`
	Timeout  int               `json:"timeout"` // seconds
}
