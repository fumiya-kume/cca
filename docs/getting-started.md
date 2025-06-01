# Getting Started with ccAgents

This guide will walk you through your first automation workflow with ccAgents.

## Prerequisites

Before starting, ensure you have:
- ✅ ccAgents installed ([Installation Guide](installation.md))
- ✅ GitHub CLI configured (`gh auth status`)
- ✅ Claude Code CLI configured (`claude auth status`)
- ✅ Access to a GitHub repository

## Quick Setup

### 1. Initialize ccAgents

```bash
# Create default configuration
ccagents init

# Verify configuration
ccagents validate
```

### 2. Configure Your Repository

Navigate to your project directory:

```bash
cd /path/to/your/project
```

Create a project-specific configuration:

```bash
# Initialize project configuration
ccagents init --project

# This creates .ccagents.yaml in your project root
```

## Your First Automation

Let's automate a simple GitHub issue. We'll use a real example.

### Step 1: Create a Test Issue

Create an issue in your GitHub repository with this content:

```markdown
Title: Add user authentication validation

Description:
We need to add input validation for user authentication forms.

Requirements:
- Validate email format
- Ensure password meets security requirements (min 8 chars, special chars)
- Add rate limiting for failed attempts
- Return appropriate error messages

Acceptance Criteria:
- [ ] Email validation using regex
- [ ] Password strength validation
- [ ] Rate limiting implementation
- [ ] Unit tests for all validation functions
- [ ] Error messages are user-friendly
```

### Step 2: Process the Issue

```bash
# Process the issue (replace with your issue URL)
ccagents process "https://github.com/your-username/your-repo/issues/1"
```

### Step 3: Monitor Progress

In another terminal, monitor the progress:

```bash
# Watch real-time progress
ccagents status --follow

# Or check current status
ccagents status
```

## Understanding the Workflow

ccAgents follows this workflow:

```
Issue Analysis → Code Context → AI Implementation → Testing → Review → Merge
```

### 1. Issue Analysis
- Parses the GitHub issue
- Extracts requirements and acceptance criteria
- Identifies affected components

### 2. Code Context Building
- Analyzes your codebase structure
- Identifies relevant files and patterns
- Understands existing conventions

### 3. AI Implementation
- Uses Claude Code to generate code
- Creates implementation following your patterns
- Generates corresponding tests

### 4. Quality Assurance
- Runs automated tests
- Performs linting and formatting
- Conducts security scans

### 5. Review Process
- Creates a pull request
- Requests reviews if configured
- Handles feedback and iterations

### 6. Merge
- Merges when all checks pass
- Updates issue status
- Cleans up temporary branches

## Interactive Mode

For more control, use interactive mode:

```bash
ccagents interactive
```

This opens a terminal UI where you can:
- Browse and select issues
- Monitor progress in real-time
- Make decisions at key points
- View logs and debug information

## Configuration Examples

### Basic Configuration

```yaml
# .ccagents.yaml
version: "1.0"

github:
  owner: "your-username"
  repo: "your-repo"
  default_branch: "main"

workflow:
  enable_auto_merge: false  # Require manual approval
  require_review: true
  run_tests: true
  security_scan: true

claude:
  model: "claude-3-sonnet-20240229"
  temperature: 0.1  # Conservative for code generation
```

### Advanced Configuration

```yaml
version: "1.0"

github:
  owner: "your-org"
  repo: "your-repo"
  api_version: "2022-11-28"
  timeout: 60s

workflow:
  enable_auto_merge: true
  require_review: true
  review_team: "core-team"
  run_tests: true
  test_command: "npm test"
  security_scan: true
  security_tools: ["semgrep", "gosec"]
  
  # Custom workflow steps
  pre_hooks:
    - "make lint"
    - "make format"
  post_hooks:
    - "make docs"

claude:
  model: "claude-3-sonnet-20240229"
  max_tokens: 8192
  temperature: 0.1
  
  # Custom prompts
  system_prompt: |
    You are a senior software engineer working on a production system.
    Follow these guidelines:
    - Write clean, maintainable code
    - Include comprehensive error handling
    - Add detailed comments for complex logic
    - Follow the existing code style

ui:
  theme: "dark"
  show_progress: true
  enable_notifications: true
  
logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "./logs/ccagents.log"
```

## Common Commands

### Processing Issues

```bash
# Process by URL
ccagents process "https://github.com/owner/repo/issues/123"

# Process by issue number (requires repo configuration)
ccagents process "#123"

# Process with custom config
ccagents process "#123" --config custom.yaml

# Dry run (analyze only, don't make changes)
ccagents process "#123" --dry-run
```

