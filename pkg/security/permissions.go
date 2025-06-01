package security

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fumiya-kume/cca/pkg/errors"
)

// Permission represents a security permission
type Permission string

const (
	// File system permissions
	PermissionReadFile    Permission = "file:read"
	PermissionWriteFile   Permission = "file:write"
	PermissionCreateFile  Permission = "file:create"
	PermissionDeleteFile  Permission = "file:delete"
	PermissionExecuteFile Permission = "file:execute"

	// Network permissions
	PermissionNetworkAccess Permission = "network:access"
	PermissionAPIAccess     Permission = "api:access"

	// Command execution permissions
	PermissionExecuteCommand  Permission = "command:execute"
	PermissionGitOperation    Permission = "git:operation"
	PermissionGitHubOperation Permission = "github:operation"

	// System permissions
	PermissionProcessSpawn Permission = "process:spawn"
	PermissionEnvironment  Permission = "env:modify"

	// Configuration permissions
	PermissionConfigRead  Permission = "config:read"
	PermissionConfigWrite Permission = "config:write"
)

// PermissionLevel represents the level of permissions granted
type PermissionLevel int

const (
	PermissionLevelNone PermissionLevel = iota
	PermissionLevelRestricted
	PermissionLevelStandard
	PermissionLevelElevated
)

// String returns string representation of permission level
func (pl PermissionLevel) String() string {
	switch pl {
	case PermissionLevelNone:
		return "none"
	case PermissionLevelRestricted:
		return "restricted"
	case PermissionLevelStandard:
		return "standard"
	case PermissionLevelElevated:
		return "elevated"
	default:
		return "unknown"
	}
}

