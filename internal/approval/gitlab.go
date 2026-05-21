package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/gitlab"
)

// GitLabStore implements approval state tracking using GitLab artifacts
type GitLabStore struct {
	client    *gitlab.Client
	projectID string
	jobID     string
	baseDir   string
	baseURL   string
	ctx       context.Context
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

	jobID := os.Getenv("CI_JOB_ID")
	if jobID == "" {
		return nil, fmt.Errorf("CI_JOB_ID environment variable is required")
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
		jobID:     jobID,
		baseDir:   baseDir,
		baseURL:   baseURL,
		ctx:       context.Background(),
	}, nil
}

// uploadArtifact uploads a file as a GitLab artifact
func (s *GitLabStore) uploadArtifact(name, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	artifactPath := filepath.Join("shiro-approvals", name)
	return s.client.UploadArtifact(s.ctx, s.projectID, s.jobID, artifactPath, content)
}

// downloadArtifact downloads a file from GitLab artifacts
func (s *GitLabStore) downloadArtifact(name string) ([]byte, error) {
	artifactPath := filepath.Join("shiro-approvals", name)
	return s.client.DownloadArtifact(s.ctx, s.projectID, s.jobID, artifactPath)
}

// CreateRequest creates a new approval request using GitLab artifacts
func (s *GitLabStore) CreateRequest(req *ApprovalRequest) error {
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	// Store locally first
	filePath := filepath.Join(s.baseDir, req.ID+".json")
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal approval request: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write approval request: %w", err)
	}

	// Upload to GitLab artifacts
	if err := s.uploadArtifact(req.ID+".json", filePath); err != nil {
		// Log error but don't fail - local storage is sufficient
		fmt.Printf("Warning: Failed to upload artifact to GitLab: %v\n", err)
	}

	return nil
}

// GetRequest retrieves an approval request by ID from GitLab artifacts
func (s *GitLabStore) GetRequest(id string) (*ApprovalRequest, error) {
	// Try to download from GitLab artifacts first
	content, err := s.downloadArtifact(id + ".json")
	if err == nil {
		var req ApprovalRequest
		if err := json.Unmarshal(content, &req); err != nil {
			return nil, fmt.Errorf("failed to unmarshal approval request: %w", err)
		}
		return &req, nil
	}

	// Fall back to local storage
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

	// Upload updated artifact to GitLab
	if err := s.uploadArtifact(req.ID+".json", filePath); err != nil {
		// Log error but don't fail - local storage is sufficient
		fmt.Printf("Warning: Failed to upload artifact to GitLab: %v\n", err)
	}

	return nil
}

// ListPending retrieves all pending approval requests for a workflow
func (s *GitLabStore) ListPending(workflowID string) ([]*ApprovalRequest, error) {
	// Try to list artifacts from GitLab
	artifactFiles, err := s.client.ListJobArtifacts(s.ctx, s.projectID, s.jobID)
	if err == nil {
		var pending []*ApprovalRequest
		for _, artifactPath := range artifactFiles {
			if !strings.HasPrefix(filepath.Clean(artifactPath), "shiro-approvals") {
				continue
			}

			content, err := s.client.DownloadArtifact(s.ctx, s.projectID, s.jobID, artifactPath)
			if err != nil {
				continue
			}

			var req ApprovalRequest
			if err := json.Unmarshal(content, &req); err != nil {
				continue
			}

			if req.WorkflowID == workflowID && req.Status == ApprovalStatusPending {
				pending = append(pending, &req)
			}
		}

		if len(pending) > 0 {
			return pending, nil
		}
	}

	// Fall back to local storage
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
