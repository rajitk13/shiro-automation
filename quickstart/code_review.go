package quickstart

import (
	"fmt"
	"os"
)

// CodeReviewTemplate implements the Template interface for GitLab code review
type CodeReviewTemplate struct{}

// NewCodeReviewTemplate creates a new code review template
func NewCodeReviewTemplate() Template {
	return &CodeReviewTemplate{}
}

// Name returns the template name
func (t *CodeReviewTemplate) Name() string {
	return "code-review"
}

// Description returns the template description
func (t *CodeReviewTemplate) Description() string {
	return "AI-powered GitLab MR code review workflow"
}

// Initialize sets up the code review template
func (t *CodeReviewTemplate) Initialize(interactive, directConfig bool, configArgs []string) error {
	// Create .shiro directory structure
	dirs := []string{
		".shiro",
		".shiro/workflows",
	}

	for _, dir := range dirs {
		if err := EnsureDir(dir); err != nil {
			return err
		}
		fmt.Printf("Created directory: %s\n", dir)
	}

	// Create code-review workflow
	workflowContent := `{
  "name": "gitlab-code-review",
  "description": "AI-powered GitLab MR code review",
  "steps": [
    {
      "id": "get-diff",
      "type": "git",
      "config": {
        "operation": "diff",
        "base": "{{env.CI_MERGE_REQUEST_DIFF_BASE_SHA}}",
        "target": "{{env.CI_MERGE_REQUEST_SOURCE_SHA}}"
      }
    },
    {
      "id": "ai-review",
      "type": "ai.generate",
      "depends_on": ["get-diff"],
      "config": {
        "prompt": "Review this code diff. Provide free text comments with file path and line number for each issue found.",
        "input": "{{steps.get-diff.diff}}"
      }
    },
    {
      "id": "post-comment",
      "type": "http.request",
      "depends_on": ["ai-review"],
      "config": {
        "method": "POST",
        "url": "{{env.CI_API_V4_URL}}/projects/{{env.CI_PROJECT_ID}}/merge_requests/{{env.CI_MERGE_REQUEST_IID}}/notes",
        "headers": {
          "JOB-TOKEN": "{{env.CI_JOB_TOKEN}}"
        },
        "body": "{{steps.ai-review.content}}"
      }
    }
  ]
}`

	if err := WriteFile(".shiro/workflows/code-review.json", workflowContent); err != nil {
		return err
	}
	fmt.Println("Created file: .shiro/workflows/code-review.json")

	// Handle config setup
	if directConfig {
		config, err := ParseDirectConfig(configArgs)
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
		return writeConfig(config)
	}

	if interactive {
		config, err := InteractiveConfig()
		if err != nil {
			return fmt.Errorf("failed to get interactive config: %w", err)
		}
		return writeConfig(config)
	}

	// Create default config template
	configContent := `# AI model configuration for code review
# Run 'shiro init -template code-review -i' to configure interactively

models:
  openai:
    type: openai
    model: gpt-4
    base_url: https://api.openai.com/v1
    api_key: "{{env.OPENAI_API_KEY}}"`
	if err := WriteFile(".shiro/config.yaml", configContent); err != nil {
		return err
	}
	fmt.Println("Created file: .shiro/config.yaml")

	// Create GitLab CI integration
	ciContent := `code-review:
  stage: review
  image: ghcr.io/rajitk13/shiro-automation:latest
  script:
    - shiro run -workflow .shiro/workflows/code-review.json -config .shiro/config.yaml
  only:
    - merge_requests`

	if _, err := os.Stat(".gitlab-ci.yml"); err == nil {
		fmt.Println("Note: .gitlab-ci.yml already exists. Add the code-review job manually.")
	} else {
		if err := WriteFile(".gitlab-ci.yml", ciContent); err != nil {
			return err
		}
		fmt.Println("Created file: .gitlab-ci.yml")
	}

	fmt.Println()
	fmt.Println("✓ GitLab code review workflow initialized successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Configure AI model in .shiro/config.yaml")
	fmt.Println("  2. Commit and push to trigger on next MR")
	fmt.Println("  3. Review will run automatically on merge requests")

	return nil
}

func writeConfig(config map[string]interface{}) error {
	configContent := `# AI model configuration
models:`
	for provider, cfg := range config {
		configContent += fmt.Sprintf("\n  %s:", provider)
		cfgMap, ok := cfg.(map[string]interface{})
		if !ok {
			continue
		}
		for k, v := range cfgMap {
			configContent += fmt.Sprintf("\n    %s: %v", k, v)
		}
	}
	if err := WriteFile(".shiro/config.yaml", configContent); err != nil {
		return err
	}
	fmt.Println("Created file: .shiro/config.yaml")
	return nil
}
