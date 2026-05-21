package approval

import (
	"fmt"
	"sync"
	"time"
)

// MemoryStore implements approval state tracking in memory
type MemoryStore struct {
	requests map[string]*ApprovalRequest
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory approval store
func NewMemoryStore() (*MemoryStore, error) {
	return &MemoryStore{
		requests: make(map[string]*ApprovalRequest),
	}, nil
}

// CreateRequest creates a new approval request in memory
func (s *MemoryStore) CreateRequest(req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.requests[req.ID]; exists {
		return fmt.Errorf("approval request already exists: %s", req.ID)
	}

	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	s.requests[req.ID] = req

	return nil
}

// GetRequest retrieves an approval request by ID
func (s *MemoryStore) GetRequest(id string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	req, exists := s.requests[id]
	if !exists {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	return req, nil
}

// UpdateRequest updates an approval request
func (s *MemoryStore) UpdateRequest(req *ApprovalRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.requests[req.ID]; !exists {
		return fmt.Errorf("approval request not found: %s", req.ID)
	}

	req.UpdatedAt = time.Now()
	s.requests[req.ID] = req

	return nil
}

// ListPending retrieves all pending approval requests for a workflow
func (s *MemoryStore) ListPending(workflowID string) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*ApprovalRequest
	for _, req := range s.requests {
		if req.WorkflowID == workflowID && req.Status == ApprovalStatusPending {
			pending = append(pending, req)
		}
	}

	return pending, nil
}

// DeleteRequest deletes an approval request
func (s *MemoryStore) DeleteRequest(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.requests[id]; !exists {
		return fmt.Errorf("approval request not found: %s", id)
	}

	delete(s.requests, id)
	return nil
}
