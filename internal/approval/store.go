package approval

import "fmt"

// Store defines the interface for approval state tracking
type Store interface {
	// CreateRequest creates a new approval request
	CreateRequest(req *ApprovalRequest) error

	// GetRequest retrieves an approval request by ID
	GetRequest(id string) (*ApprovalRequest, error)

	// UpdateRequest updates an approval request
	UpdateRequest(req *ApprovalRequest) error

	// ListPending retrieves all pending approval requests for a workflow
	ListPending(workflowID string) ([]*ApprovalRequest, error)

	// DeleteRequest deletes an approval request
	DeleteRequest(id string) error
}

// NewStore creates a new store based on the configuration
func NewStore(config *ApprovalConfig) (Store, error) {
	switch config.StoreType {
	case "gitlab":
		return NewGitLabStore(config.StoreConfig)
	case "filesystem":
		return NewFilesystemStore(config.StoreConfig)
	case "memory":
		return NewMemoryStore()
	default:
		return nil, fmt.Errorf("unknown store type: %s", config.StoreType)
	}
}
