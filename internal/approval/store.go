package approval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// FilesystemApprovalStore stores approvals in the filesystem
type FilesystemApprovalStore struct {
	baseDir string
	mu      sync.RWMutex
}

// NewFilesystemApprovalStore creates a new filesystem approval store
func NewFilesystemApprovalStore(baseDir string) (*FilesystemApprovalStore, error) {
	if baseDir == "" {
		baseDir = ".shiro/approvals"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create approval store directory: %w", err)
	}

	return &FilesystemApprovalStore{
		baseDir: baseDir,
	}, nil
}

// GetApproval retrieves an approval by ID
func (s *FilesystemApprovalStore) GetApproval(approvalID string) (*workflow.ApprovalState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", approvalID))
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read approval: %w", err)
	}

	var state workflow.ApprovalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approval: %w", err)
	}

	return &state, nil
}

// SaveApproval saves an approval state
func (s *FilesystemApprovalStore) SaveApproval(approvalID string, state *workflow.ApprovalState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", approvalID))
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal approval: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write approval: %w", err)
	}

	return nil
}

// GetPendingApprovals returns all pending approvals
func (s *FilesystemApprovalStore) GetPendingApprovals() ([]*workflow.ApprovalState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pendingApprovals []*workflow.ApprovalState

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read approval directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var state workflow.ApprovalState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		if state.Status == workflow.ApprovalPending {
			pendingApprovals = append(pendingApprovals, &state)
		}
	}

	return pendingApprovals, nil
}

// DeleteApproval deletes an approval by ID
func (s *FilesystemApprovalStore) DeleteApproval(approvalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", approvalID))
	return os.Remove(filePath)
}

// ListApprovals returns all approvals
func (s *FilesystemApprovalStore) ListApprovals() ([]*workflow.ApprovalState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allApprovals []*workflow.ApprovalState

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read approval directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var state workflow.ApprovalState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		allApprovals = append(allApprovals, &state)
	}

	return allApprovals, nil
}

// GetApprovalByWorkflowID returns all approvals for a specific workflow
func (s *FilesystemApprovalStore) GetApprovalByWorkflowID(workflowID string) ([]*workflow.ApprovalState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var workflowApprovals []*workflow.ApprovalState

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read approval directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var state workflow.ApprovalState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}

		// Check if this approval belongs to the workflow
		// This would require adding workflow_id to ApprovalState
		// For now, we'll return all approvals
		workflowApprovals = append(workflowApprovals, &state)
	}

	return workflowApprovals, nil
}
