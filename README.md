# Shiro Automation - AI-Native CI Workflow Runtime

A portable workflow orchestration runtime optimized for CI/CD environments with AI-native capabilities.

## Overview

Shiro is a production-ready Go-based workflow runtime that:
- Runs inside existing CI runners (GitLab/Jenkins/GitHub Actions/K8s Jobs)
- Executes ephemeral workflows with DAG-based scheduling
- Supports reusable modules and integrations
- Enables AI-assisted workflows with multiple provider support (Ollama, OpenAI, custom)
- Minimizes always-on infrastructure

## Features

- **Simplified CLI**: Intuitive commands with auto-detection (`shiro init`, `shiro run`, `shiro add module`)
- **Portable Runtime**: Single binary that runs in any CI environment
- **DAG Execution**: Topological sorting with dependency management
- **Module System**: Pluggable architecture with GitHub marketplace integration
- **Auto-Discovery**: Search and install modules from official GitHub repository
- **AI Providers**: Support for Ollama, OpenAI, and custom endpoints
- **State Storage**: Modular backends (GitLab artifacts, filesystem, memory)
- **GitLab Integration**: Both CI job and webhook-triggered modes
- **GitHub Integration**: GitHub Actions workflows and webhook support
- **Variable Resolution**: Template-based parameterization (inputs, env vars, step outputs)
- **Retry Logic**: Configurable retry with exponential backoff
- **.shiro Folder**: Organized project structure with auto-detection

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
shiro search module slack

# List installed modules
shiro list modules

# Remove a module
shiro remove module jira

# Get module information
shiro info module jira

# Open module documentation
shiro docs module jira
```

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
curl -sSL https://gitlab.com/rajitk13/shiro-automation/-/raw/main/scripts/install-auto.sh | bash
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

### Webhook Mode

1. Build the webhook server:
```bash
go build -o webhook-server ./cmd/webhook-server
```

2. Run the server:
```bash
export CONFIG_FILE=configs/models.yaml
export WORKFLOW_DIR=./workflows
./webhook-server
```

3. Configure GitLab or GitHub webhooks to point to your server

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

## Available Modules

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

### `git.diff`
Performs git operations.

**Config:**
- `operation` (required): `diff` or `get_changes`
- `base`: Base branch/commit
- `target`: Target branch/commit (default: HEAD)

### `ai.generate`
Generates content using AI models.

**Config:**
- `provider` (required): Provider name from config
- `model` (required): Model name
- `prompt` (required): AI prompt
- `system`: System prompt
- `temperature`: Generation temperature
- `max_tokens`: Maximum tokens

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

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - run: go build -o shiro ./cmd/runtime

      - name: Run AI Review
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: ./shiro -workflow examples/github-mr-review.json -config configs/models.yaml
```

### Webhook Mode

Configure GitHub webhook to point to your server at `/webhook/github`. The server will:
- Parse GitHub webhook events
- Match events to workflows
- Execute workflows asynchronously
- Persist state to filesystem

## GitLab CI Integration

### CI Job Mode

```yaml
stages:
  - review

ai-review:
  stage: review
  image: golang:1.21
  before_script:
    - apt-get update && apt-get install -y git
    - go build -o shiro ./cmd/runtime
  script:
    - ./shiro -workflow examples/mr-review.json -config configs/models.yaml -state-store gitlab
  only:
    - merge_requests
```

### Webhook Mode

Configure GitLab webhook to point to your server at `/webhook/gitlab`. The server will:
- Parse GitLab webhook events
- Match events to workflows
- Execute workflows asynchronously
- Persist state to filesystem

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
  webhook-server/   # Webhook receiver
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
rm -f shiro webhook-server
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

This installs:
- pre-commit (for pre-commit hooks)
- golangci-lint (for Go linting)

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

MIT

## Contributing

Contributions welcome! Please read our contributing guidelines before submitting PRs.
