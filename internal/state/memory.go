package state

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// MemoryStore is an in-memory state store
type MemoryStore struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryStore creates a new memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string][]byte),
	}
}

// Save saves the workflow execution state
func (s *MemoryStore) Save(ctx context.Context, key string, state interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	s.data[key] = data
	return nil
}

// Load loads the workflow execution state
func (s *MemoryStore) Load(ctx context.Context, key string, target interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.data[key]
	if !exists {
		return fmt.Errorf("state not found for key: %s", key)
	}

	return json.Unmarshal(data, target)
}

// Delete deletes the workflow execution state
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return nil
}

// Exists checks if a state exists
func (s *MemoryStore) Exists(ctx context.Context, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.data[key]
	return exists, nil
}
