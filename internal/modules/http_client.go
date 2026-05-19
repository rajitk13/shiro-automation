package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// HTTPModuleClient handles communication with HTTP-based modules
type HTTPModuleClient struct {
	httpClient *http.Client
	timeout    time.Duration
}

// NewHTTPModuleClient creates a new HTTP module client
func NewHTTPModuleClient(timeout time.Duration) *HTTPModuleClient {
	return &HTTPModuleClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// LoadBalancedClient handles load balancing across multiple endpoints
type LoadBalancedClient struct {
	endpoints     []string
	currentIndex  int
	healthy       map[string]bool
	mu            sync.RWMutex
	httpClient    *HTTPModuleClient
	circuitOpen   map[string]bool
	circuitTimer  map[string]*time.Timer
	retryAttempts int
}

// NewLoadBalancedClient creates a new load-balanced client
func NewLoadBalancedClient(endpoints []string, timeout time.Duration) *LoadBalancedClient {
	healthy := make(map[string]bool)
	for _, endpoint := range endpoints {
		healthy[endpoint] = true
	}

	return &LoadBalancedClient{
		endpoints:     endpoints,
		currentIndex:  0,
		healthy:       healthy,
		httpClient:    NewHTTPModuleClient(timeout),
		circuitOpen:   make(map[string]bool),
		circuitTimer:  make(map[string]*time.Timer),
		retryAttempts: 3,
	}
}

// getNextEndpoint returns the next healthy endpoint using round-robin
func (c *LoadBalancedClient) getNextEndpoint() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Find next healthy endpoint
	attempts := 0
	for attempts < len(c.endpoints) {
		endpoint := c.endpoints[c.currentIndex]
		c.currentIndex = (c.currentIndex + 1) % len(c.endpoints)

		if c.healthy[endpoint] && !c.circuitOpen[endpoint] {
			return endpoint
		}
		attempts++
	}

	// All endpoints are unhealthy or circuits are open, return first one anyway
	return c.endpoints[0]
}

// markUnhealthy marks an endpoint as unhealthy
func (c *LoadBalancedClient) markUnhealthy(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthy[endpoint] = false
}

// markHealthy marks an endpoint as healthy
func (c *LoadBalancedClient) markHealthy(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthy[endpoint] = true
	c.circuitOpen[endpoint] = false
}

// openCircuit opens the circuit for an endpoint
func (c *LoadBalancedClient) openCircuit(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.circuitOpen[endpoint] = true

	// Auto-close circuit after 30 seconds
	if timer, exists := c.circuitTimer[endpoint]; exists {
		timer.Stop()
	}

	c.circuitTimer[endpoint] = time.AfterFunc(30*time.Second, func() {
		c.markHealthy(endpoint)
	})
}

// Execute sends an execute request to an HTTP module with load balancing
func (c *LoadBalancedClient) Execute(ctx context.Context, endpoints []string, request ExecuteRequest) (ExecuteResponse, error) {
	if len(endpoints) == 1 {
		return c.httpClient.Execute(ctx, endpoints[0], request)
	}

	// Use load balancing for multiple endpoints
	lbClient := NewLoadBalancedClient(endpoints, c.httpClient.timeout)

	var lastError error
	for i := 0; i < c.retryAttempts; i++ {
		endpoint := lbClient.getNextEndpoint()
		result, err := c.httpClient.Execute(ctx, endpoint, request)
		if err == nil {
			return result, nil
		}

		lastError = err
		lbClient.markUnhealthy(endpoint)
		lbClient.openCircuit(endpoint)
	}

	return ExecuteResponse{}, fmt.Errorf("all endpoints failed after %d attempts: %w", c.retryAttempts, lastError)
}

// Metadata retrieves metadata from an HTTP module with load balancing
func (c *LoadBalancedClient) Metadata(ctx context.Context, endpoints []string) (MetadataResponse, error) {
	if len(endpoints) == 1 {
		return c.httpClient.Metadata(ctx, endpoints[0])
	}

	lbClient := NewLoadBalancedClient(endpoints, c.httpClient.timeout)

	var lastError error
	for i := 0; i < c.retryAttempts; i++ {
		endpoint := lbClient.getNextEndpoint()
		result, err := c.httpClient.Metadata(ctx, endpoint)
		if err == nil {
			return result, nil
		}

		lastError = err
		lbClient.markUnhealthy(endpoint)
	}

	return MetadataResponse{}, fmt.Errorf("all endpoints failed after %d attempts: %w", c.retryAttempts, lastError)
}

// Health checks the health of an HTTP module with load balancing
func (c *LoadBalancedClient) Health(ctx context.Context, endpoints []string) (HealthResponse, error) {
	if len(endpoints) == 1 {
		return c.httpClient.Health(ctx, endpoints[0])
	}

	lbClient := NewLoadBalancedClient(endpoints, c.httpClient.timeout)

	var lastError error
	for i := 0; i < c.retryAttempts; i++ {
		endpoint := lbClient.getNextEndpoint()
		result, err := c.httpClient.Health(ctx, endpoint)
		if err == nil {
			return result, nil
		}

		lastError = err
		lbClient.markUnhealthy(endpoint)
	}

	return HealthResponse{}, fmt.Errorf("all endpoints failed after %d attempts: %w", c.retryAttempts, lastError)
}

// Execute sends an execute request to an HTTP module
func (c *HTTPModuleClient) Execute(ctx context.Context, endpoint string, request ExecuteRequest) (ExecuteResponse, error) {
	result, err := c.sendRequest(ctx, endpoint+"/execute", request)
	if err != nil {
		return ExecuteResponse{}, err
	}
	resp, ok := result.(ExecuteResponse)
	if !ok {
		return ExecuteResponse{}, fmt.Errorf("unexpected response type for execute")
	}
	return resp, nil
}

// Metadata retrieves metadata from an HTTP module
func (c *HTTPModuleClient) Metadata(ctx context.Context, endpoint string) (MetadataResponse, error) {
	result, err := c.sendRequest(ctx, endpoint+"/metadata", nil)
	if err != nil {
		return MetadataResponse{}, err
	}
	resp, ok := result.(MetadataResponse)
	if !ok {
		return MetadataResponse{}, fmt.Errorf("unexpected response type for metadata")
	}
	return resp, nil
}

// Health checks the health of an HTTP module
func (c *HTTPModuleClient) Health(ctx context.Context, endpoint string) (HealthResponse, error) {
	result, err := c.sendRequest(ctx, endpoint+"/health", nil)
	if err != nil {
		return HealthResponse{}, err
	}
	resp, ok := result.(HealthResponse)
	if !ok {
		return HealthResponse{}, fmt.Errorf("unexpected response type for health")
	}
	return resp, nil
}

// sendRequest sends a generic HTTP request to a module endpoint
func (c *HTTPModuleClient) sendRequest(ctx context.Context, url string, body interface{}) (interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Method = "POST"
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Determine response type based on URL path
	if body != nil {
		var result ExecuteResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal execute response: %w", err)
		}
		return result, nil
	}

	// Try to determine response type
	var metadata MetadataResponse
	if err := json.Unmarshal(respBody, &metadata); err == nil && metadata.Name != "" {
		return metadata, nil
	}

	var health HealthResponse
	if err := json.Unmarshal(respBody, &health); err == nil {
		return health, nil
	}

	return nil, fmt.Errorf("failed to determine response type")
}
