# Configuration Examples

This directory contains example configurations for various scenarios and project types.

## Quick Start Examples

### [Basic Setup](basic-setup.yaml)
Simple configuration for getting started with ccAgents.

### [Development Environment](development.yaml)
Configuration optimized for development workflows.

### [Production Environment](production.yaml)
Enterprise-grade configuration for production use.

## Project Type Examples

### [Web Application](web-app.yaml)
Configuration for React/Vue/Angular web applications.

### [API Service](api-service.yaml)
Configuration for REST APIs and microservices.

### [CLI Tool](cli-tool.yaml)
Configuration for command-line applications.

### [Library/Package](library.yaml)
Configuration for npm packages, Go modules, Python packages.

### [Mobile App](mobile-app.yaml)
Configuration for React Native, Flutter mobile apps.

## Workflow Examples

### [Simple Workflow](workflows/simple.yaml)
Basic automation workflow with minimal steps.

### [Complex Workflow](workflows/complex.yaml)
Advanced workflow with multiple quality gates.

### [Security-First](workflows/security-first.yaml)
Security-focused workflow with comprehensive scanning.

### [Performance-Optimized](workflows/performance.yaml)
Performance-optimized workflow for high-throughput scenarios.

## Language-Specific Examples

### [Go Project](languages/go.yaml)
Configuration for Go applications and libraries.

### [Node.js Project](languages/nodejs.yaml)
Configuration for Node.js applications and packages.

### [Python Project](languages/python.yaml)
Configuration for Python applications and packages.

### [Java Project](languages/java.yaml)
Configuration for Java applications and Spring Boot.

### [Rust Project](languages/rust.yaml)
Configuration for Rust applications and crates.

## Team Examples

### [Small Team](teams/small-team.yaml)
Configuration for teams of 2-5 developers.

### [Large Team](teams/large-team.yaml)
Configuration for teams of 20+ developers.

### [Open Source](teams/open-source.yaml)
Configuration for open source projects.

### [Enterprise](teams/enterprise.yaml)
Configuration for enterprise environments.

## Integration Examples

### [GitHub Actions](integrations/github-actions.yaml)
Integration with GitHub Actions CI/CD.

### [Jenkins](integrations/jenkins.yaml)
Integration with Jenkins pipelines.

### [GitLab CI](integrations/gitlab-ci.yaml)
Integration with GitLab CI/CD.

### [Slack Notifications](integrations/slack.yaml)
Configuration with Slack notifications.

## Usage

To use an example configuration:

1. **Copy the example file:**
   ```bash
   cp docs/examples/basic-setup.yaml .ccagents.yaml
   ```

2. **Customize the configuration:**
   - Update repository information
   - Adjust workflow settings
   - Configure authentication

3. **Validate the configuration:**
   ```bash
   ccagents validate
   ```

4. **Test the configuration:**
   ```bash
   ccagents test --dry-run
   ```

## Contributing Examples

To contribute a new example:

1. Create a new YAML file with descriptive name
2. Add comprehensive comments explaining options
3. Include usage instructions in this README
4. Test the configuration works as expected
5. Submit a pull request

## Example Template

```yaml
# Example Configuration: [Description]
# Use case: [Specific use case]
# Team size: [Small/Medium/Large]
# Environment: [Development/Staging/Production]

version: "1.0"

# Repository configuration
github:
  owner: "example-org"           # Replace with your organization
  repo: "example-repo"           # Replace with your repository
  default_branch: "main"

# Claude AI configuration
claude:
  model: "claude-3-sonnet-20240229"
  max_tokens: 4096
  temperature: 0.1

# Workflow configuration
workflow:
  enable_auto_merge: false       # Set to true for automatic merging
  require_review: true           # Require human review
  run_tests: true               # Run tests before creating PR

# Additional configuration sections...
```

## Getting Help

- [Configuration Guide](../configuration.md)
- [Troubleshooting](../troubleshooting.md)
- [GitHub Discussions](https://github.com/fumiya-kume/cca/discussions)