package quickstart

import (
	"fmt"
	"strings"

	"github.com/manifoldco/promptui"
)

// InteractiveConfig prompts user for AI provider configuration
func InteractiveConfig() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// Select AI provider
	providerPrompt := promptui.Select{
		Label: "Select AI Provider",
		Items: []string{"OpenAI", "Ollama", "Custom"},
	}

	_, provider, err := providerPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("prompt failed: %w", err)
	}

	switch provider {
	case "OpenAI":
		config, err = promptOpenAI()
	case "Ollama":
		config, err = promptOllama()
	case "Custom":
		config, err = promptCustom()
	}

	return config, err
}

func promptOpenAI() (map[string]interface{}, error) {
	config := make(map[string]interface{})
	config["type"] = "openai"

	// API Key
	apiKeyPrompt := promptui.Prompt{
		Label: "OpenAI API Key",
		Mask:  '*',
	}
	apiKey, err := apiKeyPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("API key prompt failed: %w", err)
	}
	config["api_key"] = apiKey

	// Base URL (optional, default to OpenAI)
	baseURLPrompt := promptui.Prompt{
		Label:   "Base URL (press Enter for default: https://api.openai.com/v1)",
		Default: "https://api.openai.com/v1",
	}
	baseURL, err := baseURLPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("base URL prompt failed: %w", err)
	}
	if baseURL != "" {
		config["base_url"] = baseURL
	}

	// Model
	modelPrompt := promptui.Prompt{
		Label:   "Model (press Enter for default: gpt-4)",
		Default: "gpt-4",
	}
	model, err := modelPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("model prompt failed: %w", err)
	}
	config["model"] = model

	return config, nil
}

func promptOllama() (map[string]interface{}, error) {
	config := make(map[string]interface{})
	config["type"] = "ollama"

	// Base URL
	baseURLPrompt := promptui.Prompt{
		Label:   "Ollama Base URL",
		Default: "http://localhost:11434",
	}
	baseURL, err := baseURLPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("base URL prompt failed: %w", err)
	}
	config["base_url"] = baseURL

	// Model
	modelPrompt := promptui.Prompt{
		Label:   "Model (e.g., codellama:34b)",
		Default: "codellama:34b",
	}
	model, err := modelPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("model prompt failed: %w", err)
	}
	config["model"] = model

	return config, nil
}

func promptCustom() (map[string]interface{}, error) {
	config := make(map[string]interface{})

	// Type
	typePrompt := promptui.Prompt{
		Label: "Provider type (e.g., openai-compatible)",
	}
	providerType, err := typePrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("type prompt failed: %w", err)
	}
	config["type"] = providerType

	// Base URL
	baseURLPrompt := promptui.Prompt{
		Label: "Base URL",
	}
	baseURL, err := baseURLPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("base URL prompt failed: %w", err)
	}
	config["base_url"] = baseURL

	// API Key
	apiKeyPrompt := promptui.Prompt{
		Label: "API Key (press Enter if not required)",
		Mask:  '*',
	}
	apiKey, err := apiKeyPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("API key prompt failed: %w", err)
	}
	if apiKey != "" {
		config["api_key"] = apiKey
	}

	// Model
	modelPrompt := promptui.Prompt{
		Label: "Model",
	}
	model, err := modelPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("model prompt failed: %w", err)
	}
	config["model"] = model

	return config, nil
}

// ParseDirectConfig parses -d flags into config map
func ParseDirectConfig(flags []string) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	provider := "default"

	for _, flag := range flags {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "provider":
			provider = value
		case "api_key", "base_url", "model", "type":
			config[key] = value
		}
	}

	// If provider is specified, wrap in provider config
	if provider != "default" {
		return map[string]interface{}{
			provider: config,
		}, nil
	}

	return config, nil
}
