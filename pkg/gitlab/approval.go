package gitlab

import (
	"context"
	"fmt"
	"os"

	"github.com/rkuthiala/shiro-automation/internal/gitlab"
	"github.com/rkuthiala/shiro-automation/internal/modules"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// ApprovalModule handles GitLab MR approvals
type ApprovalModule struct {
	client *gitlab.Client
}

// NewApprovalModule creates a new GitLab approval module
func NewApprovalModule() *ApprovalModule {
	return &ApprovalModule{
		client: gitlab.NewClient(),
	}
}

// Run executes the GitLab approval check
func (m *ApprovalModule) Run(ctx context.Context, stepCtx interface{}, step interface{}) (map[string]interface{}, error) {
	wfStep, ok := step.(workflow.Step)
	if !ok {
		return nil, fmt.Errorf("invalid step type")
	}

	projectID := os.Getenv("CI_PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("not running in GitLab CI")
	}

	mrIID := os.Getenv("CI_MERGE_REQUEST_IID")
	if mrIID == "" {
		return nil, fmt.Errorf("not running in a merge request pipeline")
	}

	requiredApprovals := 1
	if req, ok := wfStep.Config["required_approvals"].(int); ok {
		requiredApprovals = req
	}

	// Check MR approval status
	approved, err := m.client.CheckMRApproval(ctx, projectID, mrIID, requiredApprovals)
	if err != nil {
		return nil, fmt.Errorf("failed to check MR approval: %w", err)
	}

	approvalCount, err := m.client.GetApprovalCount(ctx, projectID, mrIID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval count: %w", err)
	}

	return map[string]interface{}{
		"approved":       approved,
		"approval_count": approvalCount,
		"required":       requiredApprovals,
		"mr_iid":         mrIID,
		"project_id":     projectID,
	}, nil
}

// Metadata returns module metadata
func (m *ApprovalModule) Metadata() modules.ModuleMetadata {
	return modules.ModuleMetadata{
		Name:        "gitlab.approval",
		Description: "Checks GitLab MR approval status",
		InputSchema: map[string]modules.SchemaField{
			"required_approvals": {
				Type:        "number",
				Description: "Number of approvals required",
				Required:    false,
				Default:     1,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"approved": {
				Type:        "boolean",
				Description: "Whether MR has required approvals",
				Required:    true,
			},
			"approval_count": {
				Type:        "number",
				Description: "Current number of approvals",
				Required:    true,
			},
			"required": {
				Type:        "number",
				Description: "Required number of approvals",
				Required:    true,
			},
		},
	}
}
