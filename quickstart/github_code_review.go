package quickstart

import (
	"fmt"
	"os"
)

// GitHubCodeReviewTemplate implements the Template interface for GitHub code review
type GitHubCodeReviewTemplate struct{}

// NewGitHubCodeReviewTemplate creates a new GitHub code review template
func NewGitHubCodeReviewTemplate() Template {
	return &GitHubCodeReviewTemplate{}
}

// Name returns the template name
func (t *GitHubCodeReviewTemplate) Name() string {
	return "github-code-review"
}

// Description returns the template description
func (t *GitHubCodeReviewTemplate) Description() string {
	return "AI-powered GitHub PR code review workflow"
}

// Initialize sets up the GitHub code review template
func (t *GitHubCodeReviewTemplate) Initialize(interactive, directConfig bool, configArgs []string) error {
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

	// Create github-code-review workflow
	workflowContent := `{
  "name": "github-code-review",
  "description": "AI-powered GitHub PR code review",
  "steps": [
    {
      "id": "get-diff",
      "type": "git.diff",
      "config": {
        "operation": "diff",
        "base": "{{env.GITHUB_BASE_REF}}",
        "target": "{{env.GITHUB_SHA}}"
      }
    },
    {
      "id": "ai-review",
      "type": "ai.generate",
      "depends_on": ["get-diff"],
      "config": {
        "prompt": "Review this code diff:\n\n{{steps.get-diff.diff}}\n\nProvide free text comments with file path and line number for each issue found in format: 'path/to/file.go:42 - issue description'"
      }
    },
    {
      "id": "post-inline-comments",
      "type": "github",
      "depends_on": ["ai-review"],
      "config": {
        "operation": "post_inline_comments",
        "body": "{{steps.ai-review.content}}",
        "output_format": "text",
        "dedup": true
      }
    }
  ]
}`

	if err := WriteFile(".shiro/workflows/github-code-review.json", workflowContent); err != nil {
		return err
	}
	fmt.Println("Created file: .shiro/workflows/github-code-review.json")

	// Handle config setup
	if directConfig {
		config, err := ParseDirectConfig(configArgs)
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
		return writeGitHubConfig(config)
	}

	if interactive {
		config, err := InteractiveConfig()
		if err != nil {
			return fmt.Errorf("failed to get interactive config: %w", err)
		}
		return writeGitHubConfig(config)
	}

	// Create default config template
	configContent := `# AI model configuration for code review
# Run 'shiro init -template github-code-review -i' to configure interactively

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

	// Create GitHub Actions integration
	gaContent := `name: Code Review

on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions:
  contents: read
  pull-requests: write

jobs:
  code-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - name: Run Shiro Code Review
        uses: rajitk13/shiro-automation@latest
        with:
          workflow: .shiro/workflows/github-code-review.json
          config: .shiro/config.yaml
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}`

	if _, err := os.Stat(".github/workflows"); err == nil {
		fmt.Println("Note: .github/workflows directory already exists. Add the code-review workflow manually.")
	} else {
		if err := os.MkdirAll(".github/workflows", 0755); err != nil {
			return fmt.Errorf("failed to create .github/workflows directory: %w", err)
		}
		if err := WriteFile(".github/workflows/code-review.yml", gaContent); err != nil {
			return err
		}
		fmt.Println("Created file: .github/workflows/code-review.yml")
	}

	fmt.Println()
	fmt.Println("✓ GitHub code review workflow initialized successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Configure AI model in .shiro/config.yaml")
	fmt.Println("  2. Add OPENAI_API_KEY to GitHub repository secrets")
	fmt.Println("  3. Commit and push to trigger on next PR")
	fmt.Println("  4. Review will run automatically on pull requests")

	return nil
}

func writeGitHubConfig(config map[string]interface{}) error {
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
