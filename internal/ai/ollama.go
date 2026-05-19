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

// OllamaProvider implements Provider for Ollama
type OllamaProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config *ProviderConfig) (*OllamaProvider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	return &OllamaProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Generate generates a response from Ollama
func (p *OllamaProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Build Ollama API request
	ollamaReq := map[string]interface{}{
		"model":  req.Model,
		"stream": false,
	}
	
	// Add messages
	if len(req.Messages) > 0 {
		ollamaReq["messages"] = req.Messages
	} else if req.System != "" {
		// Fallback to prompt format for older Ollama versions
		ollamaReq["prompt"] = req.System
	}
	
	if req.Temperature > 0 {
		ollamaReq["options"] = map[string]interface{}{
			"temperature": req.Temperature,
		}
	}
	
	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/api/chat", p.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var ollamaResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return &GenerateResponse{
		Content:      ollamaResp.Message.Content,
		FinishReason: "stop",
	}, nil
}

// Stream generates a streaming response from Ollama
func (p *OllamaProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	
	go func() {
		defer close(ch)
		
		// Build Ollama API request
		ollamaReq := map[string]interface{}{
			"model":  req.Model,
			"stream": true,
		}
		
		if len(req.Messages) > 0 {
			ollamaReq["messages"] = req.Messages
		} else if req.System != "" {
			ollamaReq["prompt"] = req.System
		}
		
		body, err := json.Marshal(ollamaReq)
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
			return
		}
		
		url := fmt.Sprintf("%s/api/chat", p.config.BaseURL)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to create request: %w", err)}
			return
		}
		
		httpReq.Header.Set("Content-Type", "application/json")
		
		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("request failed: %w", err)}
			return
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			ch <- StreamChunk{Error: fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))}
			return
		}
		
		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}
			
			if err := decoder.Decode(&chunk); err != nil {
				if err == io.EOF {
					break
				}
				ch <- StreamChunk{Error: fmt.Errorf("failed to decode chunk: %w", err)}
				return
			}
			
			ch <- StreamChunk{
				Content: chunk.Message.Content,
				Done:    chunk.Done,
			}
			
			if chunk.Done {
				break
			}
		}
	}()
	
	return ch, nil
}

// Close cleans up Ollama provider resources
func (p *OllamaProvider) Close() error {
	p.httpClient.CloseIdleConnections()
	return nil
}
