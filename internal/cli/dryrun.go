package cli

import (
	"log"
	"os"

	"github.com/rkuthiala/shiro-automation/internal/config"
	"github.com/rkuthiala/shiro-automation/internal/runtime"
	"github.com/rkuthiala/shiro-automation/internal/workflow"
)

// PrintDryRunPlan prints the execution plan and relevant context for a workflow
func PrintDryRunPlan(wf *workflow.Workflow, cfg *config.Config, env map[string]string, logger *log.Logger) {
	// Get execution order
	execOrder, err := runtime.GetExecutionOrder(wf)
	if err != nil {
		log.Fatalf("Failed to determine execution order: %v", err)
	}

	// Print execution plan
	for i, stepID := range execOrder {
		step := wf.GetStepByID(stepID)
		if step == nil {
			continue
		}
		logger.Printf("\n%d. Step: %s", i+1, step.ID)
		logger.Printf("   Type: %s", step.Type)
		if len(step.DependsOn) > 0 {
			logger.Printf("   Depends On: %v", step.DependsOn)
		}
		if len(step.Config) > 0 {
			logger.Printf("   Config: %v keys", len(step.Config))
		}
		if step.Quiet {
			logger.Printf("   Quiet: true")
		}
	}

	// Detect relevant modules
	hasGitHub := false
	hasGitLab := false
	for _, step := range wf.Steps {
		if step.Type == "github" {
			hasGitHub = true
		}
		if step.Type == "gitlab" {
			hasGitLab = true
		}
	}

	// Show only relevant CI variables
	if hasGitHub || hasGitLab {
		logger.Println("\n--- Environment Variables ---")
		if hasGitHub {
			ghVars := []string{"GITHUB_TOKEN", "GITHUB_REPOSITORY", "GITHUB_PR_NUMBER", "GITHUB_SHA"}
			for _, varName := range ghVars {
				if val, ok := env[varName]; ok {
					logger.Printf("%s: %s", varName, val)
				} else {
					logger.Printf("%s: (not set)", varName)
				}
			}
		}
		if hasGitLab {
			glVars := []string{"CI_PROJECT_ID", "CI_MERGE_REQUEST_IID", "CI_COMMIT_SHA", "CI_COMMIT_REF_NAME", "CI_SERVER_URL"}
			for _, varName := range glVars {
				if val, ok := env[varName]; ok {
					logger.Printf("%s: %s", varName, val)
				} else {
					logger.Printf("%s: (not set)", varName)
				}
			}
		}
	}

	// Show state store only if it's non-default (not gitlab)
	if cfg.StateStore != "" && cfg.StateStore != "gitlab" {
		logger.Println("\n--- State Store ---")
		logger.Printf("Type: %s", cfg.StateStore)
	}

	logger.Println("\n=== Dry Run Complete ===")
	logger.Println("Workflow is valid and ready to execute")
	os.Exit(0)
}
