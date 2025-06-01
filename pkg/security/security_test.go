package security

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCredentialManager(t *testing.T) {
	cm := NewCredentialManager()

	// Test with environment variable
	_ = os.Setenv("GITHUB_TOKEN", "test-token")
	defer func() { _ = os.Unsetenv("GITHUB_TOKEN") }()

	token, err := cm.GetGitHubToken()
	if err != nil {
		t.Errorf("Expected to get GitHub token, got error: %v", err)
	}
	if token != "test-token" {
		t.Errorf("Expected 'test-token', got '%s'", token)
	}
}

func TestInputSanitizer(t *testing.T) {
	sanitizer := NewInputSanitizer()

	// Test safe command
	err := sanitizer.SanitizeCommand("git", []string{"status"})
	if err != nil {
		t.Errorf("Expected safe command to pass, got error: %v", err)
	}

	// Test unsafe command
	err = sanitizer.SanitizeCommand("rm", []string{"-rf", "/"})
	if err == nil {
		t.Error("Expected unsafe command to fail, but it passed")
	}

	// Test path sanitization
	cleanPath, err := sanitizer.SanitizePath("./test/file.go")
	if err != nil {
		t.Errorf("Expected safe path to pass, got error: %v", err)
	}
	if cleanPath != "test/file.go" {
		t.Errorf("Expected 'test/file.go', got '%s'", cleanPath)
	}

	// Test path traversal
	_, err = sanitizer.SanitizePath("../../../etc/passwd")
	if err == nil {
		t.Error("Expected path traversal to fail, but it passed")
	}

	// Test URL sanitization
	err = sanitizer.SanitizeURL("https://github.com/user/repo")
	if err != nil {
		t.Errorf("Expected safe URL to pass, got error: %v", err)
	}

	err = sanitizer.SanitizeURL("javascript:alert('xss')")
	if err == nil {
		t.Error("Expected unsafe URL to fail, but it passed")
	}
}

func TestSafeExecutor(t *testing.T) {
	tempDir := os.TempDir()
	executor := NewSafeExecutor(tempDir)

	ctx := context.Background()

	// Test successful command
	result, err := executor.ExecuteCommand(ctx, "echo", "hello")
	if err != nil {
		t.Errorf("Expected echo command to succeed, got error: %v", err)
		return
	}
	if result == nil {
		t.Error("Expected result to not be nil")
		return
	}
	if !result.IsSuccess() {
		t.Error("Expected command to be successful")
	}
	if result.Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", result.Command)
	}
}

func TestGitExecutor(t *testing.T) {
	tempDir := os.TempDir()
	executor := NewGitExecutor(tempDir)

	ctx := context.Background()

	// Test git status (will fail if not a git repo, but should not crash)
	_, err := executor.Status(ctx)
	// We expect an error since tempDir is not a git repo
	if err == nil {
		t.Log("Git status succeeded (unexpected but not necessarily wrong)")
	}

	// Test branch name validation
	_, err = executor.CreateBranch(ctx, "feature/test-branch")
	// Will fail due to not being in a git repo, but validation should pass
	if err == nil {
		t.Log("Create branch succeeded (unexpected but not necessarily wrong)")
	}

	// Test invalid branch name
	_, err = executor.CreateBranch(ctx, "invalid..branch")
	if err == nil {
		t.Error("Expected invalid branch name to fail validation")
	}
}

func TestPermissionManager(t *testing.T) {
	pm := NewPermissionManager(PermissionLevelStandard, nil)

	ctx := context.Background()

	// Test file read permission (should be granted)
	permCtx := &PermissionContext{
		Resource:  "test.go",
		Operation: "read",
	}

	granted, reason := pm.CheckPermission(ctx, PermissionReadFile, permCtx)
	if !granted {
		t.Errorf("Expected read permission to be granted, but was denied: %s", reason)
	}

	// Test file execution permission on standard level (should be denied for non-approved commands)
	permCtx = &PermissionContext{
		Resource:  "dangerous-command",
		Operation: "execute",
	}

	granted, _ = pm.CheckPermission(ctx, PermissionExecuteCommand, permCtx)
	if granted {
		t.Error("Expected dangerous command permission to be denied")
	}

	// Test permission level upgrade
	err := pm.UpdatePermissionLevel(ctx, PermissionLevelElevated)
	if err != nil {
		t.Errorf("Expected permission level update to succeed, got error: %v", err)
	}
}

func TestAuditLogger(t *testing.T) {
	tempDir := os.TempDir()
	logDir := filepath.Join(tempDir, "audit-test")
	defer func() { _ = os.RemoveAll(logDir) }()

	logger, err := NewAuditLogger(AuditLevelDetailed, logDir)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	ctx := context.Background()

	// Test authentication logging
	logger.LogAuthentication(ctx, "github", true, map[string]interface{}{
		"method": "token",
	})

	// Test command execution logging
	result := &ExecutionResult{
		Command:  "git",
		Args:     []string{"status"},
		ExitCode: 0,
		Duration: time.Millisecond * 100,
		Output:   "On branch main\nnothing to commit, working tree clean",
	}

	logger.LogCommandExecution(ctx, "git", []string{"status"}, result, nil)

	// Test file access logging
	logger.LogFileAccess(ctx, "read", "/path/to/file.go", true, map[string]interface{}{
		"size": 1024,
	})

	// Test security event logging
	logger.LogSecurityEvent(ctx, "PERMISSION_CHECK", "validate_access", "/sensitive/path", false, map[string]interface{}{
		"reason": "insufficient_permissions",
	})
}

func TestSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig()

	// Test validation
	err := config.Validate()
	if err != nil {
		t.Errorf("Expected default config to be valid, got error: %v", err)
	}

	// Test invalid config
	config.PermissionLevel = PermissionLevel(999)
	err = config.Validate()
	if err == nil {
		t.Error("Expected invalid permission level to fail validation")
	}

	// Reset to valid config
	config = DefaultSecurityConfig()

	// Test directory creation
	config.AuditLogDir = filepath.Join(os.TempDir(), "test-audit")
	config.WorkspaceRoot = filepath.Join(os.TempDir(), "test-workspace")
	config.TempDirectory = filepath.Join(os.TempDir(), "test-temp")

	defer func() {
		_ = os.RemoveAll(config.AuditLogDir)
		_ = os.RemoveAll(config.WorkspaceRoot)
		_ = os.RemoveAll(config.TempDirectory)
	}()

	err = config.EnsureDirectories()
	if err != nil {
		t.Errorf("Expected directory creation to succeed, got error: %v", err)
	}

	// Verify directories were created
	for _, dir := range []string{config.AuditLogDir, config.WorkspaceRoot, config.TempDirectory} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", dir)
		}
	}
}

func TestSecurityManager(t *testing.T) {
	tempDir := os.TempDir()
	config := DefaultSecurityConfig()
	config.AuditLogDir = filepath.Join(tempDir, "test-audit")
	config.WorkspaceRoot = filepath.Join(tempDir, "test-workspace")
	config.TempDirectory = filepath.Join(tempDir, "test-temp")

	defer func() {
		_ = os.RemoveAll(config.AuditLogDir)
		_ = os.RemoveAll(config.WorkspaceRoot)
		_ = os.RemoveAll(config.TempDirectory)
	}()

	sm, err := NewSecurityManager(config)
	if err != nil {
		t.Fatalf("Failed to create security manager: %v", err)
	}
	defer func() { _ = sm.Close() }()

	ctx := context.Background()

	// Test secure command execution
	result, err := sm.ExecuteSecureCommand(ctx, "echo", "test")
	if err != nil {
		t.Errorf("Expected secure command to succeed, got error: %v", err)
		return
	}
	if result == nil {
		t.Error("Expected result to not be nil")
		return
	}
	if !result.IsSuccess() {
		t.Error("Expected command to be successful")
	}

	// Test secure file access
	err = sm.SecureFileAccess(ctx, "read", "test.go")
	if err != nil {
		t.Errorf("Expected secure file access to succeed, got error: %v", err)
	}

	// Test secure network access
	err = sm.SecureNetworkAccess(ctx, "https://github.com/user/repo", "get")
	if err != nil {
		t.Errorf("Expected secure network access to succeed, got error: %v", err)
	}

	// Test unsafe network access
	err = sm.SecureNetworkAccess(ctx, "javascript:alert('xss')", "get")
	if err == nil {
		t.Error("Expected unsafe network access to fail")
	}
}

func TestAuditLevel(t *testing.T) {
	tests := []struct {
		level    AuditLevel
		expected string
	}{
		{AuditLevelNone, "none"},
		{AuditLevelBasic, "basic"},
		{AuditLevelDetailed, "detailed"},
		{AuditLevelVerbose, "verbose"},
	}

	for _, test := range tests {
		if test.level.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.level.String())
		}
	}
}

func TestPermissionLevel(t *testing.T) {
	tests := []struct {
		level    PermissionLevel
		expected string
	}{
		{PermissionLevelNone, "none"},
		{PermissionLevelRestricted, "restricted"},
		{PermissionLevelStandard, "standard"},
		{PermissionLevelElevated, "elevated"},
	}

	for _, test := range tests {
		if test.level.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.level.String())
		}
	}
}

// Test helper functions
func TestValidateServiceName(t *testing.T) {
	tests := []struct {
		name    string
		service string
		valid   bool
	}{
		{"valid service", "github", true},
		{"valid with underscore", "github_api", true},
		{"valid with hyphen", "github-api", true},
		{"empty service", "", false},
		{"invalid chars", "github@api", false},
		{"too long", string(make([]byte, 51)), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateServiceName(test.service)
			if test.valid && err != nil {
				t.Errorf("Expected %s to be valid, got error: %v", test.service, err)
			}
			if !test.valid && err == nil {
				t.Errorf("Expected %s to be invalid, but validation passed", test.service)
			}
		})
	}
}

func TestValidateCredential(t *testing.T) {
	tests := []struct {
		name       string
		credential string
		valid      bool
	}{
		{"valid token", "ghp_abcdef123456", true},
		{"empty credential", "", false},
		{"with shell chars", "token; rm -rf /", false},
		{"too long", string(make([]byte, 1001)), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCredential(test.credential)
			if test.valid && err != nil {
				t.Errorf("Expected %s to be valid, got error: %v", test.credential, err)
			}
			if !test.valid && err == nil {
				t.Errorf("Expected %s to be invalid, but validation passed", test.credential)
			}
		})
	}
}
