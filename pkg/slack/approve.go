package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/approval"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// SlackApproveModule implements the slack.approve module for human-in-loop approvals
type SlackApproveModule struct {
	slackModule   *SlackModule
	approvalStore approval.Store
	logger        *log.Logger
}

// NewSlackApproveModule creates a new Slack approval module
func NewSlackApproveModule(slackModule *SlackModule) *SlackApproveModule {
	// Create approval store based on environment variable
	approvalStoreType := os.Getenv("SHIRO_APPROVAL_STORE")
	if approvalStoreType == "" {
		approvalStoreType = "memory" // default to memory for local development
	}

	approvalConfig := &approval.ApprovalConfig{
		StoreType: approvalStoreType,
	}
	approvalStore, err := approval.NewStore(approvalConfig)
	if err != nil {
		// Fallback to memory store if creation fails
		approvalStore, _ = approval.NewStore(&approval.ApprovalConfig{StoreType: "memory"})
	}

	logger := log.New(os.Stdout, "[Approval] ", log.LstdFlags)

	return &SlackApproveModule{
		slackModule:   slackModule,
		approvalStore: approvalStore,
		logger:        logger,
	}
}

// Run executes the Slack approval workflow
func (m *SlackApproveModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
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

	timeout := 3600 // default 1 hour
	if t, ok := wfStep.Config["timeout"].(float64); ok {
		timeout = int(t)
	}

	pollInterval := 30 // default 30 seconds
	if p, ok := wfStep.Config["poll_interval"].(float64); ok {
		pollInterval = int(p)
	}

	timeoutAction := "fail"
	if ta, ok := wfStep.Config["timeout_action"].(string); ok {
		timeoutAction = ta
	}

	permissions := "anyone"
	if perm, ok := wfStep.Config["permissions"].(string); ok {
		permissions = perm
	}

	var allowedUsers []string
	if users, ok := wfStep.Config["allowed_users"].([]interface{}); ok {
		for _, user := range users {
			if userStr, ok := user.(string); ok {
				allowedUsers = append(allowedUsers, userStr)
			}
		}
	}

	// Check for non-blocking mode
	blocking := true
	if b, ok := wfStep.Config["blocking"].(bool); ok {
		blocking = b
	}

	// Validate permissions
	if permissions == "users" && len(allowedUsers) == 0 {
		return nil, fmt.Errorf("permissions mode 'users' requires allowed_users to be specified")
	}

	// Generate approval ID
	approvalID := fmt.Sprintf("approval-%d", time.Now().Unix())

	// Create approval URLs (in production, these would be real webhook endpoints)
	approveURL := fmt.Sprintf("https://your-domain.com/approve?id=%s&action=approve", approvalID)
	rejectURL := fmt.Sprintf("https://your-domain.com/approve?id=%s&action=reject", approvalID)

	// Create approval request
	approvalReq := &approval.ApprovalRequest{
		ID:         approvalID,
		WorkflowID: wfStep.ID, // Using step ID as workflow ID for simplicity
		StepID:     wfStep.ID,
		Message:    message,
		Status:     approval.ApprovalStatusPending,
		ExpiresAt:  time.Now().Add(time.Duration(timeout) * time.Second),
	}

	if err := m.approvalStore.CreateRequest(approvalReq); err != nil {
		return nil, fmt.Errorf("failed to create approval request: %w", err)
	}

	// Send approval request to Slack with interactive buttons
	slackMsg := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color": "#3AA3E3",
				"text":  message,
				"actions": []map[string]interface{}{
					{
						"type":  "button",
						"text":  "Approve",
						"url":   approveURL,
						"style": "primary",
					},
					{
						"type":  "button",
						"text":  "Reject",
						"url":   rejectURL,
						"style": "danger",
					},
				},
			},
		},
	}

	if channel != "" {
		slackMsg["channel"] = channel
	}

	body, err := json.Marshal(slackMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal approval message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create approval request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.slackModule.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send approval request to Slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("slack API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	m.logger.Printf("Approval request %s sent, waiting for approval...", approvalID)

	// Non-blocking mode: return immediately with pending status
	if !blocking {
		m.logger.Printf("Non-blocking mode: returning immediately with pending status")
		return map[string]interface{}{
			"status":      "pending",
			"approval_id": approvalID,
			"blocking":    false,
		}, nil
	}

	// Poll for approval status (blocking mode)
	pollTicker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer pollTicker.Stop()

	timeoutChan := time.After(time.Duration(timeout) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-timeoutChan:
			// Handle timeout based on timeout_action
			switch timeoutAction {
			case "fail":
				approvalReq.Status = approval.ApprovalStatusTimeout
				m.approvalStore.UpdateRequest(approvalReq)
				return nil, fmt.Errorf("approval request timed out after %d seconds", timeout)

			case "continue":
				approvalReq.Status = approval.ApprovalStatusTimeout
				m.approvalStore.UpdateRequest(approvalReq)
				return map[string]interface{}{
					"status":         "timeout",
					"approval_id":    approvalID,
					"timeout_action": timeoutAction,
				}, nil

			case "retry":
				// Resend approval request
				resp, err := m.slackModule.httpClient.Do(req)
				if err != nil {
					return nil, fmt.Errorf("failed to resend approval request: %w", err)
				}
				defer resp.Body.Close()
				// Reset timer and continue polling
				timeoutChan = time.After(time.Duration(timeout) * time.Second)
				continue

			default:
				return nil, fmt.Errorf("unknown timeout action: %s", timeoutAction)
			}

		case <-pollTicker.C:
			// Check approval status
			currentReq, err := m.approvalStore.GetRequest(approvalID)
			if err != nil {
				m.logger.Printf("Failed to check approval status: %v", err)
				continue
			}

			switch currentReq.Status {
			case approval.ApprovalStatusApproved:
				m.logger.Printf("Approval request %s approved by %s", approvalID, currentReq.ApprovedBy)
				return map[string]interface{}{
					"status":      "approved",
					"approval_id": approvalID,
					"approved_by": currentReq.ApprovedBy,
					"approved_at": currentReq.UpdatedAt,
				}, nil

			case approval.ApprovalStatusRejected:
				m.logger.Printf("Approval request %s rejected by %s", approvalID, currentReq.RejectedBy)
				return map[string]interface{}{
					"status":      "rejected",
					"approval_id": approvalID,
					"rejected_by": currentReq.RejectedBy,
					"rejected_at": currentReq.UpdatedAt,
				}, nil

			case approval.ApprovalStatusPending:
				// Still pending, continue polling
				continue

			default:
				return nil, fmt.Errorf("unknown approval status: %s", currentReq.Status)
			}
		}
	}
}

// Metadata returns module metadata
func (m *SlackApproveModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "slack.approve",
		Description: "Sends an approval request to Slack and waits for approval",
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
				Description: "Approval message",
				Required:    true,
			},
			"timeout": {
				Type:        "number",
				Description: "Timeout in seconds (default: 3600)",
				Required:    false,
				Default:     3600,
			},
			"poll_interval": {
				Type:        "number",
				Description: "Polling interval in seconds (default: 30)",
				Required:    false,
				Default:     30,
			},
			"timeout_action": {
				Type:        "string",
				Description: "Action on timeout: fail, continue, retry (default: fail)",
				Required:    false,
				Default:     "fail",
			},
			"blocking": {
				Type:        "boolean",
				Description: "Block until approval (default: true)",
				Required:    false,
				Default:     true,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"status": {
				Type:        "string",
				Description: "Approval status: approved, rejected, timeout",
				Required:    true,
			},
			"approval_id": {
				Type:        "string",
				Description: "Approval request ID",
				Required:    true,
			},
		},
	}
}
