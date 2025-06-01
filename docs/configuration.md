# Configuration Guide

ccAgents uses YAML configuration files to customize behavior. This guide covers all configuration options and examples.

## Configuration Files

### Hierarchy

ccAgents loads configuration from multiple sources in this order (later sources override earlier ones):

1. **System defaults** (built-in)
2. **Global config** (`~/.ccagents.yaml`)
3. **Project config** (`.ccagents.yaml` in project root)
4. **Environment variables** (`CCAGENTS_*`)
5. **Command-line flags** (`--config`, `--option`)

### File Locations

```bash
# Global configuration
~/.ccagents.yaml

# Project-specific configuration
.ccagents.yaml
.ccagents/config.yaml

# Custom location
export CCAGENTS_CONFIG_PATH="/path/to/config.yaml"
```

## Basic Configuration

### Minimal Configuration

```yaml
version: "1.0"

github:
  owner: "your-username"
  repo: "your-repo"

claude:
  model: "claude-3-sonnet-20240229"
```

### Standard Configuration

```yaml
version: "1.0"

# GitHub configuration
github:
  owner: "your-username"
  repo: "your-repo"
  default_branch: "main"
  api_version: "2022-11-28"
  timeout: 30s

# Claude Code configuration
claude:
  model: "claude-3-sonnet-20240229"
  max_tokens: 4096
  temperature: 0.1
  timeout: 60s

# Workflow configuration
workflow:
  enable_auto_merge: false
  require_review: true
  run_tests: true
  security_scan: true

# UI configuration
ui:
  theme: "auto"
  show_progress: true
  enable_notifications: true

# Logging configuration
logging:
  level: "info"
  format: "pretty"
  output: "stdout"
```

## Detailed Configuration Options

### GitHub Configuration

```yaml
github:
  # Repository information
  owner: "organization-name"        # GitHub username or organization
  repo: "repository-name"           # Repository name
  default_branch: "main"            # Default branch (main, master, develop)
  
  # API configuration
  api_version: "2022-11-28"         # GitHub API version
  api_url: "https://api.github.com" # GitHub API URL (for enterprise)
  timeout: 30s                      # Request timeout
  max_retries: 3                    # Maximum retry attempts
  retry_delay: 2s                   # Delay between retries
  
  # Authentication
  token: ""                         # GitHub token (use env var instead)
  app_id: 0                         # GitHub App ID (for app authentication)
  private_key_path: ""              # Path to GitHub App private key
  
  # Rate limiting
  rate_limit:
    requests_per_hour: 5000         # Rate limit (0 = no limit)
    burst_size: 100                 # Burst size for rate limiting
  
  # Pull request settings
  pr:
    template: ".github/pr_template.md"  # PR template file
    auto_delete_branch: true            # Delete branch after merge
    draft: false                        # Create as draft PR
    labels: ["automation", "ccagents"]  # Default labels
    assignees: ["maintainer"]           # Default assignees
    reviewers: ["team-lead"]            # Default reviewers
    team_reviewers: ["core-team"]       # Default team reviewers
  
  # Branch protection
  protection:
    required_status_checks: []      # Required CI checks
    enforce_admins: false           # Enforce for admins
    dismiss_stale_reviews: true     # Dismiss stale reviews
    require_code_owner_reviews: true # Require code owner review
```

### Claude Configuration

```yaml
claude:
  # Model configuration
  model: "claude-3-sonnet-20240229"  # Claude model to use
  max_tokens: 4096                   # Maximum tokens per request
  temperature: 0.1                   # Creativity (0.0-1.0, lower = more focused)
  timeout: 60s                       # Request timeout
  
  # API configuration
  api_key: ""                        # Claude API key (use env var instead)
  api_url: "https://api.anthropic.com" # API endpoint
  max_retries: 3                     # Maximum retry attempts
  retry_delay: 2s                    # Delay between retries
  
  # Rate limiting
  rate_limit:
    requests_per_minute: 60          # Rate limit
    tokens_per_minute: 240000        # Token rate limit
  
  # Prompts configuration
  prompts:
    system_prompt: |                 # Custom system prompt
      You are a senior software engineer creating production-ready code.
      Follow these principles:
      - Write clean, maintainable code
      - Include comprehensive error handling
      - Add detailed comments for complex logic
      - Follow the project's coding standards
      - Include unit tests for all functions
    
    code_style: |                    # Code style instructions
      Follow these coding standards:
      - Use descriptive variable names
      - Keep functions small and focused
      - Handle all error cases
      - Add JSDoc/GoDoc comments
      - Follow existing patterns in the codebase
  
  # Context configuration
  context:
    max_files: 20                    # Maximum files to include in context
    max_file_size: 10240             # Maximum file size in bytes
    include_tests: true              # Include test files in context
    include_docs: true               # Include documentation files
```

