package approval

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/gitlab"
)

// GitLabStore implements approval state tracking using GitLab artifacts
type GitLabStore struct {
	client    *gitlab.Client
	projectID string
	baseDir   string
	baseURL   string
}

// NewGitLabStore creates a new GitLab approval store
func NewGitLabStore(config map[string]interface{}) (*GitLabStore, error) {
	client := gitlab.NewClient()

	baseURL := os.Getenv("CI_SERVER_URL")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	projectID := os.Getenv("CI_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID environment variable is required")
	}

	baseDir := "/tmp/shiro-approvals"
	if dir, ok := config["base_dir"].(string); ok {
		baseDir = dir
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create approval directory: %w", err)
	}

	return &GitLabStore{
		client:    client,
		projectID: projectID,
		baseDir:   baseDir,
		baseURL:   baseURL,
	}, nil
}

// CreateRequest creates a new approval request using GitLab artifacts
func (s *GitLabStore) CreateRequest(req *ApprovalRequest) error {
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	// Store locally
	filePath := filepath.Join(s.baseDir, req.ID+".json")
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal approval request: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write approval request: %w", err)
	}

	// Note: In a full implementation, this would upload to GitLab artifacts
	// For now, we use filesystem storage in GitLab CI environment
	// where artifacts are automatically collected from specified directories

	return nil
}

// GetRequest retrieves an approval request by ID from local storage
func (s *GitLabStore) GetRequest(id string) (*ApprovalRequest, error) {
	filePath := filepath.Join(s.baseDir, id+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read approval request: %w", err)
	}

	var req ApprovalRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approval request: %w", err)
	}

	return &req, nil
}

// UpdateRequest updates an approval request
func (s *GitLabStore) UpdateRequest(req *ApprovalRequest) error {
	req.UpdatedAt = time.Now()

	filePath := filepath.Join(s.baseDir, req.ID+".json")
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal approval request: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write approval request: %w", err)
	}

	return nil
}

// ListPending retrieves all pending approval requests for a workflow
func (s *GitLabStore) ListPending(workflowID string) ([]*ApprovalRequest, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read approval directory: %w", err)
	}

	var pending []*ApprovalRequest
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var req ApprovalRequest
		if err := json.Unmarshal(data, &req); err != nil {
			continue
		}

		if req.WorkflowID == workflowID && req.Status == ApprovalStatusPending {
			pending = append(pending, &req)
		}
	}

	return pending, nil
}

// DeleteRequest deletes an approval request
func (s *GitLabStore) DeleteRequest(id string) error {
	filePath := filepath.Join(s.baseDir, id+".json")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete approval request: %w", err)
	}

	return nil
}

// uploadArtifact uploads a file as a GitLab artifact
func (s *GitLabStore) uploadArtifact(name, filePath string) error {
	// This would use GitLab API to upload artifacts
	// For now, it's a placeholder for future implementation
	return nil
}

// downloadArtifact downloads a file from GitLab artifacts
func (s *GitLabStore) downloadArtifact(name string) (io.ReadCloser, error) {
	// This would use GitLab API to download artifacts
	// For now, it's a placeholder for future implementation
	return nil, fmt.Errorf("not implemented")
}
