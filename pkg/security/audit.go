// Package security provides comprehensive security auditing, credential management,
// and security scanning capabilities for code repositories and applications.
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
)

// Audit result constants
const (
	auditResultSuccess = "SUCCESS"
	auditResultFailure = "FAILURE"
)

// Audit severity constants
const (
	auditSeverityInfo    = "INFO"
	auditSeverityWarning = "WARNING"
)

// Audit level string constants
const (
	auditLevelUnknown = "unknown"
)

// AuditLevel represents the level of audit logging
type AuditLevel int

const (
	AuditLevelNone AuditLevel = iota
	AuditLevelBasic
	AuditLevelDetailed
	AuditLevelVerbose
)

// String returns string representation of audit level
func (al AuditLevel) String() string {
	switch al {
	case AuditLevelNone:
		return "none"
	case AuditLevelBasic:
		return "basic"
	case AuditLevelDetailed:
		return "detailed"
	case AuditLevelVerbose:
		return "verbose"
	default:
		return auditLevelUnknown
	}
}

// AuditEvent represents a security-relevant event
type AuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	Actor     string                 `json:"actor,omitempty"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource,omitempty"`
	Result    string                 `json:"result"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Severity  string                 `json:"severity"`
	SessionID string                 `json:"session_id,omitempty"`
	ProcessID int                    `json:"process_id"`
	UserAgent string                 `json:"user_agent,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// AuditLogger provides security audit logging capabilities
type AuditLogger struct {
	level     AuditLevel
	logFile   string
	sessionID string
	mutex     sync.RWMutex
	file      *os.File
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(level AuditLevel, logDir string) (*AuditLogger, error) {
	if level == AuditLevelNone {
		return &AuditLogger{level: level}, nil
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create audit log directory").
			WithCause(err).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(true).
			WithContext("log_dir", logDir).
			Build()
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, fmt.Sprintf("audit-%s.log", timestamp))

	// #nosec G304 - logFile path is controlled by configuration and validated
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to open audit log file").
			WithCause(err).
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(true).
			WithContext("log_file", logFile).
			Build()
	}

	return &AuditLogger{
		level:     level,
		logFile:   logFile,
		sessionID: generateSessionID(),
		file:      file,
	}, nil
}

// SetSessionID sets the session ID for audit events
func (al *AuditLogger) SetSessionID(sessionID string) {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	al.sessionID = sessionID
}

// LogAuthentication logs authentication events
func (al *AuditLogger) LogAuthentication(ctx context.Context, service string, success bool, details map[string]interface{}) {
	if al.level == AuditLevelNone {
		return
	}

	result := auditResultSuccess
	severity := auditSeverityInfo
	if !success {
		result = auditResultFailure
		severity = auditSeverityWarning
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "AUTHENTICATION",
		Action:    fmt.Sprintf("authenticate_%s", service),
		Resource:  service,
		Result:    result,
		Details:   details,
		Severity:  severity,
		SessionID: al.sessionID,
		ProcessID: os.Getpid(),
	}

	al.writeEvent(event)
}

// LogCommandExecution logs command execution events
func (al *AuditLogger) LogCommandExecution(ctx context.Context, command string, args []string, result *ExecutionResult, err error) {
	if al.level == AuditLevelNone || al.level == AuditLevelBasic {
		return
	}

	eventResult := "SUCCESS"
	severity := auditSeverityInfo
	var errorMsg string

	if err != nil {
		eventResult = "FAILURE"
		severity = "ERROR"
		errorMsg = err.Error()
	}

	details := map[string]interface{}{
		"command": command,
		"args":    args,
	}

	if result != nil {
		details["exit_code"] = result.ExitCode
		details["duration"] = result.Duration.String()
		if al.level == AuditLevelVerbose {
			details["output"] = result.Output
		}
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "COMMAND_EXECUTION",
		Action:    "execute_command",
		Resource:  command,
		Result:    eventResult,
		Details:   details,
		Severity:  severity,
		SessionID: al.sessionID,
		ProcessID: os.Getpid(),
		Error:     errorMsg,
	}

	al.writeEvent(event)
}

// LogFileAccess logs file system access events
func (al *AuditLogger) LogFileAccess(ctx context.Context, operation string, path string, success bool, details map[string]interface{}) {
	if al.level == AuditLevelNone || al.level == AuditLevelBasic {
		return
	}

	result := auditResultSuccess
	severity := auditSeverityInfo
	if !success {
		result = auditResultFailure
		severity = auditSeverityWarning
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "FILE_ACCESS",
		Action:    operation,
		Resource:  path,
		Result:    result,
		Details:   details,
		Severity:  severity,
		SessionID: al.sessionID,
		ProcessID: os.Getpid(),
	}

	al.writeEvent(event)
}

