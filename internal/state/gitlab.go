package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// GitLabStore is a GitLab artifacts-based state store
type GitLabStore struct {
	artifactsDir string
}

// NewGitLabStore creates a new GitLab store
func NewGitLabStore(config map[string]interface{}) (*GitLabStore, error) {
	artifactsDir, _ := config["artifacts_dir"].(string)
	if artifactsDir == "" {
		artifactsDir = os.Getenv("CI_PROJECT_DIR")
		if artifactsDir == "" {
			artifactsDir = "."
		}
	}

	return &GitLabStore{
		artifactsDir: artifactsDir,
	}, nil
}

// Save saves the workflow execution state to GitLab artifacts
func (s *GitLabStore) Save(ctx context.Context, key string, state interface{}) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	// Include pipeline ID in key for isolation between pipelines
	pipelineID := os.Getenv("CI_PIPELINE_ID")
	if pipelineID != "" {
		key = fmt.Sprintf("%s-pipeline-%s", key, pipelineID)
	}

	// Save to artifacts directory
	stateDir := fmt.Sprintf("%s/.shiro/state", s.artifactsDir)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	filePath := fmt.Sprintf("%s/%s.json", stateDir, key)
	return os.WriteFile(filePath, data, 0644)
}

// Load loads the workflow execution state from GitLab artifacts
func (s *GitLabStore) Load(ctx context.Context, key string, target interface{}) error {
	// Include pipeline ID in key for isolation between pipelines
	pipelineID := os.Getenv("CI_PIPELINE_ID")
	if pipelineID != "" {
		key = fmt.Sprintf("%s-pipeline-%s", key, pipelineID)
	}

	stateDir := fmt.Sprintf("%s/.shiro/state", s.artifactsDir)
	filePath := fmt.Sprintf("%s/%s.json", stateDir, key)

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
func (s *GitLabStore) Delete(ctx context.Context, key string) error {
	// Include pipeline ID in key for isolation between pipelines
	pipelineID := os.Getenv("CI_PIPELINE_ID")
	if pipelineID != "" {
		key = fmt.Sprintf("%s-pipeline-%s", key, pipelineID)
	}

	stateDir := fmt.Sprintf("%s/.shiro/state", s.artifactsDir)
	filePath := fmt.Sprintf("%s/%s.json", stateDir, key)
	return os.Remove(filePath)
}

// Exists checks if a state exists
func (s *GitLabStore) Exists(ctx context.Context, key string) (bool, error) {
	// Include pipeline ID in key for isolation between pipelines
	pipelineID := os.Getenv("CI_PIPELINE_ID")
	if pipelineID != "" {
		key = fmt.Sprintf("%s-pipeline-%s", key, pipelineID)
	}

	stateDir := fmt.Sprintf("%s/.shiro/state", s.artifactsDir)
	filePath := fmt.Sprintf("%s/%s.json", stateDir, key)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
