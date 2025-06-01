// Package errors provides structured error handling for ccAgents with categorization,
// severity levels, and contextual information for better error management and debugging.
package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents the category of error
type ErrorType int

const (
	// ErrorTypeUnknown represents an unknown error type
	ErrorTypeUnknown ErrorType = iota

	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation

	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork

	// ErrorTypeAuthentication represents authentication errors
	ErrorTypeAuthentication

	// ErrorTypePermission represents permission errors
	ErrorTypePermission

	// ErrorTypeProcess represents external process errors
	ErrorTypeProcess

	// ErrorTypeConfiguration represents configuration errors
	ErrorTypeConfiguration

	// ErrorTypeFileSystem represents file system errors
	ErrorTypeFileSystem

	// ErrorTypeGit represents git operation errors
	ErrorTypeGit

	// ErrorTypeGitHub represents GitHub API errors
	ErrorTypeGitHub

	// ErrorTypeClaude represents Claude Code integration errors
	ErrorTypeClaude

	// ErrorTypeWorkflow represents workflow execution errors
	ErrorTypeWorkflow

	// ErrorTypeSystem represents system-level errors
	ErrorTypeSystem
)

// String returns a string representation of the error type
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeValidation:
		return "validation"
	case ErrorTypeNetwork:
		return "network"
	case ErrorTypeAuthentication:
		return "authentication"
	case ErrorTypePermission:
		return "permission"
	case ErrorTypeProcess:
		return "process"
	case ErrorTypeConfiguration:
		return "configuration"
	case ErrorTypeFileSystem:
		return "filesystem"
	case ErrorTypeGit:
		return "git"
	case ErrorTypeGitHub:
		return "github"
	case ErrorTypeClaude:
		return "claude"
	case ErrorTypeWorkflow:
		return "workflow"
	case ErrorTypeSystem:
		return "system"
	default:
		return "unknown"
	}
}

// Severity represents the severity level of an error
type Severity int

const (
	// SeverityLow represents low severity errors (warnings)
	SeverityLow Severity = iota

	// SeverityMedium represents medium severity errors (recoverable)
	SeverityMedium

	// SeverityHigh represents high severity errors (critical)
	SeverityHigh
)

// String returns a string representation of the severity
func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	default:
		return "unknown"
	}
}

// ccAgentsError represents a structured error with additional context
type ccAgentsError struct {
	errorType   ErrorType
	severity    Severity
	message     string
	cause       error
	context     map[string]interface{}
	recoverable bool
	suggestions []string
}

// Error implements the error interface
func (e *ccAgentsError) Error() string {
	var parts []string

	// Add error type and severity
	parts = append(parts, fmt.Sprintf("[%s:%s]", e.errorType.String(), e.severity.String()))

	// Add message
	parts = append(parts, e.message)

	// Add cause if present
	if e.cause != nil {
		parts = append(parts, fmt.Sprintf("caused by: %s", e.cause.Error()))
	}

	return strings.Join(parts, " ")
}

// Type returns the error type
func (e *ccAgentsError) Type() ErrorType {
	return e.errorType
}

// Severity returns the error severity
func (e *ccAgentsError) Severity() Severity {
	return e.severity
}

// Cause returns the underlying cause of the error
func (e *ccAgentsError) Cause() error {
	return e.cause
}

// Context returns the error context
func (e *ccAgentsError) Context() map[string]interface{} {
	return e.context
}

// IsRecoverable returns whether the error is recoverable
func (e *ccAgentsError) IsRecoverable() bool {
	return e.recoverable
}

// Suggestions returns suggested actions to resolve the error
func (e *ccAgentsError) Suggestions() []string {
	return e.suggestions
}

// Unwrap returns the underlying error for compatibility with errors.Unwrap
func (e *ccAgentsError) Unwrap() error {
	return e.cause
}

// ErrorBuilder helps construct structured errors
type ErrorBuilder struct {
	errorType   ErrorType
	severity    Severity
	message     string
	cause       error
	context     map[string]interface{}
	recoverable bool
	suggestions []string
}

// NewError creates a new error builder
func NewError(errorType ErrorType) *ErrorBuilder {
	return &ErrorBuilder{
		errorType:   errorType,
		severity:    SeverityMedium,
		context:     make(map[string]interface{}),
		recoverable: false,
		suggestions: []string{},
	}
}

// WithMessage sets the error message
func (eb *ErrorBuilder) WithMessage(message string) *ErrorBuilder {
	eb.message = message
	return eb
}

// WithMessagef sets the error message with formatting
func (eb *ErrorBuilder) WithMessagef(format string, args ...interface{}) *ErrorBuilder {
	eb.message = fmt.Sprintf(format, args...)
	return eb
}

// WithCause sets the underlying cause of the error
func (eb *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	eb.cause = cause
	return eb
}

// WithSeverity sets the error severity
func (eb *ErrorBuilder) WithSeverity(severity Severity) *ErrorBuilder {
	eb.severity = severity
	return eb
}

// WithContext adds context information
func (eb *ErrorBuilder) WithContext(key string, value interface{}) *ErrorBuilder {
	eb.context[key] = value
	return eb
}

// WithRecoverable marks the error as recoverable
func (eb *ErrorBuilder) WithRecoverable(recoverable bool) *ErrorBuilder {
	eb.recoverable = recoverable
	return eb
}

