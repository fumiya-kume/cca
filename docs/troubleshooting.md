# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with ccAgents.

## Quick Diagnostics

### System Status Check

```bash
# Check overall system status
ccagents status --verbose

# Validate configuration
ccagents validate

# Check authentication
ccagents auth status

# Test connectivity
ccagents test --all
```

### Log Analysis

```bash
# View recent logs
ccagents logs --tail 50

# Debug level logs
ccagents logs --level debug

# Follow logs in real-time
ccagents logs --follow

# Search logs for specific errors
ccagents logs --grep "error"
```

## Common Issues

### Authentication Issues

#### GitHub Authentication Fails

**Symptoms:**
- `Error: authentication failed`
- `403 Forbidden` when accessing repositories
- `GitHub API rate limit exceeded`

**Solutions:**

```bash
# Check current authentication status
gh auth status

# Re-authenticate with GitHub
gh auth logout
gh auth login --web

# For organizations, ensure proper scope
gh auth login --web --scopes "repo,read:org"

# Use personal access token
gh auth login --with-token < token.txt

# Verify access to specific repository
gh repo view owner/repo
```

**Advanced Solutions:**

```bash
# Check token permissions
gh api user

# Test repository access
gh api repos/owner/repo

# For enterprise GitHub
gh auth login --hostname github.enterprise.com
```

#### Claude Code Authentication Fails

**Symptoms:**
- `Error: Claude API authentication failed`
- `Invalid API key`
- `Rate limit exceeded`

**Solutions:**

```bash
# Check Claude authentication
claude auth status

# Re-authenticate
claude auth logout
claude auth login

# Verify API key
claude config get api_key

# Test Claude connection
claude models list

# Check usage and limits
claude usage
```

### Configuration Issues

#### Invalid Configuration

**Symptoms:**
- `Error: configuration validation failed`
- `Unknown configuration key`
- `Invalid value for parameter`

**Solutions:**

```bash
# Validate configuration
ccagents validate --config .ccagents.yaml

# Show configuration schema
ccagents config schema

# Reset to default configuration
ccagents init --force

# Check specific configuration value
ccagents config get github.timeout
```

**Common Configuration Fixes:**

```yaml
# Fix: Invalid timeout format
github:
  timeout: "30s"  # Use string with unit, not number

# Fix: Invalid model name
claude:
  model: "claude-3-sonnet-20240229"  # Use exact model name

# Fix: Invalid log level
logging:
  level: "info"  # Use lowercase: debug, info, warn, error

# Fix: Invalid boolean value
workflow:
  enable_auto_merge: false  # Use boolean, not string
```

#### Environment Variable Issues

**Symptoms:**
- Configuration not loading
- Settings being ignored
- Default values used instead of custom ones

**Solutions:**

```bash
# Check environment variables
env | grep CCAGENTS

# Set required environment variables
export CCAGENTS_CONFIG_PATH="$HOME/.ccagents.yaml"
export CCAGENTS_LOG_LEVEL="debug"

# Verify variable loading
ccagents config dump
```

### Workflow Issues

#### Issue Processing Fails

**Symptoms:**
- `Error: failed to parse issue`
- `No requirements found in issue`
- `Unable to determine implementation approach`

**Solutions:**

1. **Improve Issue Description:**

```markdown
# Bad issue example
Title: Fix bug

Description: Something is broken.

# Good issue example
Title: Fix user authentication validation bug

Description: Users can login with invalid email formats.

Steps to Reproduce:
1. Go to login page
2. Enter "invalid-email" (no @ symbol)
3. Click login
4. User is logged in (should be rejected)

Expected: Email validation should reject invalid formats
Actual: Invalid emails are accepted

Acceptance Criteria:
- [ ] Validate email format using regex
- [ ] Show error message for invalid emails
- [ ] Add unit tests for email validation
```

2. **Check Repository Context:**

```bash
# Verify repository access
ccagents context analyze

# Check if codebase is supported
ccagents validate --check-languages

# Ensure proper file structure
ls -la  # Should see source files, not just README
```

#### Code Generation Fails

