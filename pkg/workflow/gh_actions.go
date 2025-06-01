package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
)

// GitHub CLI Action Types - extends existing ActionType system
const (
	ActionTypeGitHubCLI ActionType = iota + 100 // Start after existing action types
	ActionTypeGitCommand
	ActionTypeClaudeAnalysis
	ActionTypeCodeGeneration
)

// GitHubCLIAction represents a GitHub CLI command action
type GitHubCLIAction struct {
	Command    string            `json:"command"`    // gh subcommand (e.g., "issue", "pr", "repo")
	Args       []string          `json:"args"`       // command arguments
	Flags      map[string]string `json:"flags"`      // command flags
	WorkingDir string            `json:"working_dir"` // working directory
	OutputType string            `json:"output_type"` // "json", "text", "raw"
}

// GitCommandAction represents a git command action
type GitCommandAction struct {
	Command    string            `json:"command"`    // git subcommand
	Args       []string          `json:"args"`       // command arguments
	WorkingDir string            `json:"working_dir"` // working directory
}

// GitHubCLIExecutor handles GitHub CLI command execution
type GitHubCLIExecutor struct {
	timeout time.Duration
}

// NewGitHubCLIExecutor creates a new GitHub CLI executor
func NewGitHubCLIExecutor(timeout time.Duration) *GitHubCLIExecutor {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &GitHubCLIExecutor{timeout: timeout}
}

// ExecuteGitHubCLI executes a GitHub CLI command
func (e *GitHubCLIExecutor) ExecuteGitHubCLI(ctx context.Context, action *GitHubCLIAction) (*GitHubCLIResult, error) {
	// Build command
	args := []string{action.Command}
	args = append(args, action.Args...)
	
	// Add flags
	for flag, value := range action.Flags {
		if value == "" {
			args = append(args, "--"+flag)
		} else {
			args = append(args, "--"+flag, value)
		}
	}

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	// #nosec G204 - args are validated GitHub CLI commands from internal actions
	cmd := exec.CommandContext(cmdCtx, "gh", args...)
	
	if action.WorkingDir != "" {
		cmd.Dir = action.WorkingDir
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, e.parseGitHubCLIError(string(output), err)
	}

	result := &GitHubCLIResult{
		Command:    fmt.Sprintf("gh %s", strings.Join(args, " ")),
		Output:     string(output),
		OutputType: action.OutputType,
		Success:    true,
	}

	// Parse JSON output if requested
	if action.OutputType == "json" && len(output) > 0 {
		var jsonData interface{}
		if err := json.Unmarshal(output, &jsonData); err == nil {
			result.JSONData = jsonData
		}
	}

	return result, nil
}

