# Shiro Automation - AI-Native CI Workflow Runtime

A portable workflow orchestration runtime optimized for CI/CD environments with AI-native capabilities.

## Overview

Shiro is a production-ready Go-based workflow runtime that:
- Runs inside existing CI runners (GitLab/Jenkins/GitHub Actions/K8s Jobs)
- Executes ephemeral workflows with DAG-based scheduling
- Supports reusable modules and integrations
- Enables AI-assisted workflows with multiple provider support (Ollama, OpenAI, Gemini, custom)
- Minimizes always-on infrastructure

## Features

- **Simplified CLI**: Intuitive commands with auto-detection (`shiro init`, `shiro run`, `shiro add module`)
- **Portable Runtime**: Single binary that runs in any CI environment
- **DAG Execution**: Topological sorting with dependency management
- **Module System**: Pluggable architecture with GitHub marketplace integration
- **Auto-Discovery**: Search and install modules from official GitHub repository
- **AI Providers**: Support for Ollama, OpenAI, Gemini, and custom endpoints
- **State Storage**: Modular backends (GitLab artifacts, filesystem, memory)
- **GitLab Integration**: GitLab CI workflow support
- **GitHub Integration**: GitHub Actions workflows support
- **Variable Resolution**: Template-based parameterization (inputs, env vars, step outputs)
- **Retry Logic**: Configurable retry with exponential backoff
- **Human-in-Loop Approvals**: GitLab-native manual approval workflows with Slack review notifications
- **.shiro Folder**: Organized project structure with auto-detection

## Human-in-Loop Approvals

Shiro supports human-in-loop approval workflows without external callback infrastructure. A workflow can send a Slack review notification, pause, and resume only when a user manually plays the GitLab resume job.

### Approval Configuration

Use `slack.notify` with `pause: true` to send a GitLab review link and stop execution after the notification step:

```json
{
  "id": "request_approval",
  "type": "slack.notify",
  "pause": true,
  "config": {
    "webhook_url": "{{env.SLACK_WEBHOOK_URL}}",
    "channel": "#deployments",
    "message": "Review deployment to production. Click the GitLab button and play the manual resume job to approve.",
    "gitlab_pipeline_url": "{{env.CI_SERVER_URL}}/{{env.CI_PROJECT_ID}}/-/pipelines/{{env.CI_PIPELINE_ID}}",
    "button_text": "Review in GitLab"
  }
}
```

### Configuration Options

- `webhook_url`: Slack webhook URL (required)
- `channel`: Slack channel to send to (optional)
- `message`: Approval message (required)
- `gitlab_pipeline_url`: GitLab pipeline URL for the review button (optional)
- `button_text`: Review button text (optional, default: `Review in GitLab`)
- `pause`: Set to `true` on the workflow step to stop after sending the notification

### State Storage

Workflow state can be stored in multiple backends:

```bash
shiro run -workflow .shiro/workflow.json -state-store memory
shiro run -workflow .shiro/workflow.json -state-store filesystem
shiro run -workflow .shiro/workflow.json -state-store gitlab
```

### Example Workflow

See `examples/approval-workflow.json` for a complete example with approval step.

### Approval Flow

1. Workflow reaches approval step
2. Sends Slack message with a GitLab pipeline review button
3. Saves workflow state and pauses
4. User opens GitLab and manually plays the resume job to approve
5. Resume job loads state, skips completed steps, and continues
6. If the resume job is not played, the workflow remains stopped

## Quick Start

### Initialize Your Project

```bash
# Initialize Shiro in your project (creates .shiro folder structure)
shiro init
```

This creates:
```
.shiro/
├── workflow.json          # Your workflow definition
├── config.yaml           # AI model configuration
├── modules/
│   └── registry.yaml     # Module registry
└── workflows/            # Additional workflows
```

### Run Workflows

```bash
# Run workflow (auto-detects .shiro/workflow.json)
shiro run

# Run specific workflow
shiro run -workflow examples/simple-test.json

# With custom config
shiro run -config configs/models.yaml

# With custom .shiro directory
shiro run -shiro-dir /path/to/.shiro
```

### Module Management

