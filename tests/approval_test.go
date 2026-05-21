package tests

import (
	"testing"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/approval"
)

func TestMemoryStore(t *testing.T) {
	store, err := approval.NewMemoryStore()
	if err != nil {
		t.Fatalf("Failed to create memory store: %v", err)
	}

	req := &approval.ApprovalRequest{
		ID:         "test-123",
		WorkflowID: "workflow-1",
		StepID:     "step-1",
		Message:    "Test approval request",
		Status:     approval.ApprovalStatusPending,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	// Test CreateRequest
	if err := store.CreateRequest(req); err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Test GetRequest
	retrieved, err := store.GetRequest("test-123")
	if err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	if retrieved.ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, retrieved.ID)
	}

	if retrieved.Status != approval.ApprovalStatusPending {
		t.Errorf("Expected status %s, got %s", approval.ApprovalStatusPending, retrieved.Status)
	}

	// Test UpdateRequest
	retrieved.Status = approval.ApprovalStatusApproved
	retrieved.ApprovedBy = "test-user"
	if err := store.UpdateRequest(retrieved); err != nil {
		t.Fatalf("Failed to update request: %v", err)
	}

	// Verify update
	updated, err := store.GetRequest("test-123")
	if err != nil {
		t.Fatalf("Failed to get updated request: %v", err)
	}

	if updated.Status != approval.ApprovalStatusApproved {
		t.Errorf("Expected status %s, got %s", approval.ApprovalStatusApproved, updated.Status)
	}

	if updated.ApprovedBy != "test-user" {
		t.Errorf("Expected approved_by %s, got %s", "test-user", updated.ApprovedBy)
	}

	// Test DeleteRequest
	if err := store.DeleteRequest("test-123"); err != nil {
		t.Fatalf("Failed to delete request: %v", err)
	}

	// Verify deletion
	_, err = store.GetRequest("test-123")
	if err == nil {
		t.Error("Expected error when getting deleted request, got nil")
	}
}

func TestFilesystemStore(t *testing.T) {
	config := map[string]interface{}{
		"base_dir": "/tmp/test-approvals",
	}
	store, err := approval.NewFilesystemStore(config)
	if err != nil {
		t.Fatalf("Failed to create filesystem store: %v", err)
	}

	req := &approval.ApprovalRequest{
		ID:         "test-456",
		WorkflowID: "workflow-2",
		StepID:     "step-2",
		Message:    "Test filesystem approval",
		Status:     approval.ApprovalStatusPending,
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	// Test CreateRequest
	if err := store.CreateRequest(req); err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Test GetRequest
	retrieved, err := store.GetRequest("test-456")
	if err != nil {
		t.Fatalf("Failed to get request: %v", err)
	}

	if retrieved.ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, retrieved.ID)
	}

	// Test UpdateRequest
	retrieved.Status = approval.ApprovalStatusRejected
	retrieved.RejectedBy = "test-user"
	if err := store.UpdateRequest(retrieved); err != nil {
		t.Fatalf("Failed to update request: %v", err)
	}

	// Verify update
	updated, err := store.GetRequest("test-456")
	if err != nil {
		t.Fatalf("Failed to get updated request: %v", err)
	}

	if updated.Status != approval.ApprovalStatusRejected {
		t.Errorf("Expected status %s, got %s", approval.ApprovalStatusRejected, updated.Status)
	}

	// Test DeleteRequest
	if err := store.DeleteRequest("test-456"); err != nil {
		t.Fatalf("Failed to delete request: %v", err)
	}

	// Cleanup
	_ = store.DeleteRequest("test-456")
}

func TestApprovalTypes(t *testing.T) {
	// Test ApprovalStatus constants
	if approval.ApprovalStatusPending != "pending" {
		t.Errorf("Expected pending, got %s", approval.ApprovalStatusPending)
	}
	if approval.ApprovalStatusApproved != "approved" {
		t.Errorf("Expected approved, got %s", approval.ApprovalStatusApproved)
	}
	if approval.ApprovalStatusRejected != "rejected" {
		t.Errorf("Expected rejected, got %s", approval.ApprovalStatusRejected)
	}
	if approval.ApprovalStatusTimeout != "timeout" {
		t.Errorf("Expected timeout, got %s", approval.ApprovalStatusTimeout)
	}

	// Test TimeoutAction constants
	if approval.TimeoutActionFail != "fail" {
		t.Errorf("Expected fail, got %s", approval.TimeoutActionFail)
	}
	if approval.TimeoutActionContinue != "continue" {
		t.Errorf("Expected continue, got %s", approval.TimeoutActionContinue)
	}
	if approval.TimeoutActionRetry != "retry" {
		t.Errorf("Expected retry, got %s", approval.TimeoutActionRetry)
	}

	// Test PermissionMode constants
	if approval.PermissionModeAnyone != "anyone" {
		t.Errorf("Expected anyone, got %s", approval.PermissionModeAnyone)
	}
	if approval.PermissionModeUsers != "users" {
		t.Errorf("Expected users, got %s", approval.PermissionModeUsers)
	}
	if approval.PermissionModeSlackPermissions != "slack_permissions" {
		t.Errorf("Expected slack_permissions, got %s", approval.PermissionModeSlackPermissions)
	}
}

func TestApprovalStoreFactory(t *testing.T) {
	// Test memory store creation
	config := &approval.ApprovalConfig{
		StoreType: "memory",
	}
	store, err := approval.NewStore(config)
	if err != nil {
		t.Fatalf("Failed to create memory store via factory: %v", err)
	}
	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Test filesystem store creation
	config = &approval.ApprovalConfig{
		StoreType: "filesystem",
		StoreConfig: map[string]interface{}{
			"base_dir": "/tmp/test-approvals-factory",
		},
	}
	store, err = approval.NewStore(config)
	if err != nil {
		t.Fatalf("Failed to create filesystem store via factory: %v", err)
	}
	if store == nil {
		t.Fatal("Expected non-nil store")
	}

	// Test invalid store type
	config = &approval.ApprovalConfig{
		StoreType: "invalid",
	}
	_, err = approval.NewStore(config)
	if err == nil {
		t.Error("Expected error for invalid store type, got nil")
	}
}