// ExecuteGitCommand executes a git command
func (e *GitHubCLIExecutor) ExecuteGitCommand(ctx context.Context, action *GitCommandAction) (*GitCommandResult, error) {
	// Build command
	args := append([]string{action.Command}, action.Args...)
	
	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	// #nosec G204 - args are validated git commands from internal actions
	cmd := exec.CommandContext(cmdCtx, "git", args...)
	
	if action.WorkingDir != "" {
		cmd.Dir = action.WorkingDir
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	
	result := &GitCommandResult{
		Command:    fmt.Sprintf("git %s", strings.Join(args, " ")),
		Output:     string(output),
		Success:    err == nil,
		WorkingDir: action.WorkingDir,
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result, err
}

// FetchIssueData fetches GitHub issue data using gh CLI
func (e *GitHubCLIExecutor) FetchIssueData(ctx context.Context, issueRef *types.IssueReference) (*types.IssueData, error) {
	action := &GitHubCLIAction{
		Command: "issue",
		Args:    []string{"view", fmt.Sprintf("#%d", issueRef.Number)},
		Flags: map[string]string{
			"repo":   fmt.Sprintf("%s/%s", issueRef.Owner, issueRef.Repo),
			"json":   "title,body,labels,assignees,milestone,createdAt,updatedAt,state",
		},
		OutputType: "json",
	}

	result, err := e.ExecuteGitHubCLI(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue data: %w", err)
	}

	// Parse issue data from JSON
	return e.parseIssueData(result.JSONData, issueRef)
}

// CloneRepository clones a repository using gh CLI
func (e *GitHubCLIExecutor) CloneRepository(ctx context.Context, issueRef *types.IssueReference, workDir string) (string, error) {
	repoPath := filepath.Join(workDir, fmt.Sprintf("%s-%s", issueRef.Owner, issueRef.Repo))
	
	action := &GitHubCLIAction{
		Command: "repo",
		Args:    []string{"clone", fmt.Sprintf("%s/%s", issueRef.Owner, issueRef.Repo), repoPath},
		Flags:   map[string]string{},
	}

	_, err := e.ExecuteGitHubCLI(ctx, action)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return repoPath, nil
}

// CreateBranch creates a new git branch
func (e *GitHubCLIExecutor) CreateBranch(ctx context.Context, repoPath, branchName string) error {
	action := &GitCommandAction{
		Command:    "checkout",
		Args:       []string{"-b", branchName},
		WorkingDir: repoPath,
	}

	_, err := e.ExecuteGitCommand(ctx, action)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	return nil
}

// CommitChanges commits changes to the repository
func (e *GitHubCLIExecutor) CommitChanges(ctx context.Context, repoPath, message string) error {
	// Add all changes
	addAction := &GitCommandAction{
		Command:    "add",
		Args:       []string{"."},
		WorkingDir: repoPath,
	}

	if _, err := e.ExecuteGitCommand(ctx, addAction); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit changes
	commitAction := &GitCommandAction{
		Command:    "commit",
		Args:       []string{"-m", message},
		WorkingDir: repoPath,
	}

	if _, err := e.ExecuteGitCommand(ctx, commitAction); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// PushBranch pushes the branch to origin
func (e *GitHubCLIExecutor) PushBranch(ctx context.Context, repoPath, branchName string) error {
	action := &GitCommandAction{
		Command:    "push",
		Args:       []string{"--set-upstream", "origin", branchName},
		WorkingDir: repoPath,
	}

	_, err := e.ExecuteGitCommand(ctx, action)
	if err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	return nil
}

// CreatePullRequest creates a pull request using gh CLI
func (e *GitHubCLIExecutor) CreatePullRequest(ctx context.Context, repoPath string, prRequest *PullRequestRequest) (*PullRequestResult, error) {
	action := &GitHubCLIAction{
		Command:    "pr",
		Args:       []string{"create"},
		Flags:      make(map[string]string),
		WorkingDir: repoPath,
		OutputType: "text",
	}

	// Set PR details
	action.Flags["title"] = prRequest.Title
	action.Flags["body"] = prRequest.Body
	
	if prRequest.Draft {
		action.Flags["draft"] = ""
	}
	
	if prRequest.Base != "" {
		action.Flags["base"] = prRequest.Base
	}

	if len(prRequest.Labels) > 0 {
		action.Flags["label"] = strings.Join(prRequest.Labels, ",")
	}

	if len(prRequest.Assignees) > 0 {
		action.Flags["assignee"] = strings.Join(prRequest.Assignees, ",")
	}

	result, err := e.ExecuteGitHubCLI(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	// Extract PR URL from output
	prURL := strings.TrimSpace(result.Output)
	
	return &PullRequestResult{
		URL:     prURL,
		Success: true,
		Command: result.Command,
	}, nil
}

// CheckAuthentication verifies GitHub CLI authentication
func (e *GitHubCLIExecutor) CheckAuthentication(ctx context.Context) error {
	action := &GitHubCLIAction{
		Command:    "auth",
		Args:       []string{"status"},
		Flags:      map[string]string{},
		OutputType: "text",
	}

	_, err := e.ExecuteGitHubCLI(ctx, action)
	if err != nil {
		return fmt.Errorf("GitHub authentication required: run 'gh auth login' to authenticate")
	}

	return nil
}

// parseGitHubCLIError parses GitHub CLI error messages for better user feedback
func (e *GitHubCLIExecutor) parseGitHubCLIError(output string, err error) error {
	output = strings.ToLower(output)
	
	if strings.Contains(output, "not authenticated") {
		return fmt.Errorf("GitHub authentication required: run 'gh auth login'")
	}
	
	if strings.Contains(output, "not found") {
		return fmt.Errorf("repository or issue not found: %s", output)
	}
	
	if strings.Contains(output, "permission denied") || strings.Contains(output, "forbidden") {
		return fmt.Errorf("permission denied: insufficient access to repository")
	}
	
	if strings.Contains(output, "rate limit") {
		return fmt.Errorf("GitHub API rate limit exceeded: try again later")
	}

	return fmt.Errorf("GitHub CLI error: %s", output)
}

// parseIssueData parses GitHub issue JSON data into IssueData struct
func (e *GitHubCLIExecutor) parseIssueData(jsonData interface{}, issueRef *types.IssueReference) (*types.IssueData, error) {
	dataMap, ok := jsonData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid issue data format")
	}

	issueData := e.createBaseIssueData(issueRef)

	e.parseBasicFields(dataMap, issueData)
	e.parseLabels(dataMap, issueData)
	e.parseAssignees(dataMap, issueData)
	e.parseMilestone(dataMap, issueData)
	e.parseTimestamps(dataMap, issueData)

	return issueData, nil
}

// createBaseIssueData creates the base issue data structure
func (e *GitHubCLIExecutor) createBaseIssueData(issueRef *types.IssueReference) *types.IssueData {
	return &types.IssueData{
		Number:     issueRef.Number,
		Repository: fmt.Sprintf("%s/%s", issueRef.Owner, issueRef.Repo),
	}
}

// parseBasicFields parses basic string fields from the data map
func (e *GitHubCLIExecutor) parseBasicFields(dataMap map[string]interface{}, issueData *types.IssueData) {
	if title, ok := dataMap["title"].(string); ok {
		issueData.Title = title
	}

	if body, ok := dataMap["body"].(string); ok {
		issueData.Body = body
		issueData.Description = body // Alias
	}

	if state, ok := dataMap["state"].(string); ok {
		issueData.State = state
	}
}

// parseLabels parses labels from the data map
func (e *GitHubCLIExecutor) parseLabels(dataMap map[string]interface{}, issueData *types.IssueData) {
	labelsData, ok := dataMap["labels"].([]interface{})
	if !ok {
		return
	}

	for _, labelData := range labelsData {
		if labelName := e.extractLabelName(labelData); labelName != "" {
			issueData.Labels = append(issueData.Labels, labelName)
		}
	}
}

// extractLabelName extracts the label name from label data
func (e *GitHubCLIExecutor) extractLabelName(labelData interface{}) string {
	labelMap, ok := labelData.(map[string]interface{})
	if !ok {
		return ""
	}

	name, ok := labelMap["name"].(string)
	if !ok {
		return ""
	}

	return name
}

// parseAssignees parses assignees from the data map
func (e *GitHubCLIExecutor) parseAssignees(dataMap map[string]interface{}, issueData *types.IssueData) {
	assigneesData, ok := dataMap["assignees"].([]interface{})
	if !ok {
		return
	}

	for _, assigneeData := range assigneesData {
		if login := e.extractAssigneeLogin(assigneeData); login != "" {
			issueData.Assignees = append(issueData.Assignees, login)
		}
	}
}

// extractAssigneeLogin extracts the login from assignee data
func (e *GitHubCLIExecutor) extractAssigneeLogin(assigneeData interface{}) string {
	assigneeMap, ok := assigneeData.(map[string]interface{})
	if !ok {
		return ""
	}

	login, ok := assigneeMap["login"].(string)
	if !ok {
		return ""
	}

	return login
}

// parseMilestone parses milestone from the data map
func (e *GitHubCLIExecutor) parseMilestone(dataMap map[string]interface{}, issueData *types.IssueData) {
	milestoneData, ok := dataMap["milestone"].(map[string]interface{})
	if !ok {
		return
	}

	if title, ok := milestoneData["title"].(string); ok {
		issueData.Milestone = title
	}
}

// parseTimestamps parses timestamp fields from the data map
func (e *GitHubCLIExecutor) parseTimestamps(dataMap map[string]interface{}, issueData *types.IssueData) {
	if createdAt, ok := dataMap["createdAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			issueData.CreatedAt = t
		}
	}

	if updatedAt, ok := dataMap["updatedAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			issueData.UpdatedAt = t
		}
	}
}

// GitHubCLIResult represents the result of a GitHub CLI operation
type GitHubCLIResult struct {
	Command    string      `json:"command"`
	Output     string      `json:"output"`
	OutputType string      `json:"output_type"`
	JSONData   interface{} `json:"json_data,omitempty"`
	Success    bool        `json:"success"`
}

type GitCommandResult struct {
	Command    string `json:"command"`
	Output     string `json:"output"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	WorkingDir string `json:"working_dir"`
}

type PullRequestRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Base      string   `json:"base,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Draft     bool     `json:"draft,omitempty"`
}

type PullRequestResult struct {
	URL     string `json:"url"`
	Success bool   `json:"success"`
	Command string `json:"command"`
}