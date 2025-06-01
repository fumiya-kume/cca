// Package help provides contextual help and guidance for ccAgents
package help

import (
	"fmt"
	"sort"
	"strings"
)

// Error type constants
const (
	errorTypeConfiguration = "configuration_error"
	errorTypeWorkflowFailed = "workflow_failed"
)

// HelpSystem provides contextual help and guidance
type HelpSystem struct {
	topics   map[string]*HelpTopic
	commands map[string]*CommandHelp
}

// HelpTopic represents a help topic with content and examples
type HelpTopic struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Content     string        `json:"content"`
	Examples    []HelpExample `json:"examples"`
	SeeAlso     []string      `json:"see_also"`
	Tags        []string      `json:"tags"`
	Category    string        `json:"category"`
	Difficulty  string        `json:"difficulty"`
}

// HelpExample represents an example with description
type HelpExample struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command"`
	Output      string `json:"output,omitempty"`
}

// CommandHelp represents help for a specific command
type CommandHelp struct {
	Name        string                  `json:"name"`
	Usage       string                  `json:"usage"`
	Description string                  `json:"description"`
	Options     []CommandOption         `json:"options"`
	Examples    []HelpExample           `json:"examples"`
	SeeAlso     []string                `json:"see_also"`
	Subcommands map[string]*CommandHelp `json:"subcommands,omitempty"`
}

// CommandOption represents a command-line option
type CommandOption struct {
	Name        string `json:"name"`
	Short       string `json:"short,omitempty"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
}

// NewHelpSystem creates a new help system
func NewHelpSystem() *HelpSystem {
	hs := &HelpSystem{
		topics:   make(map[string]*HelpTopic),
		commands: make(map[string]*CommandHelp),
	}

	hs.initializeTopics()
	hs.initializeCommands()

	return hs
}

// GetTopic returns help for a specific topic
func (hs *HelpSystem) GetTopic(topic string) (*HelpTopic, bool) {
	t, exists := hs.topics[strings.ToLower(topic)]
	return t, exists
}

// GetCommand returns help for a specific command
func (hs *HelpSystem) GetCommand(command string) (*CommandHelp, bool) {
	c, exists := hs.commands[strings.ToLower(command)]
	return c, exists
}

// ListTopics returns all available help topics
func (hs *HelpSystem) ListTopics() []string {
	topics := make([]string, 0, len(hs.topics))
	for topic := range hs.topics {
		topics = append(topics, topic)
	}
	sort.Strings(topics)
	return topics
}

// ListCommands returns all available commands
func (hs *HelpSystem) ListCommands() []string {
	commands := make([]string, 0, len(hs.commands))
	for command := range hs.commands {
		commands = append(commands, command)
	}
	sort.Strings(commands)
	return commands
}

// SearchTopics searches for topics by keyword
func (hs *HelpSystem) SearchTopics(keyword string) []*HelpTopic {
	keyword = strings.ToLower(keyword)
	var results []*HelpTopic

	for _, topic := range hs.topics {
		if hs.topicMatches(topic, keyword) {
			results = append(results, topic)
		}
	}

	return results
}

// topicMatches checks if a topic matches a keyword
func (hs *HelpSystem) topicMatches(topic *HelpTopic, keyword string) bool {
	// Convert keyword to lowercase for case-insensitive matching
	keyword = strings.ToLower(keyword)

	// Check title, description, and tags
	if strings.Contains(strings.ToLower(topic.Title), keyword) ||
		strings.Contains(strings.ToLower(topic.Description), keyword) ||
		strings.Contains(strings.ToLower(topic.Content), keyword) {
		return true
	}

	// Check tags
	for _, tag := range topic.Tags {
		if strings.Contains(strings.ToLower(tag), keyword) {
			return true
		}
	}

	return false
}

// GetQuickStart returns quick start guide
func (hs *HelpSystem) GetQuickStart() string {
	return `# ccAgents Quick Start

## 1. Initial Setup
   ccagents init                    # Initialize configuration
   ccagents validate               # Validate setup

## 2. Authentication
   gh auth login                   # GitHub authentication
   claude auth                     # Claude authentication

## 3. Process an Issue
   ccagents process "#123"         # Process issue by number
   ccagents process "https://..."  # Process issue by URL

## 4. Monitor Progress
   ccagents status                 # Check current status
   ccagents logs --follow          # Monitor in real-time

## 5. Get Help
   ccagents help                   # General help
   ccagents help <topic>           # Topic-specific help
   ccagents <command> --help       # Command-specific help

For detailed guides, see: https://docs.ccagents.dev/getting-started`
}

