# API Documentation

This directory contains comprehensive API documentation for ccAgents, including both internal APIs and external integration interfaces.

## Overview

ccAgents provides several types of APIs:

1. **CLI Commands** - Command-line interface for user interactions
2. **Configuration API** - YAML-based configuration system
3. **Plugin API** - Extension points for custom functionality
4. **REST API** - HTTP endpoints for external integrations
5. **Webhook API** - GitHub webhook handlers
6. **SDK API** - Go package interfaces for library usage

## Documentation Structure

### Core APIs

- [CLI Commands](cli.md) - Complete command-line reference
- [Configuration](configuration.md) - Configuration schema and options
- [REST API](rest.md) - HTTP endpoints and responses
- [Webhooks](webhooks.md) - GitHub webhook integration

### Integration APIs

- [Plugin System](plugins.md) - Create custom plugins and extensions
- [SDK Reference](sdk.md) - Go package documentation
- [GitHub Integration](github.md) - GitHub API usage and patterns
- [Claude Integration](claude.md) - Claude Code API integration

### Developer Guides

- [API Authentication](authentication.md) - API authentication methods
- [Error Handling](errors.md) - Error codes and handling patterns
- [Rate Limiting](rate-limiting.md) - API rate limits and best practices
- [Monitoring](monitoring.md) - API monitoring and observability

## Quick Start

### Using the CLI API

```bash
# Basic workflow
ccagents process "https://github.com/owner/repo/issues/123"

# Check status
ccagents status --verbose

# Get help
ccagents help <topic>
```

### Using the REST API

```bash
# Start API server
ccagents server --port 8080

# Process issue via API
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{"issue_url": "https://github.com/owner/repo/issues/123"}'
```

### Using the Go SDK

```go
package main

import (
    "context"
    "github.com/fumiya-kume/cca/pkg/workflow"
    "github.com/fumiya-kume/cca/pkg/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }
    
    engine := workflow.NewEngine(cfg)
    ctx := context.Background()
    
    result, err := engine.ProcessIssue(ctx, "https://github.com/owner/repo/issues/123")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Workflow ID: %s\n", result.WorkflowID)
}
```

## API Versioning

ccAgents follows semantic versioning for its APIs:

- **Major Version Changes** - Breaking changes to existing APIs
- **Minor Version Changes** - New features, backward compatible
- **Patch Version Changes** - Bug fixes, backward compatible

Current API versions:
- CLI API: `v1.0`
- REST API: `v1`
- SDK API: `v1.0.0`
- Configuration API: `v1.0`

## Common Patterns

### Authentication

Most APIs require authentication with GitHub and Claude:

```bash
# Set up authentication
export CCAGENTS_GITHUB_TOKEN="your_token"
export CCAGENTS_CLAUDE_API_KEY="your_key"
```

### Error Handling

All APIs return structured errors:

```json
{
  "error": {
    "code": "E001",
    "message": "Authentication failed",
    "details": "GitHub token is invalid or expired"
  }
}
```

### Async Operations

Long-running operations return workflow IDs:

```json
{
  "workflow_id": "wf_123456789",
  "status": "running",
  "created_at": "2024-01-01T00:00:00Z"
}
```

## Best Practices

### Rate Limiting

- Respect GitHub and Claude API rate limits
- Implement exponential backoff for retries
- Use caching when appropriate

### Security

- Never log or expose API keys
- Use HTTPS for all external communications
- Validate all input parameters

### Monitoring

- Monitor API response times and error rates
- Set up alerts for critical failures
- Log structured data for debugging

## Examples

See the [examples](../examples/) directory for complete configuration examples and usage patterns.

## Support

- [GitHub Issues](https://github.com/fumiya-kume/cca/issues) - Bug reports and feature requests
- [Discussions](https://github.com/fumiya-kume/cca/discussions) - Community support
- [Security](https://github.com/fumiya-kume/cca/security/policy) - Security issues

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines on contributing to the API documentation.