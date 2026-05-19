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

- **Portable Runtime**: Single binary that runs in any CI environment
- **DAG Execution**: Topological sorting with dependency management
- **Module System**: Pluggable architecture for extensibility
- **AI Providers**: Support for Ollama, OpenAI, and custom endpoints
- **State Storage**: Modular backends (GitLab artifacts, filesystem, memory)
- **GitLab Integration**: Both CI job and webhook-triggered modes
- **GitHub Integration**: GitHub Actions workflows and webhook support
- **Variable Resolution**: Template-based parameterization (inputs, env vars, step outputs)
- **Retry Logic**: Configurable retry with exponential backoff

## Quick Start

### CLI Mode (GitHub Actions)

1. Build the runtime:
```bash
go build -o shiro ./cmd/runtime
```

2. Create a workflow JSON file (see `examples/` directory)

3. Run the workflow:
```bash
./shiro -workflow examples/github-mr-review.json -config configs/models.yaml
```

### CLI Mode (GitLab CI Job)

1. Build the runtime:
```bash
go build -o shiro ./cmd/runtime
```

2. Create a workflow JSON file (see `examples/` directory)

3. Run the workflow:
```bash
./shiro -workflow examples/mr-review.json -config configs/models.yaml
```

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
  codellama:
    provider: ollama
    model: codellama:34b
    base_url: http://localhost:11434
  
  gpt-4:
    provider: openai
    model: gpt-4
    base_url: https://api.openai.com/v1
    api_key: "{{env.OPENAI_API_KEY}}"
```

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
examples/           # Example workflows
configs/            # Configuration templates
```

### Building

```bash
# Build runtime
go build -o shiro ./cmd/runtime

# Build webhook server
go build -o webhook-server ./cmd/webhook-server

# Build both
go build ./...
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## License

MIT

## Contributing

Contributions welcome! Please read our contributing guidelines before submitting PRs.