// GetContextualHelp provides help based on current context
func (hs *HelpSystem) GetContextualHelp(context string, command string, args []string) string {
	switch context {
	case "authentication_failed":
		return hs.getAuthenticationHelp()
	case errorTypeConfiguration:
		return hs.getConfigurationHelp()
	case "github_error":
		return hs.getGitHubHelp()
	case "claude_error":
		return hs.getClaudeHelp()
	case errorTypeWorkflowFailed:
		return hs.getWorkflowHelp()
	default:
		if cmd, exists := hs.GetCommand(command); exists {
			return hs.formatCommandHelp(cmd)
		}
		return hs.GetQuickStart()
	}
}

// initializeTopics initializes built-in help topics
func (hs *HelpSystem) initializeTopics() {
	// Configuration topic
	hs.topics["configuration"] = &HelpTopic{
		Title:       "Configuration",
		Description: "How to configure ccAgents for your project",
		Category:    "setup",
		Difficulty:  "beginner",
		Content: `ccAgents uses YAML configuration files to customize behavior.

Configuration files are loaded in this order:
1. System defaults
2. Global config (~/.ccagents.yaml)
3. Project config (.ccagents.yaml)
4. Environment variables
5. Command-line flags`,
		Examples: []HelpExample{
			{
				Title:       "Initialize configuration",
				Description: "Create a default configuration file",
				Command:     "ccagents init",
			},
			{
				Title:       "Validate configuration",
				Description: "Check if configuration is valid",
				Command:     "ccagents validate",
			},
			{
				Title:       "View current configuration",
				Description: "Display the current configuration",
				Command:     "ccagents config dump",
			},
		},
		SeeAlso: []string{"authentication", "workflows", "github"},
		Tags:    []string{"config", "setup", "yaml", "initialization"},
	}

	// Authentication topic
	hs.topics["authentication"] = &HelpTopic{
		Title:       "Authentication",
		Description: "Setting up authentication with GitHub and Claude",
		Category:    "setup",
		Difficulty:  "beginner",
		Content: `ccAgents requires authentication with both GitHub and Claude Code.

GitHub Authentication:
- Uses GitHub CLI (gh) for authentication
- Supports personal access tokens and GitHub Apps
- Requires repo and workflow permissions

Claude Authentication:
- Uses Claude Code CLI for authentication
- Requires valid Anthropic API key
- Supports both personal and team accounts`,
		Examples: []HelpExample{
			{
				Title:       "GitHub authentication",
				Description: "Authenticate with GitHub",
				Command:     "gh auth login",
			},
			{
				Title:       "Claude authentication",
				Description: "Authenticate with Claude Code",
				Command:     "claude auth",
			},
			{
				Title:       "Check authentication status",
				Description: "Verify authentication is working",
				Command:     "ccagents auth status",
			},
		},
		SeeAlso: []string{"configuration", "github", "claude"},
		Tags:    []string{"auth", "github", "claude", "setup", "tokens"},
	}

	// Workflows topic
	hs.topics["workflows"] = &HelpTopic{
		Title:       "Workflows",
		Description: "Understanding and customizing ccAgents workflows",
		Category:    "automation",
		Difficulty:  "intermediate",
		Content: `ccAgents workflows automate the process from issue to merged PR.

Standard workflow steps:
1. Issue Analysis - Parse requirements
2. Code Context - Understand codebase
3. Implementation - Generate code with AI
4. Quality Assurance - Run tests and scans
5. Review - Create PR and handle feedback
6. Merge - Complete the automation

Workflows are highly customizable through configuration.`,
		Examples: []HelpExample{
			{
				Title:       "Process an issue",
				Description: "Start a workflow for a GitHub issue",
				Command:     "ccagents process \"#123\"",
			},
			{
				Title:       "Monitor workflow",
				Description: "Check workflow status",
				Command:     "ccagents status",
			},
			{
				Title:       "List workflows",
				Description: "Show all workflows",
				Command:     "ccagents workflow list",
			},
		},
		SeeAlso: []string{"configuration", "issues", "pull-requests"},
		Tags:    []string{"workflow", "automation", "process", "ci", "cd"},
	}

	// Troubleshooting topic
	hs.topics["troubleshooting"] = &HelpTopic{
		Title:       "Troubleshooting",
		Description: "Common issues and solutions",
		Category:    "support",
		Difficulty:  "beginner",
		Content: `Common troubleshooting steps:

1. Check system status: ccagents status --verbose
2. Validate configuration: ccagents validate
3. Check authentication: ccagents auth status
4. Review logs: ccagents logs --level debug
5. Test connectivity: ccagents test --all

Most issues are related to:
- Authentication problems
- Configuration errors
- Network connectivity
- Missing dependencies`,
		Examples: []HelpExample{
			{
				Title:       "Debug mode",
				Description: "Run with detailed logging",
				Command:     "CCAGENTS_LOG_LEVEL=debug ccagents process \"#123\"",
			},
			{
				Title:       "Check connectivity",
				Description: "Test GitHub and Claude connectivity",
				Command:     "ccagents test --all",
			},
			{
				Title:       "Validate setup",
				Description: "Check if everything is configured correctly",
				Command:     "ccagents validate --verbose",
			},
		},
		SeeAlso: []string{"authentication", "configuration", "logs"},
		Tags:    []string{"troubleshooting", "debug", "issues", "problems", "errors"},
	}
}

