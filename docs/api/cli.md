# CLI API Reference

This document provides comprehensive reference for the ccAgents command-line interface.

## Global Options

These options are available for all commands:

```bash
--config, -c <path>    Path to configuration file (default: .ccagents.yaml)
--verbose, -v          Enable verbose output
--quiet, -q            Suppress non-essential output
--help, -h             Show help for command
--version              Show version information
```

## Environment Variables

```bash
CCAGENTS_CONFIG_PATH      Path to configuration file
CCAGENTS_LOG_LEVEL        Log level (debug, info, warn, error)
CCAGENTS_LOG_FORMAT       Log format (pretty, json, logfmt)
CCAGENTS_GITHUB_TOKEN     GitHub personal access token
CCAGENTS_CLAUDE_API_KEY   Claude API key
```

## Commands

### `ccagents init`

Initialize ccAgents configuration in the current directory.

**Usage:**
```bash
ccagents init [options]
```

**Options:**
- `--force` - Overwrite existing configuration
- `--template <name>` - Use specific configuration template
- `--global` - Create global configuration in `~/.ccagents.yaml`

**Examples:**
```bash
# Initialize with default configuration
ccagents init

# Force overwrite existing configuration
ccagents init --force

# Use production template
ccagents init --template production

# Create global configuration
ccagents init --global
```

**Exit Codes:**
- `0` - Success
- `1` - Configuration already exists (use --force to overwrite)
- `2` - Permission denied

---

### `ccagents process`

Process a GitHub issue and create an automated implementation.

**Usage:**
```bash
ccagents process <issue> [options]
```

**Arguments:**
- `<issue>` - GitHub issue URL or number (e.g., "#123" or full URL)

**Options:**
- `--dry-run` - Analyze only, don't make changes
- `--branch <name>` - Specify custom branch name
- `--draft` - Create draft pull request
- `--assignee <user>` - Assign PR to specific user
- `--reviewer <user>` - Request review from specific user
- `--label <label>` - Add labels to PR (can be used multiple times)
- `--timeout <duration>` - Set custom timeout (e.g., "30m", "1h")

**Examples:**
```bash
# Process issue by URL
ccagents process "https://github.com/owner/repo/issues/123"

# Process issue by number (requires repo config)
ccagents process "#123"

# Dry run analysis
ccagents process "#123" --dry-run

# Create draft PR with specific reviewer
ccagents process "#123" --draft --reviewer @senior-dev

# Add multiple labels
ccagents process "#123" --label "enhancement" --label "backend"
```

**Exit Codes:**
- `0` - Success
- `1` - General error
- `2` - Configuration error
- `3` - Authentication error
- `4` - Issue not found or inaccessible
- `5` - Workflow execution failed

---

### `ccagents status`

Show current status of ccAgents workflows.

**Usage:**
```bash
ccagents status [options]
```

**Options:**
- `--follow, -f` - Follow status updates in real-time
- `--format <format>` - Output format (text, json, yaml)
- `--filter <filter>` - Filter by status (running, completed, failed)
- `--workflow <id>` - Show specific workflow status
- `--since <duration>` - Show workflows since duration (e.g., "1h", "24h")

**Examples:**
```bash
# Show current status
ccagents status

# Follow status in real-time
ccagents status --follow

# Show JSON output
ccagents status --format json

# Show only running workflows
ccagents status --filter running

# Show specific workflow
ccagents status --workflow wf_123456789
```

**Sample Output:**
```
ccAgents Status
===============

Active Workflows: 2
Completed Today: 5
Failed Today: 0

ID           Status    Issue                    Progress    Duration
wf_123456789 running   feat: add user auth (#123) 60%        5m 30s
wf_987654321 running   fix: login bug (#124)      30%        2m 15s

Recent Completions:
wf_111222333 completed feat: dashboard (#122)     100%       15m 45s
```

---

### `ccagents workflow`

Manage workflows and workflow execution.

**Usage:**
```bash
ccagents workflow <subcommand> [options]
```

#### `ccagents workflow list`

List all workflows.

**Options:**
- `--status <status>` - Filter by status (pending, running, completed, failed)
- `--limit <number>` - Limit number of results
- `--since <duration>` - Show workflows since duration

**Example:**
```bash
ccagents workflow list --status failed --limit 10
```

#### `ccagents workflow show <id>`

Show detailed information about a specific workflow.

**Example:**
```bash
ccagents workflow show wf_123456789
```

#### `ccagents workflow retry <id>`

Retry a failed workflow.

**Options:**
- `--step <step>` - Retry from specific step
- `--force` - Force retry even if not failed

**Example:**
```bash
ccagents workflow retry wf_123456789 --step generate-code
```

#### `ccagents workflow cancel <id>`

Cancel a running workflow.

**Example:**
```bash
ccagents workflow cancel wf_123456789
```

#### `ccagents workflow logs <id>`

Show logs for a specific workflow.

**Options:**
- `--follow, -f` - Follow logs in real-time
- `--tail <number>` - Show last N lines
- `--level <level>` - Filter by log level

**Example:**
```bash
ccagents workflow logs wf_123456789 --follow --level debug
```

---

### `ccagents config`

Manage ccAgents configuration.

**Usage:**
```bash
ccagents config <subcommand> [options]
```

#### `ccagents config get <key>`

Get a configuration value.

**Example:**
```bash
ccagents config get github.timeout
ccagents config get claude.model
```

#### `ccagents config set <key> <value>`

Set a configuration value.

**Example:**
```bash
ccagents config set github.timeout "60s"
ccagents config set claude.model "claude-3-sonnet-20240229"
```

#### `ccagents config dump`

Show current configuration.