**Symptoms:**
- `Error: Claude request failed`
- `Generated code doesn't compile`
- `Implementation doesn't match requirements`

**Solutions:**

1. **Check Claude Configuration:**

```yaml
claude:
  model: "claude-3-sonnet-20240229"  # Use most capable model
  max_tokens: 8192                   # Increase for complex tasks
  temperature: 0.1                   # Low temperature for code
```

2. **Improve Context:**

```bash
# Analyze codebase context
ccagents context build --verbose

# Check if relevant files are detected
ccagents context files

# Add context hints in issue
```

#### Tests Fail During Automation

**Symptoms:**
- `Error: test execution failed`
- `Tests pass locally but fail in automation`
- `Missing test dependencies`

**Solutions:**

1. **Check Test Configuration:**

```yaml
workflow:
  test_command: "make test"        # Ensure command is correct
  test_timeout: "10m"             # Increase timeout if needed
  test_environment:               # Set required environment
    NODE_ENV: "test"
    DATABASE_URL: "sqlite://test.db"
```

2. **Debug Test Environment:**

```bash
# Run tests manually with same environment
ccagents test --dry-run

# Check test dependencies
ccagents validate --check-tests

# View test output
ccagents logs --filter test
```

#### Pull Request Creation Fails

**Symptoms:**
- `Error: failed to create pull request`
- `Branch protection rules prevent push`
- `Required status checks missing`

**Solutions:**

1. **Check Branch Protection:**

```bash
# View branch protection rules
gh api repos/owner/repo/branches/main/protection

# Check required status checks
gh api repos/owner/repo/branches/main/protection/required_status_checks
```

2. **Update Workflow Configuration:**

```yaml
workflow:
  target_branch: "main"              # Ensure correct target
  pr_template: ".github/pr_template.md"  # Use custom template
  draft_pr: true                     # Create as draft first
  require_review: true               # Follow protection rules
```

### Performance Issues

#### Slow Processing

**Symptoms:**
- Long response times
- Timeouts during code generation
- High memory usage

**Solutions:**

1. **Optimize Configuration:**

```yaml
claude:
  model: "claude-3-haiku-20240307"  # Use faster model for simple tasks
  max_tokens: 4096                  # Reduce token limit
  timeout: "30s"                    # Set appropriate timeout

workflow:
  parallel_tasks: 2                 # Reduce parallelism
  batch_size: 5                     # Smaller batches
```

2. **System Optimization:**

```bash
# Check system resources
ccagents status --system

# Monitor performance
ccagents metrics --follow

# Clear cache if needed
ccagents cache clear
```

#### Memory Issues

**Symptoms:**
- `Out of memory` errors
- System becomes unresponsive
- Process killed by OS

**Solutions:**

```yaml
# Reduce memory usage
performance:
  max_buffer_size: 1024     # Reduce buffer size
  gc_threshold: 512         # More aggressive GC
  max_concurrent_ops: 2     # Reduce concurrency

logging:
  buffer_size: 100          # Smaller log buffer
  max_entries: 1000         # Limit log entries
```

### Network Issues

#### Connection Timeouts

**Symptoms:**
- `Error: connection timeout`
- `Request timeout`
- Intermittent failures

**Solutions:**

```yaml
# Increase timeouts
github:
  timeout: "60s"
  max_retries: 5
  retry_delay: "2s"

claude:
  timeout: "120s"
  max_retries: 3
```

#### Rate Limiting

**Symptoms:**
- `Rate limit exceeded`
- `Too many requests`
- `API quota exhausted`

**Solutions:**

1. **GitHub Rate Limits:**

```bash
# Check current rate limit
gh api rate_limit

# Use GitHub App instead of personal token
# Configure in GitHub settings

# Implement request throttling
```

2. **Claude Rate Limits:**

```bash
# Check usage
claude usage

# Reduce request frequency
ccagents config set claude.rate_limit 10  # requests per minute
```

## Advanced Troubleshooting

### Debug Mode

Enable debug mode for detailed logging:

