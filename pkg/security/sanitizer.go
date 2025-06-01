package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fumiya-kume/cca/pkg/errors"
)

// InputSanitizer provides input validation and sanitization
type InputSanitizer struct {
	allowedCommands map[string]bool
	blockedPatterns []*regexp.Regexp
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer() *InputSanitizer {
	// Define allowed commands for safe execution
	allowedCommands := map[string]bool{
		"git":       true,
		"gh":        true,
		"go":        true,
		"make":      true,
		"npm":       true,
		"yarn":      true,
		"docker":    true,
		"kubectl":   true,
		"helm":      true,
		"terraform": true,
		"echo":      true, // For testing
		"ls":        true, // For testing
		"cat":       true, // For testing
	}

	// Define blocked patterns that indicate potential security issues
	blockedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`[;&|\$` + "`" + `]`),          // Shell metacharacters
		regexp.MustCompile(`\$\([^)]*\)`),                 // Command substitution
		regexp.MustCompile("`" + `[^` + "`" + `]*` + "`"), // Backtick command substitution
		regexp.MustCompile(`\.\./`),                       // Directory traversal
		regexp.MustCompile(`^/etc/`),                      // System directories
		regexp.MustCompile(`^/usr/bin/`),                  // System binaries
		regexp.MustCompile(`^/bin/`),                      // System binaries
		regexp.MustCompile(`rm\s+-rf\s+/`),                // Dangerous delete operations
		regexp.MustCompile(`sudo`),                        // Privilege escalation
		regexp.MustCompile(`curl.*\|.*sh`),                // Pipe to shell
		regexp.MustCompile(`wget.*\|.*sh`),                // Pipe to shell
	}

	return &InputSanitizer{
		allowedCommands: allowedCommands,
		blockedPatterns: blockedPatterns,
	}
}

// SanitizeCommand validates and sanitizes a command before execution
func (s *InputSanitizer) SanitizeCommand(command string, args []string) error {
	// Validate command name
	if err := s.validateCommand(command); err != nil {
		return err
	}

	// Validate all arguments
	for i, arg := range args {
		if err := s.validateArgument(arg); err != nil {
			return errors.NewError(errors.ErrorTypeValidation).
				WithMessage(fmt.Sprintf("invalid argument at position %d", i)).
				WithCause(err).
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(false).
				WithContext("command", command).
				WithContext("argument", arg).
				WithContext("position", i).
				Build()
		}
	}

	return nil
}

// SanitizePath validates and normalizes file paths
func (s *InputSanitizer) SanitizePath(path string) (string, error) {
	if path == "" {
		return "", errors.ValidationError("path cannot be empty")
	}

	// Clean the path to resolve .. and . elements
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return "", errors.NewError(errors.ErrorTypePermission).
			WithMessage("path traversal attempt detected").
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("original_path", path).
			WithContext("clean_path", cleanPath).
			WithSuggestion("Use absolute paths or paths relative to project root").
			Build()
	}

	// Check for access to sensitive system directories
	sensitiveDirectories := []string{
		"/etc/", "/usr/", "/bin/", "/sbin/", "/boot/", "/sys/", "/proc/",
		"/root/", "/var/log/", "/var/spool/", "/var/mail/",
	}

	for _, sensitive := range sensitiveDirectories {
		if strings.HasPrefix(cleanPath, sensitive) {
			return "", errors.NewError(errors.ErrorTypePermission).
				WithMessage("access to sensitive directory denied").
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(false).
				WithContext("path", cleanPath).
				WithContext("sensitive_directory", sensitive).
				WithSuggestion("Use paths within the project directory").
				Build()
		}
	}

	return cleanPath, nil
}

