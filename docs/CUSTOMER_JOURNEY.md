# Customer Journey: Simplified Shiro Experience

## Overview

Shiro provides an intuitive, developer-friendly experience for workflow automation. This guide walks you through the complete customer journey from initial setup to advanced usage.

## Phase 1: Getting Started (5 minutes)

### Step 1: Initialize Your Project

```bash
# Navigate to your GitLab project
cd your-project

# Initialize Shiro
shiro init
```

**What happens:**
- Creates `.shiro/` folder structure
- Sets up example `workflow.json`
- Creates `config.yaml` for AI models
- Initializes module registry
- Updates `.gitignore` to exclude sensitive config

**Result:**
```
your-project/
├── .shiro/
│   ├── workflow.json          # Your workflow definition
│   ├── config.yaml           # AI model configuration
│   ├── modules/
│   │   └── registry.yaml     # Module registry
│   └── workflows/            # Additional workflows
├── .gitignore               # Updated to exclude .shiro/
└── your code...
```

### Step 2: Run Your First Workflow

```bash
shiro run
```

**What happens:**
- Automatically detects `.shiro/workflow.json`
- Loads configuration from `.shiro/config.yaml`
- Executes your workflow
- Shows results in real-time

**Result:**
```
[Shiro] Loaded workflow: example-workflow
[Shiro] Starting workflow: example-workflow
[Shiro] Step step1 completed: true
=== Workflow Results ===
Step: step1
  Success: true
  Output: { "level": "info", "message": "Hello from Shiro!" }
```

## Phase 2: Adding Modules (10 minutes)

### Step 3: Discover and Add Modules

```bash
# Search for available modules
shiro search module jira
shiro search module slack
shiro search module github
```

**What happens:**
- Searches GitHub for modules tagged with `shiro-module`
- Shows module name, stars, and description
- Helps you discover community modules

**Result:**
```
Found module: jira-module
Repository: github.com/rkuthiala/jira-module
Stars: 42
Description: Integrate with Jira for issue tracking
```

### Step 4: Add a Module

```bash
# Add official module (auto-discovers)
shiro add module jira

# OR add custom module from GitHub
shiro add module github.com/your-org/custom-module
```

**What happens:**
- Auto-discovers module from official repository
- Fetches metadata from GitHub
- Adds module to your registry
- Shows configuration instructions

**Result:**
```
Auto-discovering module 'jira' from official repository...
Found module: jira-module
Repository: github.com/rkuthiala/jira-module
Stars: 42
Description: Integrate with Jira for issue tracking
Module 'jira' added successfully!
Source: github.com/rkuthiala/jira-module
To use this module, configure its endpoints in .shiro/modules/registry.yaml
```

### Step 5: Configure Your Module

Edit `.shiro/modules/registry.yaml`:

```yaml
modules:
  jira:
    name: "Jira Integration"
    type: "http"
    endpoints:
      - http://localhost:8080
    config: ".shiro/modules/jira/config.yaml"
    version: "1.0.0"
    description: "Integrate with Jira for issue tracking"
```

### Step 6: Use Module in Workflow

Edit `.shiro/workflow.json`:

```json
{
  "name": "jira-workflow",
  "steps": [
    {
      "id": "create-issue",
      "type": "jira",
      "operation": "create_issue",
      "config": {
        "project": "PROJ",
        "summary": "New issue from workflow"
      }
    }
  ]
}
```

### Step 7: Run Your Enhanced Workflow

```bash
shiro run
```

**What happens:**
- Loads your workflow with Jira integration
- Executes Jira operation
- Shows results

## Phase 3: GitLab CI Integration (15 minutes)

### Step 8: Add Shiro to GitLab CI

Edit `.gitlab-ci.yml`:

```yaml
stages:
  - test
  - deploy
  - notify

# Install Shiro in CI
before_script:
  - wget -O shiro https://github.com/rkuthiala/shiro-automation/releases/latest/download/shiro
  - chmod +x shiro

test-workflow:
  stage: test
  script:
    - ./shiro run
  artifacts:
    when: always
    paths:
      - workflow-results.json

deploy-app:
  stage: deploy
  script:
    - ./shiro run -workflow .shiro/workflows/deploy.json
  only:
    - main

notify-team:
  stage: notify
  script:
    - ./shiro run -workflow .shiro/workflows/notify.json
  when: on_failure
```

### Step 9: Commit and Push

```bash
git add .shiro/
git commit -m "Add Shiro workflow automation"
git push
```

**What happens:**
- `.shiro/` is excluded from git (via .gitignore)
- Only your workflow definitions are tracked
- CI automatically runs your workflows

## Phase 4: Advanced Usage (Optional)

### Multiple Workflows