// initializeCommands initializes command help
func (hs *HelpSystem) initializeCommands() {
	// Process command
	hs.commands["process"] = &CommandHelp{
		Name:        "process",
		Usage:       "ccagents process <issue>",
		Description: "Process a GitHub issue and create an automated implementation",
		Options: []CommandOption{
			{
				Name:        "config",
				Short:       "c",
				Description: "Path to configuration file",
				Type:        "string",
				Default:     ".ccagents.yaml",
			},
			{
				Name:        "dry-run",
				Description: "Analyze only, don't make changes",
				Type:        "bool",
			},
			{
				Name:        "verbose",
				Short:       "v",
				Description: "Enable verbose output",
				Type:        "bool",
			},
		},
		Examples: []HelpExample{
			{
				Title:       "Process by URL",
				Description: "Process an issue using its GitHub URL",
				Command:     "ccagents process \"https://github.com/owner/repo/issues/123\"",
			},
			{
				Title:       "Process by number",
				Description: "Process an issue by number (requires repo config)",
				Command:     "ccagents process \"#123\"",
			},
			{
				Title:       "Dry run",
				Description: "Analyze issue without making changes",
				Command:     "ccagents process \"#123\" --dry-run",
			},
		},
		SeeAlso: []string{"status", "workflow", "validate"},
	}

	// Status command
	hs.commands["status"] = &CommandHelp{
		Name:        "status",
		Usage:       "ccagents status [options]",
		Description: "Show current status of ccAgents workflows",
		Options: []CommandOption{
			{
				Name:        "follow",
				Short:       "f",
				Description: "Follow status updates in real-time",
				Type:        "bool",
			},
			{
				Name:        "verbose",
				Short:       "v",
				Description: "Show detailed status information",
				Type:        "bool",
			},
			{
				Name:        "format",
				Description: "Output format (text, json, yaml)",
				Type:        "string",
				Default:     "text",
			},
		},
		Examples: []HelpExample{
			{
				Title:       "Check status",
				Description: "Show current workflow status",
				Command:     "ccagents status",
			},
			{
				Title:       "Follow status",
				Description: "Monitor status changes in real-time",
				Command:     "ccagents status --follow",
			},
			{
				Title:       "Detailed status",
				Description: "Show verbose status information",
				Command:     "ccagents status --verbose",
			},
		},
		SeeAlso: []string{"process", "workflow", "logs"},
	}

	// Config command
	hs.commands["config"] = &CommandHelp{
		Name:        "config",
		Usage:       "ccagents config <subcommand>",
		Description: "Manage ccAgents configuration",
		Subcommands: map[string]*CommandHelp{
			"get": {
				Name:        "get",
				Usage:       "ccagents config get <key>",
				Description: "Get a configuration value",
				Examples: []HelpExample{
					{
						Title:   "Get GitHub timeout",
						Command: "ccagents config get github.timeout",
					},
				},
			},
			"set": {
				Name:        "set",
				Usage:       "ccagents config set <key> <value>",
				Description: "Set a configuration value",
				Examples: []HelpExample{
					{
						Title:   "Set Claude model",
						Command: "ccagents config set claude.model claude-3-sonnet-20240229",
					},
				},
			},
			"dump": {
				Name:        "dump",
				Usage:       "ccagents config dump",
				Description: "Show current configuration",
			},
		},
		SeeAlso: []string{"init", "validate"},
	}
}

