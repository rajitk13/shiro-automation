package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// SlackModule implements the slack.notify module
type SlackModule struct {
	httpClient *http.Client
}

// NewSlackModule creates a new Slack module
func NewSlackModule() *SlackModule {
	return &SlackModule{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Run executes the Slack notification
func (m *SlackModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	// Type assert to get the step
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	// Extract configuration
	webhookURL, ok := wfStep.Config["webhook_url"].(string)
	if !ok {
		return nil, fmt.Errorf("webhook_url is required")
	}

	channel, _ := wfStep.Config["channel"].(string)
	message, ok := wfStep.Config["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message is required")
	}

	username, _ := wfStep.Config["username"].(string)
	iconEmoji, _ := wfStep.Config["icon_emoji"].(string)

	// Check if this is an interactive approval
	isApproval := false
	if wfStep.Approval != nil {
		isApproval = true
	}

	// Build Slack message
	slackMsg := map[string]interface{}{
		"text": message,
	}

	if channel != "" {
		slackMsg["channel"] = channel
	}
	if username != "" {
		slackMsg["username"] = username
	}
	if iconEmoji != "" {
		slackMsg["icon_emoji"] = iconEmoji
	}

	// Add attachments if provided
	if attachments, ok := wfStep.Config["attachments"].([]interface{}); ok {
		slackMsg["attachments"] = attachments
	}

	// Add interactive buttons for approval
	if isApproval {
		approvalURL, _ := wfStep.Config["approval_url"].(string)
		if approvalURL == "" {
			// Check environment variable first
			approvalURL = os.Getenv("SHIRO_APPROVAL_WEBHOOK_URL")
			if approvalURL == "" {
				// For CI/CD, use GitLab MR URL instead of webhook
				mrURL := os.Getenv("CI_MERGE_REQUEST_URL")
				if mrURL != "" {
					approvalURL = mrURL + "/approvals"
				} else {
					approvalURL = "http://localhost:8080/approve" // Default for local dev
				}
			}
		}
		approvalID := wfStep.Approval.ApprovalID

		approvalAttachment := map[string]interface{}{
			"text": "Please approve or reject this request",
			"actions": []map[string]interface{}{
				{
					"type":  "button",
					"text":  "Approve",
					"style": "primary",
					"url":   fmt.Sprintf("%s?approval_id=%s&decision=approved", approvalURL, approvalID),
				},
				{
					"type":  "button",
					"text":  "Reject",
					"style": "danger",
					"url":   fmt.Sprintf("%s?approval_id=%s&decision=rejected", approvalURL, approvalID),
				},
			},
		}

		if slackMsg["attachments"] == nil {
			slackMsg["attachments"] = []interface{}{approvalAttachment}
		} else {
			slackMsg["attachments"] = append(slackMsg["attachments"].([]interface{}), approvalAttachment)
		}
	}

	// Send to Slack
	body, err := json.Marshal(slackMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("slack API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	result := map[string]interface{}{
		"sent":    true,
		"channel": channel,
		"message": message,
		"status":  "success",
	}

	if isApproval {
		result["approval_id"] = wfStep.Approval.ApprovalID
		result["is_approval"] = true
	}

	return result, nil
}

// Metadata returns module metadata
func (m *SlackModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "slack.notify",
		Description: "Sends a notification to Slack via webhook",
		InputSchema: map[string]modules.SchemaField{
			"webhook_url": {
				Type:        "string",
				Description: "Slack webhook URL",
				Required:    true,
			},
			"channel": {
				Type:        "string",
				Description: "Slack channel to send to",
				Required:    false,
			},
			"message": {
				Type:        "string",
				Description: "Message content",
				Required:    true,
			},
			"username": {
				Type:        "string",
				Description: "Bot username",
				Required:    false,
				Default:     "Shiro",
			},
			"icon_emoji": {
				Type:        "string",
				Description: "Bot icon emoji",
				Required:    false,
				Default:     ":robot_face:",
			},
			"attachments": {
				Type:        "array",
				Description: "Slack message attachments",
				Required:    false,
			},
			"approval_url": {
				Type:        "string",
				Description: "URL for approval webhook",
				Required:    false,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"sent": {
				Type:        "boolean",
				Description: "Whether the message was sent successfully",
				Required:    true,
			},
			"channel": {
				Type:        "string",
				Description: "Channel the message was sent to",
				Required:    true,
			},
			"message": {
				Type:        "string",
				Description: "Message content",
				Required:    true,
			},
			"status": {
				Type:        "string",
				Description: "Status of the operation",
				Required:    true,
			},
		},
	}
}
