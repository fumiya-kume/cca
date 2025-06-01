# ccAgents - AI-Powered GitHub Issue to Pull Request Automation

[![Build Status](https://img.shields.io/github/workflow/status/fumiya-kume/cca/CI)](https://github.com/fumiya-kume/cca/actions)
[![Go Version](https://img.shields.io/github/go-mod/go-version/fumiya-kume/cca)](https://github.com/fumiya-kume/cca)
[![Release](https://img.shields.io/github/v/release/fumiya-kume/cca)](https://github.com/fumiya-kume/cca/releases)
[![License](https://img.shields.io/github/license/fumiya-kume/cca)](LICENSE)

**ccAgents** is an intelligent automation tool that transforms GitHub issues into fully implemented and merged pull requests using Claude Code as an AI sub-agent. It provides a seamless workflow from issue analysis to code implementation, review, and deployment.

## 🎯 What ccAgents Does

ccAgents automates the entire software development lifecycle:

1. **Issue Analysis** - Parses GitHub issues and extracts requirements
2. **Code Context Building** - Analyzes your codebase and understands the architecture
3. **AI-Powered Implementation** - Uses Claude Code to write code, tests, and documentation
4. **Quality Assurance** - Runs tests, linting, and security checks
5. **Review & Iteration** - Handles review feedback and iterative improvements
6. **Automated Deployment** - Merges changes when all checks pass

## ✨ Key Features

- **🤖 AI-Powered Development**: Leverages Claude Code for intelligent code generation
- **🔄 End-to-End Automation**: From issue to merged PR with minimal human intervention
- **🎯 Context-Aware**: Deep understanding of your codebase architecture and conventions
- **🛡️ Security-First**: Comprehensive security scanning and credential management
- **📊 Performance Optimized**: Built-in performance monitoring and optimization
- **🔍 Quality Assurance**: Automated testing, linting, and code review
- **📈 Observability**: Comprehensive monitoring and debugging capabilities
- **🎨 Beautiful TUI**: Intuitive terminal user interface with real-time progress
- **🔧 Highly Configurable**: Extensive configuration options for any workflow
- **🌐 Multi-Platform**: Cross-platform support for macOS, Linux, and Windows

## 🚀 Quick Start

### Prerequisites

- Go 1.24 or later
- Git
- GitHub CLI (`gh`) configured with authentication
- Claude Code CLI installed and configured
- Docker (for DevContainer development)
- VS Code with DevContainer extension (for local development)

### Installation

```bash
# Install from source
git clone https://github.com/fumiya-kume/cca.git
cd cca
go install ./cmd/ccagents

# Or install from releases
curl -sSL https://github.com/fumiya-kume/cca/releases/latest/download/install.sh | bash
```

### Initial Setup

1. **Configure GitHub Authentication**:
   ```bash
   gh auth login
   ```

2. **Configure Claude Code**:
   ```bash
   claude auth
   ```

3. **Initialize ccAgents Configuration**:
   ```bash
   ccagents init
   ```

### Basic Usage

```bash
# Process a GitHub issue
ccagents process "https://github.com/owner/repo/issues/123"

# Process with custom configuration
ccagents process "#123" --config custom-config.yaml

# Interactive mode
ccagents interactive

# Monitor progress
ccagents status

# Show help
ccagents help
```

## 📖 Documentation

### Core Concepts

- **[Architecture Overview](docs/architecture.md)** - Understanding ccAgents' design
- **[Workflow Engine](docs/workflow.md)** - How the automation workflow works
- **[Configuration Guide](docs/configuration.md)** - Customizing ccAgents behavior
- **[Security Model](docs/security.md)** - Security features and best practices

### User Guides

- **[Installation Guide](docs/installation.md)** - Detailed installation instructions
- **[Getting Started](docs/getting-started.md)** - Your first automation
- **[Configuration Examples](docs/examples/)** - Common configuration patterns
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and solutions

### Advanced Topics

- **[Custom Workflows](docs/advanced/workflows.md)** - Creating custom automation workflows
- **[Plugin Development](docs/advanced/plugins.md)** - Extending ccAgents functionality
- **[Performance Tuning](docs/advanced/performance.md)** - Optimizing performance
- **[Monitoring & Observability](docs/advanced/monitoring.md)** - Monitoring your automations

### Developer Documentation

- **[API Reference](docs/api/)** - Complete API documentation
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to ccAgents
- **[Development Setup](docs/development.md)** - Setting up a development environment
- **[DevContainer Setup](#-devcontainer-development)** - Using DevContainers for consistent development
- **[Testing Guide](docs/testing.md)** - Testing strategies and frameworks

## 🛠️ Configuration

ccAgents uses YAML configuration files for customization:

```yaml
# .ccagents.yaml
version: "1.0"

claude:
  model: "claude-3-sonnet-20240229"
  max_tokens: 4096
  temperature: 0.1

github:
  api_version: "2022-11-28"
  timeout: 30s

workflow:
  enable_auto_merge: true
  require_review: true
  run_tests: true
  security_scan: true

ui:
  theme: "dark"
  show_progress: true
  enable_notifications: true
```

See the [Configuration Guide](docs/configuration.md) for complete options.

## 🔧 Commands

### Core Commands

- `ccagents process <issue>` - Process a GitHub issue
- `ccagents interactive` - Start interactive mode
- `ccagents status` - Show current status
- `ccagents config` - Manage configuration
- `ccagents init` - Initialize new configuration

### Workflow Commands

- `ccagents workflow list` - List available workflows
- `ccagents workflow run <name>` - Run a specific workflow
- `ccagents workflow status` - Show workflow status

### Monitoring Commands

- `ccagents monitor` - Start monitoring dashboard
- `ccagents logs` - View application logs
- `ccagents metrics` - Show performance metrics

### Utility Commands

- `ccagents validate` - Validate configuration
- `ccagents version` - Show version information
- `ccagents help` - Show help information

## 🎯 Use Cases

### Development Teams

- **Feature Implementation**: Automatically implement features from detailed issues
- **Bug Fixes**: Analyze bug reports and generate fixes with tests
- **Documentation**: Generate and maintain documentation from code changes
- **Refactoring**: Automated code refactoring and optimization

### Open Source Projects

- **Contributor Onboarding**: Help new contributors with automated implementations
- **Issue Triage**: Automatically categorize and prioritize issues
- **Maintenance**: Automated dependency updates and security patches
- **Release Management**: Automated changelog generation and release preparation

### Enterprise Workflows

- **Compliance**: Ensure all changes meet security and compliance requirements
- **Quality Gates**: Automated quality checks and approval workflows
- **Integration**: Seamless integration with existing CI/CD pipelines
- **Monitoring**: Comprehensive observability and performance monitoring

## 🔒 Security

ccAgents takes security seriously:

- **🔐 Secure Credential Management**: Encrypted storage of API keys and tokens
- **🛡️ Security Scanning**: Automated vulnerability detection in code changes
- **🔍 Code Analysis**: Static analysis for security issues and best practices
- **📝 Audit Logging**: Comprehensive audit trails for all operations
- **🚫 Least Privilege**: Minimal permissions required for operation

See our [Security Guide](docs/security.md) for detailed information.

## 📊 Performance

ccAgents is designed for performance:

- **⚡ Fast Processing**: Optimized algorithms for rapid code analysis
- **🔄 Concurrent Operations**: Parallel processing of multiple tasks
- **💾 Smart Caching**: Intelligent caching to reduce API calls
- **📈 Monitoring**: Built-in performance monitoring and alerting
- **🎯 Resource Optimization**: Efficient memory and CPU usage

## 🧪 Testing

ccAgents includes comprehensive testing:

```bash
# Run all tests
make test

# Run specific test suites
make test-unit
make test-integration
make test-e2e

# Run benchmarks
make benchmark

# Generate test coverage
make coverage
```

## 🐛 Troubleshooting

### Common Issues

**Issue**: Claude Code authentication fails
```bash
# Solution: Reconfigure authentication
claude auth logout
claude auth login
```

**Issue**: GitHub API rate limiting
```bash
# Solution: Use GitHub token with higher limits
gh auth login --with-token < token.txt
```

**Issue**: Configuration validation errors
```bash
# Solution: Validate configuration
ccagents validate --config .ccagents.yaml
```

See the [Troubleshooting Guide](docs/troubleshooting.md) for more solutions.

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

#### Local Development
```bash
# Clone the repository
git clone https://github.com/fumiya-kume/cca.git
cd cca

# Install dependencies
go mod download

# Run tests
make test

# Start development server
make dev
```

#### 🐳 DevContainer Development

For a consistent development environment across all platforms, use DevContainers:

```bash
# Prerequisites: Docker and VS Code with DevContainer extension

# 1. Clone and open in VS Code
git clone https://github.com/fumiya-kume/cca.git
cd cca
code .

# 2. VS Code will prompt to "Reopen in Container" - click it
# Or use Command Palette: "Dev Containers: Reopen in Container"

# 3. The container will build automatically with:
#    - Go 1.24.3
#    - All development tools (golangci-lint, goimports, gofumpt)
#    - Pre-configured VS Code extensions
#    - Git and GitHub CLI

# 4. Start developing immediately:
make test
make dev
```

**DevContainer Features:**
- ✅ Consistent environment across team members
- ✅ Pre-installed development tools
- ✅ VS Code extensions automatically configured
- ✅ Works on Windows, macOS, and Linux
- ✅ Used in CI/CD for environment consistency

### Code Style

- Follow Go best practices and conventions
- Write comprehensive tests for new features
- Update documentation for any changes
- Use conventional commit messages

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Anthropic** for Claude Code and Claude AI
- **GitHub** for the GitHub API and platform
- **Charm** for Bubble Tea TUI framework
- **Go Community** for excellent tooling and libraries

## 📞 Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/fumiya-kume/cca/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fumiya-kume/cca/discussions)
- **Email**: [support@ccagents.dev](mailto:support@ccagents.dev)

## 🔗 Links

- **Website**: [ccagents.dev](https://ccagents.dev)
- **Documentation**: [docs.ccagents.dev](https://docs.ccagents.dev)
- **Blog**: [blog.ccagents.dev](https://blog.ccagents.dev)
- **Twitter**: [@ccagents](https://twitter.com/ccagents)

---

<p align="center">
  <img src="docs/assets/logo.png" alt="ccAgents Logo" width="100">
  <br>
  <em>Automate your GitHub workflow with AI</em>
</p>