// WithSuggestion adds a suggested action
func (eb *ErrorBuilder) WithSuggestion(suggestion string) *ErrorBuilder {
	eb.suggestions = append(eb.suggestions, suggestion)
	return eb
}

// WithSuggestions adds multiple suggested actions
func (eb *ErrorBuilder) WithSuggestions(suggestions ...string) *ErrorBuilder {
	eb.suggestions = append(eb.suggestions, suggestions...)
	return eb
}

// Build creates the final error
func (eb *ErrorBuilder) Build() error {
	return &ccAgentsError{
		errorType:   eb.errorType,
		severity:    eb.severity,
		message:     eb.message,
		cause:       eb.cause,
		context:     eb.context,
		recoverable: eb.recoverable,
		suggestions: eb.suggestions,
	}
}

// Convenience functions for common error types

// ValidationError creates a validation error
func ValidationError(message string) error {
	return NewError(ErrorTypeValidation).
		WithMessage(message).
		WithSeverity(SeverityLow).
		WithRecoverable(true).
		Build()
}

// NetworkError creates a network error
func NetworkError(cause error) error {
	return NewError(ErrorTypeNetwork).
		WithMessage("network operation failed").
		WithCause(cause).
		WithSeverity(SeverityMedium).
		WithRecoverable(true).
		WithSuggestion("Check your internet connection").
		WithSuggestion("Verify proxy settings if applicable").
		Build()
}

// AuthenticationError creates an authentication error
func AuthenticationError(service string) error {
	return NewError(ErrorTypeAuthentication).
		WithMessagef("authentication failed for %s", service).
		WithSeverity(SeverityHigh).
		WithRecoverable(true).
		WithContext("service", service).
		WithSuggestion(fmt.Sprintf("Re-authenticate with %s", service)).
		Build()
}

// ProcessError creates a process execution error
func ProcessError(command string, exitCode int, cause error) error {
	return NewError(ErrorTypeProcess).
		WithMessagef("process '%s' failed with exit code %d", command, exitCode).
		WithCause(cause).
		WithSeverity(SeverityMedium).
		WithRecoverable(true).
		WithContext("command", command).
		WithContext("exit_code", exitCode).
		Build()
}

// ConfigurationError creates a configuration error
func ConfigurationError(message string) error {
	return NewError(ErrorTypeConfiguration).
		WithMessage(message).
		WithSeverity(SeverityHigh).
		WithRecoverable(true).
		WithSuggestion("Check your configuration file").
		WithSuggestion("Run 'ccagents config validate' to verify settings").
		Build()
}

// GitError creates a git operation error
func GitError(operation string, cause error) error {
	return NewError(ErrorTypeGit).
		WithMessagef("git %s failed", operation).
		WithCause(cause).
		WithSeverity(SeverityMedium).
		WithRecoverable(true).
		WithContext("operation", operation).
		WithSuggestion("Check git repository status").
		WithSuggestion("Ensure you have proper git permissions").
		Build()
}

// GitHubError creates a GitHub API error
func GitHubError(operation string, cause error) error {
	return NewError(ErrorTypeGitHub).
		WithMessagef("GitHub %s failed", operation).
		WithCause(cause).
		WithSeverity(SeverityMedium).
		WithRecoverable(true).
		WithContext("operation", operation).
		WithSuggestion("Check GitHub authentication").
		WithSuggestion("Verify repository permissions").
		WithSuggestion("Check GitHub API rate limits").
		Build()
}

// ClaudeError creates a Claude Code integration error
func ClaudeError(operation string, cause error) error {
	return NewError(ErrorTypeClaude).
		WithMessagef("Claude Code %s failed", operation).
		WithCause(cause).
		WithSeverity(SeverityMedium).
		WithRecoverable(true).
		WithContext("operation", operation).
		WithSuggestion("Ensure Claude Code is installed and accessible").
		WithSuggestion("Check Claude Code authentication").
		WithSuggestion("Verify Claude Code version compatibility").
		Build()
}

// WorkflowError creates a workflow execution error
func WorkflowError(stage string, cause error) error {
	return NewError(ErrorTypeWorkflow).
		WithMessagef("workflow stage '%s' failed", stage).
		WithCause(cause).
		WithSeverity(SeverityMedium).
		WithRecoverable(true).
		WithContext("stage", stage).
		WithSuggestion("Review workflow configuration").
		WithSuggestion("Check stage dependencies").
		Build()
}

// Type checking functions

// IsType checks if an error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	if ccErr, ok := err.(*ccAgentsError); ok {
		return ccErr.Type() == errorType
	}
	return false
}

// IsSeverity checks if an error has a specific severity
func IsSeverity(err error, severity Severity) bool {
	if ccErr, ok := err.(*ccAgentsError); ok {
		return ccErr.Severity() == severity
	}
	return false
}

// IsRecoverable checks if an error is recoverable
func IsRecoverable(err error) bool {
	if ccErr, ok := err.(*ccAgentsError); ok {
		return ccErr.IsRecoverable()
	}
	return false
}

// GetSuggestions extracts suggestions from an error
func GetSuggestions(err error) []string {
	if ccErr, ok := err.(*ccAgentsError); ok {
		return ccErr.Suggestions()
	}
	return []string{}
}

// GetContext extracts context from an error
func GetContext(err error) map[string]interface{} {
	if ccErr, ok := err.(*ccAgentsError); ok {
		return ccErr.Context()
	}
	return nil
}
