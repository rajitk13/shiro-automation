package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

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
	case "post_inline_comments":
		return m.postInlineComments(ctx, &wfStep)
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
			"operation": {
				Type:        "string",
				Description: "Operation to perform: post_comment, post_inline_comments, get_commit_info, get_user_info, get_mr_participants, get_files_changed",
				Required:    true,
			},
			"body": {
				Type:        "string",
				Description: "Comment body to post (for post_comment and post_inline_comments with text format)",
				Required:    false,
			},
			"comments": {
				Type:        "array",
				Description: "Array of comment objects for post_inline_comments with JSON format",
				Required:    false,
			},
			"output_format": {
				Type:        "string",
				Description: "Output format for post_inline_comments: json or text (default: text)",
				Required:    false,
			},
			"api_type": {
				Type:        "string",
				Description: "API type for post_inline_comments: notes or discussions (default: discussions)",
				Required:    false,
			},
			"dedup": {
				Type:        "boolean",
				Description: "Enable deduplication for post_inline_comments (default: true)",
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
				Description: "Operation success status",
			},
			"message": {
				Type:        "string",
				Description: "Success or error message",
			},
			"info": {
				Type:        "object",
				Description: "Commit or user information (for get_commit_info and get_user_info operations)",
			},
			"participants": {
				Type:        "array",
				Description: "List of MR participants (for get_mr_participants operation)",
			},
			"files": {
				Type:        "array",
				Description: "List of changed files (for get_files_changed operation)",
			},
			"posted_count": {
				Type:        "integer",
				Description: "Number of comments posted (for post_inline_comments operation)",
			},
			"skipped_count": {
				Type:        "integer",
				Description: "Number of comments skipped due to deduplication (for post_inline_comments operation)",
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

// Comment represents a code review comment
type Comment struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Comment string `json:"comment"`
}

// postInlineComments posts line-by-line code review comments
func (m *GitLabModule) postInlineComments(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	// Get configuration options
	outputFormat, _ := step.Config["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "text"
	}

	apiType, _ := step.Config["api_type"].(string)
	if apiType == "" {
		apiType = "discussions"
	}

	dedup := true
	if dedupVal, ok := step.Config["dedup"].(bool); ok {
		dedup = dedupVal
	}

	projectID := GetProjectID()
	mrIID := GetMRID()
	baseSHA := GetCommitSHA()

	if projectID == "" {
		return nil, fmt.Errorf("CI_PROJECT_ID is required")
	}
	if mrIID == "" {
		return nil, fmt.Errorf("CI_MERGE_REQUEST_IID is required")
	}

	// Parse comments from AI output
	var comments []Comment
	var err error

	if outputFormat == "json" {
		comments, err = m.parseJSONComments(step)
	} else {
		comments, err = m.parseTextComments(step)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	if len(comments) == 0 {
		return map[string]interface{}{
			"success":       true,
			"message":       "No comments to post",
			"posted_count":  0,
			"skipped_count": 0,
		}, nil
	}

	// Get existing comments for deduplication
	var existingComments []map[string]interface{}
	if dedup {
		existingComments, err = m.client.GetMRNotes(ctx, projectID, mrIID)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing comments: %w", err)
		}
	}

	// Post comments
	postedCount := 0
	skippedCount := 0

	for _, comment := range comments {
		// Check for duplicates
		if dedup && m.isDuplicate(comment, existingComments) {
			skippedCount++
			continue
		}

		// Post comment based on API type
		if apiType == "discussions" {
			err = m.client.PostMRDiscussion(ctx, projectID, mrIID, comment.Comment, comment.File, comment.Line, comment.Line, baseSHA, baseSHA)
		} else {
			// Use notes API for general comments
			noteBody := fmt.Sprintf("%s:%d - %s", comment.File, comment.Line, comment.Comment)
			err = m.client.PostMRComment(ctx, projectID, mrIID, noteBody)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to post comment on %s:%d: %w", comment.File, comment.Line, err)
		}

		postedCount++
	}

	return map[string]interface{}{
		"success":       true,
		"message":       fmt.Sprintf("Posted %d comments, skipped %d", postedCount, skippedCount),
		"posted_count":  postedCount,
		"skipped_count": skippedCount,
		"comments":      comments,
	}, nil
}

// parseJSONComments parses JSON formatted comments from AI output
func (m *GitLabModule) parseJSONComments(step *workflow.Step) ([]Comment, error) {
	commentsData, ok := step.Config["comments"]
	if !ok {
		return nil, fmt.Errorf("comments field is required for JSON output format")
	}

	// Try to parse as JSON array
	var comments []Comment
	switch v := commentsData.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &comments); err != nil {
			return nil, fmt.Errorf("failed to parse JSON comments: %w", err)
		}
	case []interface{}:
		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			comment := Comment{
				File:    getString(itemMap, "file"),
				Line:    getInt(itemMap, "line"),
				Comment: getString(itemMap, "comment"),
			}
			if comment.File != "" && comment.Comment != "" {
				comments = append(comments, comment)
			}
		}
	default:
		return nil, fmt.Errorf("invalid comments format")
	}

	return comments, nil
}

// parseTextComments parses free text comments with file:line format
func (m *GitLabModule) parseTextComments(step *workflow.Step) ([]Comment, error) {
	content, ok := step.Config["body"].(string)
	if !ok {
		return nil, fmt.Errorf("body field is required for text output format")
	}

	// Pattern to match file:line format
	// Examples: "path/to/file.go:42 - issue description"
	re := regexp.MustCompile(`([^\s:]+):(\d+)\s*[-:]\s*(.+)`)
	matches := re.FindAllStringSubmatch(content, -1)

	var comments []Comment
	for _, match := range matches {
		if len(match) >= 4 {
			line := 0
			if _, err := fmt.Sscanf(match[2], "%d", &line); err == nil {
				comments = append(comments, Comment{
					File:    match[1],
					Line:    line,
					Comment: strings.TrimSpace(match[3]),
				})
			}
		}
	}

	return comments, nil
}

// isDuplicate checks if a comment already exists
func (m *GitLabModule) isDuplicate(comment Comment, existingComments []map[string]interface{}) bool {
	for _, existing := range existingComments {
		body, _ := existing["body"].(string)
		// Check if the comment body contains the file and line
		if strings.Contains(body, comment.File) && strings.Contains(body, fmt.Sprintf("%d", comment.Line)) {
			return true
		}
	}
	return false
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// getInt safely extracts an int value from a map
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}
