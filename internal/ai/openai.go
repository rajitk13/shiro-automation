package ai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIProvider implements Provider for OpenAI-compatible APIs
type OpenAIProvider struct {
	config     *ProviderConfig
	httpClient *http.Client
}

// NewOpenAIProvider creates a new OpenAI-compatible provider
func NewOpenAIProvider(config *ProviderConfig) (*OpenAIProvider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI provider")
	}

	// Check if API key is still a template (not resolved)
	if strings.HasPrefix(config.APIKey, "{{env.") && strings.HasSuffix(config.APIKey, "}}") {
		return nil, fmt.Errorf("API key template %s was not resolved - ensure the environment variable is set", config.APIKey)
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second // Increased default timeout for slower models
	}

	// Create HTTP client with optional TLS skip
	transport := &http.Transport{}
	if config.SkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &OpenAIProvider{
		config: config,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

// Generate generates a response from OpenAI-compatible API
func (p *OpenAIProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Build OpenAI API request
	openaiReq := map[string]interface{}{
		"model": req.Model,
	}

	// Add messages
	messages := []Message{}
	if req.System != "" {
		messages = append(messages, Message{Role: "system", Content: req.System})
	}
	messages = append(messages, req.Messages...)
	openaiReq["messages"] = messages

	if req.Temperature > 0 {
		openaiReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		openaiReq["max_tokens"] = req.MaxTokens
	}
	if req.TopP > 0 {
		openaiReq["top_p"] = req.TopP
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Read response body with size limit
	body, err = io.ReadAll(io.LimitReader(resp.Body, 50*1024*1024)) // 50MB limit
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *Usage `json:"usage"`
	}

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &GenerateResponse{
		Content:      openaiResp.Choices[0].Message.Content,
		FinishReason: openaiResp.Choices[0].FinishReason,
		Usage:        openaiResp.Usage,
	}, nil
}

// Stream generates a streaming response from OpenAI-compatible API
func (p *OpenAIProvider) Stream(ctx context.Context, req *GenerateRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)

	go func() {
		defer close(ch)

		// Build OpenAI API request
		openaiReq := map[string]interface{}{
			"model":  req.Model,
			"stream": true,
		}

		messages := []Message{}
		if req.System != "" {
			messages = append(messages, Message{Role: "system", Content: req.System})
		}
		messages = append(messages, req.Messages...)
		openaiReq["messages"] = messages

		if req.Temperature > 0 {
			openaiReq["temperature"] = req.Temperature
		}
		if req.MaxTokens > 0 {
			openaiReq["max_tokens"] = req.MaxTokens
		}

		body, err := json.Marshal(openaiReq)
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to marshal request: %w", err)}
			return
		}

		url := fmt.Sprintf("%s/chat/completions", p.config.BaseURL)
		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			ch <- StreamChunk{Error: fmt.Errorf("failed to create request: %w", err)}
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

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

		// Read SSE stream
		reader := newSSEReader(resp.Body)
		for {
			line, err := reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					break
				}
				ch <- StreamChunk{Error: fmt.Errorf("failed to read stream: %w", err)}
				return
			}

			if line == "" {
				continue
			}

			if line == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
					FinishReason *string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue // Skip invalid lines
			}

			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				if content != "" {
					ch <- StreamChunk{Content: content}
				}

				if chunk.Choices[0].FinishReason != nil {
					ch <- StreamChunk{Done: true}
					return
				}
			}
		}
	}()

	return ch, nil
}

// Close cleans up OpenAI provider resources
func (p *OpenAIProvider) Close() error {
	p.httpClient.CloseIdleConnections()
	return nil
}

// sseReader reads Server-Sent Events
type sseReader struct {
	reader *bufio.Reader
}

func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{reader: bufio.NewReader(r)}
}

func (r *sseReader) ReadLine() (string, error) {
	for {
		line, err := r.reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			return strings.TrimPrefix(line, "data: "), nil
		}
	}
}