// LogNetworkAccess logs network access events
func (al *AuditLogger) LogNetworkAccess(ctx context.Context, operation string, url string, success bool, details map[string]interface{}) {
	if al.level == AuditLevelNone {
		return
	}

	result := auditResultSuccess
	severity := auditSeverityInfo
	if !success {
		result = auditResultFailure
		severity = auditSeverityWarning
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "NETWORK_ACCESS",
		Action:    operation,
		Resource:  url,
		Result:    result,
		Details:   details,
		Severity:  severity,
		SessionID: al.sessionID,
		ProcessID: os.Getpid(),
	}

	al.writeEvent(event)
}

// LogSecurityEvent logs general security events
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, eventType string, action string, resource string, success bool, details map[string]interface{}) {
	if al.level == AuditLevelNone {
		return
	}

	result := auditResultSuccess
	severity := auditSeverityInfo
	if !success {
		result = auditResultFailure
		severity = "ERROR"
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		Action:    action,
		Resource:  resource,
		Result:    result,
		Details:   details,
		Severity:  severity,
		SessionID: al.sessionID,
		ProcessID: os.Getpid(),
	}

	al.writeEvent(event)
}

// LogPermissionCheck logs permission validation events
func (al *AuditLogger) LogPermissionCheck(ctx context.Context, operation string, resource string, granted bool, reason string) {
	if al.level == AuditLevelNone || al.level == AuditLevelBasic {
		return
	}

	result := "GRANTED"
	severity := auditSeverityInfo
	if !granted {
		result = "DENIED"
		severity = auditSeverityWarning
	}

	details := map[string]interface{}{
		"reason": reason,
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "PERMISSION_CHECK",
		Action:    operation,
		Resource:  resource,
		Result:    result,
		Details:   details,
		Severity:  severity,
		SessionID: al.sessionID,
		ProcessID: os.Getpid(),
	}

	al.writeEvent(event)
}

// LogConfigurationChange logs configuration changes
func (al *AuditLogger) LogConfigurationChange(ctx context.Context, key string, oldValue interface{}, newValue interface{}) {
	if al.level == AuditLevelNone || al.level == AuditLevelBasic {
		return
	}

	details := map[string]interface{}{
		"key": key,
	}

	// Only log values in verbose mode
	if al.level == AuditLevelVerbose {
		details["old_value"] = oldValue
		details["new_value"] = newValue
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "CONFIGURATION_CHANGE",
		Action:    "update_configuration",
		Resource:  key,
		Result:    "SUCCESS",
		Details:   details,
		Severity:  "INFO",
		SessionID: al.sessionID,
		ProcessID: os.Getpid(),
	}

	al.writeEvent(event)
}

// writeEvent writes an audit event to the log file
func (al *AuditLogger) writeEvent(event *AuditEvent) {
	if al.level == AuditLevelNone || al.file == nil {
		return
	}

	al.mutex.Lock()
	defer al.mutex.Unlock()

	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		// If we can't serialize the event, write a basic error entry
		errorEvent := fmt.Sprintf(`{"timestamp":"%s","event_type":"AUDIT_ERROR","error":"failed to serialize event: %s"}%s`,
			time.Now().Format(time.RFC3339), err.Error(), "\n")
		if _, err := al.file.WriteString(errorEvent); err != nil {
			// Unable to write error event to audit log
			// This is a critical failure but we can't do much more
			return
		}
		return
	}

	// Write event with newline
	if _, err := al.file.WriteString(string(eventData) + "\n"); err != nil {
		// Critical: failed to write audit event
		// This could indicate disk space or permission issues
		return
	}
	if err := al.file.Sync(); err != nil {
		// Log sync error but continue as data may have been written
		// Sync errors can happen due to filesystem issues
		return
	}
}

// Close closes the audit logger and its resources
func (al *AuditLogger) Close() error {
	al.mutex.Lock()
	defer al.mutex.Unlock()

	if al.file != nil {
		// Log closure event
		event := &AuditEvent{
			Timestamp: time.Now(),
			EventType: "AUDIT_SESSION",
			Action:    "close_session",
			Result:    "SUCCESS",
			Severity:  "INFO",
			SessionID: al.sessionID,
			ProcessID: os.Getpid(),
		}

		eventData, err := json.Marshal(event)
		if err != nil {
			// If we can't marshal the close event, just close the file
			return al.file.Close()
		}
		if _, writeErr := al.file.WriteString(string(eventData) + "\n"); writeErr != nil {
			// Log write error but continue with close
			// File close is more important than writing final event
			// Close the file and return the close error instead
			if closeErr := al.file.Close(); closeErr != nil {
				return closeErr
			}
			return writeErr
		}

		return al.file.Close()
	}

	return nil
}