### Workflow Configuration

```yaml
workflow:
  # Automation settings
  enable_auto_merge: false           # Auto-merge when all checks pass
  require_review: true               # Require human review
  review_timeout: 24h                # Timeout for review
  
  # Quality assurance
  run_tests: true                    # Run tests before creating PR
  test_command: "make test"          # Command to run tests
  test_timeout: 10m                  # Test execution timeout
  test_environment:                  # Environment variables for tests
    NODE_ENV: "test"
    DATABASE_URL: "sqlite://test.db"
  
  # Security scanning
  security_scan: true                # Enable security scanning
  security_tools:                    # Security tools to run
    - "semgrep"
    - "gosec"
    - "npm-audit"
  security_timeout: 5m               # Security scan timeout
  
  # Linting and formatting
  run_linter: true                   # Run linter
  lint_command: "make lint"          # Linting command
  run_formatter: true                # Run code formatter
  format_command: "make format"      # Formatting command
  
  # Custom hooks
  pre_hooks:                         # Commands to run before processing
    - "make deps"
    - "make generate"
  post_hooks:                        # Commands to run after processing
    - "make docs"
    - "make changelog"
  
  # Branch management
  branch_prefix: "ccagents/"         # Prefix for created branches
  branch_cleanup: true               # Clean up merged branches
  target_branch: "main"              # Target branch for PRs
  
  # Parallel processing
  parallel_tasks: 4                  # Number of parallel tasks
  batch_size: 10                     # Batch size for operations
  
  # Retry configuration
  max_retries: 3                     # Maximum retry attempts
  retry_delay: 30s                   # Delay between retries
  retry_backoff: "exponential"       # Backoff strategy (linear, exponential)
```

### UI Configuration

```yaml
ui:
  # Theme settings
  theme: "auto"                      # Theme (auto, light, dark)
  color_scheme: "default"            # Color scheme
  
  # Display options
  show_progress: true                # Show progress indicators
  show_spinner: true                 # Show loading spinners
  enable_notifications: true         # Enable desktop notifications
  
  # Layout
  layout: "compact"                  # Layout style (compact, spacious)
  max_width: 120                     # Maximum terminal width
  
  # Interactive features
  enable_mouse: true                 # Enable mouse support
  enable_shortcuts: true             # Enable keyboard shortcuts
  
  # Output formatting
  timestamp_format: "15:04:05"       # Timestamp format
  date_format: "2006-01-02"          # Date format
  duration_format: "human"           # Duration format (human, precise)
```

### Logging Configuration

```yaml
logging:
  # Log level
  level: "info"                      # Log level (debug, info, warn, error)
  
  # Output format
  format: "pretty"                   # Format (pretty, json, logfmt)
  
  # Output destination
  output: "stdout"                   # Output (stdout, stderr, file)
  file_path: "./logs/ccagents.log"   # Log file path (when output=file)
  
  # File rotation
  max_size: 100                      # Max file size in MB
  max_age: 30                        # Max age in days
  max_backups: 5                     # Max backup files
  compress: true                     # Compress old log files
  
  # Filtering
  filters:                           # Log filters
    - "github.api"                   # Filter GitHub API logs
    - "claude.requests"              # Filter Claude request logs
  
  # Sampling
  sampling:
    enabled: false                   # Enable log sampling
    rate: 0.1                        # Sampling rate (0.0-1.0)
```

### Performance Configuration

```yaml
performance:
  # Memory management
  max_memory: "1GB"                  # Maximum memory usage
  gc_percent: 100                    # Garbage collection target
  
  # Caching
  cache:
    enabled: true                    # Enable caching
    size: 100                        # Cache size (MB)
    ttl: 1h                          # Cache TTL
    cleanup_interval: 10m            # Cache cleanup interval
  
  # Connection pooling
  connections:
    max_idle: 10                     # Maximum idle connections
    max_open: 100                    # Maximum open connections
    max_lifetime: 1h                 # Connection lifetime
  
  # Resource limits
  limits:
    max_goroutines: 1000             # Maximum goroutines
    max_files: 1000                  # Maximum open files
    max_cpu_percent: 80              # Maximum CPU usage
```

