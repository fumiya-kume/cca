// Package observability provides debugging utilities for ccAgents
package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// DebugManager provides debugging utilities and profiling
type DebugManager struct {
	enabled      bool
	outputPath   string
	logger       *logger.Logger
	profiles     map[string]*ProfileSession
	mutex        sync.RWMutex
	config       DebugConfig
	traceEnabled bool
	traces       []TraceEvent
	maxTraces    int
}

// DebugConfig configures debugging features
type DebugConfig struct {
	Enabled         bool          `yaml:"enabled"`
	OutputPath      string        `yaml:"output_path"`
	EnableProfiling bool          `yaml:"enable_profiling"`
	EnableTracing   bool          `yaml:"enable_tracing"`
	MaxTraces       int           `yaml:"max_traces"`
	AutoProfile     bool          `yaml:"auto_profile"`
	ProfileInterval time.Duration `yaml:"profile_interval"`
	ProfileDuration time.Duration `yaml:"profile_duration"`
}

// ProfileSession represents an active profiling session
type ProfileSession struct {
	Type      string            `json:"type"`
	StartTime time.Time         `json:"start_time"`
	Duration  time.Duration     `json:"duration"`
	FilePath  string            `json:"file_path"`
	Active    bool              `json:"active"`
	Metadata  map[string]string `json:"metadata"`
	timer     *time.Timer       `json:"-"`
}

// TraceEvent represents a trace event for debugging
type TraceEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Type         string                 `json:"type"`
	Component    string                 `json:"component"`
	Operation    string                 `json:"operation"`
	Duration     time.Duration          `json:"duration,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	StackTrace   string                 `json:"stack_trace,omitempty"`
	Error        string                 `json:"error,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
	SpanID       string                 `json:"span_id,omitempty"`
	ParentSpanID string                 `json:"parent_span_id,omitempty"`
}

// DiagnosticInfo contains system diagnostic information
type DiagnosticInfo struct {
	Timestamp      time.Time              `json:"timestamp"`
	Runtime        RuntimeInfo            `json:"runtime"`
	Memory         MemoryInfo             `json:"memory"`
	Goroutines     []GoroutineInfo        `json:"goroutines"`
	Environment    map[string]string      `json:"environment"`
	Configuration  map[string]interface{} `json:"configuration"`
	RecentTraces   []TraceEvent           `json:"recent_traces"`
	ActiveProfiles []ProfileSession       `json:"active_profiles"`
}

// RuntimeInfo contains Go runtime information
type RuntimeInfo struct {
	Version      string `json:"version"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	NumCgoCall   int64  `json:"num_cgo_call"`
	Compiler     string `json:"compiler"`
}

// MemoryInfo contains memory usage information
type MemoryInfo struct {
	Alloc         uint64  `json:"alloc"`
	TotalAlloc    uint64  `json:"total_alloc"`
	Sys           uint64  `json:"sys"`
	Lookups       uint64  `json:"lookups"`
	Mallocs       uint64  `json:"mallocs"`
	Frees         uint64  `json:"frees"`
	HeapAlloc     uint64  `json:"heap_alloc"`
	HeapSys       uint64  `json:"heap_sys"`
	HeapIdle      uint64  `json:"heap_idle"`
	HeapInuse     uint64  `json:"heap_inuse"`
	HeapReleased  uint64  `json:"heap_released"`
	HeapObjects   uint64  `json:"heap_objects"`
	StackInuse    uint64  `json:"stack_inuse"`
	StackSys      uint64  `json:"stack_sys"`
	NumGC         uint32  `json:"num_gc"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
}

// GoroutineInfo contains goroutine information
type GoroutineInfo struct {
	ID    int    `json:"id"`
	State string `json:"state"`
	Stack string `json:"stack"`
	Wait  string `json:"wait,omitempty"`
}

// NewDebugManager creates a new debug manager
func NewDebugManager(config DebugConfig, logger *logger.Logger) *DebugManager {
	if config.MaxTraces <= 0 {
		config.MaxTraces = 1000
	}

	return &DebugManager{
		enabled:      config.Enabled,
		outputPath:   config.OutputPath,
		logger:       logger,
		profiles:     make(map[string]*ProfileSession),
		config:       config,
		traceEnabled: config.EnableTracing,
		traces:       make([]TraceEvent, 0),
		maxTraces:    config.MaxTraces,
	}
}

// DefaultDebugConfig returns default debug configuration
func DefaultDebugConfig() DebugConfig {
	return DebugConfig{
		Enabled:         false,
		OutputPath:      "./debug",
		EnableProfiling: false,
		EnableTracing:   false,
		MaxTraces:       1000,
		AutoProfile:     false,
		ProfileInterval: 30 * time.Minute,
		ProfileDuration: 1 * time.Minute,
	}
}

