package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rkuthiala/shiro-automation/internal/config"
	"github.com/rkuthiala/shiro-automation/internal/state"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// ApproveCommand handles the approval command
func ApproveCommand(args []string) {
	if len(args) < 1 {
		printApproveHelp()
		os.Exit(1)
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "list":
		listPendingApprovals()
	case "approve":
		approveRequest(subArgs)
	case "reject":
		rejectRequest(subArgs)
	case "help":
		printApproveHelp()
	default:
		fmt.Printf("Unknown approval command: %s\n", subcommand)
		printApproveHelp()
		os.Exit(1)
	}
}

// listPendingApprovals lists all pending approvals
func listPendingApprovals() {
	cfg, err := config.LoadConfig(".shiro")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create state store
	stateFactory := state.NewStoreFactory()
	_, err = stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	// Load all workflow states
	fmt.Println("Pending Approvals:")
	fmt.Println()

	// In a real implementation, this would query a dedicated approval store
	// For now, we'll just show a message
	fmt.Println("No pending approvals found.")
	fmt.Println()
	fmt.Println("Note: Approval tracking requires workflow execution with approval steps.")
}

// approveRequest approves a pending approval
func approveRequest(args []string) {
	if len(args) < 1 {
		log.Fatal("Approval ID is required")
	}

	approvalID := args[0]
	reason := ""
	if len(args) > 1 {
		reason = args[1]
	}

	cfg, err := config.LoadConfig(".shiro")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create state store
	stateFactory := state.NewStoreFactory()
	_, err = stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	// In a real implementation, this would:
	// 1. Load the approval state from the approval store
	// 2. Add the approval record
	// 3. Update the approval status
	// 4. Trigger workflow resumption if all approvals are met

	fmt.Printf("Approving request: %s\n", approvalID)
	if reason != "" {
		fmt.Printf("Reason: %s\n", reason)
	}

	// Create approval record
	record := workflow.ApprovalRecord{
		ApproverID: os.Getenv("USER"), // Use current user as approver
		Decision:   "approved",
		Reason:     reason,
		Timestamp:  time.Now().Unix(),
	}

	// In a real implementation, save this to the approval store
	fmt.Println("Approval record created:")
	approvalJSON, _ := json.MarshalIndent(record, "  ", "  ")
	fmt.Printf("  %s\n", string(approvalJSON))

	fmt.Println()
	fmt.Println("Note: Full approval tracking and workflow resumption requires approval store implementation.")
}

// rejectRequest rejects a pending approval
func rejectRequest(args []string) {
	if len(args) < 1 {
		log.Fatal("Approval ID is required")
	}

	approvalID := args[0]
	reason := ""
	if len(args) > 1 {
		reason = args[1]
	}

	if reason == "" {
		log.Fatal("Reason is required for rejection")
	}

	cfg, err := config.LoadConfig(".shiro")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create state store
	stateFactory := state.NewStoreFactory()
	_, err = stateFactory.Create(cfg.StateStore, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Failed to create state store: %v", err)
	}

	fmt.Printf("Rejecting request: %s\n", approvalID)
	fmt.Printf("Reason: %s\n", reason)

	// Create approval record
	record := workflow.ApprovalRecord{
		ApproverID: os.Getenv("USER"),
		Decision:   "rejected",
		Reason:     reason,
		Timestamp:  time.Now().Unix(),
	}

	fmt.Println("Rejection record created:")
	approvalJSON, _ := json.MarshalIndent(record, "  ", "  ")
	fmt.Printf("  %s\n", string(approvalJSON))

	fmt.Println()
	fmt.Println("Note: Full approval tracking and workflow resumption requires approval store implementation.")
}

// printApproveHelp prints help for the approve command
func printApproveHelp() {
	fmt.Println("Usage: shiro approve <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list              List pending approvals")
	fmt.Println("  approve <id>      Approve a pending request")
	fmt.Println("  reject <id>       Reject a pending request")
	fmt.Println("  help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shiro approve list")
	fmt.Println("  shiro approve approve jira-access-123")
	fmt.Println("  shiro approve reject jira-access-123 \"Access not authorized\"")
}
