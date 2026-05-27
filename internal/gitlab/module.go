package gitlab

import (
	"context"
	"fmt"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// GitLabModule implements GitLab operations as a workflow module
type GitLabModule struct {
	client *Client
}

// NewGitLabModule creates a new GitLab module
func NewGitLabModule() *GitLabModule {
	return &GitLabModule{
		client: NewClient(),
	}
}

// Run executes a GitLab operation
func (m *GitLabModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	// Type assert to get the step
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	operation, ok := wfStep.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation is required")
	}

	switch operation {
	case "post_comment":
		return m.postComment(ctx, &wfStep)
	case "get_commit_info":
		return m.getCommitInfo(ctx, &wfStep)
	case "get_user_info":
		return m.getUserInfo(ctx, &wfStep)
	case "get_mr_participants":
		return m.getMRParticipants(ctx, &wfStep)
	case "get_files_changed":
		return m.getFilesChanged(ctx, &wfStep)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// Metadata returns module metadata
func (m *GitLabModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "gitlab",
		Description: "GitLab operations for merge requests, commits, and users",
		InputSchema: map[string]modules.SchemaField{
			"operation": modules.SchemaField{
				Type:        "string",
				Description: "Operation to perform: post_comment, get_commit_info, get_user_info, get_mr_participants, get_files_changed",
				Required:    true,
			},
			"body": modules.SchemaField{
				Type:        "string",
				Description: "Comment body to post (for post_comment operation)",
				Required:    false,
			},
			"commit_sha": modules.SchemaField{
				Type:        "string",
				Description: "Commit SHA to get info for (for get_commit_info operation, defaults to CI_COMMIT_SHA)",
				Required:    false,
			},
			"user_id": modules.SchemaField{
				Type:        "string",
				Description: "User ID to get info for (for get_user_info operation)",
				Required:    false,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"success": modules.SchemaField{
				Type:        "boolean",
				Description: "Operation success status",
			},
			"message": modules.SchemaField{
				Type:        "string",
				Description: "Success or error message",
			},
			"info": modules.SchemaField{
				Type:        "object",
				Description: "Commit or user information (for get_commit_info and get_user_info operations)",
			},
			"participants": modules.SchemaField{
				Type:        "array",
				Description: "List of MR participants (for get_mr_participants operation)",
			},
			"files": modules.SchemaField{
				Type:        "array",
				Description: "List of changed files (for get_files_changed operation)",
			},
		},
	}
}

// postComment posts a comment to a merge request
func (m *GitLabModule) postComment(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	body, ok := step.Config["body"].(string)
	if !ok {
		return nil, fmt.Errorf("body is required for post_comment operation")
	}

	projectID := GetProjectID()
	mrIID := GetMRID()

	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID is required")
	}
	if mrIID == "" {
		return nil, fmt.Errorf("CI_MERGE_REQUEST_IID is required")
	}

	if err := m.client.PostMRComment(ctx, projectID, mrIID, body); err != nil {
		return nil, fmt.Errorf("failed to post comment: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Comment posted successfully",
	}, nil
}

// getCommitInfo gets information about a specific commit
func (m *GitLabModule) getCommitInfo(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	commitSHA, ok := step.Config["commit_sha"].(string)
	if !ok || commitSHA == "" {
		commitSHA = GetCommitSHA()
	}

	projectID := GetProjectID()
	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID is required")
	}

	info, err := m.client.GetCommitInfo(ctx, projectID, commitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info: %w", err)
	}

	return map[string]interface{}{
		"info": info,
	}, nil
}

// getUserInfo gets information about a user
func (m *GitLabModule) getUserInfo(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	userID, ok := step.Config["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id is required for get_user_info operation")
	}

	info, err := m.client.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return map[string]interface{}{
		"info": info,
	}, nil
}

// getMRParticipants gets participants in a merge request
func (m *GitLabModule) getMRParticipants(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	projectID := GetProjectID()
	mrIID := GetMRID()

	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID is required")
	}
	if mrIID == "" {
		return nil, fmt.Errorf("CI_MERGE_REQUEST_IID is required")
	}

	participants, err := m.client.GetMRParticipants(ctx, projectID, mrIID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MR participants: %w", err)
	}

	return map[string]interface{}{
		"participants": participants,
	}, nil
}

// getFilesChanged gets list of changed files in a merge request
func (m *GitLabModule) getFilesChanged(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	projectID := GetProjectID()
	mrIID := GetMRID()

	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID is required")
	}
	if mrIID == "" {
		return nil, fmt.Errorf("CI_MERGE_REQUEST_IID is required")
	}

	files, err := m.client.GetFilesChanged(ctx, projectID, mrIID)
	if err != nil {
		return nil, fmt.Errorf("failed to get files changed: %w", err)
	}

	return map[string]interface{}{
		"files": files,
	}, nil
}
