package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client is a GitLab API client
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient creates a new GitLab client
func NewClient() *Client {
	baseURL := os.Getenv("CI_SERVER_URL")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		token = os.Getenv("CI_JOB_TOKEN")
	}

	return &Client{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{},
	}
}

// NewClientWithConfig creates a new GitLab client with custom config
func NewClientWithConfig(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{},
	}
}

// GetProjectID returns the GitLab project ID from environment
func GetProjectID() string {
	return os.Getenv("CI_PROJECT_ID")
}

// GetMRID returns the GitLab merge request ID from environment
func GetMRID() string {
	return os.Getenv("CI_MERGE_REQUEST_IID")
}

// GetCommitSHA returns the current commit SHA from environment
func GetCommitSHA() string {
	return os.Getenv("CI_COMMIT_SHA")
}

// GetBranch returns the current branch from environment
func GetBranch() string {
	return os.Getenv("CI_COMMIT_REF_NAME")
}

// GetDiff gets the diff for a merge request
func (c *Client) GetDiff(ctx context.Context, projectID, mrIID string) (string, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s/changes", c.baseURL, projectID, mrIID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("PRIVATE-TOKEN", c.token)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Changes []struct {
			Diff string `json:"diff"`
		} `json:"changes"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	
	var diffBuilder bytes.Buffer
	for _, change := range result.Changes {
		diffBuilder.WriteString(change.Diff)
	}
	
	return diffBuilder.String(), nil
}

// GetMRInfo gets information about a merge request
func (c *Client) GetMRInfo(ctx context.Context, projectID, mrIID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s", c.baseURL, projectID, mrIID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("PRIVATE-TOKEN", c.token)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	
	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return info, nil
}

// PostMRComment posts a comment to a merge request
func (c *Client) PostMRComment(ctx context.Context, projectID, mrIID, body string) error {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s/notes", c.baseURL, projectID, mrIID)
	
	payload := map[string]string{
		"body": body,
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("PRIVATE-TOKEN", c.token)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}
	
	return nil
}
