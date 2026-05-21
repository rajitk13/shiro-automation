package approval

import "time"

// ApprovalStatus represents the status of an approval request
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusTimeout  ApprovalStatus = "timeout"
)

// ApprovalRequest represents an approval request
type ApprovalRequest struct {
	ID         string                 `json:"id"`
	WorkflowID string                 `json:"workflow_id"`
	StepID     string                 `json:"step_id"`
	Message    string                 `json:"message"`
	Status     ApprovalStatus         `json:"status"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
	ApprovedBy string                 `json:"approved_by,omitempty"`
	RejectedBy string                 `json:"rejected_by,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ApprovalConfig represents approval configuration
type ApprovalConfig struct {
	StoreType     string                 `json:"store_type"`     // gitlab, filesystem, memory
	Timeout       int                    `json:"timeout"`        // timeout in seconds
	TimeoutAction string                 `json:"timeout_action"` // fail, continue, retry
	PollInterval  int                    `json:"poll_interval"`  // polling interval in seconds
	Permissions   string                 `json:"permissions"`    // anyone, users, slack_permissions
	AllowedUsers  []string               `json:"allowed_users,omitempty"`
	StoreConfig   map[string]interface{} `json:"store_config,omitempty"`
}

// TimeoutAction represents the action to take on timeout
type TimeoutAction string

const (
	TimeoutActionFail     TimeoutAction = "fail"
	TimeoutActionContinue TimeoutAction = "continue"
	TimeoutActionRetry    TimeoutAction = "retry"
)

// PermissionMode represents the approval permission mode
type PermissionMode string

const (
	PermissionModeAnyone           PermissionMode = "anyone"
	PermissionModeUsers            PermissionMode = "users"
	PermissionModeSlackPermissions PermissionMode = "slack_permissions"
)