// getAuthenticationHelp returns authentication-specific help
func (hs *HelpSystem) getAuthenticationHelp() string {
	return `# Authentication Help

It looks like you're having authentication issues. Here's how to fix them:

## GitHub Authentication
1. Check status: gh auth status
2. Re-authenticate: gh auth login --web
3. Verify permissions: gh api user

## Claude Authentication
1. Check status: claude auth status
2. Re-authenticate: claude auth
3. Test connection: claude models list

## Common Issues
- Expired tokens: Re-authenticate with both services
- Missing permissions: Ensure repo and workflow scopes
- Rate limits: Wait or upgrade to higher limits

## Getting Help
- Run: ccagents validate
- Check: ccagents auth status
- Docs: https://docs.ccagents.dev/authentication`
}

// getConfigurationHelp returns configuration-specific help
func (hs *HelpSystem) getConfigurationHelp() string {
	return `# Configuration Help

Configuration issues detected. Here's how to resolve them:

## Validate Configuration
ccagents validate --verbose

## Common Fixes
1. Check YAML syntax: Use proper indentation and quotes
2. Verify required fields: github.owner, github.repo
3. Check data types: Use strings for timeouts ("30s"), booleans for flags

## Reset Configuration
ccagents init --force

## Get Help
- View schema: ccagents config schema
- See examples: https://docs.ccagents.dev/examples
- Troubleshooting: https://docs.ccagents.dev/troubleshooting`
}

// getGitHubHelp returns GitHub-specific help
func (hs *HelpSystem) getGitHubHelp() string {
	return `# GitHub Help

GitHub-related issues detected. Here are solutions:

## Check GitHub Access
gh repo view owner/repo

## Common Issues
- Repository not found: Check owner/repo in configuration
- Permission denied: Ensure token has repo access
- Rate limiting: Wait or use GitHub App authentication
- Branch protection: Update workflow configuration

## Test Connection
ccagents test github

## Documentation
https://docs.ccagents.dev/github-integration`
}

// getClaudeHelp returns Claude-specific help
func (hs *HelpSystem) getClaudeHelp() string {
	return `# Claude Help

Claude Code issues detected. Here are solutions:

## Check Claude Status
claude auth status

## Common Issues
- Invalid API key: Re-authenticate with claude auth
- Rate limiting: Reduce request frequency or upgrade plan
- Model not available: Check available models with claude models list
- Timeout: Increase timeout in configuration

## Test Connection
ccagents test claude

## Documentation
https://docs.ccagents.dev/claude-integration`
}