```bash
# Add a module (auto-discovers from GitHub)
shiro add module jira

# Add module from GitHub URL
shiro add module github.com/user/custom-module

# Search for modules
shiro search module jira

# List available modules
shiro list modules

# Remove a module
shiro remove module jira

# Install module from GitHub
shiro install module github.com/user/custom-module

# Display module information
shiro info module jira

# Open module documentation
shiro docs module jira
```

### Quickstart Templates

Shiro provides quickstart templates for common use cases. These templates scaffold complete workflows with pre-configured settings.

```bash
# List available templates
shiro init -help

# Initialize with a specific template
shiro init -template code-review
```

#### GitLab Code Review Template

The `code-review` template sets up an AI-powered GitLab MR code review workflow:

```bash
# Initialize with default config template
shiro init -template code-review

# Initialize with interactive config setup
shiro init -template code-review -i

# Initialize with direct config values
shiro init -template code-review -d provider=openai -d api_key=sk-... -d model=gpt-4
```

This creates:
- `.shiro/workflows/code-review.json` — AI-powered review workflow
- `.shiro/config.yaml` — AI model configuration
- `.gitlab-ci.yml` — GitLab CI integration (if not exists)

The workflow:
1. Gets git diff from the merge request
2. Sends diff to AI provider for code review
3. Posts AI-generated comments to the MR using GitLab CI job token

**Supported providers:** OpenAI, Ollama, Gemini, Custom (configured interactively with `-i`)

**Interactive mode (-i):** Prompts for AI provider selection (OpenAI/Ollama/Gemini/Custom) and configuration

**Direct mode (-d):** Pass config values as flags: `-d provider=openai -d api_key=sk-... -d model=gpt-4`

### CLI Mode

After installation, run workflows from anywhere:

```bash
# Quick test (hello world)
shiro hello_world

# Simple test (no LLM required)
shiro run examples/simple-test.json

# Simple print example
shiro run examples/print-example.json

# AI PR review (requires LLM configuration)
shiro run examples/mr-review.json

# With custom config
shiro run examples/mr-review.json -config configs/models.yaml

# With filesystem state store
shiro run examples/github-mr-review.json -state-store filesystem

# Shorthand (run is default)
shiro examples/simple-test.json
```

### Validation

```bash
# Validate workflow JSON only
shiro validate -workflow .shiro/workflow.json

# Validate workflow + cross-check against CI configuration
shiro validate -workflow .shiro/workflow.json -ci .gitlab-ci.yml

# Validate with GitHub Actions workflow
shiro validate -workflow .shiro/workflow.json -ci .github/workflows/deploy.yml
```

The `--ci` flag cross-checks your workflow against the CI pipeline configuration to catch common misconfigurations:

**GitLab CI checks:**
- Pause steps require a `when: manual` resume job with `needs:` dependency
- Jobs using `-state-store gitlab` must expose `.shiro/` as an artifact
- Initial jobs should use `-fresh` flag, resume jobs should not

**GitHub Actions checks:**
- Pause steps should use environment protection rules (no native manual gate)
- `-state-store gitlab` is GitLab-specific — use filesystem with artifacts instead

### View Help

```bash
shiro help
```

### Download Pre-built Binaries

Pre-built binaries are available for multiple platforms. Download the latest version from GitHub releases:

**Linux:**
- Linux (AMD64): [shiro-linux-amd64](https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-linux-amd64)
- Linux (ARM64): [shiro-linux-arm64](https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-linux-arm64)

**macOS:**
- macOS (Intel): [shiro-darwin-amd64](https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-darwin-amd64)
- macOS (Apple Silicon): [shiro-darwin-arm64](https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-darwin-arm64)

**Windows:**
- Windows (AMD64): [shiro-windows-amd64.exe](https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-windows-amd64.exe)

Download the binary for your platform, make it executable (Linux/macOS), and run:

```bash
# Linux/macOS
curl -LO https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-<platform>
chmod +x shiro-<platform>
./shiro-<platform> help

# Windows
curl -LO https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-windows-amd64.exe
shiro-windows-amd64.exe help
```

### Installation

**Option 1: Auto-detect and install (Recommended)**
```bash
curl -sSL https://raw.githubusercontent.com/rajitk13/shiro-automation/master/scripts/install-auto.sh | bash
```

This script automatically detects your platform and installs the correct binary.

