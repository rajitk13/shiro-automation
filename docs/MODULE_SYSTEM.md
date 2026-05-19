# Decentralized Module System

The Shiro workflow automation system now supports a decentralized module architecture that enables external contributors to create custom modules for SaaS applications (Asana, Jira, Confluence, etc.) without modifying the core codebase.

## Overview

The module system supports two types of modules:

- **Built-in modules**: Compiled into the main binary (Slack, Git, Print, AI)
- **HTTP modules**: Separate binaries communicating via HTTP API

## Advanced Features

### Load Balancing
HTTP modules can run multiple instances with automatic load balancing:
- Round-robin load balancing across endpoints
- Health check-based endpoint removal
- Circuit breaker pattern for failed endpoints
- Retry logic with automatic failover

```yaml
jira:
  name: "Jira Integration"
  type: "http"
  endpoints:
    - http://localhost:8080
    - http://localhost:8081
    - http://localhost:8082
  config: "modules/jira/config.yaml"
```

### Operation-Based Routing
Modules can support multiple operations through a single endpoint:
- Operation parameter in execute requests
- Complex modules with multiple capabilities
- Clean workflow configuration

```json
{
  "type": "jira",
  "operation": "create_issue",
  "config": {
    "project": "PROJ",
    "summary": "New issue"
  }
}
```

### GitHub Marketplace
Discover and install modules from GitHub:
- Search modules by topic (`shiro-module`)
- Install modules from GitHub repositories
- Automatic metadata extraction from README
- Star-based discovery ranking

```bash
shiro module search jira
shiro module install github.com/user/jira-module
```

## Architecture

### Module Registry

The `modules/registry.yaml` file lists all available modules and their configuration:

```yaml
modules:
  slack:
    name: "Slack Notifications"
    type: "builtin"
    description: "Send notifications to Slack channels"
    version: "1.0.0"
  
  jira:
    name: "Jira Integration"
    type: "http"
    endpoint: "http://localhost:8082"
    config: "modules/jira/config.yaml"
    version: "1.0.0"
    description: "Integrate with Jira for issue tracking"
```

### Module Discovery

The system auto-discovers modules from the `modules/` directory and validates HTTP modules via health checks.

### HTTP API Contract

HTTP modules must implement three endpoints:

#### POST /execute
Executes the module with workflow step data.

**Request:**
```json
{
  "step_id": "step1",
  "step_type": "your.module",
  "config": {
    "param1": "value1"
  },
  "input": {
    "data": "input data"
  },
  "context": {
    "workflow_id": "workflow-123"
  }
}
```

**Response:**
```json
{
  "success": true,
  "output": {
    "result": "operation result"
  },
  "error": ""
}
```

#### GET /metadata
Returns module metadata including input/output schemas.

**Response:**
```json
{
  "name": "Your Module Name",
  "description": "Description of what your module does",
  "version": "1.0.0",
  "input_schema": {
    "param1": {
      "type": "string",
      "description": "Parameter description",
      "required": true
    }
  },
  "output_schema": {
    "result": {
      "type": "string",
      "description": "Result description",
      "required": true
    }
  }
}
```

#### GET /health
Health check endpoint.

**Response:**
```json
{
  "healthy": true,
  "message": "Module is healthy"
}
```

## Module Management

### List Modules

```bash
shiro module list
```

### Add HTTP Module

```bash
shiro module add \
  -name jira \
  -type http \
  -endpoint http://localhost:8082 \
  -config modules/jira/config.yaml \
  -description "Jira Integration"
```

### Remove Module

```bash
shiro module remove jira
```

## Creating Custom Modules

### Quick Start

1. Copy the module template:
```bash
cp -r examples/module-template your-module
cd your-module
```

2. Customize `main.go` with your module logic
3. Update `config.yaml` with your module configuration
4. Build and run your module:
```bash
go mod init your-module
go get github.com/rkuthiala/shiro-automation/internal/modules
go build -o your-module main.go
./your-module
```

5. Register your module:
```bash
shiro module add \
  -name your-module \
  -type http \
  -endpoint http://localhost:8080 \
  -config your-module/config.yaml \
  -description "Your module description"
```

### Module Template

See `examples/module-template/` for a complete module template including:
- `main.go` - Module implementation
- `config.yaml` - Module configuration
- `README.md` - Module documentation

### Module Configuration

Each module can have its own configuration file:

```yaml
name: "Your Module Name"
version: "1.0.0"
description: "Description of what your module does"

api:
  port: 8080
  timeout: 30s

settings:
  timeout: 30
  retries: 3

auth:
  type: "api_key"
  env_var: "YOUR_API_KEY"
```

## Using Modules in Workflows

### Built-in Modules

```json
{
  "steps": [
    {
      "id": "notify",
      "type": "slack.notify",
      "config": {
        "webhook_url": "{{env.SLACK_WEBHOOK_URL}}",
        "channel": "#alerts",
        "message": "Workflow completed"
      }
    }
  ]
}
```

### HTTP Modules

```json
{
  "steps": [
    {
      "id": "jira_issue",
      "type": "jira",
      "config": {
        "project": "PROJ",
        "summary": "New issue from workflow",
        "description": "Created automatically"
      }
    }
  ]
}
```

## Module Distribution

### Community Registry

Modules can be distributed via a community registry (similar to npm for packages).

### Git-based Installation

Install modules from Git repositories:
```bash
shiro module install github.com/user/jira-module
```

### Manual Installation

Copy module files to your `modules/` directory and register them manually.

## Best Practices

1. **Error Handling**: Always return proper error messages
2. **Timeouts**: Implement proper timeout handling
3. **Logging**: Add meaningful logging for debugging
4. **Configuration**: Use environment variables for sensitive data
5. **Testing**: Test modules independently before integration
6. **Documentation**: Document module inputs and outputs

## Security

- Use environment variables for API keys and secrets
- Implement proper authentication in your modules
- Validate all input parameters
- Use HTTPS for module communication in production

## Examples

### Slack Module (Built-in)

```json
{
  "type": "slack.notify",
  "config": {
    "webhook_url": "{{env.SLACK_WEBHOOK_URL}}",
    "channel": "#alerts",
    "message": "Alert: Build failed"
  }
}
```

### Custom Jira Module (HTTP)

```json
{
  "type": "jira",
  "config": {
    "project": "PROJ",
    "issue_type": "Bug",
    "summary": "Build failure detected",
    "description": "Workflow execution failed"
  }
}
```

## Troubleshooting

### Module Not Found

Ensure your module is registered in `modules/registry.yaml`:
```bash
shiro module list
```

### Connection Refused

Check that your HTTP module is running:
```bash
curl http://localhost:8080/health
```

### Timeout Errors

Increase timeout in module configuration:
```yaml
api:
  timeout: 60s
```

### Authentication Errors

Verify environment variables are set:
```bash
echo $YOUR_API_KEY
```

## Module Development Kit

For detailed module development instructions, see `examples/module-template/README.md`.

## Migration Guide

### From Built-in to HTTP Module

1. Extract module logic from main binary
2. Implement HTTP API endpoints
3. Create module configuration
4. Register as HTTP module
5. Test independently

### Backward Compatibility

Built-in modules continue to work without changes. The system supports both module types simultaneously.

## Support

For issues or questions:
- Check existing module implementations in `pkg/` directory
- Review the module template in `examples/module-template/`
- Test modules independently before integration
- Use `shiro module help` for CLI assistance