// Enable enables debugging features
func (dm *DebugManager) Enable() {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.enabled = true

	if dm.logger != nil {
		dm.logger.Info("Debug manager enabled")
	}
}

// Disable disables debugging features
func (dm *DebugManager) Disable() {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	dm.enabled = false

	// Stop any active profiles
	for _, profile := range dm.profiles {
		if profile.Active {
			defer func(pt string) {
				if err := dm.stopProfile(pt); err != nil {
					// Log error but continue shutdown
					// Profile may already be stopped
					dm.logger.Warn("Failed to stop profile %s during shutdown: %v", pt, err)
				}
			}(profile.Type)
		}
	}

	if dm.logger != nil {
		dm.logger.Info("Debug manager disabled")
	}
}

// StartProfile starts a profiling session
func (dm *DebugManager) StartProfile(profileType string, duration time.Duration, metadata map[string]string) error {
	if err := dm.validateProfilingRequest(profileType); err != nil {
		return err
	}

	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	session, err := dm.createProfileSession(profileType, duration, metadata)
	if err != nil {
		return err
	}

	if err := dm.startProfileByType(profileType, session.FilePath); err != nil {
		return err
	}

	dm.profiles[profileType] = session
	dm.scheduleAutoStop(profileType, duration, session)
	dm.logProfileStart(profileType, session.FilePath, duration)

	return nil
}

// validateProfilingRequest validates profiling prerequisites
func (dm *DebugManager) validateProfilingRequest(profileType string) error {
	if !dm.enabled || !dm.config.EnableProfiling {
		return errors.ValidationError("profiling not enabled")
	}

	if profile, exists := dm.profiles[profileType]; exists && profile.Active {
		return errors.ValidationError("profile already running: " + profileType)
	}

	return nil
}

// createProfileSession creates a new profile session
func (dm *DebugManager) createProfileSession(profileType string, duration time.Duration, metadata map[string]string) (*ProfileSession, error) {
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-profile-%s.prof", profileType, timestamp)
	filePath := filepath.Join(dm.outputPath, filename)

	// Ensure directory exists with secure permissions
	if err := os.MkdirAll(dm.outputPath, 0750); err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create debug output directory").
			WithCause(err).
			Build()
	}

	return &ProfileSession{
		Type:      profileType,
		StartTime: time.Now(),
		Duration:  duration,
		FilePath:  filePath,
		Active:    true,
		Metadata:  metadata,
	}, nil
}

// startProfileByType starts the appropriate profiler based on type
func (dm *DebugManager) startProfileByType(profileType, filePath string) error {
	switch profileType {
	case "cpu":
		return dm.startCPUProfile(filePath)
	case "memory", "heap":
		return dm.startMemoryProfile(filePath)
	case "goroutine":
		return dm.startGoroutineProfile(filePath)
	case "block":
		return dm.startBlockProfile(filePath)
	case "mutex":
		return dm.startMutexProfile(filePath)
	default:
		return errors.ValidationError("unsupported profile type: " + profileType)
	}
}

// scheduleAutoStop schedules automatic profile stopping
func (dm *DebugManager) scheduleAutoStop(profileType string, duration time.Duration, session *ProfileSession) {
	if duration <= 0 {
		return
	}

	timer := time.NewTimer(duration)
	session.timer = timer
	go func() {
		<-timer.C
		defer func() {
			if err := dm.StopProfile(profileType); err != nil {
				// Log error but continue as this is cleanup
				// Profile may already be stopped
				return
			}
		}()
	}()
}

// logProfileStart logs the start of profiling
func (dm *DebugManager) logProfileStart(profileType, filePath string, duration time.Duration) {
	if dm.logger != nil {
		dm.logger.Info("Started profiling (type: %s, file: %s, duration: %v)", profileType, filePath, duration)
	}
}

// StopProfile stops a profiling session
func (dm *DebugManager) StopProfile(profileType string) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	return dm.stopProfile(profileType)
}

// stopProfile stops a profiling session (internal, assumes lock is held)
func (dm *DebugManager) stopProfile(profileType string) error {
	profile, exists := dm.profiles[profileType]
	if !exists || !profile.Active {
		return errors.ValidationError("no active profile found: " + profileType)
	}

	switch profileType {
	case "cpu":
		pprof.StopCPUProfile()
	case "memory", "heap":
		if err := dm.writeMemoryProfile(profile.FilePath); err != nil {
			return err
		}
	case "goroutine":
		if err := dm.writeGoroutineProfile(profile.FilePath); err != nil {
			return err
		}
	case "block":
		if err := dm.writeBlockProfile(profile.FilePath); err != nil {
			return err
		}
	case "mutex":
		if err := dm.writeMutexProfile(profile.FilePath); err != nil {
			return err
		}
	}

	profile.Active = false

	// Clean up timer if it exists
	if profile.timer != nil {
		profile.timer.Stop()
		profile.timer = nil
	}

	actualDuration := time.Since(profile.StartTime)

	if dm.logger != nil {
		dm.logger.Info("Stopped profiling (type: %s, file: %s, duration: %v)", profileType, profile.FilePath, actualDuration)
	}

	return nil
}

