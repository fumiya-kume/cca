# Contributing to ccAgents

Thank you for your interest in contributing to ccAgents! This guide will help you get started with contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Documentation](#documentation)
- [Community](#community)

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.21 or later** - [Install Go](https://golang.org/doc/install)
- **Git** - [Install Git](https://git-scm.com/downloads)
- **GitHub CLI** - [Install gh](https://cli.github.com/)
- **Claude Code** - [Install claude](https://claude.ai/code)
- **Make** - For running build tasks

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/your-username/cca.git
cd cca
```

3. Add the original repository as upstream:

```bash
git remote add upstream https://github.com/fumiya-kume/cca.git
```

## Development Setup

### Environment Setup

1. **Install dependencies:**

```bash
make deps
```

2. **Set up development configuration:**

```bash
cp docs/examples/development.yaml .ccagents.yaml
# Edit .ccagents.yaml with your settings
```

3. **Set up authentication:**

```bash
gh auth login
claude auth
```

4. **Verify setup:**

```bash
make test
make lint
make build
```

### Development Workflow

1. **Create a feature branch:**

```bash
git checkout -b feature/your-feature-name
```

2. **Make your changes and commit:**

```bash
git add .
git commit -m "feat: add your feature description"
```

3. **Keep your branch updated:**

```bash
git fetch upstream
git rebase upstream/main
```

4. **Push to your fork:**

```bash
git push origin feature/your-feature-name
```

### Project Structure

```
cca/
├── cmd/                    # CLI commands
├── internal/               # Internal packages
│   ├── types/             # Type definitions
│   └── utils/             # Utility functions
├── pkg/                    # Public packages
│   ├── analysis/          # Code analysis
│   ├── claude/            # Claude integration
│   ├── github/            # GitHub integration
│   ├── workflow/          # Workflow engine
│   └── ...
├── docs/                   # Documentation
├── test/                   # Test files
├── Makefile               # Build tasks
└── go.mod                 # Go modules
```

### Build Tasks

We use Make for common development tasks:

```bash
# Build the application
make build

# Run tests
make test

# Run linting
make lint

# Format code
make format

# Generate documentation
make docs

# Run security scan
make security

# Clean build artifacts
make clean

# Install development tools
make tools

# Run all checks (recommended before PR)
make check
```

## Contributing Guidelines

### Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` and `goimports` for formatting
- Follow the existing code style in the project
- Write clear, concise comments for public APIs
- Use meaningful variable and function names

### Commit Messages

We follow [Conventional Commits](https://conventionalcommits.org/) specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(workflow): add support for custom templates

fix(github): handle rate limiting gracefully

docs: update installation instructions

test(claude): add unit tests for prompt generation
```

### Branch Naming

Use descriptive branch names:

```
feature/add-custom-templates
fix/github-rate-limiting
docs/update-api-reference
refactor/workflow-engine
```

## Pull Request Process

### Before Creating a PR

1. Ensure your code follows the project style
2. Add or update tests for your changes
3. Update documentation if needed
4. Run all checks locally:

```bash
make check
```

### Creating a Pull Request

1. **Create PR from your fork:**
   - Go to your fork on GitHub
   - Click "New pull request"
   - Select your feature branch

2. **Use the PR template:**
   - Fill out the provided PR template
   - Include clear description of changes
   - Reference related issues

3. **PR Checklist:**
   - [ ] Code follows project style guidelines
   - [ ] Self-review of the code completed
   - [ ] Tests added/updated for new functionality
   - [ ] Documentation updated if needed
   - [ ] All CI checks pass
   - [ ] Changes are backwards compatible (or breaking changes documented)

### PR Review Process

1. **Automated Checks:**
   - Tests must pass
   - Linting must pass
   - Security scans must pass
   - Build must succeed

2. **Code Review:**
   - At least one maintainer review required
   - Address all review comments
   - Keep discussions professional and constructive

3. **Merge:**
   - PRs are merged by maintainers
   - Usually squash-merged to keep history clean
   - Branch will be deleted after merge

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./pkg/workflow/...

# Run integration tests
make test-integration

# Run end-to-end tests
make test-e2e
```

### Writing Tests

- Write unit tests for all new functionality
- Use table-driven tests where appropriate
- Mock external dependencies
- Aim for >80% test coverage
- Include both positive and negative test cases

**Example test structure:**

```go
func TestWorkflowEngine_ProcessIssue(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Result
        wantErr bool
    }{
        {
            name:    "valid issue URL",
            input:   "https://github.com/owner/repo/issues/123",
            want:    &Result{WorkflowID: "wf_123"},
            wantErr: false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Test Categories

1. **Unit Tests** - Test individual functions/methods
2. **Integration Tests** - Test component interactions
3. **End-to-End Tests** - Test complete workflows
4. **Performance Tests** - Test performance characteristics

## Documentation

### Documentation Requirements

- Update README.md for major changes
- Add/update API documentation for new APIs
- Include code examples in documentation
- Update configuration documentation for new options
- Add troubleshooting information for common issues

### Documentation Structure

```
docs/
├── README.md              # Main documentation
├── installation.md        # Installation guide
├── getting-started.md     # Getting started tutorial
├── configuration.md       # Configuration reference
├── troubleshooting.md     # Troubleshooting guide
├── api/                   # API documentation
├── examples/              # Configuration examples
└── CONTRIBUTING.md        # This file
```

### Writing Documentation

- Use clear, concise language
- Include practical examples
- Keep documentation up-to-date with code changes
- Use proper Markdown formatting
- Include diagrams where helpful

## Community

### Getting Help

- **GitHub Discussions** - For questions and general discussion
- **GitHub Issues** - For bug reports and feature requests
- **Discord** - For real-time chat and community support

### Reporting Issues

When reporting bugs, please include:

- ccAgents version (`ccagents version`)
- Operating system and version
- Go version (`go version`)
- Steps to reproduce the issue
- Expected vs actual behavior
- Relevant logs or error messages
- Configuration (sanitized, no secrets)

Use the issue templates when available.

### Feature Requests

For feature requests:

- Check existing issues first
- Describe the problem you're trying to solve
- Explain why this feature would be valuable
- Provide implementation ideas if you have them
- Consider contributing the feature yourself

### Security Issues

For security vulnerabilities:

- **DO NOT** create public issues
- Follow our [Security Policy](../SECURITY.md)
- Report to security@ccagents.dev
- Allow time for investigation and fix

## Development Tips

### Debugging

- Use the `--verbose` flag for detailed output
- Set `CCAGENTS_LOG_LEVEL=debug` for debug logs
- Use Go debugger (delve) for complex issues
- Check logs in `~/.ccagents/logs/`

### Performance

- Profile code using Go pprof tools
- Monitor memory usage
- Consider concurrent operations carefully
- Cache expensive operations when appropriate

### Common Pitfalls

- Always handle errors properly
- Don't ignore context cancellation
- Be careful with goroutine lifecycle
- Avoid blocking operations in hot paths
- Clean up resources (files, connections)

## Release Process

For maintainers:

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create release branch
4. Run full test suite
5. Create GitHub release
6. Update documentation
7. Announce release

## Questions?

If you have questions about contributing:

- Check the documentation first
- Search existing issues and discussions
- Ask in GitHub Discussions
- Join our Discord community

Thank you for contributing to ccAgents! 🎉