// PermissionRule defines a permission rule
type PermissionRule struct {
	Permission Permission             `json:"permission"`
	Level      PermissionLevel        `json:"level"`
	Resources  []string               `json:"resources,omitempty"`
	Conditions map[string]interface{} `json:"conditions,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
}

// PermissionContext provides context for permission checks
type PermissionContext struct {
	UserID    string
	SessionID string
	Resource  string
	Operation string
	Metadata  map[string]interface{}
}

// PermissionManager manages security permissions
type PermissionManager struct {
	rules       map[Permission]*PermissionRule
	level       PermissionLevel
	auditLogger *AuditLogger
	mutex       sync.RWMutex
}

// NewPermissionManager creates a new permission manager
func NewPermissionManager(level PermissionLevel, auditLogger *AuditLogger) *PermissionManager {
	pm := &PermissionManager{
		rules:       make(map[Permission]*PermissionRule),
		level:       level,
		auditLogger: auditLogger,
	}

	// Set up default permission rules based on level
	pm.setupDefaultRules()

	return pm
}

// CheckPermission checks if a permission is granted
func (pm *PermissionManager) CheckPermission(ctx context.Context, permission Permission, permCtx *PermissionContext) (bool, string) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Get permission rule
	rule, exists := pm.rules[permission]
	if !exists {
		reason := "permission not defined"
		if pm.auditLogger != nil {
			pm.auditLogger.LogPermissionCheck(ctx, string(permission), permCtx.Resource, false, reason)
		}
		return false, reason
	}

	// Check permission level
	if rule.Level > pm.level {
		reason := fmt.Sprintf("insufficient permission level (required: %s, current: %s)",
			rule.Level.String(), pm.level.String())
		if pm.auditLogger != nil {
			pm.auditLogger.LogPermissionCheck(ctx, string(permission), permCtx.Resource, false, reason)
		}
		return false, reason
	}

	// Check resource restrictions
	if len(rule.Resources) > 0 && permCtx.Resource != "" {
		if !pm.matchesResource(permCtx.Resource, rule.Resources) {
			reason := "resource not permitted"
			if pm.auditLogger != nil {
				pm.auditLogger.LogPermissionCheck(ctx, string(permission), permCtx.Resource, false, reason)
			}
			return false, reason
		}
	}

	// Check additional conditions
	if len(rule.Conditions) > 0 {
		if !pm.checkConditions(rule.Conditions, permCtx) {
			reason := "condition check failed"
			if pm.auditLogger != nil {
				pm.auditLogger.LogPermissionCheck(ctx, string(permission), permCtx.Resource, false, reason)
			}
			return false, reason
		}
	}

	// Permission granted
	reason := "permission granted"
	if pm.auditLogger != nil {
		pm.auditLogger.LogPermissionCheck(ctx, string(permission), permCtx.Resource, true, reason)
	}
	return true, reason
}

// CheckFileAccess checks file access permissions
func (pm *PermissionManager) CheckFileAccess(ctx context.Context, operation string, path string) error {
	permCtx := &PermissionContext{
		Resource:  path,
		Operation: operation,
	}

	var permission Permission
	switch operation {
	case "read":
		permission = PermissionReadFile
	case "write", "update":
		permission = PermissionWriteFile
	case "create":
		permission = PermissionCreateFile
	case "delete":
		permission = PermissionDeleteFile
	case "execute":
		permission = PermissionExecuteFile
	default:
		return errors.NewError(errors.ErrorTypePermission).
			WithMessage(fmt.Sprintf("unknown file operation: %s", operation)).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("operation", operation).
			WithContext("path", path).
			Build()
	}

	granted, reason := pm.CheckPermission(ctx, permission, permCtx)
	if !granted {
		return errors.NewError(errors.ErrorTypePermission).
			WithMessage(fmt.Sprintf("file %s permission denied", operation)).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("operation", operation).
			WithContext("path", path).
			WithContext("reason", reason).
			Build()
	}

	return nil
}

// CheckCommandExecution checks command execution permissions
func (pm *PermissionManager) CheckCommandExecution(ctx context.Context, command string, args []string) error {
	permCtx := &PermissionContext{
		Resource:  command,
		Operation: "execute",
		Metadata: map[string]interface{}{
			"args": args,
		},
	}

	granted, reason := pm.CheckPermission(ctx, PermissionExecuteCommand, permCtx)
	if !granted {
		return errors.NewError(errors.ErrorTypePermission).
			WithMessage("command execution permission denied").
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("command", command).
			WithContext("args", args).
			WithContext("reason", reason).
			Build()
	}

	return nil
}

// CheckNetworkAccess checks network access permissions
func (pm *PermissionManager) CheckNetworkAccess(ctx context.Context, url string, operation string) error {
	permCtx := &PermissionContext{
		Resource:  url,
		Operation: operation,
	}

	granted, reason := pm.CheckPermission(ctx, PermissionNetworkAccess, permCtx)
	if !granted {
		return errors.NewError(errors.ErrorTypePermission).
			WithMessage("network access permission denied").
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(false).
			WithContext("url", url).
			WithContext("operation", operation).
			WithContext("reason", reason).
			Build()
	}

	return nil
}

// UpdatePermissionLevel updates the current permission level
func (pm *PermissionManager) UpdatePermissionLevel(ctx context.Context, level PermissionLevel) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	oldLevel := pm.level
	pm.level = level

	if pm.auditLogger != nil {
		pm.auditLogger.LogConfigurationChange(ctx, "permission_level", oldLevel.String(), level.String())
	}

	// Update default rules for new level
	pm.setupDefaultRules()

	return nil
}

// AddPermissionRule adds a custom permission rule
func (pm *PermissionManager) AddPermissionRule(ctx context.Context, rule *PermissionRule) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.rules[rule.Permission] = rule

	if pm.auditLogger != nil {
		pm.auditLogger.LogConfigurationChange(ctx, "permission_rule", nil, rule)
	}

	return nil
}

// RemovePermissionRule removes a permission rule
func (pm *PermissionManager) RemovePermissionRule(ctx context.Context, permission Permission) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	oldRule, exists := pm.rules[permission]
	if exists {
		delete(pm.rules, permission)

		if pm.auditLogger != nil {
			pm.auditLogger.LogConfigurationChange(ctx, "permission_rule", oldRule, nil)
		}
	}

	return nil
}

// GetPermissionRules returns all current permission rules
func (pm *PermissionManager) GetPermissionRules() map[Permission]*PermissionRule {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	rules := make(map[Permission]*PermissionRule)
	for k, v := range pm.rules {
		rules[k] = v
	}

	return rules
}

// setupDefaultRules sets up default permission rules based on the current level
func (pm *PermissionManager) setupDefaultRules() {
	switch pm.level {
	case PermissionLevelNone:
		// No permissions granted

	case PermissionLevelRestricted:
		// Basic read-only permissions
		pm.rules[PermissionReadFile] = &PermissionRule{
			Permission: PermissionReadFile,
			Level:      PermissionLevelRestricted,
			Resources:  []string{"*.go", "*.md", "*.txt", "*.json", "*.yaml", "*.yml"},
			Reason:     "restricted read access to common file types",
		}
		pm.rules[PermissionConfigRead] = &PermissionRule{
			Permission: PermissionConfigRead,
			Level:      PermissionLevelRestricted,
			Reason:     "read-only configuration access",
		}

	case PermissionLevelStandard:
		// Standard development permissions
		pm.rules[PermissionReadFile] = &PermissionRule{
			Permission: PermissionReadFile,
			Level:      PermissionLevelRestricted,
			Reason:     "standard file read access",
		}
		pm.rules[PermissionWriteFile] = &PermissionRule{
			Permission: PermissionWriteFile,
			Level:      PermissionLevelStandard,
			Resources:  []string{"!*.key", "!*.pem", "!*.p12", "!/etc/*", "!/root/*"},
			Reason:     "standard file write access with security restrictions",
		}
		pm.rules[PermissionCreateFile] = &PermissionRule{
			Permission: PermissionCreateFile,
			Level:      PermissionLevelStandard,
			Reason:     "standard file creation access",
		}
		pm.rules[PermissionExecuteCommand] = &PermissionRule{
			Permission: PermissionExecuteCommand,
			Level:      PermissionLevelStandard,
			Resources:  []string{"git", "gh", "go", "make", "npm", "yarn", "echo", "ls", "cat"},
			Reason:     "execute approved development commands",
		}
		pm.rules[PermissionGitOperation] = &PermissionRule{
			Permission: PermissionGitOperation,
			Level:      PermissionLevelStandard,
			Reason:     "git operation access",
		}
		pm.rules[PermissionGitHubOperation] = &PermissionRule{
			Permission: PermissionGitHubOperation,
			Level:      PermissionLevelStandard,
			Reason:     "GitHub operation access",
		}
		pm.rules[PermissionNetworkAccess] = &PermissionRule{
			Permission: PermissionNetworkAccess,
			Level:      PermissionLevelStandard,
			Resources:  []string{"https://github.com/*", "https://api.github.com/*"},
			Reason:     "access to approved development services",
		}
		pm.rules[PermissionConfigRead] = &PermissionRule{
			Permission: PermissionConfigRead,
			Level:      PermissionLevelRestricted,
			Reason:     "configuration read access",
		}
		pm.rules[PermissionConfigWrite] = &PermissionRule{
			Permission: PermissionConfigWrite,
			Level:      PermissionLevelStandard,
			Reason:     "configuration write access",
		}

	case PermissionLevelElevated:
		// Elevated permissions for advanced operations
		pm.rules[PermissionReadFile] = &PermissionRule{
			Permission: PermissionReadFile,
			Level:      PermissionLevelRestricted,
			Reason:     "elevated file read access",
		}
		pm.rules[PermissionWriteFile] = &PermissionRule{
			Permission: PermissionWriteFile,
			Level:      PermissionLevelStandard,
			Reason:     "elevated file write access",
		}
		pm.rules[PermissionCreateFile] = &PermissionRule{
			Permission: PermissionCreateFile,
			Level:      PermissionLevelStandard,
			Reason:     "elevated file creation access",
		}
		pm.rules[PermissionDeleteFile] = &PermissionRule{
			Permission: PermissionDeleteFile,
			Level:      PermissionLevelElevated,
			Resources:  []string{"!*.key", "!*.pem", "!/etc/*", "!/root/*"},
			Reason:     "elevated file deletion with restrictions",
		}
		pm.rules[PermissionExecuteCommand] = &PermissionRule{
			Permission: PermissionExecuteCommand,
			Level:      PermissionLevelElevated,
			Reason:     "elevated command execution access",
		}
		pm.rules[PermissionProcessSpawn] = &PermissionRule{
			Permission: PermissionProcessSpawn,
			Level:      PermissionLevelElevated,
			Reason:     "process spawning access",
		}
		pm.rules[PermissionNetworkAccess] = &PermissionRule{
			Permission: PermissionNetworkAccess,
			Level:      PermissionLevelElevated,
			Reason:     "elevated network access",
		}
		pm.rules[PermissionAPIAccess] = &PermissionRule{
			Permission: PermissionAPIAccess,
			Level:      PermissionLevelElevated,
			Reason:     "API access permissions",
		}
		pm.rules[PermissionConfigRead] = &PermissionRule{
			Permission: PermissionConfigRead,
			Level:      PermissionLevelRestricted,
			Reason:     "configuration read access",
		}
		pm.rules[PermissionConfigWrite] = &PermissionRule{
			Permission: PermissionConfigWrite,
			Level:      PermissionLevelStandard,
			Reason:     "configuration write access",
		}
	}
}

// matchesResource checks if a resource matches any of the allowed patterns
func (pm *PermissionManager) matchesResource(resource string, patterns []string) bool {
	hasPositivePattern := false
	matchedPositive := false

	for _, pattern := range patterns {
		// Handle negation patterns (starting with !)
		if strings.HasPrefix(pattern, "!") {
			negPattern := pattern[1:]
			if matched, err := filepath.Match(negPattern, resource); err == nil && matched {
				return false
			}
			if strings.HasPrefix(resource, negPattern) {
				return false
			}
		} else {
			hasPositivePattern = true
			// Positive pattern matching
			if matched, err := filepath.Match(pattern, resource); err == nil && matched {
				matchedPositive = true
			}
			if strings.HasPrefix(resource, pattern) {
				matchedPositive = true
			}
			// Handle wildcard patterns for URLs
			if strings.Contains(pattern, "*") {
				basePattern := strings.TrimSuffix(pattern, "*")
				if strings.HasPrefix(resource, basePattern) {
					matchedPositive = true
				}
			}
		}
	}

	// If we have positive patterns, require at least one match
	if hasPositivePattern {
		return matchedPositive
	}

	// If we only have negative patterns, allow by default
	return true
}

// checkConditions evaluates additional permission conditions
func (pm *PermissionManager) checkConditions(conditions map[string]interface{}, permCtx *PermissionContext) bool {
	for key, expected := range conditions {
		switch key {
		case "operation":
			if permCtx.Operation != expected {
				return false
			}
		case "user_id":
			if permCtx.UserID != expected {
				return false
			}
		case "session_id":
			if permCtx.SessionID != expected {
				return false
			}
		default:
			// Check metadata
			if permCtx.Metadata != nil {
				if actual, exists := permCtx.Metadata[key]; !exists || actual != expected {
					return false
				}
			}
		}
	}

	return true
}
