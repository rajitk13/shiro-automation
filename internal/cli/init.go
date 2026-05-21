package cli

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// InitCommand handles the project initialization command
func InitCommand(args []string) {
	fmt.Println("Initializing Shiro project...")

	// Create .shiro and modules directory structure
	dirs := []string{
		".shiro",
		".shiro/workflows",
		"modules",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		fmt.Printf("Created directory: %s\n", dir)
	}

	// Create example workflow.json
	workflowContent := `{
  "name": "example-workflow",
  "description": "Example workflow for Shiro",
  "steps": [
    {
      "id": "step1",
      "type": "print",
      "config": {
        "level": "info",
        "message": "Hello from Shiro!"
      }
    }
  ]
}`

	if err := os.WriteFile(".shiro/workflow.json", []byte(workflowContent), 0644); err != nil {
		log.Fatalf("Failed to create workflow.json: %v", err)
	}
	fmt.Println("Created file: .shiro/workflow.json")

	// Create example config.yaml
	configContent := `# Shiro configuration
# This file configures AI models and other settings
# Environment variables can be used with {{env.VARIABLE_NAME}} syntax

models:
  # Example AI model configuration with environment variables
  # openai-custom:
  #   type: openai
  #   model: "gpt-4"
  #   base_url: "{{env.OPENAI_BASE_URL}}"
  #   api_key: "{{env.OPENAI_API_KEY}}"
  
  # ollama:
  #   type: ollama
  #   model: "codellama:34b"
  #   base_url: "http://localhost:11434"
  
  # openai:
  #   type: openai
  #   model: "gpt-4"
  #   base_url: "https://api.openai.com/v1"
  #   api_key: "{{env.OPENAI_API_KEY}}"`

	if err := os.WriteFile(".shiro/config.yaml", []byte(configContent), 0644); err != nil {
		log.Fatalf("Failed to create config.yaml: %v", err)
	}
	fmt.Println("Created file: .shiro/config.yaml")

	// Create module registry
	registryContent := `modules:
  # Built-in modules
  slack:
    name: "Slack Notifications"
    type: "builtin"
    description: "Send notifications to Slack channels"
  
  git:
    name: "Git Operations"
    type: "builtin"
    description: "Perform Git operations (diff, clone, etc.)"
  
  print:
    name: "Print Module"
    type: "builtin"
    description: "Print messages to console with different log levels"
  
  ai:
    name: "AI Generation"
    type: "builtin"
    description: "Generate content using AI providers"
  
  # Example HTTP module configuration
  # jira:
  #   name: "Jira Integration"
  #   type: "http"
  #   endpoints:
  #     - http://localhost:8080
  #   config: ".shiro/modules/jira/config.yaml"
  #   version: "1.0.0"
  #   description: "Integrate with Jira for issue tracking"
  #   source: "github.com/your-org/jira-module"
  #   docs: "https://github.com/your-org/jira-module/blob/main/README.md"`

	if err := os.WriteFile(".shiro/modules/registry.yaml", []byte(registryContent), 0644); err != nil {
		log.Fatalf("Failed to create .shiro/modules/registry.yaml: %v", err)
	}

	fmt.Println("Created file: .shiro/modules/registry.yaml")

	// Create .gitignore entry
	gitignorePath := ".gitignore"
	gitignoreContent := `
# Shiro configuration
.shiro/
.shiro/modules/
`

	// Append to .gitignore if it exists, otherwise create it
	if _, err := os.Stat(gitignorePath); err == nil {
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			log.Fatalf("Failed to read .gitignore: %v", err)
		}
		if !strings.Contains(string(content), ".shiro/") {
			f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("Failed to open .gitignore: %v", err)
			}
			defer f.Close()
			if _, err := f.WriteString(gitignoreContent); err != nil {
				log.Fatalf("Failed to write to .gitignore: %v", err)
			}
			fmt.Println("Updated .gitignore to exclude .shiro/")
		}
	} else {
		if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
			log.Fatalf("Failed to create .gitignore: %v", err)
		}
		fmt.Println("Created file: .gitignore")
	}

	fmt.Println()
	fmt.Println("✓ Shiro project initialized successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Customize .shiro/workflow.json with your workflow")
	fmt.Println("  2. Configure AI models in .shiro/config.yaml")
	fmt.Println("  3. Add modules: shiro add module <module-name>")
	fmt.Println("  4. Run your workflow: shiro run")
	fmt.Println()
	fmt.Println("For more information: shiro help")
}