// SanitizeURL validates URLs for safety
func (s *InputSanitizer) SanitizeURL(url string) error {
	if url == "" {
		return errors.ValidationError("URL cannot be empty")
	}

	// Check for safe URL schemes
	allowedSchemes := []string{"https://", "http://", "git://", "ssh://"}
	hasValidScheme := false

	for _, scheme := range allowedSchemes {
		if strings.HasPrefix(url, scheme) {
			hasValidScheme = true
			break
		}
	}

	if !hasValidScheme {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("unsupported URL scheme").
			WithSeverity(errors.SeverityMedium).
			WithRecoverable(true).
			WithContext("url", url).
			WithSuggestion("Use https://, http://, git://, or ssh:// URLs").
			Build()
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"javascript:", "data:", "file:", "ftp://",
		"localhost", "127.0.0.1", "0.0.0.0",
		"internal", "local", "admin",
	}

	lowerURL := strings.ToLower(url)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerURL, pattern) {
			return errors.NewError(errors.ErrorTypeValidation).
				WithMessage("potentially unsafe URL pattern detected").
				WithSeverity(errors.SeverityMedium).
				WithRecoverable(true).
				WithContext("url", url).
				WithContext("pattern", pattern).
				WithSuggestion("Use public repository URLs only").
				Build()
		}
	}

	return nil
}

// SanitizeEnvironmentVariable validates environment variable values
func (s *InputSanitizer) SanitizeEnvironmentVariable(key, value string) error {
	// Validate key
	if err := s.validateEnvKey(key); err != nil {
		return err
	}

	// Validate value
	if err := s.validateEnvValue(value); err != nil {
		return err
	}

	return nil
}

// validateCommand checks if a command is allowed for execution
func (s *InputSanitizer) validateCommand(command string) error {
	if command == "" {
		return errors.ValidationError("command cannot be empty")
	}

	// Extract just the command name (no path)
	commandName := filepath.Base(command)

	// Check if command is in allowed list
	if !s.allowedCommands[commandName] {
		return errors.NewError(errors.ErrorTypePermission).
			WithMessage(fmt.Sprintf("command '%s' not allowed", commandName)).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("command", command).
			WithSuggestion("Use only approved commands for security").
			Build()
	}

	return nil
}

// validateArgument checks command arguments for safety
func (s *InputSanitizer) validateArgument(arg string) error {
	// Check against blocked patterns
	for _, pattern := range s.blockedPatterns {
		if pattern.MatchString(arg) {
			return errors.NewError(errors.ErrorTypeValidation).
				WithMessage("argument contains unsafe pattern").
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(false).
				WithContext("argument", arg).
				WithContext("pattern", pattern.String()).
				WithSuggestion("Remove shell metacharacters and unsafe patterns").
				Build()
		}
	}

	return nil
}

// validateEnvKey validates environment variable keys
func (s *InputSanitizer) validateEnvKey(key string) error {
	if key == "" {
		return errors.ValidationError("environment variable key cannot be empty")
	}

	// Environment variable keys should be alphanumeric with underscores
	validKeyPattern := regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	if !validKeyPattern.MatchString(key) {
		return errors.ValidationError("invalid environment variable key format")
	}

	// Check for sensitive system variables
	sensitiveKeys := []string{
		"PATH", "LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH",
		"HOME", "USER", "SHELL", "PWD",
		"SUDO_USER", "SUDO_UID", "SUDO_GID",
	}

	for _, sensitive := range sensitiveKeys {
		if key == sensitive {
			return errors.NewError(errors.ErrorTypePermission).
				WithMessage("modification of system environment variable not allowed").
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(false).
				WithContext("key", key).
				Build()
		}
	}

	return nil
}

// validateEnvValue validates environment variable values
func (s *InputSanitizer) validateEnvValue(value string) error {
	// Check for shell injection patterns
	for _, pattern := range s.blockedPatterns {
		if pattern.MatchString(value) {
			return errors.NewError(errors.ErrorTypeValidation).
				WithMessage("environment variable value contains unsafe pattern").
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(false).
				WithContext("pattern", pattern.String()).
				WithSuggestion("Remove shell metacharacters from environment values").
				Build()
		}
	}

	return nil
}

// EscapeShellArg safely escapes an argument for shell execution
func (s *InputSanitizer) EscapeShellArg(arg string) string {
	// For additional safety, quote arguments that contain spaces or special characters
	specialChars := " \t\n\r\"'\\$`&|;(){}[]<>?*"
	if strings.ContainsAny(arg, specialChars) {
		// Escape single quotes and wrap in single quotes
		escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
		return "'" + escaped + "'"
	}
	return arg
}
