package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Client is a GitHub API client
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient creates a new GitHub client
func NewClient() *Client {
	baseURL := os.Getenv("GITHUB_API_URL")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	token := os.Getenv("GITHUB_TOKEN")

	return &Client{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{},
	}
}

// NewClientWithConfig creates a new GitHub client with custom config
func NewClientWithConfig(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{},
	}
}

// GetRepository returns the GitHub repository from environment
func GetRepository() string {
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return os.Getenv("GITHUB_REPOSITORY")
	}
	return repo
}

// GetPRNumber returns the GitHub pull request number from environment
func GetPRNumber() string {
	// GitHub Actions sets this for pull_request events
	if num := os.Getenv("GITHUB_PR_NUMBER"); num != "" {
		return num
	}

	// Extract from GITHUB_REF for pull requests
	ref := os.Getenv("GITHUB_REF")
	if len(ref) > 11 && ref[:11] == "refs/pull/" {
		// Format: refs/pull/123/merge
		for i := 12; i < len(ref); i++ {
			if ref[i] == '/' {
				return ref[11:i]
			}
		}
	}

	return ""
}

// GetCommitSHA returns the current commit SHA from environment
func GetCommitSHA() string {
	return os.Getenv("GITHUB_SHA")
}

// GetBranch returns the current branch from environment
func GetBranch() string {
	ref := os.Getenv("GITHUB_REF")
	if ref == "" {
		return ""
	}

	// Handle refs/heads/branch-name
	if len(ref) > 11 && ref[:11] == "refs/heads/" {
		return ref[11:]
	}

	return ref
}

// GetOwner returns the repository owner
func GetOwner() string {
	repo := GetRepository()
	if repo == "" {
		return ""
	}

	// Format: owner/repo
	for i := 0; i < len(repo); i++ {
		if repo[i] == '/' {
			return repo[:i]
		}
	}

	return repo
}

// GetRepoName returns the repository name
func GetRepoName() string {
	repo := GetRepository()
	if repo == "" {
		return ""
	}

	// Format: owner/repo
	for i := len(repo) - 1; i >= 0; i-- {
		if repo[i] == '/' {
			return repo[i+1:]
		}
	}

	return repo
}

// GetDiff gets the diff for a pull request
func (c *Client) GetDiff(ctx context.Context, owner, repo, prNumber string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%s", c.baseURL, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var prInfo struct {
		DiffURL string `json:"diff_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&prInfo); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Fetch the diff
	diffReq, err := http.NewRequestWithContext(ctx, "GET", prInfo.DiffURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create diff request: %w", err)
	}

	if c.token != "" {
		diffReq.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}

	diffResp, err := c.client.Do(diffReq)
	if err != nil {
		return "", fmt.Errorf("diff request failed: %w", err)
	}
	defer diffResp.Body.Close()

	if diffResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(diffResp.Body)
		return "", fmt.Errorf("unexpected diff status code %d: %s", diffResp.StatusCode, string(body))
	}

	diff, err := io.ReadAll(diffResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read diff: %w", err)
	}

	return string(diff), nil
}

// GetPRInfo gets information about a pull request
func (c *Client) GetPRInfo(ctx context.Context, owner, repo, prNumber string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%s", c.baseURL, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

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

// PostPRComment posts a comment to a pull request
func (c *Client) PostPRComment(ctx context.Context, owner, repo, prNumber, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", c.baseURL, owner, repo, prNumber)

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

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
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

// ReviewComment represents a review comment on a specific line
type ReviewComment struct {
	Path     string `json:"path"`
	Position int    `json:"position,omitempty"`
	Line     int    `json:"line,omitempty"`
	Body     string `json:"body"`
}

// PostReviewComment posts an inline review comment on a pull request
func (c *Client) PostReviewComment(ctx context.Context, owner, repo, prNumber, body, path string, position int, commitID string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%s/comments", c.baseURL, owner, repo, prNumber)

	payload := map[string]interface{}{
		"body":      body,
		"path":      path,
		"position":  position,
		"commit_id": commitID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
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

// GetPRReviewComments gets all review comments on a pull request
func (c *Client) GetPRReviewComments(ctx context.Context, owner, repo, prNumber string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%s/comments", c.baseURL, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	var comments []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return comments, nil
}

// CreateReview creates a full review with multiple comments
func (c *Client) CreateReview(ctx context.Context, owner, repo, prNumber, body string, comments []ReviewComment, commitID string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%s/reviews", c.baseURL, owner, repo, prNumber)

	payload := map[string]interface{}{
		"body":      body,
		"comments":  comments,
		"event":     "COMMENT",
		"commit_id": commitID,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetPRDiff gets the diff for a pull request
func (c *Client) GetPRDiff(ctx context.Context, owner, repo, prNumber string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%s", c.baseURL, owner, repo, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3.diff")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	diff, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read diff: %w", err)
	}

	return string(diff), nil
}