### Observability Configuration

```yaml
observability:
  # Metrics
  metrics:
    enabled: true                    # Enable metrics collection
    port: 9090                       # Metrics server port
    path: "/metrics"                 # Metrics endpoint path
  
  # Tracing
  tracing:
    enabled: false                   # Enable distributed tracing
    endpoint: "http://jaeger:14268"  # Tracing endpoint
    service_name: "ccagents"         # Service name for tracing
  
  # Health checks
  health:
    enabled: true                    # Enable health checks
    port: 8080                       # Health check port
    path: "/health"                  # Health check endpoint
    
  # Profiling
  profiling:
    enabled: false                   # Enable profiling
    port: 6060                       # Profiling port
```

## Environment Variables

### Setting Environment Variables

```bash
# Configuration file path
export CCAGENTS_CONFIG_PATH="$HOME/.ccagents.yaml"

# Logging configuration
export CCAGENTS_LOG_LEVEL="debug"
export CCAGENTS_LOG_FORMAT="json"

# GitHub configuration
export CCAGENTS_GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
export CCAGENTS_GITHUB_OWNER="my-org"
export CCAGENTS_GITHUB_REPO="my-repo"

# Claude configuration
export CCAGENTS_CLAUDE_API_KEY="sk-xxxxxxxxxxxx"
export CCAGENTS_CLAUDE_MODEL="claude-3-sonnet-20240229"

# Workflow configuration
export CCAGENTS_WORKFLOW_AUTO_MERGE="false"
export CCAGENTS_WORKFLOW_REQUIRE_REVIEW="true"
```

### Variable Naming Convention

Environment variables follow this pattern:
```
CCAGENTS_<SECTION>_<KEY>=<VALUE>
```

Examples:
- `CCAGENTS_GITHUB_TOKEN` → `github.token`
- `CCAGENTS_CLAUDE_MODEL` → `claude.model`
- `CCAGENTS_WORKFLOW_RUN_TESTS` → `workflow.run_tests`

## Configuration Examples

### Development Environment

```yaml
version: "1.0"

github:
  owner: "mycompany"
  repo: "myproject"
  default_branch: "develop"

claude:
  model: "claude-3-haiku-20240307"  # Faster model for development
  temperature: 0.2                  # Slightly more creative

workflow:
  enable_auto_merge: false          # Manual review required
  require_review: true
  run_tests: true
  test_command: "npm test -- --coverage"
  security_scan: true

ui:
  theme: "dark"
  show_progress: true

logging:
  level: "debug"
  format: "pretty"
```

### Production Environment

```yaml
version: "1.0"

github:
  owner: "mycompany"
  repo: "myproject"
  default_branch: "main"
  api_version: "2022-11-28"
  timeout: 60s
  max_retries: 5

claude:
  model: "claude-3-sonnet-20240229"  # Most capable model
  max_tokens: 8192
  temperature: 0.05                  # Very conservative
  timeout: 120s

workflow:
  enable_auto_merge: false          # Always require human review
  require_review: true
  review_timeout: 48h               # Longer review window
  run_tests: true
  test_command: "make test-all"
  test_timeout: 30m
  security_scan: true
  security_tools: ["semgrep", "gosec", "snyk"]
  
  pre_hooks:
    - "make lint"
    - "make security-check"
  post_hooks:
    - "make docs"
    - "make changelog"

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "/var/log/ccagents/ccagents.log"
  max_size: 100
  max_backups: 10
  compress: true

performance:
  max_memory: "2GB"
  cache:
    enabled: true
    size: 500
    ttl: 4h
```

### Open Source Project