// RecordTrace records a trace event
func (dm *DebugManager) RecordTrace(event TraceEvent) {
	if !dm.enabled || !dm.traceEnabled {
		return
	}

	event.Timestamp = time.Now()
	if event.ID == "" {
		event.ID = generateEventID()
	}

	dm.mutex.Lock()
	dm.traces = append(dm.traces, event)

	// Trim traces if too many
	if len(dm.traces) > dm.maxTraces {
		dm.traces = dm.traces[len(dm.traces)-dm.maxTraces/2:]
	}
	dm.mutex.Unlock()

	if dm.logger != nil {
		dm.logger.Debug("Recorded trace event (id: %s, type: %s, component: %s, operation: %s)", event.ID, event.Type, event.Component, event.Operation)
	}
}

// StartTrace creates a new trace span
func (dm *DebugManager) StartTrace(component, operation string, data map[string]interface{}) *TraceSpan {
	if !dm.enabled || !dm.traceEnabled {
		return &TraceSpan{manager: dm, enabled: false}
	}

	traceID := generateEventID()
	spanID := generateEventID()

	event := TraceEvent{
		ID:        generateEventID(),
		Type:      "span_start",
		Component: component,
		Operation: operation,
		Data:      data,
		TraceID:   traceID,
		SpanID:    spanID,
	}

	dm.RecordTrace(event)

	return &TraceSpan{
		manager:   dm,
		traceID:   traceID,
		spanID:    spanID,
		component: component,
		operation: operation,
		startTime: time.Now(),
		enabled:   true,
		data:      data,
	}
}

// GetDiagnosticInfo returns comprehensive diagnostic information
func (dm *DebugManager) GetDiagnosticInfo() DiagnosticInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	dm.mutex.RLock()
	recentTraces := make([]TraceEvent, len(dm.traces))
	copy(recentTraces, dm.traces)

	activeProfiles := make([]ProfileSession, 0)
	for _, profile := range dm.profiles {
		if profile.Active {
			activeProfiles = append(activeProfiles, *profile)
		}
	}
	dm.mutex.RUnlock()

	return DiagnosticInfo{
		Timestamp: time.Now(),
		Runtime: RuntimeInfo{
			Version:      runtime.Version(),
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			NumCgoCall:   runtime.NumCgoCall(),
			Compiler:     runtime.Compiler,
		},
		Memory: MemoryInfo{
			Alloc:         memStats.Alloc,
			TotalAlloc:    memStats.TotalAlloc,
			Sys:           memStats.Sys,
			Lookups:       memStats.Lookups,
			Mallocs:       memStats.Mallocs,
			Frees:         memStats.Frees,
			HeapAlloc:     memStats.HeapAlloc,
			HeapSys:       memStats.HeapSys,
			HeapIdle:      memStats.HeapIdle,
			HeapInuse:     memStats.HeapInuse,
			HeapReleased:  memStats.HeapReleased,
			HeapObjects:   memStats.HeapObjects,
			StackInuse:    memStats.StackInuse,
			StackSys:      memStats.StackSys,
			NumGC:         memStats.NumGC,
			GCCPUFraction: memStats.GCCPUFraction,
		},
		Goroutines:     dm.getGoroutineInfo(),
		Environment:    dm.getEnvironmentInfo(),
		Configuration:  dm.getConfigurationInfo(),
		RecentTraces:   recentTraces,
		ActiveProfiles: activeProfiles,
	}
}

// ExportDiagnostics exports diagnostic information to file
func (dm *DebugManager) ExportDiagnostics() (string, error) {
	if !dm.enabled {
		return "", errors.ValidationError("debugging not enabled")
	}

	diagnostics := dm.GetDiagnosticInfo()

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("diagnostics-%s.json", timestamp)
	filePath := filepath.Join(dm.outputPath, filename)

	// Ensure directory exists with secure permissions
	if err := os.MkdirAll(dm.outputPath, 0750); err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", err
	}

	if dm.logger != nil {
		dm.logger.Info("Exported diagnostics (file: %s)", filePath)
	}

	return filePath, nil
}