```bash
# Create additional workflows
cat > .shiro/workflows/deploy.json << 'EOF'
{
  "name": "deploy-workflow",
  "steps": [...]
}
EOF

# Run specific workflow
shiro run -workflow .shiro/workflows/deploy.json
```

### Custom Configuration Location

```bash
# Use custom .shiro directory
shiro run -shiro-dir /path/to/custom/.shiro
```

### Module Management

```bash
# List all modules
shiro list modules

# Get module information
shiro info module jira

# Open module documentation
shiro docs module jira

# Remove a module
shiro remove module jira
```

## Complete Example: E-commerce Deployment

### Scenario: Automated E-commerce Deployment

**Workflow:** `.shiro/workflow.json`
```json
{
  "name": "ecommerce-deploy",
  "steps": [
    {
      "id": "test",
      "type": "git",
      "operation": "diff",
      "config": {
        "branch": "main"
      }
    },
    {
      "id": "build",
      "type": "docker",
      "operation": "build",
      "config": {
        "image": "myapp:latest"
      }
    },
    {
      "id": "deploy",
      "type": "kubernetes",
      "operation": "deploy",
      "config": {
        "namespace": "production"
      }
    },
    {
      "id": "notify",
      "type": "slack",
      "config": {
        "channel": "#deployments",
        "message": "Deployment completed successfully"
      }
    }
  ]
}
```

**GitLab CI:** `.gitlab-ci.yml`
```yaml
deploy-production:
  stage: deploy
  script:
    - ./shiro run
  only:
    - main
  environment:
    name: production
```

## Key Benefits

### 1. Simplicity
- **Before:** `shiro run -workflow examples/simple-test.json -config configs/models.yaml`
- **After:** `shiro run`

### 2. Discovery
- **Before:** Manually searching GitHub for modules
- **After:** `shiro search module jira`

### 3. Installation
- **Before:** `shiro module add -name jira -type http -endpoint http://localhost:8080`
- **After:** `shiro add module jira`

### 4. Organization
- **Before:** Scattered config files across the project
- **After:** Everything in `.shiro/` folder

### 5. Git Integration
- **Before:** Manually managing .gitignore
- **After:** Automatic .gitignore updates

## Common Use Cases

### Use Case 1: Development Team

**Goal:** Automate code review workflow

**Steps:**
1. `shiro init`
2. `shiro add module github`
3. Create workflow for MR review
4. Add to GitLab CI
5. Every MR automatically triggers workflow

### Use Case 2: DevOps Team

**Goal:** Automate deployment pipeline

**Steps:**
1. `shiro init`
2. `shiro add module kubernetes`
3. `shiro add module slack`
4. Create deployment workflow
5. Add to GitLab CI
6. Automated deployments with notifications

### Use Case 3: QA Team

**Goal:** Automate testing workflow

**Steps:**
1. `shiro init`
2. `shiro add module jira`
3. Create testing workflow
4. Add to GitLab CI
5. Automated test execution with issue tracking

## Troubleshooting

### Module Not Found

**Problem:** Workflow references module that isn't installed

**Solution:**
```bash
shiro add module <module-name>
shiro run
```

### Configuration Issues

**Problem:** Workflow fails with config error

**Solution:**
```bash
# Check your config
cat .shiro/config.yaml

# Validate workflow
shiro run -workflow .shiro/workflow.json -config .shiro/config.yaml
```

### GitLab CI Issues

**Problem:** Workflow fails in CI but works locally

**Solution:**
```yaml
# Add debugging to CI
debug-workflow:
  stage: test
  script:
    - ./shiro run -v
  only:
    - debug
```

## Best Practices

1. **Version Control:** Keep workflow definitions in git, exclude sensitive config
2. **Modular Workflows:** Create separate workflows for different stages
3. **Module Reuse:** Use community modules when possible
4. **Testing:** Test workflows locally before adding to CI
5. **Documentation:** Document custom workflows for team members

## Support Resources

- **Documentation:** https://github.com/rkuthiala/shiro-automation/docs
- **Module Marketplace:** Search GitHub for `shiro-module` topic
- **Issues:** https://github.com/rkuthiala/shiro-automation/issues
- **Examples:** Check `examples/` folder for workflow templates

## Summary

The simplified Shiro experience provides:

- **5-minute setup:** `shiro init` and you're ready
- **Intuitive commands:** Natural language like `shiro add module jira`
- **Auto-discovery:** Find and install modules easily
- **GitLab ready:** Works seamlessly with CI/CD
- **Organization:** Everything in `.shiro/` folder
- **No complexity:** Sensible defaults, minimal configuration

From initialization to production deployment, Shiro provides a frictionless experience for workflow automation.