**Option 2: Download and install to PATH**
```bash
# Download for your platform (example for Linux AMD64)
curl -LO https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-linux-amd64

# Make executable
chmod +x shiro-linux-amd64

# Move to PATH
sudo mv shiro-linux-amd64 /usr/local/bin/shiro
```

**Option 3: Build from source**
```bash
go build -o shiro ./cmd/runtime
# Then use ./shiro or add to your PATH
```

**Option 4: Use the install script**
```bash
make install
# or
./scripts/install.sh
```

This installs `shiro` to `/usr/local/bin`, allowing you to run it from anywhere.

### Docker Image

For CI/CD environments, use the pre-built Docker image to avoid downloading the binary in each job:

```bash
# Pull the image
docker pull ghcr.io/rajitk13/shiro-automation:latest

# Run shiro
docker run --rm ghcr.io/rajitk13/shiro-automation:latest shiro help
```

**GitLab CI usage:**

```yaml
test-jira:
  stage: test
  image: ghcr.io/rajitk13/shiro-automation:latest
  script:
    - shiro run
```

#### Image variants

Two image variants are published:

- **`slim` (default)** — `latest`, `vX.Y.Z`. Alpine base with the `shiro` binary, `git`, and `curl`. Use this for built-in modules and for subprocess modules installed as pre-built binaries. Small image.
- **`toolchain`** — `latest-go`, `vX.Y.Z-go`. Based on `golang:1.23-alpine` and includes the full Go toolchain. Use this **only** if you run go-run subprocess modules (`shiro add module github.com/...` that execute via `go run`).

```bash
# Common case (small image)
docker pull ghcr.io/rajitk13/shiro-automation:latest

# go-run subprocess modules
docker pull ghcr.io/rajitk13/shiro-automation:latest-go
```

Both variants include the `shiro` binary, `git`, and `curl`, support `linux/amd64` and `linux/arm64`, and embed the release version (`shiro --version`).

**Building locally:**

```bash
# Default slim image
docker build -t shiro .

# Toolchain image (go-run support)
docker build --target toolchain -t shiro:go .
```

## Simplified User Experience

Shiro provides a streamlined developer experience with sensible defaults:

### Before vs After

**Before:**
```bash
shiro run -workflow examples/simple-test.json -config configs/models.yaml
shiro module add -name jira -type http -endpoint http://localhost:8080
```

**After:**
```bash
shiro init
shiro run
shiro add module jira
```

### Key Improvements

- **Auto-detection**: `shiro run` automatically finds `.shiro/workflow.json` and `.shiro/config.yaml`
- **Intuitive Commands**: Natural language like `shiro add module jira` instead of complex flags
- **GitHub Integration**: Auto-discover modules from official repository
- **Organized Structure**: Everything in `.shiro/` folder with clear separation
- **Quick Setup**: `shiro init` creates complete project structure in seconds

For a complete customer journey guide, see [docs/CUSTOMER_JOURNEY.md](docs/CUSTOMER_JOURNEY.md).

## Workflow Definition

Workflows are defined in JSON:

```json
{
  "name": "my-workflow",
  "inputs": {
    "param1": "value1"
  },
  "steps": [
    {
      "id": "step1",
      "type": "module.type",
      "config": {
        "option": "value"
      },
      "depends_on": []
    }
  ]
}
```

### Dry Run Mode

Validate and preview workflow execution without actually running it:

```bash
shiro run -dry-run
```

**What dry-run does:**
- Validates workflow structure and configuration
- Shows execution order (DAG)
- Displays module types and configurations
- Lists available environment variables
- Shows state store configuration
- Exits without executing any modules

**Use cases:**
- Validate workflows before deployment
- Debug workflow configuration issues
- Preview execution plan
- CI/CD pipeline testing without side effects

**Example output:**
```
=== Dry Run Mode ===
Workflow will be validated but not executed
Workflow: my-workflow
Total Steps: 3

--- Execution Plan (DAG Order) ---

1. Step: step1
   Type: git.diff
   Config: 2 keys

2. Step: step2
   Type: ai.generate
   Depends On: [step1]
   Config: 3 keys

3. Step: step3
   Type: print
   Depends On: [step2]

--- Environment Variables ---
CI_PROJECT_ID: 12345
CI_MERGE_REQUEST_IID: 678
CI_COMMIT_SHA: abc123
CI_COMMIT_REF_NAME: main
CI_SERVER_URL: https://gitlab.com

--- State Store ---
Type: gitlab

=== Dry Run Complete ===
Workflow is valid and ready to execute
```