// RotateLogs rotates audit logs based on size or age
func (al *AuditLogger) RotateLogs() error {
	if al.level == AuditLevelNone || al.file == nil {
		return nil
	}

	al.mutex.Lock()
	defer al.mutex.Unlock()

	// Get file info
	fileInfo, err := al.file.Stat()
	if err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to get audit log file info").
			WithCause(err).
			WithSeverity(errors.SeverityMedium).
			WithRecoverable(true).
			Build()
	}

	// Rotate if file is larger than 100MB
	const maxFileSize = 100 * 1024 * 1024
	if fileInfo.Size() > maxFileSize {
		// Close current file
		if err := al.file.Close(); err != nil {
			// Log error but continue rotation
			// File may already be closed or have IO errors
			return err
		}

		// Rename current file with timestamp
		timestamp := time.Now().Format("20060102-150405")
		rotatedName := fmt.Sprintf("%s.%s", al.logFile, timestamp)

		if err := os.Rename(al.logFile, rotatedName); err != nil {
			return errors.NewError(errors.ErrorTypeFileSystem).
				WithMessage("failed to rotate audit log file").
				WithCause(err).
				WithSeverity(errors.SeverityMedium).
				WithRecoverable(true).
				Build()
		}

		// Create new file
		file, err := os.OpenFile(al.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return errors.NewError(errors.ErrorTypeFileSystem).
				WithMessage("failed to create new audit log file").
				WithCause(err).
				WithSeverity(errors.SeverityHigh).
				WithRecoverable(true).
				Build()
		}

		al.file = file
	}

	return nil
}

// generateSessionID generates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("ccagents-%d-%d", time.Now().Unix(), os.Getpid())
}

// AuditReporter provides audit log analysis and reporting
type AuditReporter struct {
	logDir string
}

// NewAuditReporter creates a new audit reporter
func NewAuditReporter(logDir string) *AuditReporter {
	return &AuditReporter{
		logDir: logDir,
	}
}

// GetSecuritySummary generates a security summary from audit logs
func (ar *AuditReporter) GetSecuritySummary(since time.Time) (*SecuritySummary, error) {
	summary := &SecuritySummary{
		StartTime:         since,
		EndTime:           time.Now(),
		TotalEvents:       0,
		FailedEvents:      0,
		SecurityEvents:    0,
		CommandExecutions: 0,
		FileAccesses:      0,
		NetworkAccesses:   0,
		EventsByType:      make(map[string]int),
		EventsBySeverity:  make(map[string]int),
	}

	// Read all audit log files in the directory
	files, err := filepath.Glob(filepath.Join(ar.logDir, "audit-*.log"))
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to read audit log directory").
			WithCause(err).
			WithSeverity(errors.SeverityMedium).
			WithRecoverable(true).
			Build()
	}

	for _, file := range files {
		if err := ar.processLogFile(file, since, summary); err != nil {
			// Log error but continue processing other files
			continue
		}
	}

	return summary, nil
}

// processLogFile processes a single audit log file
func (ar *AuditReporter) processLogFile(filename string, since time.Time, summary *SecuritySummary) error {
	// #nosec G304 - filename is validated and comes from trusted configuration
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // File close in defer is best effort

	decoder := json.NewDecoder(file)

	for decoder.More() {
		var event AuditEvent
		if err := decoder.Decode(&event); err != nil {
			continue // Skip malformed entries
		}

		// Filter by time
		if event.Timestamp.Before(since) {
			continue
		}

		// Update summary statistics
		summary.TotalEvents++
		summary.EventsByType[event.EventType]++
		summary.EventsBySeverity[event.Severity]++

		if event.Result == "FAILURE" {
			summary.FailedEvents++
		}

		switch event.EventType {
		case "COMMAND_EXECUTION":
			summary.CommandExecutions++
		case "FILE_ACCESS":
			summary.FileAccesses++
		case "NETWORK_ACCESS":
			summary.NetworkAccesses++
		default:
			summary.SecurityEvents++
		}
	}

	return nil
}

// SecuritySummary contains audit log analysis results
type SecuritySummary struct {
	StartTime         time.Time      `json:"start_time"`
	EndTime           time.Time      `json:"end_time"`
	TotalEvents       int            `json:"total_events"`
	FailedEvents      int            `json:"failed_events"`
	SecurityEvents    int            `json:"security_events"`
	CommandExecutions int            `json:"command_executions"`
	FileAccesses      int            `json:"file_accesses"`
	NetworkAccesses   int            `json:"network_accesses"`
	EventsByType      map[string]int `json:"events_by_type"`
	EventsBySeverity  map[string]int `json:"events_by_severity"`
}
