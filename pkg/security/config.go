package security

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
)

// SecurityConfig represents security configuration options
type SecurityConfig struct {
	// Permission settings
	PermissionLevel PermissionLevel `json:"permission_level" yaml:"permission_level"`
	AllowElevated   bool            `json:"allow_elevated" yaml:"allow_elevated"`
	RestrictPaths   []string        `json:"restrict_paths" yaml:"restrict_paths"`
	AllowedCommands []string        `json:"allowed_commands" yaml:"allowed_commands"`

	// Audit settings
	AuditLevel         AuditLevel `json:"audit_level" yaml:"audit_level"`
	AuditLogDir        string     `json:"audit_log_dir" yaml:"audit_log_dir"`
	AuditRetentionDays int        `json:"audit_retention_days" yaml:"audit_retention_days"`

	// Credential settings
	CredentialStore    string   `json:"credential_store" yaml:"credential_store"`
	RequireCredentials []string `json:"require_credentials" yaml:"require_credentials"`

	// Command execution settings
	CommandTimeout        time.Duration `json:"command_timeout" yaml:"command_timeout"`
	MaxConcurrentCommands int           `json:"max_concurrent_commands" yaml:"max_concurrent_commands"`
	SandboxMode           bool          `json:"sandbox_mode" yaml:"sandbox_mode"`

	// Network settings
	AllowedURLPatterns []string      `json:"allowed_url_patterns" yaml:"allowed_url_patterns"`
	BlockedURLPatterns []string      `json:"blocked_url_patterns" yaml:"blocked_url_patterns"`
	NetworkTimeout     time.Duration `json:"network_timeout" yaml:"network_timeout"`

	// File system settings
	WorkspaceRoot string `json:"workspace_root" yaml:"workspace_root"`
	TempDirectory string `json:"temp_directory" yaml:"temp_directory"`
	MaxFileSize   int64  `json:"max_file_size" yaml:"max_file_size"`

	// Session settings
	SessionTimeout time.Duration `json:"session_timeout" yaml:"session_timeout"`
	MaxSessions    int           `json:"max_sessions" yaml:"max_sessions"`
	RequireAuth    bool          `json:"require_auth" yaml:"require_auth"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Use current directory as fallback if home directory cannot be determined
		homeDir = "."
	}

	return &SecurityConfig{
		// Permission settings
		PermissionLevel: PermissionLevelStandard,
		AllowElevated:   false,
		RestrictPaths:   []string{"/etc", "/root", "/usr/bin", "/bin", "/sbin"},
		AllowedCommands: []string{"git", "gh", "go", "make", "npm", "yarn", "docker"},

		// Audit settings
		AuditLevel:         AuditLevelDetailed,
		AuditLogDir:        filepath.Join(homeDir, ".ccagents", "logs"),
		AuditRetentionDays: 30,

		// Credential settings
		CredentialStore:    "keychain",
		RequireCredentials: []string{"github"},

		// Command execution settings
		CommandTimeout:        30 * time.Second,
		MaxConcurrentCommands: 5,
		SandboxMode:           true,

		// Network settings
		AllowedURLPatterns: []string{"https://github.com/*", "https://api.github.com/*"},
		BlockedURLPatterns: []string{"localhost", "127.0.0.1", "0.0.0.0", "internal", "admin"},
		NetworkTimeout:     30 * time.Second,

		// File system settings
		WorkspaceRoot: filepath.Join(homeDir, ".ccagents", "workspaces"),
		TempDirectory: filepath.Join(homeDir, ".ccagents", "tmp"),
		MaxFileSize:   100 * 1024 * 1024, // 100MB

		// Session settings
		SessionTimeout: 2 * time.Hour,
		MaxSessions:    3,
		RequireAuth:    true,
	}
}

// Validate validates the security configuration
func (sc *SecurityConfig) Validate() error {
	// Validate permission level
	if sc.PermissionLevel < PermissionLevelNone || sc.PermissionLevel > PermissionLevelElevated {
		return errors.ValidationError("invalid permission level")
	}

	// Validate audit level
	if sc.AuditLevel < AuditLevelNone || sc.AuditLevel > AuditLevelVerbose {
		return errors.ValidationError("invalid audit level")
	}

	// Validate directories
	if sc.AuditLogDir == "" {
		return errors.ValidationError("audit log directory cannot be empty")
	}

	if sc.WorkspaceRoot == "" {
		return errors.ValidationError("workspace root cannot be empty")
	}

	if sc.TempDirectory == "" {
		return errors.ValidationError("temp directory cannot be empty")
	}

	// Validate timeouts
	if sc.CommandTimeout <= 0 {
		return errors.ValidationError("command timeout must be positive")
	}

	if sc.NetworkTimeout <= 0 {
		return errors.ValidationError("network timeout must be positive")
	}

	if sc.SessionTimeout <= 0 {
		return errors.ValidationError("session timeout must be positive")
	}

	// Validate limits
	if sc.MaxConcurrentCommands <= 0 {
		return errors.ValidationError("max concurrent commands must be positive")
	}

	if sc.MaxSessions <= 0 {
		return errors.ValidationError("max sessions must be positive")
	}

	if sc.MaxFileSize <= 0 {
		return errors.ValidationError("max file size must be positive")
	}

	if sc.AuditRetentionDays <= 0 {
		return errors.ValidationError("audit retention days must be positive")
	}

	return nil
}

// EnsureDirectories creates required directories with proper permissions
func (sc *SecurityConfig) EnsureDirectories() error {
	directories := []struct {
		path        string
		permissions os.FileMode
	}{
		{sc.AuditLogDir, 0700},   // Owner read/write/execute only
		{sc.WorkspaceRoot, 0750}, // Restricted permissions
		{sc.TempDirectory, 0700}, // Owner read/write/execute only
	}

	for _, dir := range directories {
		if err := os.MkdirAll(dir.path, dir.permissions); err != nil {
			return errors.NewError(errors.ErrorTypeFileSystem).
				WithMessage(fmt.Sprintf("failed to create directory: %s", dir.path)).
				WithCause(err).
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(true).
				WithContext("directory", dir.path).
				WithContext("permissions", dir.permissions.String()).
				Build()
		}
	}

	return nil
}

// SecurityManager provides centralized security management
type SecurityManager struct {
	config        *SecurityConfig
	permissionMgr *PermissionManager
	credentialMgr *CredentialManager
	auditLogger   *AuditLogger
	sanitizer     *InputSanitizer
	executor      *SafeExecutor
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(config *SecurityConfig) (*SecurityManager, error) {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, errors.NewError(errors.ErrorTypeConfiguration).
			WithMessage("invalid security configuration").
			WithCause(err).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(true).
			Build()
	}

	// Ensure required directories exist
	if err := config.EnsureDirectories(); err != nil {
		return nil, err
	}

	// Create audit logger
	auditLogger, err := NewAuditLogger(config.AuditLevel, config.AuditLogDir)
	if err != nil {
		return nil, err
	}

	// Create permission manager
	permissionMgr := NewPermissionManager(config.PermissionLevel, auditLogger)

	// Create credential manager
	credentialMgr := NewCredentialManager()

	// Create input sanitizer
	sanitizer := NewInputSanitizer()

	// Create safe executor
	executor := NewSafeExecutor(config.WorkspaceRoot).
		WithTimeout(config.CommandTimeout)

	sm := &SecurityManager{
		config:        config,
		permissionMgr: permissionMgr,
		credentialMgr: credentialMgr,
		auditLogger:   auditLogger,
		sanitizer:     sanitizer,
		executor:      executor,
	}

	return sm, nil
}

// GetConfig returns the security configuration
func (sm *SecurityManager) GetConfig() *SecurityConfig {
	return sm.config
}

// GetPermissionManager returns the permission manager
func (sm *SecurityManager) GetPermissionManager() *PermissionManager {
	return sm.permissionMgr
}

// GetCredentialManager returns the credential manager
func (sm *SecurityManager) GetCredentialManager() *CredentialManager {
	return sm.credentialMgr
}

// GetAuditLogger returns the audit logger
func (sm *SecurityManager) GetAuditLogger() *AuditLogger {
	return sm.auditLogger
}

// GetSanitizer returns the input sanitizer
func (sm *SecurityManager) GetSanitizer() *InputSanitizer {
	return sm.sanitizer
}

// GetExecutor returns the safe executor
func (sm *SecurityManager) GetExecutor() *SafeExecutor {
	return sm.executor
}

// ValidateCredentials validates that required credentials are available
func (sm *SecurityManager) ValidateCredentials(ctx context.Context) error {
	if sm.auditLogger != nil {
		sm.auditLogger.LogSecurityEvent(ctx, "CREDENTIAL_VALIDATION", "validate_credentials", "", true, nil)
	}

	return sm.credentialMgr.ValidateCredentials()
}

// ExecuteSecureCommand executes a command with full security validation
func (sm *SecurityManager) ExecuteSecureCommand(ctx context.Context, command string, args ...string) (*ExecutionResult, error) {
	// Check command execution permission
	permCtx := &PermissionContext{
		Resource:  command,
		Operation: "execute",
		Metadata: map[string]interface{}{
			"args": args,
		},
	}

	granted, reason := sm.permissionMgr.CheckPermission(ctx, PermissionExecuteCommand, permCtx)
	if !granted {
		return nil, errors.NewError(errors.ErrorTypePermission).
			WithMessage("command execution permission denied").
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("command", command).
			WithContext("reason", reason).
			Build()
	}

	// Sanitize command and arguments
	if err := sm.sanitizer.SanitizeCommand(command, args); err != nil {
		return nil, err
	}

	// Execute command
	result, err := sm.executor.ExecuteCommand(ctx, command, args...)

	// Log execution
	if sm.auditLogger != nil {
		sm.auditLogger.LogCommandExecution(ctx, command, args, result, err)
	}

	return result, err
}

// SecureFileAccess performs file access with security validation
func (sm *SecurityManager) SecureFileAccess(ctx context.Context, operation string, path string) error {
	// Sanitize path
	cleanPath, err := sm.sanitizer.SanitizePath(path)
	if err != nil {
		return err
	}

	// Check file access permission
	if err := sm.permissionMgr.CheckFileAccess(ctx, operation, cleanPath); err != nil {
		return err
	}

	// Log file access
	if sm.auditLogger != nil {
		sm.auditLogger.LogFileAccess(ctx, operation, cleanPath, true, nil)
	}

	return nil
}

// SecureNetworkAccess validates network access requests
func (sm *SecurityManager) SecureNetworkAccess(ctx context.Context, url string, operation string) error {
	// Sanitize URL
	if err := sm.sanitizer.SanitizeURL(url); err != nil {
		return err
	}

	// Check network access permission
	if err := sm.permissionMgr.CheckNetworkAccess(ctx, url, operation); err != nil {
		return err
	}

	// Log network access
	if sm.auditLogger != nil {
		sm.auditLogger.LogNetworkAccess(ctx, operation, url, true, nil)
	}

	return nil
}

// Close closes the security manager and releases resources
func (sm *SecurityManager) Close() error {
	if sm.auditLogger != nil {
		return sm.auditLogger.Close()
	}
	return nil
}

// CleanupExpiredSessions removes expired audit logs and temporary files
func (sm *SecurityManager) CleanupExpiredSessions(ctx context.Context) error {
	// Rotate audit logs
	if sm.auditLogger != nil {
		if err := sm.auditLogger.RotateLogs(); err != nil {
			return err
		}
	}

	// Clean up temporary files older than retention period
	cutoff := time.Now().AddDate(0, 0, -sm.config.AuditRetentionDays)

	err := filepath.Walk(sm.config.TempDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() && info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				if sm.auditLogger != nil {
					sm.auditLogger.LogFileAccess(ctx, "cleanup", path, true, map[string]interface{}{
						"reason": "expired_temp_file",
						"age":    time.Since(info.ModTime()).String(),
					})
				}
			}
		}

		return nil
	})

	return err
}