```bash
# Run with debug logging
CCAGENTS_LOG_LEVEL=debug ccagents process "#123"

# Enable trace logging for network requests
CCAGENTS_TRACE=true ccagents process "#123"

# Save debug output to file
ccagents process "#123" --debug > debug.log 2>&1
```

### Profiling Performance

```bash
# Profile CPU usage
ccagents profile cpu --duration 30s

# Profile memory usage
ccagents profile memory

# Generate performance report
ccagents metrics export --format json > metrics.json
```

### Network Debugging

```bash
# Test connectivity
ccagents test connectivity

# Check DNS resolution
ccagents test dns

# Test SSL certificates
ccagents test ssl

# Monitor network traffic
ccagents monitor network
```

### Database Issues

If using persistent storage:

```bash
# Check database connection
ccagents db status

# Run database migrations
ccagents db migrate

# Repair database if corrupted
ccagents db repair

# Reset database (WARNING: loses data)
ccagents db reset --confirm
```

## Recovery Procedures

### Recover from Failed Workflow

```bash
# List failed workflows
ccagents workflow list --status failed

# Get workflow details
ccagents workflow show <workflow-id>

# Retry from last checkpoint
ccagents workflow retry <workflow-id>

# Retry specific step
ccagents workflow retry <workflow-id> --step generate-code
```

### Recover from Corrupted State

```bash
# Reset workflow state
ccagents workflow reset <workflow-id>

# Clear temporary files
ccagents clean --temp

# Reset configuration
ccagents init --reset

# Rebuild context
ccagents context rebuild
```

### Emergency Recovery

```bash
# Stop all running workflows
ccagents stop --all

# Clean up temporary branches
ccagents clean --branches

# Reset to safe state
ccagents reset --safe

# Restore from backup
ccagents restore --from backup.tar.gz
```

## Error Reference

### Common Error Codes

| Code | Message | Solution |
|------|---------|----------|
| E001 | Authentication failed | Re-authenticate with `gh auth login` |
| E002 | Invalid configuration | Run `ccagents validate` |
| E003 | Network timeout | Check internet connection, increase timeouts |
| E004 | Rate limit exceeded | Wait or upgrade API limits |
| E005 | Repository not found | Check repository URL and permissions |
| E006 | Branch protection violation | Update workflow configuration |
| E007 | Test execution failed | Check test command and environment |
| E008 | Code generation failed | Improve issue description, check Claude config |
| E009 | Merge conflict | Resolve conflicts manually |
| E010 | Insufficient permissions | Check GitHub token permissions |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Authentication error |
| 4 | Network error |
| 5 | Permission error |
| 6 | Timeout error |
| 130 | Interrupted by user |

## Reporting Issues

If you can't resolve an issue:

### Gather Information

```bash
# Collect system information
ccagents diagnose > diagnose.txt

# Export configuration (sanitized)
ccagents config export --safe > config.yaml

# Export recent logs
ccagents logs --since 1h > logs.txt
```

### Create Bug Report

Include this information:

1. **System Information:**
   - OS and version
   - ccAgents version
   - Go version
   - GitHub CLI version
   - Claude CLI version

2. **Configuration:**
   - Sanitized configuration file
   - Environment variables (without secrets)

3. **Error Details:**
   - Complete error message
   - Steps to reproduce
   - Expected vs actual behavior

4. **Logs:**
   - Relevant log entries
   - Debug output if available

### Submit Report

- **GitHub Issues**: [Create Issue](https://github.com/fumiya-kume/cca/issues/new)
- **Security Issues**: [Security Policy](https://github.com/fumiya-kume/cca/security/policy)
- **Feature Requests**: [Discussions](https://github.com/fumiya-kume/cca/discussions)

## Getting Help

### Documentation
- [Configuration Guide](configuration.md)
- [API Reference](api/)
- [Examples](examples/)

### Community
- [GitHub Discussions](https://github.com/fumiya-kume/cca/discussions)
- [Discord Server](https://discord.gg/ccagents)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/ccagents)

### Professional Support
- [Enterprise Support](https://ccagents.dev/support)
- [Consulting Services](https://ccagents.dev/consulting)
- [Training Programs](https://ccagents.dev/training)