// Package observability provides telemetry and session tracking for ccAgents
package observability

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// TelemetryCollector collects anonymous usage telemetry (optional)
type TelemetryCollector struct {
	enabled       bool
	sessionID     string
	startTime     time.Time
	events        []TelemetryEvent
	mutex         sync.RWMutex
	logger        *logger.Logger
	config        TelemetryConfig
	flushInterval time.Duration
	maxEvents     int
}

// TelemetryConfig configures telemetry collection
type TelemetryConfig struct {
	Enabled           bool          `yaml:"enabled"`
	FlushInterval     time.Duration `yaml:"flush_interval"`
	MaxEvents         int           `yaml:"max_events"`
	OutputPath        string        `yaml:"output_path"`
	IncludeStackTrace bool          `yaml:"include_stack_trace"`
	AnonymizeData     bool          `yaml:"anonymize_data"`
}

// TelemetryEvent represents a telemetry event
type TelemetryEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	Data      map[string]interface{} `json:"data"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Platform  PlatformInfo           `json:"platform"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

// PlatformInfo contains platform information
type PlatformInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
}

// SessionTracker tracks user sessions and activities
type SessionTracker struct {
	sessionID    string
	startTime    time.Time
	lastActivity time.Time
	activities   []SessionActivity
	mutex        sync.RWMutex
	logger       *logger.Logger
	isActive     bool
	metadata     map[string]interface{}
}

