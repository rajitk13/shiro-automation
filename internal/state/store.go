package state

import (
	"context"
	"fmt"
)

// StateStore is the interface for workflow state persistence
type StateStore interface {
	// Save saves the workflow execution state
	Save(ctx context.Context, key string, state interface{}) error

	// Load loads the workflow execution state
	Load(ctx context.Context, key string, target interface{}) error

	// Delete deletes the workflow execution state
	Delete(ctx context.Context, key string) error

	// Exists checks if a state exists
	Exists(ctx context.Context, key string) (bool, error)
}

// StoreFactory creates state stores based on configuration
type StoreFactory struct{}

// NewStoreFactory creates a new store factory
func NewStoreFactory() *StoreFactory {
	return &StoreFactory{}
}

// Create creates a state store based on the type
func (f *StoreFactory) Create(storeType string, config map[string]interface{}) (StateStore, error) {
	switch storeType {
	case "filesystem":
		return NewFilesystemStore(config)
	case "gitlab":
		return NewGitLabStore(config)
	case "memory":
		return NewMemoryStore(), nil
	default:
		return nil, fmt.Errorf("unknown store type: %s", storeType)
	}
}