```yaml
version: "1.0"

github:
  owner: "myorg"
  repo: "open-source-project"
  pr:
    template: ".github/pull_request_template.md"
    labels: ["enhancement", "automated"]
    auto_delete_branch: true

claude:
  model: "claude-3-sonnet-20240229"
  temperature: 0.1
  prompts:
    system_prompt: |
      You are contributing to an open source project.
      Follow these guidelines:
      - Write clean, well-documented code
      - Follow the project's contributing guidelines
      - Include comprehensive tests
      - Be respectful to the community
      - Follow semantic versioning principles

workflow:
  enable_auto_merge: false          # Community review required
  require_review: true
  run_tests: true
  test_command: "npm run test:ci"
  security_scan: true
  
  pre_hooks:
    - "npm run lint"
    - "npm run type-check"
  post_hooks:
    - "npm run docs:generate"

ui:
  theme: "auto"
  enable_notifications: false       # Less intrusive for contributors

logging:
  level: "info"
  format: "pretty"
  filters: ["github.api"]           # Reduce noise
```

### Enterprise Setup

```yaml
version: "1.0"

github:
  api_url: "https://github.company.com/api/v3"
  owner: "engineering"
  repo: "microservice-auth"
  app_id: 12345
  private_key_path: "/etc/ccagents/github-app.pem"
  
  protection:
    required_status_checks: ["ci", "security-scan", "sonar"]
    enforce_admins: true
    require_code_owner_reviews: true

claude:
  model: "claude-3-sonnet-20240229"
  api_url: "https://claude-proxy.company.com"
  max_tokens: 8192
  temperature: 0.05
  
  context:
    max_files: 50
    include_tests: true
    include_docs: true

workflow:
  enable_auto_merge: false
  require_review: true
  review_timeout: 72h
  
  run_tests: true
  test_command: "make test-enterprise"
  test_timeout: 45m
  test_environment:
    ENV: "test"
    DATABASE_URL: "postgresql://test:test@db:5432/test"
    REDIS_URL: "redis://redis:6379/0"
  
  security_scan: true
  security_tools: ["semgrep", "gosec", "snyk", "sonar"]
  security_timeout: 15m
  
  pre_hooks:
    - "make deps"
    - "make lint"
    - "make security-baseline"
  post_hooks:
    - "make docs"
    - "make metrics"
    - "make compliance-check"

observability:
  metrics:
    enabled: true
    port: 9090
  tracing:
    enabled: true
    endpoint: "http://jaeger.monitoring:14268"
    service_name: "ccagents-enterprise"
  health:
    enabled: true
    port: 8080

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "/var/log/ccagents/enterprise.log"
  max_size: 500
  max_backups: 30
  compress: true

performance:
  max_memory: "4GB"
  cache:
    enabled: true
    size: 1000
    ttl: 8h
  connections:
    max_open: 200
    max_lifetime: 2h
```

## Configuration Validation

### Validate Configuration

```bash
# Validate current configuration
ccagents validate

# Validate specific file
ccagents validate --config custom.yaml

# Check configuration syntax only
ccagents validate --syntax-only

# Validate with warnings
ccagents validate --strict
```

### Configuration Schema

```bash
# Show configuration schema
ccagents config schema

# Generate example configuration
ccagents config example > example.yaml

# Show current configuration
ccagents config dump

# Get specific configuration value
ccagents config get github.timeout
```

### Configuration Testing

```bash
# Test configuration with dry run
ccagents test --config custom.yaml --dry-run

# Test authentication
ccagents test auth

# Test GitHub connectivity
ccagents test github

# Test Claude connectivity
ccagents test claude
```

## Best Practices

### Security

1. **Never commit secrets to configuration files**
2. **Use environment variables for sensitive data**
3. **Set appropriate file permissions** (600 for config files)
4. **Use GitHub Apps instead of personal tokens** in production
5. **Enable security scanning** in all environments

### Performance

1. **Use appropriate Claude models** (Haiku for simple tasks, Sonnet for complex)
2. **Configure reasonable timeouts** to prevent hanging
3. **Enable caching** for better performance
4. **Limit parallel operations** based on system capacity
5. **Monitor resource usage** and adjust limits accordingly

### Maintainability

1. **Document custom configurations** with comments
2. **Use configuration inheritance** (global → project → environment)
3. **Version control project configurations**
4. **Test configurations** before deploying
5. **Keep configurations DRY** using templates or includes

### Monitoring

1. **Enable observability features** in production
2. **Configure appropriate log levels** for each environment
3. **Set up alerts** for configuration errors
4. **Monitor performance metrics**
5. **Regular configuration audits**

## Troubleshooting Configuration

See the [Troubleshooting Guide](troubleshooting.md#configuration-issues) for help with configuration issues.