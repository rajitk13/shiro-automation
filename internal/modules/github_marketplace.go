package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GitHubClient handles GitHub API interactions for module marketplace
type GitHubClient struct {
	httpClient *http.Client
	token      string // Optional GitHub token for private repos
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{},
		token:      token,
	}
}

// SearchResult represents a GitHub search result
type SearchResult struct {
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	HTMLURL     string   `json:"html_url"`
	Stargazers  int      `json:"stargazers_count"`
	Language    string   `json:"language"`
	UpdatedAt   string   `json:"updated_at"`
	Topics      []string `json:"topics"`
}

// GitHubModuleMetadata represents metadata for a module from GitHub
type GitHubModuleMetadata struct {
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	Repository  string   `json:"repository"`
	Stars       int      `json:"stars"`
	Language    string   `json:"language"`
	UpdatedAt   string   `json:"updated_at"`
	Topics      []string `json:"topics"`
}

// SearchModules searches GitHub for shiro modules
func (c *GitHubClient) SearchModules(query string) ([]SearchResult, error) {
	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s+topic:shiro-module+language:go&sort=stars&order=desc", query)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var searchResponse struct {
		Items []SearchResult `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return searchResponse.Items, nil
}

// GetModuleMetadata retrieves module metadata from a GitHub repository
func (c *GitHubClient) GetModuleMetadata(repo string) (*GitHubModuleMetadata, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var repoData struct {
		Name          string   `json:"name"`
		FullName      string   `json:"full_name"`
		Description   string   `json:"description"`
		HTMLURL       string   `json:"html_url"`
		Stargazers    int      `json:"stargazers_count"`
		Language      string   `json:"language"`
		UpdatedAt     string   `json:"updated_at"`
		Topics        []string `json:"topics"`
		DefaultBranch string   `json:"default_branch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Fetch README for additional metadata
	readmeURL := fmt.Sprintf("https://api.github.com/repos/%s/readme", repo)
	readmeReq, err := http.NewRequest("GET", readmeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create README request: %w", err)
	}

	if c.token != "" {
		readmeReq.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
		readmeReq.Header.Add("Accept", "application/vnd.github.v3.raw")
	}

	readmeResp, err := c.httpClient.Do(readmeReq)
	if err == nil && readmeResp.StatusCode == http.StatusOK {
		defer readmeResp.Body.Close()
		// README content can be parsed for additional metadata
		_ = readmeResp.Body // Can be used for extracting documentation
	}

	return &GitHubModuleMetadata{
		Name:        repoData.Name,
		FullName:    repoData.FullName,
		Description: repoData.Description,
		Repository:  repoData.HTMLURL,
		Stars:       repoData.Stargazers,
		Language:    repoData.Language,
		UpdatedAt:   repoData.UpdatedAt,
		Topics:      repoData.Topics,
	}, nil
}

// ParseGitHubRepo parses a GitHub repository URL or owner/repo string
func ParseGitHubRepo(input string) (string, error) {
	input = strings.TrimSpace(input)

	// Handle various input formats
	if strings.HasPrefix(input, "https://github.com/") {
		parts := strings.Split(input, "/")
		if len(parts) >= 5 {
			return fmt.Sprintf("%s/%s", parts[3], parts[4]), nil
		}
	}

	if strings.HasPrefix(input, "github.com/") {
		parts := strings.Split(input, "/")
		if len(parts) >= 3 {
			return fmt.Sprintf("%s/%s", parts[1], parts[2]), nil
		}
	}

	// Assume owner/repo format
	if strings.Contains(input, "/") {
		parts := strings.Split(input, "/")
		if len(parts) == 2 {
			return fmt.Sprintf("%s/%s", parts[0], parts[1]), nil
		}
	}

	return "", fmt.Errorf("invalid GitHub repository format: %s", input)
}
