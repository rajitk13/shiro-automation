# Module Development Kit

This template helps you create custom modules for the Shiro workflow automation system with support for operation-based routing and load balancing.

## Module Structure

```
your-module/
├── main.go           # Module implementation
├── config.yaml       # Module configuration
├── go.mod            # Go module dependencies
└── README.md         # Module documentation
```

## Getting Started

1. **Copy the template** to your module directory
2. **Customize main.go** with your module logic
3. **Update config.yaml** with your module configuration
4. **Build and run** your module

## Module API Contract

Your module must implement three HTTP endpoints:

### POST /execute
Executes the module with workflow step data.

**Request:**
```json
{
  "step_id": "step1",
  "step_type": "your.module",
  "operation": "create",  // Optional: specific operation to execute
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

### GET /metadata
Returns module metadata.

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

### GET /health
Health check endpoint.

**Response:**
```json
{
  "healthy": true,
  "message": "Module is healthy"
}
```

## Operation-Based Routing

Your module can support multiple operations by checking the `operation` field in the request:

```go
func (s *ModuleServer) executeModule(ctx context.Context, req modules.ExecuteRequest) (modules.ExecuteResponse, error) {
    operation := req.Operation
    if operation == "" {
        operation = "execute" // Default operation
    }

    switch operation {
    case "create":
        return s.executeCreate(req)
    case "update":
        return s.executeUpdate(req)
    case "delete":
        return s.executeDelete(req)
    default:
        return s.executeDefault(req)
    }
}
```

**Usage in workflow:**
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

## Load Balancing

For high-availability modules, you can run multiple instances and configure load balancing in the module registry:

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

The system will automatically:
- Use round-robin load balancing across endpoints
- Remove unhealthy endpoints from rotation
- Implement circuit breaker pattern for failed endpoints
- Retry failed requests with automatic failover

## Configuration

Update `config.yaml` with your module settings:

```yaml
name: "Your Module Name"
version: "1.0.0"
description: "Description of what your module does"

api:
  port: 8080
  timeout: 30s

settings:
  # Module-specific configuration
  timeout: 30
  retries: 3

auth:
  type: "none"  # Options: none, api_key, oauth, basic
```

## Building Your Module

```bash
# Initialize Go module
go mod init your-module

# Add dependencies
go get github.com/rkuthiala/shiro-automation/internal/modules

# Build
go build -o your-module main.go

# Run
./your-module
```

## Registering Your Module

Once your module is running, add it to the Shiro module registry:

```bash
shiro module add \
  -name your-module \
  -type http \
  -endpoint http://localhost:8080 \
  -config your-module/config.yaml \
  -description "Your module description"
```

For load-balanced modules:
```bash
shiro module add \
  -name your-module \
  -type http \
  -endpoints http://localhost:8080,http://localhost:8081,http://localhost:8082 \
  -config your-module/config.yaml \
  -description "Your module description"
```

## GitHub Marketplace Integration

To publish your module to the GitHub marketplace:

1. **Tag your module repository** with `shiro-module` topic
2. **Add a README.md** with module documentation
3. **Register with shiro:**
```bash
shiro module install github.com/your-username/your-module
```

Users can then discover and install your module:
```bash
shiro module search your-module
shiro module install github.com/your-username/your-module
```

## Example Modules

See the existing modules for reference:
- `pkg/slack/` - Slack notification module
- `pkg/git/` - Git operations module
- `pkg/print/` - Console output module

## Best Practices

1. **Error Handling**: Always return proper error messages
2. **Timeouts**: Implement proper timeout handling
3. **Logging**: Add meaningful logging for debugging
4. **Configuration**: Use environment variables for sensitive data
5. **Testing**: Test your module independently before integration
6. **Documentation**: Document your module's inputs and outputs
7. **Operations**: Use operation-based routing for complex modules
8. **Health Checks**: Implement robust health check endpoints

## Environment Variables

Use environment variables for sensitive data:

```go
apiKey := os.Getenv("YOUR_API_KEY")
```

Configure in your module's config.yaml:
```yaml
auth:
  type: "api_key"
  env_var: "YOUR_API_KEY"
```

## Testing Your Module

Test your module independently using curl:

```bash
# Health check
curl http://localhost:8080/health

# Get metadata
curl http://localhost:8080/metadata

# Execute module
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "step_id": "test",
    "step_type": "your.module",
    "config": {"param": "value"},
    "input": {},
    "context": {}
  }'

# Execute module with specific operation
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "step_id": "test",
    "step_type": "your.module",
    "operation": "create",
    "config": {"param": "value"},
    "input": {},
    "context": {}
  }'
```

## Troubleshooting

- **Module not found**: Ensure your module is registered in `modules/registry.yaml`
- **Connection refused**: Check that your module is running on the correct port
- **Timeout errors**: Increase the timeout in your module configuration
- **Authentication errors**: Verify your API keys and environment variables
- **Load balancing issues**: Ensure all endpoints are healthy and accessible

## Support

For issues or questions:
- Check existing module implementations
- Review the API contract documentation
- Test your module independently before integration
- Search GitHub for similar modules: `shiro module search <query>`