### Workflow Management

```bash
# List active workflows
ccagents workflow list

# Check workflow status
ccagents workflow status <workflow-id>

# Cancel a workflow
ccagents workflow cancel <workflow-id>

# Retry a failed workflow
ccagents workflow retry <workflow-id>
```

### Monitoring

```bash
# Real-time status
ccagents status --follow

# Show logs
ccagents logs --level debug --follow

# Performance metrics
ccagents metrics

# Health check
ccagents health
```

## Working with Different Issue Types

### Feature Requests

```markdown
Title: Add dark mode support

Description: Users want dark mode for better nighttime usage.

Requirements:
- Toggle between light and dark themes
- Persist user preference
- Smooth transitions
- Accessibility compliance
```

### Bug Reports

```markdown
Title: Fix memory leak in user session handling

Description: Memory usage increases over time in production.

Steps to Reproduce:
1. Login multiple users
2. Let sessions timeout
3. Memory usage doesn't decrease

Expected: Memory should be freed when sessions expire
Actual: Memory usage continues to grow
```

### Refactoring Tasks

```markdown
Title: Refactor authentication module for better testability

Description: Current auth module is tightly coupled and hard to test.

Goals:
- Extract interfaces for dependencies
- Improve test coverage to >90%
- Maintain backward compatibility
- Add integration tests
```

## Best Practices

### 1. Write Clear Issues

- **Be specific**: Include exact requirements and acceptance criteria
- **Provide context**: Explain why the change is needed
- **Include examples**: Show expected inputs/outputs
- **Reference related code**: Point to relevant files or functions

### 2. Repository Setup

- **Configure branch protection**: Require reviews and status checks
- **Set up CI/CD**: Ensure automated testing runs
- **Define coding standards**: Use linters and formatters
- **Document conventions**: Maintain coding style guides

### 3. Review Configuration

```yaml
workflow:
  # Always require human review for critical changes
  require_review: true
  
  # Use auto-merge only for low-risk changes
  enable_auto_merge: false
  
  # Run comprehensive tests
  test_command: "make test-all"
  
  # Enable security scanning
  security_scan: true
```

### 4. Monitor and Iterate

- **Check logs regularly**: `ccagents logs --level info`
- **Review metrics**: `ccagents metrics`
- **Update configuration**: Based on experience and needs
- **Provide feedback**: Help improve the AI's performance

## Troubleshooting

### Common Issues

**Issue**: ccAgents can't access the repository
```bash
# Check GitHub authentication
gh auth status

# Verify repository access
gh repo view owner/repo
```

**Issue**: Claude Code requests fail
```bash
# Check Claude authentication
claude auth status

# Test Claude connection
claude models list
```

**Issue**: Tests fail during automation
```bash
# Run tests manually to debug
ccagents validate --run-tests

# Check test configuration
ccagents config get workflow.test_command
```

**Issue**: Pull request creation fails
```bash
# Check branch permissions
gh api repos/owner/repo/branches/main/protection

# Verify CI status
ccagents status --verbose
```

### Getting Help

1. **Check status**: `ccagents status --verbose`
2. **View logs**: `ccagents logs --level debug`
3. **Validate config**: `ccagents validate`
4. **Read docs**: [Troubleshooting Guide](troubleshooting.md)
5. **Ask community**: [GitHub Discussions](https://github.com/fumiya-kume/cca/discussions)

## Next Steps

Now that you've completed your first automation:

1. **Explore advanced features**: [Advanced Workflows](advanced/workflows.md)
2. **Customize configuration**: [Configuration Guide](configuration.md)
3. **Set up monitoring**: [Monitoring Guide](advanced/monitoring.md)
4. **Join the community**: [GitHub Discussions](https://github.com/fumiya-kume/cca/discussions)

## Example Projects

Check out these example projects using ccAgents:

- **[Simple Web App](examples/web-app/)** - Basic authentication and CRUD operations
- **[CLI Tool](examples/cli-tool/)** - Command-line utility with comprehensive testing
- **[API Service](examples/api-service/)** - REST API with security and monitoring
- **[React Component Library](examples/component-lib/)** - UI components with Storybook

## Tips for Success

1. **Start small**: Begin with simple issues to understand the workflow
2. **Be descriptive**: Better issue descriptions lead to better implementations
3. **Review outputs**: Always review generated code before merging
4. **Iterate**: Improve your configuration based on results
5. **Stay updated**: Keep ccAgents and dependencies up to date

Happy automating! 🚀