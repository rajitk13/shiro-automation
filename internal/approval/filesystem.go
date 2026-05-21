package approval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FilesystemStore implements approval state tracking on the filesystem
type FilesystemStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFilesystemStore creates a new filesystem approval store
func NewFilesystemStore(config map[string]interface{}) (*FilesystemStore, error) {
	baseDir := ".shiro/approvals"
	if dir, ok := config["base_dir"].(string); ok {
		baseDir = dir
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create approval directory: %w", err)
	}

	return &FilesystemStore{
		baseDir: baseDir,
	}, nil
}

// CreateRequest creates a new approval request on the filesystem
func (s *FilesystemStore) CreateRequest(req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.baseDir, req.ID+".json")
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("approval request already exists: %s", req.ID)
	}

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal approval request: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write approval request: %w", err)
	}

	return nil
}

// GetRequest retrieves an approval request by ID
func (s *FilesystemStore) GetRequest(id string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
func (s *FilesystemStore) UpdateRequest(req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.baseDir, req.ID+".json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("approval request not found: %s", req.ID)
	}

	req.UpdatedAt = time.Now()

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
func (s *FilesystemStore) ListPending(workflowID string) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*ApprovalRequest

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read approval directory: %w", err)
	}

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
func (s *FilesystemStore) DeleteRequest(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.baseDir, id+".json")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete approval request: %w", err)
	}

	return nil
}