**Options:**
- `--format <format>` - Output format (yaml, json)
- `--safe` - Hide sensitive values

**Example:**
```bash
ccagents config dump --safe
```

#### `ccagents config schema`

Show configuration schema.

**Example:**
```bash
ccagents config schema
```

#### `ccagents config validate`

Validate current configuration.

**Options:**
- `--strict` - Enable strict validation
- `--file <path>` - Validate specific file

**Example:**
```bash
ccagents config validate --strict
```

---

### `ccagents validate`

Validate ccAgents configuration and setup.

**Usage:**
```bash
ccagents validate [options]
```

**Options:**
- `--config <path>` - Validate specific configuration file
- `--syntax-only` - Check syntax only, skip connectivity tests
- `--strict` - Enable strict validation with warnings
- `--check-auth` - Validate authentication credentials
- `--check-connectivity` - Test network connectivity
- `--check-dependencies` - Verify required dependencies

**Examples:**
```bash
# Full validation
ccagents validate

# Syntax check only
ccagents validate --syntax-only

# Strict validation with warnings
ccagents validate --strict

# Validate specific file
ccagents validate --config production.yaml
```

---

### `ccagents auth`

Manage authentication credentials.

**Usage:**
```bash
ccagents auth <subcommand> [options]
```

#### `ccagents auth status`

Check authentication status.

**Options:**
- `--verbose` - Show detailed status
- `--test` - Test API connectivity

**Example:**
```bash
ccagents auth status --verbose
```

#### `ccagents auth setup`

Set up authentication interactively.

**Example:**
```bash
ccagents auth setup
```

#### `ccagents auth refresh`

Refresh authentication tokens.

**Example:**
```bash
ccagents auth refresh
```

---

### `ccagents test`

Test ccAgents functionality and connectivity.

**Usage:**
```bash
ccagents test [options]
```

**Options:**
- `--all` - Run all tests
- `--github` - Test GitHub connectivity
- `--claude` - Test Claude connectivity
- `--config` - Test configuration
- `--auth` - Test authentication
- `--dry-run` - Perform dry run tests only

**Examples:**
```bash
# Run all tests
ccagents test --all

# Test specific service
ccagents test --github

# Dry run tests
ccagents test --dry-run
```

---

### `ccagents logs`

View ccAgents logs.

**Usage:**
```bash
ccagents logs [options]
```

**Options:**
- `--follow, -f` - Follow logs in real-time
- `--tail <number>` - Show last N lines
- `--level <level>` - Filter by log level (debug, info, warn, error)
- `--since <duration>` - Show logs since duration
- `--grep <pattern>` - Search for specific pattern
- `--format <format>` - Output format (text, json)
- `--filter <component>` - Filter by component

**Examples:**
```bash
# Show recent logs
ccagents logs --tail 50

# Follow logs in real-time
ccagents logs --follow

# Debug level logs
ccagents logs --level debug

# Search for errors
ccagents logs --grep "error"

# Filter by workflow component
ccagents logs --filter workflow
```

---

### `ccagents help`

Get help information.

**Usage:**
```bash
ccagents help [topic]
```

**Topics:**
- `getting-started` - Getting started guide
- `configuration` - Configuration help
- `authentication` - Authentication setup
- `workflows` - Workflow information
- `troubleshooting` - Troubleshooting guide

**Examples:**
```bash
# General help
ccagents help

# Specific topic
ccagents help configuration

# Command help
ccagents process --help
```

---

### `ccagents version`

Show version information.

**Usage:**
```bash
ccagents version [options]
```

**Options:**
- `--short` - Show short version only
- `--build` - Show build information

**Example:**
```bash
ccagents version --build
```

---

### `ccagents server`

Start ccAgents API server.

**Usage:**
```bash
ccagents server [options]
```

**Options:**
- `--port <port>` - Server port (default: 8080)
- `--host <host>` - Server host (default: localhost)
- `--cors` - Enable CORS
- `--tls-cert <path>` - TLS certificate file
- `--tls-key <path>` - TLS private key file

**Example:**
```bash
ccagents server --port 8080 --cors
```

---

## Aliases

Some commands have shorter aliases:

```bash
ccagents st     # status
ccagents wf     # workflow
ccagents cfg    # config
ccagents proc   # process
ccagents val    # validate
```

## Shell Completion

Enable shell completion for better CLI experience:

**Bash:**
```bash
source <(ccagents completion bash)
echo 'source <(ccagents completion bash)' >> ~/.bashrc
```

**Zsh:**
```bash
source <(ccagents completion zsh)
echo 'source <(ccagents completion zsh)' >> ~/.zshrc
```

**Fish:**
```bash
ccagents completion fish | source
ccagents completion fish > ~/.config/fish/completions/ccagents.fish
```

## Common Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Authentication error |
| 4 | Network error |
| 5 | Permission error |
| 6 | Timeout error |
| 130 | Interrupted by user (Ctrl+C) |

## Examples and Workflows

### Basic Workflow

```bash
# 1. Initialize configuration
ccagents init

# 2. Validate setup
ccagents validate

# 3. Process an issue
ccagents process "https://github.com/owner/repo/issues/123"

# 4. Monitor progress
ccagents status --follow
```

### Development Workflow

```bash
# Use development configuration
ccagents init --template development

# Process issue with draft PR
ccagents process "#123" --draft --reviewer @team-lead

# Monitor specific workflow
ccagents workflow logs wf_123456789 --follow
```

### Troubleshooting Workflow

```bash
# Check system status
ccagents status --verbose

# Validate configuration
ccagents validate --strict

# Check authentication
ccagents auth status --test

# View recent errors
ccagents logs --level error --since 1h
```