// getWorkflowHelp returns workflow-specific help
func (hs *HelpSystem) getWorkflowHelp() string {
	return `# Workflow Help

Workflow execution issues detected. Here are solutions:

## Check Workflow Status
ccagents workflow status

## Common Issues
- Test failures: Check test command and environment
- Security scan failures: Review security tool configuration
- Merge conflicts: Resolve conflicts manually
- Review requirements: Ensure proper review configuration

## Debug Workflow
ccagents logs --level debug --filter workflow

## Retry Workflow
ccagents workflow retry <workflow-id>

## Documentation
https://docs.ccagents.dev/workflows`
}

// formatCommandHelp formats command help for display
func (hs *HelpSystem) formatCommandHelp(cmd *CommandHelp) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", cmd.Name))
	sb.WriteString(fmt.Sprintf("**Usage:** %s\n\n", cmd.Usage))
	sb.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))

	if len(cmd.Options) > 0 {
		sb.WriteString("## Options\n\n")
		for _, opt := range cmd.Options {
			sb.WriteString(fmt.Sprintf("- **--%s", opt.Name))
			if opt.Short != "" {
				sb.WriteString(fmt.Sprintf(", -%s", opt.Short))
			}
			sb.WriteString(fmt.Sprintf("**: %s", opt.Description))
			if opt.Default != "" {
				sb.WriteString(fmt.Sprintf(" (default: %s)", opt.Default))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(cmd.Examples) > 0 {
		sb.WriteString("## Examples\n\n")
		for _, ex := range cmd.Examples {
			sb.WriteString(fmt.Sprintf("**%s:**\n", ex.Title))
			if ex.Description != "" {
				sb.WriteString(fmt.Sprintf("%s\n", ex.Description))
			}
			sb.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", ex.Command))
		}
	}

	if len(cmd.SeeAlso) > 0 {
		sb.WriteString("## See Also\n\n")
		for _, ref := range cmd.SeeAlso {
			sb.WriteString(fmt.Sprintf("- %s\n", ref))
		}
	}

	return sb.String()
}

// FormatHelpTopic formats a help topic for display
func (hs *HelpSystem) FormatHelpTopic(topic *HelpTopic) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", topic.Title))
	sb.WriteString(fmt.Sprintf("%s\n\n", topic.Description))

	if topic.Content != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", topic.Content))
	}

	if len(topic.Examples) > 0 {
		sb.WriteString("## Examples\n\n")
		for _, ex := range topic.Examples {
			sb.WriteString(fmt.Sprintf("### %s\n", ex.Title))
			if ex.Description != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", ex.Description))
			}
			sb.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", ex.Command))
			if ex.Output != "" {
				sb.WriteString(fmt.Sprintf("Expected output:\n```\n%s\n```\n\n", ex.Output))
			}
		}
	}

	if len(topic.SeeAlso) > 0 {
		sb.WriteString("## See Also\n\n")
		for _, ref := range topic.SeeAlso {
			sb.WriteString(fmt.Sprintf("- %s\n", ref))
		}
	}

	if len(topic.Tags) > 0 {
		sb.WriteString("\n---\n")
		sb.WriteString(fmt.Sprintf("**Category:** %s | **Difficulty:** %s | **Tags:** %s\n",
			topic.Category, topic.Difficulty, strings.Join(topic.Tags, ", ")))
	}

	return sb.String()
}

