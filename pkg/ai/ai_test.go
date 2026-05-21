package ai

import (
	"context"
	"strings"
	"testing"

	aiprovider "github.com/rkuthiala/shiro-automation/internal/ai"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

type fakeProvider struct {
	lastRequest *aiprovider.GenerateRequest
}

func (p *fakeProvider) Generate(ctx context.Context, req *aiprovider.GenerateRequest) (*aiprovider.GenerateResponse, error) {
	p.lastRequest = req
	return &aiprovider.GenerateResponse{Content: "ok"}, nil
}

func (p *fakeProvider) Stream(ctx context.Context, req *aiprovider.GenerateRequest) (<-chan aiprovider.StreamChunk, error) {
	ch := make(chan aiprovider.StreamChunk)
	close(ch)
	return ch, nil
}

func (p *fakeProvider) Close() error {
	return nil
}

func TestAIModuleUsesProviderDefaultModel(t *testing.T) {
	provider := &fakeProvider{}
	module := NewAIModule()
	module.AddProviderWithDefaultModel("openai-custom", provider, "gpt-4.1")

	_, err := module.Run(context.Background(), nil, workflow.Step{
		ID:   "review",
		Type: "ai.generate",
		Config: map[string]interface{}{
			"provider": "openai-custom",
			"prompt":   "review this",
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if provider.lastRequest.Model != "gpt-4.1" {
		t.Fatalf("model = %s, want gpt-4.1", provider.lastRequest.Model)
	}
}

func TestAIModuleUsesOnlyProviderAsDefault(t *testing.T) {
	provider := &fakeProvider{}
	module := NewAIModule()
	module.AddProviderWithDefaultModel("openai-custom", provider, "gpt-4.1")

	_, err := module.Run(context.Background(), nil, workflow.Step{
		ID:   "review",
		Type: "ai.generate",
		Config: map[string]interface{}{
			"prompt": "review this",
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if provider.lastRequest.Model != "gpt-4.1" {
		t.Fatalf("model = %s, want gpt-4.1", provider.lastRequest.Model)
	}
}

func TestAIModuleRequiresModelWhenNoDefaultExists(t *testing.T) {
	module := NewAIModule()
	module.AddProviderWithDefaultModel("openai-custom", &fakeProvider{}, "")

	_, err := module.Run(context.Background(), nil, workflow.Step{
		ID:   "review",
		Type: "ai.generate",
		Config: map[string]interface{}{
			"provider": "openai-custom",
			"prompt":   "review this",
		},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want missing model error")
	}
	if !strings.Contains(err.Error(), "model is required for provider openai-custom") {
		t.Fatalf("error = %v, want missing model details", err)
	}
}
