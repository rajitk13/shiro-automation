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
	baseURL   string
	token     string
	tokenType string // "private" or "job"
	client    *http.Client
}

// NewClient creates a new GitLab client
func NewClient() *Client {
	baseURL := os.Getenv("CI_SERVER_URL")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	token := os.Getenv("GITLAB_TOKEN")
	tokenType := "private"
	if token == "" {
		token = os.Getenv("CI_JOB_TOKEN")
		tokenType = "job"
	}

	return &Client{
		baseURL:   baseURL,
		token:     token,
		tokenType: tokenType,
		client:    &http.Client{},
	}
}

// NewClientWithConfig creates a new GitLab client with custom config
func NewClientWithConfig(baseURL, token string) *Client {
	return &Client{
		baseURL:   baseURL,
		token:     token,
		tokenType: "private",
		client:    &http.Client{},
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

// setAuthToken sets the appropriate authentication header based on token type
func (c *Client) setAuthToken(req *http.Request) {
	if c.tokenType == "job" {
		req.Header.Set("JOB-TOKEN", c.token)
	} else {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}
}

// GetDiff gets the diff for a merge request
func (c *Client) GetDiff(ctx context.Context, projectID, mrIID string) (string, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s/changes", c.baseURL, projectID, mrIID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)

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

	c.setAuthToken(req)

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

	c.setAuthToken(req)
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

// UploadArtifact uploads a file as a GitLab job artifact
func (c *Client) UploadArtifact(ctx context.Context, projectID, jobID, artifactPath string, content []byte) error {
	url := fmt.Sprintf("%s/api/v4/projects/%s/jobs/%s/artifacts/%s", c.baseURL, projectID, jobID, artifactPath)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DownloadArtifact downloads a file from GitLab job artifacts
func (c *Client) DownloadArtifact(ctx context.Context, projectID, jobID, artifactPath string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/jobs/%s/artifacts/%s", c.baseURL, projectID, jobID, artifactPath)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

// GetCommitInfo gets information about a specific commit
func (c *Client) GetCommitInfo(ctx context.Context, projectID, commitSHA string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s", c.baseURL, projectID, commitSHA)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)

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

// GetUserInfo gets information about a user
func (c *Client) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v4/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)

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

// GetMRParticipants gets participants in a merge request
func (c *Client) GetMRParticipants(ctx context.Context, projectID, mrIID string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s/participants", c.baseURL, projectID, mrIID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var participants []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&participants); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return participants, nil
}

// GetFilesChanged gets list of changed files in a merge request
func (c *Client) GetFilesChanged(ctx context.Context, projectID, mrIID string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%s/changes", c.baseURL, projectID, mrIID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Changes []map[string]interface{} `json:"changes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Changes, nil
}

// ListJobArtifacts lists all artifacts for a job
func (c *Client) ListJobArtifacts(ctx context.Context, projectID, jobID string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v4/projects/%s/jobs/%s/artifacts", c.baseURL, projectID, jobID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthToken(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var paths []string
	for _, file := range result.Files {
		paths = append(paths, file.Path)
	}

	return paths, nil
}