// GetTroubleshootingHelp returns help for specific error types
func (hs *HelpSystem) GetTroubleshootingHelp(errorType string, errorMessage string) string {
	switch errorType {
	case "network_timeout":
		return `# Network Timeout Help

**Issue:** Connection timed out while communicating with external services.

**Quick Fixes:**
1. Check internet connection: ping github.com
2. Increase timeout: ccagents config set github.timeout 60s
3. Retry operation: ccagents retry --last

**Advanced Solutions:**
- Configure proxy if behind corporate firewall
- Check DNS resolution: nslookup api.github.com
- Test connectivity: ccagents test --connectivity`

	case "authentication_error":
		return `# Authentication Error Help

**Issue:** Failed to authenticate with GitHub or Claude services.

**GitHub Authentication:**
1. Check status: gh auth status
2. Re-authenticate: gh auth login --web
3. Verify permissions: gh api user

**Claude Authentication:**
1. Check status: claude auth status
2. Re-authenticate: claude auth
3. Verify API key: claude config get api_key`

	case errorTypeConfiguration:
		return `# Configuration Error Help

**Issue:** Invalid or missing configuration detected.

**Solutions:**
1. Validate config: ccagents validate --verbose
2. Check syntax: ccagents config check
3. Reset to defaults: ccagents init --force
4. View schema: ccagents config schema`

	case errorTypeWorkflowFailed:
		return `# Workflow Failure Help

**Issue:** Workflow execution encountered an error.

**Recovery Steps:**
1. Check status: ccagents workflow status
2. View logs: ccagents logs --filter workflow
3. Retry workflow: ccagents workflow retry <id>
4. Reset if needed: ccagents workflow reset <id>`

	default:
		return fmt.Sprintf(`# General Troubleshooting

**Error:** %s

**General Steps:**
1. Check system status: ccagents status --verbose
2. Validate configuration: ccagents validate
3. Check logs: ccagents logs --tail 20
4. Get help: ccagents help troubleshooting

**Need more help?**
- Documentation: https://docs.ccagents.dev
- Issues: https://github.com/fumiya-kume/cca/issues
- Community: https://discord.gg/ccagents`, errorMessage)
	}
}

// GetSuggestedCommands returns suggested commands based on context
func (hs *HelpSystem) GetSuggestedCommands(context string) []string {
	switch context {
	case "new_user":
		return []string{
			"ccagents init",
			"ccagents validate",
			"gh auth login",
			"claude auth",
			"ccagents help getting-started",
		}

	case errorTypeWorkflowFailed:
		return []string{
			"ccagents workflow status",
			"ccagents logs --filter workflow",
			"ccagents workflow retry",
			"ccagents help troubleshooting",
		}

	case "authentication_failed":
		return []string{
			"gh auth status",
			"gh auth login --web",
			"claude auth status",
			"claude auth",
			"ccagents auth status",
		}

	case errorTypeConfiguration:
		return []string{
			"ccagents validate --verbose",
			"ccagents config dump",
			"ccagents config schema",
			"ccagents init --force",
		}

	default:
		return []string{
			"ccagents help",
			"ccagents status",
			"ccagents validate",
		}
	}
}

// SearchCommands searches for commands matching a query
func (hs *HelpSystem) SearchCommands(query string) []*CommandHelp {
	query = strings.ToLower(query)
	var results []*CommandHelp

	for _, cmd := range hs.commands {
		if hs.commandMatches(cmd, query) {
			results = append(results, cmd)
		}
	}

	return results
}

// commandMatches checks if a command matches a search query
func (hs *HelpSystem) commandMatches(cmd *CommandHelp, query string) bool {
	// Convert query to lowercase for case-insensitive matching
	query = strings.ToLower(query)

	if strings.Contains(strings.ToLower(cmd.Name), query) ||
		strings.Contains(strings.ToLower(cmd.Description), query) ||
		strings.Contains(strings.ToLower(cmd.Usage), query) {
		return true
	}

	// Check options
	for _, opt := range cmd.Options {
		if strings.Contains(strings.ToLower(opt.Name), query) ||
			strings.Contains(strings.ToLower(opt.Description), query) {
			return true
		}
	}

	return false
}

// GetCommandByName gets command help by exact name match
func (hs *HelpSystem) GetCommandByName(name string) *CommandHelp {
	cmd, exists := hs.commands[strings.ToLower(name)]
	if !exists {
		return nil
	}
	return cmd
}

// AddCustomTopic adds a custom help topic
func (hs *HelpSystem) AddCustomTopic(key string, topic *HelpTopic) {
	hs.topics[strings.ToLower(key)] = topic
}

// AddCustomCommand adds a custom command help
func (hs *HelpSystem) AddCustomCommand(key string, cmd *CommandHelp) {
	hs.commands[strings.ToLower(key)] = cmd
}
