package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FilesystemStore is a filesystem-based state store
type FilesystemStore struct {
	baseDir string
}

// NewFilesystemStore creates a new filesystem store
func NewFilesystemStore(config map[string]interface{}) (*FilesystemStore, error) {
	baseDir, ok := config["base_dir"].(string)
	if !ok {
		baseDir = "./state"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	return &FilesystemStore{
		baseDir: baseDir,
	}, nil
}

// Save saves the workflow execution state
func (s *FilesystemStore) Save(ctx context.Context, key string, state interface{}) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.baseDir, key+".json")
	return os.WriteFile(filePath, data, 0644)
}

// Load loads the workflow execution state
func (s *FilesystemStore) Load(ctx context.Context, key string, target interface{}) error {
	filePath := filepath.Join(s.baseDir, key+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("state not found for key: %s", key)
		}
		return err
	}

	return json.Unmarshal(data, target)
}

// Delete deletes the workflow execution state
func (s *FilesystemStore) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(s.baseDir, key+".json")
	return os.Remove(filePath)
}

// Exists checks if a state exists
func (s *FilesystemStore) Exists(ctx context.Context, key string) (bool, error) {
	filePath := filepath.Join(s.baseDir, key+".json")
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
