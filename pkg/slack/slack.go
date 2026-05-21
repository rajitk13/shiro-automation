package slack

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// SlackModule implements the slack.notify module
type SlackModule struct {
	httpClient *http.Client
}

// NewSlackModule creates a new Slack module
func NewSlackModule(skipTLSVerify bool) *SlackModule {
	transport := &http.Transport{}
	if skipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &SlackModule{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
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
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required")
	}
	if err := validateHTTPURL("webhook_url", webhookURL); err != nil {
		return nil, err
	}

	channel, _ := wfStep.Config["channel"].(string)
	message, ok := wfStep.Config["message"].(string)
	if !ok || message == "" {
		return nil, fmt.Errorf("message is required")
	}
	username, _ := wfStep.Config["username"].(string)
	iconEmoji, _ := wfStep.Config["icon_emoji"].(string)

	// Check for GitLab pipeline URL for approval link
	gitlabPipelineURL, _ := wfStep.Config["gitlab_pipeline_url"].(string)
	if gitlabPipelineURL != "" {
		if err := validateHTTPURL("gitlab_pipeline_url", gitlabPipelineURL); err != nil {
			return nil, err
		}
	}
	buttonText, _ := wfStep.Config["button_text"].(string)
	if buttonText == "" {
		buttonText = "Review in GitLab"
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

	// Add GitLab review button if pipeline URL is provided
	if gitlabPipelineURL != "" {
		slackMsg["blocks"] = []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": message,
				},
			},
			{
				"type": "actions",
				"actions": []map[string]interface{}{
					{
						"type": "button",
						"text": map[string]interface{}{
							"type": "plain_text",
							"text": buttonText,
						},
						"url":   gitlabPipelineURL,
						"style": "primary",
					},
				},
			},
		}
	}

	// Add attachments if provided
	if attachments, ok := wfStep.Config["attachments"].([]interface{}); ok {
		slackMsg["attachments"] = attachments
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

	return map[string]interface{}{
		"sent":                true,
		"channel":             channel,
		"message":             message,
		"gitlab_pipeline_url": gitlabPipelineURL,
		"status":              "success",
	}, nil
}

func validateHTTPURL(field string, rawURL string) error {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("%s must be a valid URL", field)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("%s must include a host", field)
	}
	return nil
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
			"gitlab_pipeline_url": {
				Type:        "string",
				Description: "GitLab pipeline URL for review button",
				Required:    false,
			},
			"button_text": {
				Type:        "string",
				Description: "Button text for GitLab review link",
				Required:    false,
				Default:     "Review in GitLab",
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
			"gitlab_pipeline_url": {
				Type:        "string",
				Description: "GitLab pipeline URL used for review button",
				Required:    false,
			},
		},
	}
}