## Available Modules

### Subprocess Modules

Subprocess modules are external programs that communicate with Shiro via JSON over stdin/stdout. They can be:
- **Binary mode**: Pre-compiled binary downloaded from GitHub releases
- **Go-run mode**: Executed via `go run` directly from a GitHub repo (fallback if no binary available)

Install any subprocess module:
```bash
shiro add module github.com/your-org/your-module
```

Shiro auto-detects the best execution mode based on the module's `module.yaml`.

#### Official Subprocess Modules

| Module | Repo | Operations |
|--------|------|------------|
| `jira` | [shiro-automation-jira-datacenter](https://github.com/rajitk13/shiro-automation-jira-datacenter) | `create_issue`, `get_issue`, `update_issue`, `add_comment`, `transition_issue`, `search_issues` |

---

### `print`
Prints output to console with optional log levels and colors.

**Config:**
- `message` (required): Message to print
- `level`: Log level (info, debug, error, warning), default: info
- `format`: Output format (text, json), default: text

**Log Level Colors:**
- info: green
- debug: gray
- error: red
- warning: yellow

**Example:**
```json
{
  "id": "log_output",
  "type": "print",
  "config": {
    "message": "Step output: {{steps.review.content}}",
    "level": "info",
    "format": "text"
  }
}
```

### `slack.notify`
Sends notifications to Slack via webhooks.

**Config:**
- `webhook_url` (required): Slack webhook URL
- `channel`: Target channel
- `message` (required): Message content
- `username`: Bot username
- `icon_emoji`: Bot icon
- `gitlab_pipeline_url`: GitLab pipeline URL for a review button
- `button_text`: Review button label

### `git.diff`
Performs git operations.

**Config:**
- `operation` (required): `diff` or `get_changes`
- `base`: Base branch/commit
- `target`: Target branch/commit (default: HEAD)

### `gitlab`
GitLab operations for merge requests, commits, and users.

**Config:**
- `operation` (required): Operation to perform: `post_comment`, `post_inline_comments`, `get_commit_info`, `get_user_info`, `get_mr_participants`, `get_files_changed`
- `body`: Comment body to post (for `post_comment` and `post_inline_comments` with text format)
- `comments`: Array of comment objects for `post_inline_comments` with JSON format
- `output_format`: Output format for `post_inline_comments`: `json` or `text` (default: `text`)
- `api_type`: API type for `post_inline_comments`: `notes` (general MR comments) or `discussions` (inline line comments, default: `discussions`)
- `dedup`: Enable deduplication for `post_inline_comments` (default: `true`)
- `commit_sha`: Commit SHA to get info for (for `get_commit_info` operation, defaults to `CI_COMMIT_SHA`)
- `user_id`: User ID to get info for (for `get_user_info` operation)

**Environment Variables:**
- `CI_PROJECT_ID`: GitLab project ID (required for most operations)
- `CI_MERGE_REQUEST_IID`: Merge request IID (required for MR operations)
- `CI_JOB_TOKEN`: GitLab CI job token (default authentication)
- `GITLAB_TOKEN`: Personal access token (alternative to job token, required for MR comments with `api` scope)

**Example - Simple Comment:**
```json
{
  "id": "post-comment",
  "type": "gitlab",
  "depends_on": ["ai-review"],
  "config": {
    "operation": "post_comment",
    "body": "{{steps.ai-review.content}}"
  }
}
```

**Example - Inline Comments (Text Format):**
```json
{
  "id": "post-inline-comments",
  "type": "gitlab",
  "depends_on": ["ai-review"],
  "config": {
    "operation": "post_inline_comments",
    "body": "{{steps.ai-review.content}}",
    "output_format": "text",
    "api_type": "discussions",
    "dedup": true
  }
}
```

**Example - Inline Comments (JSON Format):**
```json
{
  "id": "post-inline-comments",
  "type": "gitlab",
  "depends_on": ["ai-review"],
  "config": {
    "operation": "post_inline_comments",
    "comments": "{{steps.ai-review.comments}}",
    "output_format": "json",
    "api_type": "discussions",
    "dedup": true
  }
}
```

**AI Prompt for Text Format:**
```
Review this code diff and provide comments in format: 'path/to/file.go:42 - issue description'
```

**AI Prompt for JSON Format:**
```
Review this code diff and return JSON array:
[
  {"file": "path/to/file.go", "line": 42, "comment": "issue description"}
]
```

### `ai.generate`
Generates content using AI models.

**Config:**
- `provider` (required): Provider name from config
- `model` (required): Model name
- `prompt` (required): AI prompt
- `system`: System prompt
- `temperature`: Generation temperature
- `max_tokens`: Maximum tokens

### `jira` (subprocess module)
Jira Data Center integration.

**Install:**
```bash
shiro add module github.com/rajitk13/shiro-automation-jira-datacenter
```

**Required env vars:** `JIRA_BASE_URL`, `JIRA_API_TOKEN`

**Config:**
- `operation` (required): `create_issue`, `get_issue`, `update_issue`, `add_comment`, `transition_issue`, `search_issues`
- `project`: Jira project key (required for `create_issue`)
- `summary`: Issue summary (required for `create_issue`)
- `issue_key`: Existing issue key (required for get/update/comment/transition)
- `description`, `issue_type`, `priority`, `labels`, `assignee`: Optional fields
- `jql`: JQL query (required for `search_issues`)
- `transition_id`: Transition ID (required for `transition_issue`)
- `comment`: Comment body (required for `add_comment`)

## Variable Resolution

Templates support:
- `{{inputs.variable}}`: Workflow inputs
- `{{env.VARIABLE}}`: Environment variables
- `{{steps.step_id.output}}`: Step outputs
- `{{memory.key}}`: Shared memory

Example:
```json
{
  "message": "Review for {{inputs.repository}}: {{steps.review.content}}"
}
```

## AI Provider Configuration

Configure AI providers in YAML:

```yaml
models:
  # Ollama local models
  codellama:
    type: ollama
    model: codellama:34b
    base_url: http://localhost:11434

  # OpenAI-compatible providers
  gpt-4:
    type: openai
    model: gpt-4
    base_url: https://api.openai.com/v1
    api_key: "{{env.OPENAI_API_KEY}}"

  # Custom OpenAI-compatible endpoint (e.g., local LLM server)
  custom-llm:
    type: openai
    model: custom-model
    base_url: http://localhost:8000/v1
    api_key: "sk-custom-key"

  # OpenAI Custom with environment variables
  openai-custom:
    type: openai
    model: "{{env.OPENAI_CUSTOM_MODEL}}"
    base_url: "{{env.OPENAI_CUSTOM_BASE_URL}}"
    api_key: "{{env.OPENAI_CUSTOM_API_KEY}}"

  # Gemini (Google AI Studio)
  gemini:
    type: gemini
    model: gemini-1.5-pro
    api_key: "{{env.GEMINI_API_KEY}}"
    api_type: "google-ai-studio"

  # Gemini (Vertex AI)
  gemini-vertex:
    type: gemini
    model: gemini-1.5-pro
    api_key: "{{env.GOOGLE_ACCESS_TOKEN}}"
    api_type: "vertex-ai"
    project_id: "{{env.GOOGLE_PROJECT_ID}}"
    location: "us-central1"
```

### Environment Variable Resolution

Shiro supports environment variable resolution in configuration files using the `{{env.VARIABLE_NAME}}` syntax. This is particularly useful for sensitive data like API keys and base URLs that shouldn't be hardcoded in your repository.

**Example:**
```yaml
models:
  openai-custom:
    type: openai
    model: "{{env.OPENAI_CUSTOM_MODEL}}"
    base_url: "{{env.OPENAI_CUSTOM_BASE_URL}}"
    api_key: "{{env.OPENAI_CUSTOM_API_KEY}}"
```

**Usage in CI/CD:**
- Set environment variables in your CI/CD pipeline (GitLab CI/CD variables, GitHub Secrets, etc.)
- Shiro will automatically resolve them at runtime
- If an environment variable is not set, the literal string will be used

## GitHub Actions Integration

### CI Job Mode

```yaml
name: AI PR Review

on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  ai-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install Shiro
        run: |
          curl -LOk https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-linux-amd64
          chmod +x shiro-linux-amd64
          sudo mv shiro-linux-amd64 /usr/local/bin/shiro

      - name: Run AI Review
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: shiro run
```

## GitLab CI Integration

### CI Job Mode

```yaml
stages:
  - review

ai-review:
  stage: review
  image: golang:1.23
  before_script:
    - curl -LOk https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-linux-arm64
    - chmod +x shiro-linux-arm64
    - mv shiro-linux-arm64 /usr/local/bin/shiro
  script:
    - shiro run
  rules:
    - if: $CI_PIPELINE_SOURCE == 'merge_request_event'
```

### Using Subprocess Modules (e.g. Jira)

Subprocess modules run as external processes. They can be installed from GitHub repos and executed via pre-built binary or `go run` mode (no binary required).

```yaml
stages:
  - test

test-jira:
  stage: test
  image: golang:1.23
  before_script:
    - curl -LOk https://github.com/rajitk13/shiro-automation/releases/latest/download/shiro-linux-arm64
    - chmod +x shiro-linux-arm64
    - mv shiro-linux-arm64 /usr/local/bin/shiro
  script:
    - SHIRO_INSECURE_TLS=1 shiro add module github.com/your-org/your-module
    - export GOSUMDB=off
    - export GOPROXY=direct
    - export GIT_SSL_NO_VERIFY=1
    - shiro run
  variables:
    MY_SERVICE_URL: "https://your-service.example.com"
```

Workflow JSON for subprocess module:

```json
{
  "name": "test-jira",
  "steps": [
    {
      "id": "create-jira-issue",
      "type": "jira",
      "config": {
        "operation": "create_issue",
        "project": "DEV",
        "summary": "Automated issue from CI",
        "description": "Created by Shiro workflow",
        "issue_type": "Task"
      }
    }
  ]
}
```

## State Storage

Configure state storage backend:

```bash
# GitLab artifacts (default in CI)
./shiro -state-store gitlab ...

# Filesystem
./shiro -state-store filesystem ...

# Memory (ephemeral)
./shiro -state-store memory ...
```

## Architecture

```
Trigger Adapters (GitLab/Jenkins/GitHub)
              ↓
      Workflow Runtime (Go)
              ↓
         DAG Executor
              ↓
      Module/Plugin System
              ↓
    AI / Integrations / Compute
```

## Development

### Project Structure

```
cmd/
  runtime/          # CLI binary
internal/
  runtime/          # Core execution engine
  workflow/         # Workflow definitions
  modules/          # Module system
  ai/               # AI provider abstraction
  state/            # State storage
  gitlab/           # GitLab integration
  github/           # GitHub integration
pkg/
  slack/            # Slack module
  git/              # Git operations module
  ai/               # AI generation module
  print/            # Console print module
examples/           # Example workflows
configs/            # Configuration templates
scripts/           # Development scripts
```

### Building

```bash
# Build runtime
shiro build

# Run tests
shiro test

# Clean build artifacts
rm -f shiro
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

### Linting

The project uses golangci-lint for Go code linting and pre-commit hooks for automated checks.

#### Setup Development Environment

```bash
# Run the setup script (recommended)
./setup.sh

# Or manually configure Git hooks
git config core.hooksPath .githooks
chmod +x .githooks/pre-commit
```

The setup script configures:
- Git hooks path to use `.githooks/` directory
- Pre-commit hook for automatic formatting, linting, and testing checks

#### Manual Setup

```bash
# Install pre-commit
pip install pre-commit

# Install pre-commit hooks
pre-commit install

# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
```

#### Run Linters

```bash
# Run pre-commit on all files
pre-commit run --all-files

# Run golangci-lint
golangci-lint run
```

### Conventional Commits

This project follows [Conventional Commits](https://www.conventionalcommits.org/) specification.

#### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

#### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Test additions or changes
- `chore`: Build process or auxiliary tool changes
- `ci`: CI/CD changes
- `build`: Build system changes
- `revert`: Revert a previous commit

#### Examples

```bash
feat(ai): add OpenAI provider support
fix(runtime): resolve variable resolution bug
docs(readme): update installation instructions
```

## License

Apache License 2.0

## Contributing

Contributions welcome! Please read our contributing guidelines before submitting PRs.
