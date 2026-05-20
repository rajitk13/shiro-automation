package approval

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// WebhookHandler handles approval webhooks
type WebhookHandler struct {
	approvalStore ApprovalStore
}

// ApprovalStore defines the interface for storing approval state
type ApprovalStore interface {
	GetApproval(approvalID string) (*workflow.ApprovalState, error)
	SaveApproval(approvalID string, state *workflow.ApprovalState) error
	GetPendingApprovals() ([]*workflow.ApprovalState, error)
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(store ApprovalStore) *WebhookHandler {
	return &WebhookHandler{
		approvalStore: store,
	}
}

// HandleApproval handles an approval webhook request
func (h *WebhookHandler) HandleApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	approvalID := r.URL.Query().Get("approval_id")
	if approvalID == "" {
		http.Error(w, "approval_id is required", http.StatusBadRequest)
		return
	}

	var payload struct {
		Decision string `json:"decision"` // approved, rejected
		Reason   string `json:"reason,omitempty"`
		UserID   string `json:"user_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if payload.Decision != "approved" && payload.Decision != "rejected" {
		http.Error(w, "decision must be 'approved' or 'rejected'", http.StatusBadRequest)
		return
	}

	// Get current approval state
	state, err := h.approvalStore.GetApproval(approvalID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get approval: %v", err), http.StatusInternalServerError)
		return
	}

	if state == nil {
		http.Error(w, "Approval not found", http.StatusNotFound)
		return
	}

	// Check if approval has expired
	if time.Now().Unix() > state.ExpiresAt {
		state.Status = workflow.ApprovalTimedOut
		if err := h.approvalStore.SaveApproval(approvalID, state); err != nil {
			log.Printf("Failed to update expired approval: %v", err)
		}
		http.Error(w, "Approval has expired", http.StatusBadRequest)
		return
	}

	// Add approval record
	userID := payload.UserID
	if userID == "" {
		userID = "unknown"
	}

	record := workflow.ApprovalRecord{
		ApproverID: userID,
		Decision:   payload.Decision,
		Reason:     payload.Reason,
		Timestamp:  time.Now().Unix(),
	}

	if state.Approvals == nil {
		state.Approvals = make(map[string]workflow.ApprovalRecord)
	}
	state.Approvals[userID] = record

	// Check if approval is complete
	approvedCount := 0
	rejectedCount := 0
	for _, r := range state.Approvals {
		if r.Decision == "approved" {
			approvedCount++
		} else if r.Decision == "rejected" {
			rejectedCount++
		}
	}

	// Update status based on approvals
	if rejectedCount > 0 {
		state.Status = workflow.ApprovalRejected
	} else if approvedCount >= 1 { // Default to 1 approval required
		state.Status = workflow.ApprovalApproved
		state.DecisionData = map[string]interface{}{
			"approved_by": userID,
			"timestamp":   time.Now().Unix(),
		}
	}

	// Save updated state
	if err := h.approvalStore.SaveApproval(approvalID, state); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save approval: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Approval %s", payload.Decision),
	})
}

// HandlePendingApprovals returns a list of pending approvals
func (h *WebhookHandler) HandlePendingApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	approvals, err := h.approvalStore.GetPendingApprovals()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pending approvals: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approvals)
}

// GenerateApprovalURL generates an approval URL for a given approval ID
func GenerateApprovalURL(baseURL, approvalID string) string {
	return fmt.Sprintf("%s/approve?approval_id=%s", baseURL, approvalID)
}
