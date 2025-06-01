package security

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
)

// SafeExecutor provides secure command execution with validation
type SafeExecutor struct {
	sanitizer   *InputSanitizer
	workingDir  string
	timeout     time.Duration
	environment map[string]string
}

// NewSafeExecutor creates a new safe command executor
func NewSafeExecutor(workingDir string) *SafeExecutor {
	return &SafeExecutor{
		sanitizer:   NewInputSanitizer(),
		workingDir:  workingDir,
		timeout:     30 * time.Second,
		environment: make(map[string]string),
	}
}

// WithTimeout sets the execution timeout
func (se *SafeExecutor) WithTimeout(timeout time.Duration) *SafeExecutor {
	se.timeout = timeout
	return se
}

// WithEnvironment adds environment variables for execution
func (se *SafeExecutor) WithEnvironment(key, value string) *SafeExecutor {
	if err := se.sanitizer.SanitizeEnvironmentVariable(key, value); err == nil {
		se.environment[key] = value
	}
	return se
}

// ExecuteCommand safely executes a command with validation
func (se *SafeExecutor) ExecuteCommand(ctx context.Context, command string, args ...string) (*ExecutionResult, error) {
	// Validate the command and arguments
	if err := se.sanitizer.SanitizeCommand(command, args); err != nil {
		return nil, err
	}

	// Validate and clean the working directory
	cleanWorkingDir, err := se.sanitizer.SanitizePath(se.workingDir)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("invalid working directory").
			WithCause(err).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("working_dir", se.workingDir).
			Build()
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, se.timeout)
	defer cancel()

	// Create the command
	cmd := exec.CommandContext(execCtx, command, args...)
	cmd.Dir = cleanWorkingDir

	// Set up environment
	cmd.Env = append(cmd.Environ(), se.buildEnvironment()...)

	// Execute the command
	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &ExecutionResult{
		Command:    command,
		Args:       args,
		WorkingDir: cleanWorkingDir,
		Output:     string(output),
		Duration:   duration,
		ExitCode:   0,
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = -1
		}

		// Determine error type based on context
		var errorType errors.ErrorType
		switch {
		case execCtx.Err() == context.DeadlineExceeded:
			errorType = errors.ErrorTypeProcess
		case strings.Contains(err.Error(), "not found"):
			errorType = errors.ErrorTypeConfiguration
		default:
			errorType = errors.ErrorTypeProcess
		}

		return result, errors.NewError(errorType).
			WithMessage(fmt.Sprintf("command execution failed: %s", command)).
			WithCause(err).
			WithSeverity(errors.SeverityMedium).
			WithRecoverable(true).
			WithContext("command", command).
			WithContext("args", args).
			WithContext("exit_code", result.ExitCode).
			WithContext("output", result.Output).
			WithContext("duration", duration.String()).
			Build()
	}

	return result, nil
}

// ExecutionResult contains the results of a command execution
type ExecutionResult struct {
	Command    string
	Args       []string
	WorkingDir string
	Output     string
	Duration   time.Duration
	ExitCode   int
}

// IsSuccess returns true if the command executed successfully
func (er *ExecutionResult) IsSuccess() bool {
	return er.ExitCode == 0
}

// String returns a string representation of the execution result
func (er *ExecutionResult) String() string {
	status := "SUCCESS"
	if !er.IsSuccess() {
		status = fmt.Sprintf("FAILED (exit code: %d)", er.ExitCode)
	}

	return fmt.Sprintf("Command: %s %s\nStatus: %s\nDuration: %v\nOutput: %s",
		er.Command, strings.Join(er.Args, " "), status, er.Duration, er.Output)
}