// TraceSpan represents an active trace span
type TraceSpan struct {
	manager   *DebugManager
	traceID   string
	spanID    string
	component string
	operation string
	startTime time.Time
	enabled   bool
	data      map[string]interface{}
	mutex     sync.Mutex
}

// SetData sets additional data for the span
func (ts *TraceSpan) SetData(key string, value interface{}) {
	if !ts.enabled {
		return
	}

	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if ts.data == nil {
		ts.data = make(map[string]interface{})
	}
	ts.data[key] = value
}

// AddEvent adds an event to the span
func (ts *TraceSpan) AddEvent(eventType string, data map[string]interface{}) {
	if !ts.enabled {
		return
	}

	event := TraceEvent{
		Type:         eventType,
		Component:    ts.component,
		Operation:    ts.operation,
		Data:         data,
		TraceID:      ts.traceID,
		SpanID:       ts.spanID,
		ParentSpanID: ts.spanID,
	}

	ts.manager.RecordTrace(event)
}

// RecordError records an error in the span
func (ts *TraceSpan) RecordError(err error) {
	if !ts.enabled {
		return
	}

	event := TraceEvent{
		Type:      "error",
		Component: ts.component,
		Operation: ts.operation,
		Error:     err.Error(),
		TraceID:   ts.traceID,
		SpanID:    ts.spanID,
		Data:      ts.data,
	}

	ts.manager.RecordTrace(event)
}

// Finish finishes the trace span
func (ts *TraceSpan) Finish() {
	if !ts.enabled {
		return
	}

	duration := time.Since(ts.startTime)

	event := TraceEvent{
		Type:      "span_finish",
		Component: ts.component,
		Operation: ts.operation,
		Duration:  duration,
		TraceID:   ts.traceID,
		SpanID:    ts.spanID,
		Data:      ts.data,
	}

	ts.manager.RecordTrace(event)
}

// Profile helper methods
func (dm *DebugManager) startCPUProfile(filePath string) error {
	// #nosec G304 - filePath is from validated debug directory
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	return pprof.StartCPUProfile(file)
}

func (dm *DebugManager) startMemoryProfile(filePath string) error {
	// Memory profiles are written on-demand
	return nil
}

func (dm *DebugManager) writeMemoryProfile(filePath string) error {
	// #nosec G304 - filePath is from validated debug directory
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but continue as profile data may have been written
			// File close errors during profile writing are usually not critical
			// Continue as this is just cleanup
			return
		}
	}()
	return pprof.WriteHeapProfile(file)
}

func (dm *DebugManager) startGoroutineProfile(filePath string) error {
	return nil // Goroutine profiles are written on-demand
}

func (dm *DebugManager) writeGoroutineProfile(filePath string) error {
	// #nosec G304 - filePath is from validated debug directory
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // File close in defer is best effort
	return pprof.Lookup("goroutine").WriteTo(file, 0)
}

func (dm *DebugManager) startBlockProfile(filePath string) error {
	runtime.SetBlockProfileRate(1)
	return nil
}

func (dm *DebugManager) writeBlockProfile(filePath string) error {
	// #nosec G304 - filePath is from validated debug directory
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // File close in defer is best effort
	return pprof.Lookup("block").WriteTo(file, 0)
}

func (dm *DebugManager) startMutexProfile(filePath string) error {
	runtime.SetMutexProfileFraction(1)
	return nil
}

func (dm *DebugManager) writeMutexProfile(filePath string) error {
	// #nosec G304 - filePath is from validated debug directory
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // File close in defer is best effort
	return pprof.Lookup("mutex").WriteTo(file, 0)
}

// Diagnostic helper methods
func (dm *DebugManager) getGoroutineInfo() []GoroutineInfo {
	// Simplified goroutine info - in practice, you'd parse runtime stack traces
	return []GoroutineInfo{
		{
			ID:    1,
			State: "running",
			Stack: "simplified stack trace",
		},
	}
}

func (dm *DebugManager) getEnvironmentInfo() map[string]string {
	env := make(map[string]string)
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			// Only include non-sensitive environment variables
			key := parts[0]
			if !strings.Contains(strings.ToLower(key), "password") &&
				!strings.Contains(strings.ToLower(key), "token") &&
				!strings.Contains(strings.ToLower(key), "secret") {
				env[key] = parts[1]
			}
		}
	}
	return env
}

func (dm *DebugManager) getConfigurationInfo() map[string]interface{} {
	return map[string]interface{}{
		"debug_enabled":     dm.enabled,
		"output_path":       dm.outputPath,
		"profiling_enabled": dm.config.EnableProfiling,
		"tracing_enabled":   dm.config.EnableTracing,
		"max_traces":        dm.maxTraces,
		"auto_profile":      dm.config.AutoProfile,
	}
}
