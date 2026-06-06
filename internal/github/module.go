package github

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// GitHubModule implements the GitHub module for workflow operations
type GitHubModule struct {
	client *Client
}

// NewGitHubModule creates a new GitHub module
func NewGitHubModule() *GitHubModule {
	return &GitHubModule{
		client: NewClient(),
	}
}

func init() {
	modules.RegisterBuiltin("github", func() (interface{}, error) {
		return NewGitHubModule(), nil
	})
}

// Execute executes a GitHub module step
func (m *GitHubModule) Execute(ctx context.Context, wfStep *workflow.Step) (map[string]interface{}, error) {
	operation, ok := wfStep.Config["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation is required")
	}

	switch operation {
	case "get_diff":
		return m.getDiff(ctx, wfStep)
	case "post_comment":
		return m.postComment(ctx, wfStep)
	case "post_inline_comments":
		return m.postInlineComments(ctx, wfStep)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// Metadata returns module metadata
func (m *GitHubModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "github",
		Description: "GitHub API operations for pull requests and repositories",
		InputSchema: map[string]modules.SchemaField{
			"operation": {
				Type:        "string",
				Description: "Operation to perform: get_diff, post_comment, post_inline_comments",
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
			"dedup": {
				Type:        "boolean",
				Description: "Enable deduplication for post_inline_comments (default: true)",
				Required:    false,
			},
			"commit_id": {
				Type:        "string",
				Description: "Commit SHA for review comments (defaults to GITHUB_SHA)",
				Required:    false,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"success": {
				Type:        "boolean",
				Description: "Whether the operation succeeded",
			},
			"message": {
				Type:        "string",
				Description: "Status message",
			},
			"posted_count": {
				Type:        "integer",
				Description: "Number of comments posted (for post_inline_comments)",
			},
			"skipped_count": {
				Type:        "integer",
				Description: "Number of comments skipped due to deduplication (for post_inline_comments)",
			},
			"comments": {
				Type:        "array",
				Description: "Array of posted comments (for post_inline_comments)",
			},
		},
	}
}

// postComment posts a general comment to a pull request
func (m *GitHubModule) postComment(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	body, ok := step.Config["body"].(string)
	if !ok {
		return nil, fmt.Errorf("body is required for post_comment operation")
	}

	owner := GetOwner()
	repo := GetRepoName()
	prNumber := GetPRNumber()

	if owner == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY_OWNER is required")
	}
	if repo == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY is required")
	}
	if prNumber == "" {
		return nil, fmt.Errorf("GITHUB_PR_NUMBER is required")
	}

	if err := m.client.PostPRComment(ctx, owner, repo, prNumber, body); err != nil {
		return nil, fmt.Errorf("failed to post comment: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "Comment posted successfully",
	}, nil
}

// getDiff gets the diff for a pull request using GitHub API
func (m *GitHubModule) getDiff(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	owner := GetOwner()
	repo := GetRepoName()
	prNumber := GetPRNumber()

	if owner == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY_OWNER is required")
	}
	if repo == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY is required")
	}
	if prNumber == "" {
		return nil, fmt.Errorf("GITHUB_PR_NUMBER is required")
	}

	diff, err := m.client.GetPRDiff(ctx, owner, repo, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR diff: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"diff":    diff,
	}, nil
}

// Comment represents a code review comment
type Comment struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Comment string `json:"comment"`
}

// postInlineComments posts line-by-line code review comments
func (m *GitHubModule) postInlineComments(ctx context.Context, step *workflow.Step) (map[string]interface{}, error) {
	// Get configuration options
	outputFormat, _ := step.Config["output_format"].(string)
	if outputFormat == "" {
		outputFormat = "text"
	}

	dedup := true
	if dedupVal, ok := step.Config["dedup"].(bool); ok {
		dedup = dedupVal
	}

	commitID := GetCommitSHA()
	if customCommitID, ok := step.Config["commit_id"].(string); ok && customCommitID != "" {
		commitID = customCommitID
	}

	owner := GetOwner()
	repo := GetRepoName()
	prNumber := GetPRNumber()

	if owner == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY_OWNER is required")
	}
	if repo == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY is required")
	}
	if prNumber == "" {
		return nil, fmt.Errorf("GITHUB_PR_NUMBER is required")
	}
	if commitID == "" {
		return nil, fmt.Errorf("GITHUB_SHA is required")
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
		existingComments, err = m.client.GetPRReviewComments(ctx, owner, repo, prNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing comments: %w", err)
		}
	}

	// Convert to review comments and post
	postedCount := 0
	skippedCount := 0
	var reviewComments []ReviewComment

	for _, comment := range comments {
		// Check for duplicates
		if dedup && m.isDuplicate(comment, existingComments) {
			skippedCount++
			continue
		}

		reviewComments = append(reviewComments, ReviewComment{
			Path:     comment.File,
			Position: comment.Line,
			Body:     comment.Comment,
		})
		postedCount++
	}

	// Post as a single review with all comments
	if len(reviewComments) > 0 {
		body := "AI Code Review"
		if err := m.client.CreateReview(ctx, owner, repo, prNumber, body, reviewComments, commitID); err != nil {
			return nil, fmt.Errorf("failed to create review: %w", err)
		}
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
func (m *GitHubModule) parseJSONComments(step *workflow.Step) ([]Comment, error) {
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
func (m *GitHubModule) parseTextComments(step *workflow.Step) ([]Comment, error) {
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
func (m *GitHubModule) isDuplicate(comment Comment, existingComments []map[string]interface{}) bool {
	for _, existing := range existingComments {
		if path, ok := existing["path"].(string); ok && path == comment.File {
			if body, ok := existing["body"].(string); ok && strings.Contains(body, comment.Comment) {
				return true
			}
		}
	}
	return false
}

// Helper functions for parsing
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	switch val := m[key].(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		var i int
		fmt.Sscanf(val, "%d", &i)
		return i
	}
	return 0
}