// buildEnvironment creates the environment variable list
func (se *SafeExecutor) buildEnvironment() []string {
	var env []string
	for key, value := range se.environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

// GitExecutor provides safe git command execution
type GitExecutor struct {
	executor *SafeExecutor
}

// NewGitExecutor creates a new git command executor
func NewGitExecutor(repoPath string) *GitExecutor {
	return &GitExecutor{
		executor: NewSafeExecutor(repoPath),
	}
}

// Status executes git status
func (ge *GitExecutor) Status(ctx context.Context) (*ExecutionResult, error) {
	return ge.executor.ExecuteCommand(ctx, "git", "status", "--porcelain")
}

// Add executes git add with path validation
func (ge *GitExecutor) Add(ctx context.Context, paths ...string) (*ExecutionResult, error) {
	// Validate all paths
	for _, path := range paths {
		if _, err := ge.executor.sanitizer.SanitizePath(path); err != nil {
			return nil, errors.NewError(errors.ErrorTypeValidation).
				WithMessage("invalid path for git add").
				WithCause(err).
				WithSeverity(errors.SeverityMedium).
				WithRecoverable(true).
				WithContext("path", path).
				Build()
		}
	}

	args := append([]string{"add"}, paths...)
	return ge.executor.ExecuteCommand(ctx, "git", args...)
}

// Commit executes git commit with message validation
func (ge *GitExecutor) Commit(ctx context.Context, message string) (*ExecutionResult, error) {
	if message == "" {
		return nil, errors.ValidationError("commit message cannot be empty")
	}

	// Validate commit message for safety
	if len(message) > 2000 {
		return nil, errors.ValidationError("commit message too long")
	}

	return ge.executor.ExecuteCommand(ctx, "git", "commit", "-m", message)
}

// Push executes git push
func (ge *GitExecutor) Push(ctx context.Context, remote, branch string) (*ExecutionResult, error) {
	if remote == "" {
		remote = "origin"
	}
	if branch == "" {
		return nil, errors.ValidationError("branch name cannot be empty")
	}

	return ge.executor.ExecuteCommand(ctx, "git", "push", remote, branch)
}

// Checkout executes git checkout with branch validation
func (ge *GitExecutor) Checkout(ctx context.Context, branch string) (*ExecutionResult, error) {
	if branch == "" {
		return nil, errors.ValidationError("branch name cannot be empty")
	}

	// Validate branch name format
	if strings.Contains(branch, "..") || strings.Contains(branch, " ") {
		return nil, errors.ValidationError("invalid branch name format")
	}

	return ge.executor.ExecuteCommand(ctx, "git", "checkout", branch)
}

// CreateBranch executes git checkout -b with branch validation
func (ge *GitExecutor) CreateBranch(ctx context.Context, branch string) (*ExecutionResult, error) {
	if branch == "" {
		return nil, errors.ValidationError("branch name cannot be empty")
	}

	// Validate branch name format
	if strings.Contains(branch, "..") || strings.Contains(branch, " ") {
		return nil, errors.ValidationError("invalid branch name format")
	}

	return ge.executor.ExecuteCommand(ctx, "git", "checkout", "-b", branch)
}

// GitHubExecutor provides safe GitHub CLI execution
type GitHubExecutor struct {
	executor *SafeExecutor
}

// NewGitHubExecutor creates a new GitHub CLI executor
func NewGitHubExecutor(workingDir string) *GitHubExecutor {
	return &GitHubExecutor{
		executor: NewSafeExecutor(workingDir),
	}
}

// GetIssue retrieves issue information
func (ghe *GitHubExecutor) GetIssue(ctx context.Context, repo string, issueNumber int) (*ExecutionResult, error) {
	if repo == "" {
		return nil, errors.ValidationError("repository cannot be empty")
	}
	if issueNumber <= 0 {
		return nil, errors.ValidationError("issue number must be positive")
	}

	return ghe.executor.ExecuteCommand(ctx, "gh", "issue", "view",
		fmt.Sprintf("%d", issueNumber),
		"--repo", repo,
		"--json", "number,title,body,labels,assignees,milestone,state,comments")
}

// CreatePR creates a pull request
func (ghe *GitHubExecutor) CreatePR(ctx context.Context, title, body, base string) (*ExecutionResult, error) {
	if title == "" {
		return nil, errors.ValidationError("PR title cannot be empty")
	}
	if base == "" {
		base = "main"
	}

	return ghe.executor.ExecuteCommand(ctx, "gh", "pr", "create",
		"--title", title,
		"--body", body,
		"--base", base)
}

// CommandValidator provides additional command validation
type CommandValidator struct {
	sanitizer *InputSanitizer
}

// NewCommandValidator creates a new command validator
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		sanitizer: NewInputSanitizer(),
	}
}

// ValidateGitOperation validates git operations for safety
func (cv *CommandValidator) ValidateGitOperation(operation string, args []string) error {
	// Define safe git operations
	safeOperations := map[string]bool{
		"status":   true,
		"add":      true,
		"commit":   true,
		"push":     true,
		"pull":     true,
		"checkout": true,
		"branch":   true,
		"diff":     true,
		"log":      true,
		"show":     true,
		"clone":    true,
		"fetch":    true,
		"merge":    true,
		"rebase":   true,
		"stash":    true,
		"tag":      true,
	}

	if !safeOperations[operation] {
		return errors.NewError(errors.ErrorTypePermission).
			WithMessage(fmt.Sprintf("git operation '%s' not allowed", operation)).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("operation", operation).
			WithSuggestion("Use only approved git operations").
			Build()
	}

	// Validate arguments
	for _, arg := range args {
		if err := cv.sanitizer.validateArgument(arg); err != nil {
			return err
		}
	}

	return nil
}

// ValidateFileOperation validates file operations for safety
func (cv *CommandValidator) ValidateFileOperation(operation string, paths []string) error {
	// Define safe file operations
	safeOperations := map[string]bool{
		"read":   true,
		"write":  true,
		"create": true,
		"delete": true,
		"copy":   true,
		"move":   true,
	}

	if !safeOperations[operation] {
		return errors.NewError(errors.ErrorTypePermission).
			WithMessage(fmt.Sprintf("file operation '%s' not allowed", operation)).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("operation", operation).
			Build()
	}

	// Validate all paths
	for _, path := range paths {
		if _, err := cv.sanitizer.SanitizePath(path); err != nil {
			return err
		}

		// Additional checks for sensitive files
		if cv.isSensitiveFile(path) {
			return errors.NewError(errors.ErrorTypePermission).
				WithMessage("access to sensitive file denied").
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(false).
				WithContext("path", path).
				WithSuggestion("Avoid accessing system configuration files").
				Build()
		}
	}

	return nil
}

// isSensitiveFile checks if a file is sensitive and should be protected
func (cv *CommandValidator) isSensitiveFile(path string) bool {
	cleanPath := filepath.Clean(path)

	sensitiveFiles := []string{
		"/etc/passwd", "/etc/shadow", "/etc/sudoers",
		"/etc/ssh/", "/root/", "/.ssh/",
		"id_rsa", "id_ed25519", "*.key", "*.pem",
		".env", ".secrets", "config.json",
	}

	for _, sensitive := range sensitiveFiles {
		if strings.Contains(cleanPath, sensitive) {
			return true
		}
	}

	return false
}