// SessionActivity represents an activity within a session
type SessionActivity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   time.Duration          `json:"duration,omitempty"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
}

// NewTelemetryCollector creates a new telemetry collector
func NewTelemetryCollector(config TelemetryConfig, logger *logger.Logger) *TelemetryCollector {
	sessionID := generateSessionID()

	return &TelemetryCollector{
		enabled:       config.Enabled,
		sessionID:     sessionID,
		startTime:     time.Now(),
		events:        make([]TelemetryEvent, 0),
		logger:        logger,
		config:        config,
		flushInterval: config.FlushInterval,
		maxEvents:     config.MaxEvents,
	}
}

// DefaultTelemetryConfig returns default telemetry configuration
func DefaultTelemetryConfig() TelemetryConfig {
	return TelemetryConfig{
		Enabled:           false, // Disabled by default for privacy
		FlushInterval:     5 * time.Minute,
		MaxEvents:         1000,
		OutputPath:        "./telemetry",
		IncludeStackTrace: false,
		AnonymizeData:     true,
	}
}

// IsEnabled returns whether telemetry is enabled
func (tc *TelemetryCollector) IsEnabled() bool {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()
	return tc.enabled
}

// Enable enables telemetry collection
func (tc *TelemetryCollector) Enable() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.enabled = true

	if tc.logger != nil {
		tc.logger.Info("Telemetry collection enabled (session_id: %s)", tc.sessionID)
	}
}

// Disable disables telemetry collection
func (tc *TelemetryCollector) Disable() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.enabled = false

	if tc.logger != nil {
		tc.logger.Info("Telemetry collection disabled")
	}
}

// RecordEvent records a telemetry event
func (tc *TelemetryCollector) RecordEvent(eventType string, data map[string]interface{}) {
	if !tc.IsEnabled() {
		return
	}

	event := TelemetryEvent{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		SessionID: tc.sessionID,
		Data:      data,
		Platform:  getPlatformInfo(),
	}

	if tc.config.AnonymizeData {
		event.Data = tc.anonymizeData(data)
	}

	tc.mutex.Lock()
	tc.events = append(tc.events, event)

	// Trim events if too many
	if len(tc.events) > tc.maxEvents {
		tc.events = tc.events[len(tc.events)-tc.maxEvents/2:]
	}
	tc.mutex.Unlock()

	if tc.logger != nil {
		tc.logger.Debug("Recorded telemetry event (type: %s, session_id: %s)", eventType, tc.sessionID)
	}
}

// RecordTiming records a timing event
func (tc *TelemetryCollector) RecordTiming(eventType string, duration time.Duration, data map[string]interface{}) {
	if !tc.IsEnabled() {
		return
	}

	if data == nil {
		data = make(map[string]interface{})
	}
	data["duration_ms"] = duration.Milliseconds()

	tc.RecordEvent(eventType, data)
}

// RecordError records an error event
func (tc *TelemetryCollector) RecordError(eventType string, err error, data map[string]interface{}) {
	if !tc.IsEnabled() {
		return
	}

	if data == nil {
		data = make(map[string]interface{})
	}

	data["error"] = err.Error()
	data["error_type"] = fmt.Sprintf("%T", err)

	if tc.config.IncludeStackTrace {
		data["stack_trace"] = getStackTrace()
	}

	tc.RecordEvent(eventType, data)
}

// GetEvents returns all collected events
func (tc *TelemetryCollector) GetEvents() []TelemetryEvent {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	events := make([]TelemetryEvent, len(tc.events))
	copy(events, tc.events)
	return events
}

// FlushEvents exports and clears collected events
func (tc *TelemetryCollector) FlushEvents() error {
	if !tc.IsEnabled() {
		return nil
	}

	tc.mutex.Lock()
	events := make([]TelemetryEvent, len(tc.events))
	copy(events, tc.events)
	tc.events = tc.events[:0] // Clear events
	tc.mutex.Unlock()

	if len(events) == 0 {
		return nil
	}

	// Export events
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("telemetry-%s-%s.json", tc.sessionID, timestamp)
	filepath := filepath.Join(tc.config.OutputPath, filename)

	// Ensure directory exists
	if err := os.MkdirAll(tc.config.OutputPath, 0750); err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create telemetry directory").
			WithCause(err).
			Build()
	}

	// Export data
	exportData := map[string]interface{}{
		"session_id":  tc.sessionID,
		"start_time":  tc.startTime,
		"export_time": time.Now(),
		"event_count": len(events),
		"events":      events,
		"platform":    getPlatformInfo(),
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return err
	}

	if tc.logger != nil {
		tc.logger.Debug("Flushed telemetry events (count: %d, file: %s)", len(events), filepath)
	}

	return nil
}

// GetSummary returns telemetry summary
func (tc *TelemetryCollector) GetSummary() map[string]interface{} {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	return map[string]interface{}{
		"enabled":        tc.enabled,
		"session_id":     tc.sessionID,
		"start_time":     tc.startTime,
		"uptime":         time.Since(tc.startTime),
		"event_count":    len(tc.events),
		"max_events":     tc.maxEvents,
		"flush_interval": tc.flushInterval,
		"platform":       getPlatformInfo(),
	}
}

// anonymizeData anonymizes sensitive data in telemetry
func (tc *TelemetryCollector) anonymizeData(data map[string]interface{}) map[string]interface{} {
	anonymized := make(map[string]interface{})

	sensitiveKeys := []string{
		"password", "token", "key", "secret", "auth", "credential",
		"email", "username", "user", "name", "path", "file",
	}

	for key, value := range data {
		keyLower := strings.ToLower(key)
		isSensitive := false

		for _, sensitive := range sensitiveKeys {
			if strings.Contains(keyLower, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			anonymized[key] = "[REDACTED]"
		} else {
			anonymized[key] = value
		}
	}

	return anonymized
}

// NewSessionTracker creates a new session tracker
func NewSessionTracker(logger *logger.Logger) *SessionTracker {
	return &SessionTracker{
		sessionID:    generateSessionID(),
		startTime:    time.Now(),
		lastActivity: time.Now(),
		activities:   make([]SessionActivity, 0),
		logger:       logger,
		isActive:     true,
		metadata:     make(map[string]interface{}),
	}
}

// GetSessionID returns the session ID
func (st *SessionTracker) GetSessionID() string {
	st.mutex.RLock()
	defer st.mutex.RUnlock()
	return st.sessionID
}

// SetMetadata sets session metadata
func (st *SessionTracker) SetMetadata(key string, value interface{}) {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	st.metadata[key] = value
	st.lastActivity = time.Now()
}

// RecordActivity records a session activity
func (st *SessionTracker) RecordActivity(activityType string, success bool, err error, metadata map[string]interface{}) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	activity := SessionActivity{
		ID:        generateEventID(),
		Type:      activityType,
		Timestamp: time.Now(),
		Success:   success,
		Metadata:  metadata,
	}

	if err != nil {
		activity.Error = err.Error()
	}

	st.activities = append(st.activities, activity)
	st.lastActivity = time.Now()

	if st.logger != nil {
		st.logger.Debug("Recorded session activity (session_id: %s, type: %s, success: %t)", st.sessionID, activityType, success)
	}
}

// RecordTimedActivity records an activity with duration
func (st *SessionTracker) RecordTimedActivity(activityType string, duration time.Duration, success bool, err error, metadata map[string]interface{}) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	activity := SessionActivity{
		ID:        generateEventID(),
		Type:      activityType,
		Timestamp: time.Now(),
		Duration:  duration,
		Success:   success,
		Metadata:  metadata,
	}

	if err != nil {
		activity.Error = err.Error()
	}

	st.activities = append(st.activities, activity)
	st.lastActivity = time.Now()

	if st.logger != nil {
		st.logger.Debug("Recorded timed session activity (session_id: %s, type: %s, duration: %v, success: %t)", st.sessionID, activityType, duration, success)
	}
}

// GetSessionSummary returns session summary
func (st *SessionTracker) GetSessionSummary() map[string]interface{} {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	// Calculate statistics
	totalActivities := len(st.activities)
	successfulActivities := 0
	totalDuration := time.Duration(0)

	activityTypes := make(map[string]int)

	for _, activity := range st.activities {
		if activity.Success {
			successfulActivities++
		}
		if activity.Duration > 0 {
			totalDuration += activity.Duration
		}
		activityTypes[activity.Type]++
	}

	successRate := 0.0
	if totalActivities > 0 {
		successRate = float64(successfulActivities) / float64(totalActivities) * 100
	}

	return map[string]interface{}{
		"session_id":            st.sessionID,
		"start_time":            st.startTime,
		"last_activity":         st.lastActivity,
		"duration":              time.Since(st.startTime),
		"is_active":             st.isActive,
		"total_activities":      totalActivities,
		"successful_activities": successfulActivities,
		"success_rate":          successRate,
		"total_duration":        totalDuration,
		"activity_types":        activityTypes,
		"metadata":              st.metadata,
	}
}

// EndSession ends the current session
func (st *SessionTracker) EndSession() {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.isActive = false

	if st.logger != nil {
		st.logger.Info("Session ended (session_id: %s, duration: %v, activities: %d)", st.sessionID, time.Since(st.startTime), len(st.activities))
	}
}

// Utility functions
func generateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// If crypto rand fails, use time-based fallback
		return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixMicro())
	}
	return hex.EncodeToString(bytes)
}

func generateEventID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// If crypto rand fails, use time-based fallback
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func getPlatformInfo() PlatformInfo {
	return PlatformInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
	}
}

func getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}
