package gitlab

import (
	"context"
	"fmt"

	"github.com/rkuthiala/shiro-automation/internal/gitlab"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// GitLabModule implements GitLab operations as a workflow module
type GitLabModule struct {
	client *gitlab.Client
}

// NewGitLabModule creates a new GitLab module
func NewGitLabModule() *GitLabModule {
	return &GitLabModule{
		client: gitlab.NewClient(),
	}
}

// Run executes a GitLab operation
func (m *GitLabModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
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
		return m.postComment(ctx, wfStep)
	case "get_commit_info":
		return m.getCommitInfo(ctx, wfStep)
	case "get_user_info":
		return m.getUserInfo(ctx, wfStep)
	case "get_mr_participants":
		return m.getMRParticipants(ctx, wfStep)
	case "get_files_changed":
		return m.getFilesChanged(ctx, wfStep)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// postComment posts a comment to a merge request
func (m *GitLabModule) postComment(ctx context.Context, step workflow.Step) (map[string]interface{}, error) {
	body, ok := step.Config["body"].(string)
	if !ok || body == "" {
		return nil, fmt.Errorf("body is required for post_comment operation")
	}

	projectID := gitlab.GetProjectID()
	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID environment variable not set")
	}

	mrIID := gitlab.GetMRID()
	if mrIID == "" {
		return nil, fmt.Errorf("CI_MERGE_REQUEST_IID environment variable not set")
	}

	if err := m.client.PostMRComment(ctx, projectID, mrIID, body); err != nil {
		return nil, fmt.Errorf("failed to post MR comment: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"mr_iid":  mrIID,
	}, nil
}

// getCommitInfo gets information about a specific commit
func (m *GitLabModule) getCommitInfo(ctx context.Context, step workflow.Step) (map[string]interface{}, error) {
	commitSHA, ok := step.Config["commit_sha"].(string)
	if !ok || commitSHA == "" {
		commitSHA = gitlab.GetCommitSHA()
	}

	projectID := gitlab.GetProjectID()
	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID environment variable not set")
	}

	info, err := m.client.GetCommitInfo(ctx, projectID, commitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"info":    info,
	}, nil
}

// getUserInfo gets information about a user
func (m *GitLabModule) getUserInfo(ctx context.Context, step workflow.Step) (map[string]interface{}, error) {
	userID, ok := step.Config["user_id"].(string)
	if !ok || userID == "" {
		return nil, fmt.Errorf("user_id is required for get_user_info operation")
	}

	info, err := m.client.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"info":    info,
	}, nil
}

// getMRParticipants gets participants in a merge request
func (m *GitLabModule) getMRParticipants(ctx context.Context, step workflow.Step) (map[string]interface{}, error) {
	projectID := gitlab.GetProjectID()
	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID environment variable not set")
	}

	mrIID := gitlab.GetMRID()
	if mrIID == "" {
		return nil, fmt.Errorf("CI_MERGE_REQUEST_IID environment variable not set")
	}

	participants, err := m.client.GetMRParticipants(ctx, projectID, mrIID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MR participants: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"participants": participants,
	}, nil
}

// getFilesChanged gets list of changed files in a merge request
func (m *GitLabModule) getFilesChanged(ctx context.Context, step workflow.Step) (map[string]interface{}, error) {
	projectID := gitlab.GetProjectID()
	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID environment variable not set")
	}

	mrIID := gitlab.GetMRID()
	if mrIID == "" {
		return nil, fmt.Errorf("CI_MERGE_REQUEST_IID environment variable not set")
	}

	files, err := m.client.GetFilesChanged(ctx, projectID, mrIID)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"files":   files,
	}, nil
}

// Metadata returns module metadata
func (m *GitLabModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "gitlab",
		Description: "GitLab operations for merge requests, commits, and users",
		InputSchema: map[string]modules.SchemaField{
			"operation": {
				Type:        "string",
				Description: "Operation to perform: post_comment, get_commit_info, get_user_info, get_mr_participants, get_files_changed",
				Required:    true,
			},
			"body": {
				Type:        "string",
				Description: "Comment body to post (for post_comment operation)",
				Required:    false,
			},
			"commit_sha": {
				Type:        "string",
				Description: "Commit SHA to get info for (for get_commit_info operation, defaults to CI_COMMIT_SHA)",
				Required:    false,
			},
			"user_id": {
				Type:        "string",
				Description: "User ID to get info for (for get_user_info operation)",
				Required:    false,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"success": {
				Type:        "boolean",
				Description: "Whether the operation succeeded",
				Required:    true,
			},
			"mr_iid": {
				Type:        "string",
				Description: "Merge request IID (for post_comment operation)",
				Required:    false,
			},
			"info": {
				Type:        "object",
				Description: "Information object (for get_commit_info and get_user_info operations)",
				Required:    false,
			},
			"participants": {
				Type:        "array",
				Description: "List of MR participants (for get_mr_participants operation)",
				Required:    false,
			},
			"files": {
				Type:        "array",
				Description: "List of changed files (for get_files_changed operation)",
				Required:    false,
			},
		},
	}
}
