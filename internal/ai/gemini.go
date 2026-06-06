package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeminiProvider implements Provider for Google Gemini AI
type GeminiProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
	apiType    string // "google-ai-studio" or "vertex-ai"
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(config *ProviderConfig) (*GeminiProvider, error) {
	// Determine API type (default to google-ai-studio)
	apiType := "google-ai-studio"
	if config.Type == "gemini" {
		if apiTypeVal, ok := config.Metadata["api_type"].(string); ok {
			apiType = apiTypeVal
		}
	}

	// Set default base URL based on API type
	if config.BaseURL == "" {
		switch apiType {
		case "vertex-ai":
			// Vertex AI requires project_id and location in the URL
			// These will be set from config metadata
			config.BaseURL = "https://aiplatform.googleapis.com"
		default:
			config.BaseURL = "https://generativelanguage.googleapis.com"
		}
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &GeminiProvider{
		config:     config,
		httpClient: &http.Client{Timeout: timeout},
		apiType:    apiType,
	}, nil
}

// Generate generates a response from Gemini API
func (p *GeminiProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	var url string
	var body []byte
	var err error

	switch p.apiType {
	case "vertex-ai":
		url, body, err = p.buildVertexAIRequest(req)
	default:
		url, body, err = p.buildGoogleAIStudioRequest(req)
	}

	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Set authentication header
	switch p.apiType {
	case "vertex-ai":
		// Vertex AI uses Bearer token (OAuth2) or API key
		if p.config.APIKey != "" {
			httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
		}
	default:
		// Google AI Studio uses API key as query parameter or header
		if p.config.APIKey != "" {
			httpReq.Header.Set("x-goog-api-key", p.config.APIKey)
		}
	}

	// Add custom headers if provided
	for key, value := range p.config.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Parse response based on API type
	switch p.apiType {
	case "vertex-ai":
		return p.parseVertexAIResponse(resp.Body)
	default:
		return p.parseGoogleAIStudioResponse(resp.Body)
	}
}

// buildGoogleAIStudioRequest builds request for Google AI Studio API
func (p *GeminiProvider) buildGoogleAIStudioRequest(req *GenerateRequest) (string, []byte, error) {
	// Google AI Studio REST API format
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": req.Messages[0].Content},
				},
			},
		},
	}

	// Add system instruction if provided
	if req.System != "" {
		geminiReq["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]string{
				{"text": req.System},
			},
		}
	}

	// Add generation config
	genConfig := map[string]interface{}{}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.TopP > 0 {
		genConfig["topP"] = req.TopP
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL with model name
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s",
		p.config.BaseURL, req.Model, p.config.APIKey)

	return url, body, nil
}

// buildVertexAIRequest builds request for Vertex AI API
func (p *GeminiProvider) buildVertexAIRequest(req *GenerateRequest) (string, []byte, error) {
	// Get project ID and location from config metadata
	projectID, _ := p.config.Metadata["project_id"].(string)
	location, _ := p.config.Metadata["location"].(string)
	if projectID == "" {
		projectID = "default-project"
	}
	if location == "" {
		location = "us-central1"
	}

	// Vertex AI REST API format
	vertexReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": req.Messages[0].Content}},
			},
		},
	}

	// Add system instruction if provided
	if req.System != "" {
		vertexReq["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]string{{"text": req.System}},
		}
	}

	// Add generation config
	genConfig := map[string]interface{}{}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if req.TopP > 0 {
		genConfig["topP"] = req.TopP
	}
	if len(genConfig) > 0 {
		vertexReq["generationConfig"] = genConfig
	}

	body, err := json.Marshal(vertexReq)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL for Vertex AI
	url := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		p.config.BaseURL, projectID, location, req.Model)

	return url, body, nil
}

// parseGoogleAIStudioResponse parses response from Google AI Studio
func (p *GeminiProvider) parseGoogleAIStudioResponse(body io.Reader) (*GenerateResponse, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata *struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	content := ""
	if len(resp.Candidates[0].Content.Parts) > 0 {
		content = resp.Candidates[0].Content.Parts[0].Text
	}

	usage := &Usage{}
	if resp.UsageMetadata != nil {
		usage.PromptTokens = resp.UsageMetadata.PromptTokenCount
		usage.CompletionTokens = resp.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return &GenerateResponse{
		Content:      content,
		FinishReason: resp.Candidates[0].FinishReason,
		Usage:        usage,
	}, nil
}

// parseVertexAIResponse parses response from Vertex AI
func (p *GeminiProvider) parseVertexAIResponse(body io.Reader) (*GenerateResponse, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata *struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	content := ""
	if len(resp.Candidates[0].Content.Parts) > 0 {
		content = resp.Candidates[0].Content.Parts[0].Text
	}

	usage := &Usage{}
	if resp.UsageMetadata != nil {
		usage.PromptTokens = resp.UsageMetadata.PromptTokenCount
		usage.CompletionTokens = resp.UsageMetadata.CandidatesTokenCount
		usage.TotalTokens = resp.UsageMetadata.TotalTokenCount
	}

	return &GenerateResponse{
		Content:      content,
		FinishReason: resp.Candidates[0].FinishReason,
		Usage:        usage,
	}, nil
}

// Stream generates a streaming response from Gemini API
func (p *GeminiProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)

	go func() {
		defer close(ch)
		// Streaming not yet implemented for Gemini
		ch <- StreamChunk{Error: fmt.Errorf("streaming not yet implemented for Gemini provider")}
	}()

	return ch, nil
}

// Close cleans up Gemini provider resources
func (p *GeminiProvider) Close() error {
	p.httpClient.CloseIdleConnections()
	return nil
}
