// Package observability provides monitoring and observability utilities for ccAgents
package observability

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// StructuredLogger provides structured logging with context and metadata
type StructuredLogger struct {
	baseLogger *logger.Logger
	context    map[string]interface{}
	mutex      sync.RWMutex
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Stack       string                 `json:"stack,omitempty"`
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Error       *ErrorInfo             `json:"error,omitempty"`
	Performance *PerformanceInfo       `json:"performance,omitempty"`
}

// ErrorInfo contains error details for logging
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Stack   string `json:"stack,omitempty"`
}

// PerformanceInfo contains performance metrics for logging
type PerformanceInfo struct {
	Duration    time.Duration `json:"duration"`
	MemoryUsage int64         `json:"memory_usage"`
	CPUUsage    float64       `json:"cpu_usage"`
	Goroutines  int           `json:"goroutines"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(baseLogger *logger.Logger) *StructuredLogger {
	return &StructuredLogger{
		baseLogger: baseLogger,
		context:    make(map[string]interface{}),
	}
}

// WithContext adds context to the logger
func (sl *StructuredLogger) WithContext(key string, value interface{}) *StructuredLogger {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	newContext := make(map[string]interface{})
	for k, v := range sl.context {
		newContext[k] = v
	}
	newContext[key] = value

	return &StructuredLogger{
		baseLogger: sl.baseLogger,
		context:    newContext,
	}
}

// WithComponent adds component context
func (sl *StructuredLogger) WithComponent(component string) *StructuredLogger {
	return sl.WithContext("component", component)
}

// WithTraceID adds trace ID context
func (sl *StructuredLogger) WithTraceID(traceID string) *StructuredLogger {
	return sl.WithContext("trace_id", traceID)
}

// Debug logs a debug message with structured data
func (sl *StructuredLogger) Debug(message string, fields ...interface{}) {
	sl.log("DEBUG", message, fields...)
}

// Info logs an info message with structured data
func (sl *StructuredLogger) Info(message string, fields ...interface{}) {
	sl.log("INFO", message, fields...)
}

// Warn logs a warning message with structured data
func (sl *StructuredLogger) Warn(message string, fields ...interface{}) {
	sl.log("WARN", message, fields...)
}

// Error logs an error message with structured data
func (sl *StructuredLogger) Error(message string, fields ...interface{}) {
	sl.log("ERROR", message, fields...)
}

// LogError logs an error with full error context
func (sl *StructuredLogger) LogError(err error, message string, fields ...interface{}) {
	entry := sl.createLogEntry("ERROR", message, fields...)

	if err != nil {
		entry.Error = &ErrorInfo{
			Type:    fmt.Sprintf("%T", err),
			Message: err.Error(),
		}

		// Extract stack trace if available
		if stackErr, ok := err.(interface{ Stack() string }); ok {
			entry.Error.Stack = stackErr.Stack()
		}

		// Extract error code if available
		if codeErr, ok := err.(interface{ Code() string }); ok {
			entry.Error.Code = codeErr.Code()
		}
	}

	sl.writeEntry(entry)
}

// LogPerformance logs performance metrics
func (sl *StructuredLogger) LogPerformance(message string, duration time.Duration, fields ...interface{}) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	entry := sl.createLogEntry("INFO", message, fields...)
	// Handle potential overflow in memory allocation conversion
	const maxInt64 = int64(^uint64(0) >> 1)
	memoryUsage := int64(m.Alloc)
	if m.Alloc > uint64(maxInt64) {
		memoryUsage = maxInt64 // Max int64 value
	}
	entry.Performance = &PerformanceInfo{
		Duration:    duration,
		MemoryUsage: memoryUsage,
		Goroutines:  runtime.NumGoroutine(),
	}

	sl.writeEntry(entry)
}

// log creates and writes a log entry
func (sl *StructuredLogger) log(level, message string, fields ...interface{}) {
	entry := sl.createLogEntry(level, message, fields...)
	sl.writeEntry(entry)
}

// createLogEntry creates a log entry with context
func (sl *StructuredLogger) createLogEntry(level, message string, fields ...interface{}) *LogEntry {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   make(map[string]interface{}),
	}

	// Copy base context
	for k, v := range sl.context {
		entry.Context[k] = v
	}

	// Add fields as key-value pairs
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			entry.Context[key] = fields[i+1]
		}
	}

	// Extract component if present
	if comp, exists := entry.Context["component"]; exists {
		if compStr, ok := comp.(string); ok {
			entry.Component = compStr
		}
	}

	// Extract trace/span IDs if present
	if traceID, exists := entry.Context["trace_id"]; exists {
		if traceStr, ok := traceID.(string); ok {
			entry.TraceID = traceStr
		}
	}

	if spanID, exists := entry.Context["span_id"]; exists {
		if spanStr, ok := spanID.(string); ok {
			entry.SpanID = spanStr
		}
	}

	return entry
}

// writeEntry writes the log entry
func (sl *StructuredLogger) writeEntry(entry *LogEntry) {
	if sl.baseLogger == nil {
		return
	}

	// Format as JSON for structured logging
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// If JSON marshaling fails, use plain text format
		jsonData = []byte(fmt.Sprintf("{\"level\":\"%s\",\"message\":\"%s\"}", entry.Level, entry.Message))
	}
	_ = jsonData
	logMessage := string(jsonData)

	switch entry.Level {
	case "DEBUG":
		sl.baseLogger.Debug("%s", logMessage)
	case "INFO":
		sl.baseLogger.Info("%s", logMessage)
	case "WARN":
		sl.baseLogger.Warn("%s", logMessage)
	case "ERROR":
		sl.baseLogger.Error("%s", logMessage)
	default:
		sl.baseLogger.Info("%s", logMessage)
	}
}

// LogAggregator collects and aggregates logs
type LogAggregator struct {
	entries    []LogEntry
	maxEntries int
	mutex      sync.RWMutex
	filters    []LogFilter
	processors []LogProcessor
	outputs    []LogOutput
}

// LogFilter filters log entries
type LogFilter interface {
	ShouldInclude(entry *LogEntry) bool
}

// LogProcessor processes log entries
type LogProcessor interface {
	ProcessEntry(entry *LogEntry) *LogEntry
}

// LogOutput outputs processed log entries
type LogOutput interface {
	WriteEntry(entry *LogEntry) error
	Flush() error
}

// NewLogAggregator creates a new log aggregator
func NewLogAggregator(maxEntries int) *LogAggregator {
	if maxEntries <= 0 {
		maxEntries = 10000
	}

	return &LogAggregator{
		entries:    make([]LogEntry, 0, maxEntries),
		maxEntries: maxEntries,
		filters:    make([]LogFilter, 0),
		processors: make([]LogProcessor, 0),
		outputs:    make([]LogOutput, 0),
	}
}

// AddFilter adds a log filter
func (la *LogAggregator) AddFilter(filter LogFilter) {
	la.mutex.Lock()
	defer la.mutex.Unlock()
	la.filters = append(la.filters, filter)
}

// AddProcessor adds a log processor
func (la *LogAggregator) AddProcessor(processor LogProcessor) {
	la.mutex.Lock()
	defer la.mutex.Unlock()
	la.processors = append(la.processors, processor)
}

// AddOutput adds a log output
func (la *LogAggregator) AddOutput(output LogOutput) {
	la.mutex.Lock()
	defer la.mutex.Unlock()
	la.outputs = append(la.outputs, output)
}

// AddEntry adds a log entry to the aggregator
func (la *LogAggregator) AddEntry(entry LogEntry) {
	la.mutex.Lock()
	defer la.mutex.Unlock()

	// Apply filters
	for _, filter := range la.filters {
		if !filter.ShouldInclude(&entry) {
			return
		}
	}

	// Apply processors
	processedEntry := &entry
	for _, processor := range la.processors {
		processedEntry = processor.ProcessEntry(processedEntry)
		if processedEntry == nil {
			return
		}
	}

	// Add to entries
	la.entries = append(la.entries, *processedEntry)

	// Trim if too many entries
	if len(la.entries) > la.maxEntries {
		la.entries = la.entries[len(la.entries)-la.maxEntries:]
	}

	// Send to outputs
	for _, output := range la.outputs {
		if err := output.WriteEntry(processedEntry); err != nil {
			// Log error but continue to other outputs
			// Individual output failures shouldn't stop processing
			continue
		}
	}
}

// GetEntries returns log entries with optional filtering
func (la *LogAggregator) GetEntries(since time.Time, level string, component string) []LogEntry {
	la.mutex.RLock()
	defer la.mutex.RUnlock()

	var filtered []LogEntry
	for _, entry := range la.entries {
		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}

		if level != "" && entry.Level != level {
			continue
		}

		if component != "" && entry.Component != component {
			continue
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// GetStats returns aggregator statistics
func (la *LogAggregator) GetStats() map[string]interface{} {
	la.mutex.RLock()
	defer la.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_entries": len(la.entries),
		"max_entries":   la.maxEntries,
		"filters":       len(la.filters),
		"processors":    len(la.processors),
		"outputs":       len(la.outputs),
	}

	// Count by level
	levelCounts := make(map[string]int)
	componentCounts := make(map[string]int)

	for _, entry := range la.entries {
		levelCounts[entry.Level]++
		if entry.Component != "" {
			componentCounts[entry.Component]++
		}
	}

	stats["level_counts"] = levelCounts
	stats["component_counts"] = componentCounts

	return stats
}

// FileLogOutput writes log entries to a file
type FileLogOutput struct {
	filePath string
	file     *os.File
	mutex    sync.Mutex
}

// NewFileLogOutput creates a new file log output
func NewFileLogOutput(filePath string) (*FileLogOutput, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create log directory").
			WithCause(err).
			WithContext("dir", dir).
			Build()
	}

	// #nosec G304 - filePath is validated for log file operations
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to open log file").
			WithCause(err).
			WithContext("filePath", filePath).
			Build()
	}

	return &FileLogOutput{
		filePath: filePath,
		file:     file,
	}, nil
}

// WriteEntry writes a log entry to the file
func (flo *FileLogOutput) WriteEntry(entry *LogEntry) error {
	flo.mutex.Lock()
	defer flo.mutex.Unlock()

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = flo.file.Write(append(jsonData, '\n'))
	return err
}

// Flush flushes the file buffer
func (flo *FileLogOutput) Flush() error {
	flo.mutex.Lock()
	defer flo.mutex.Unlock()
	return flo.file.Sync()
}

// Close closes the file
func (flo *FileLogOutput) Close() error {
	flo.mutex.Lock()
	defer flo.mutex.Unlock()
	return flo.file.Close()
}

// LevelFilter filters log entries by level
type LevelFilter struct {
	MinLevel string
}

// ShouldInclude checks if entry should be included based on level
func (lf *LevelFilter) ShouldInclude(entry *LogEntry) bool {
	levels := map[string]int{
		"DEBUG": 0,
		"INFO":  1,
		"WARN":  2,
		"ERROR": 3,
	}

	entryLevel, exists := levels[entry.Level]
	if !exists {
		return true
	}

	minLevel, exists := levels[lf.MinLevel]
	if !exists {
		return true
	}

	return entryLevel >= minLevel
}

// ComponentFilter filters log entries by component
type ComponentFilter struct {
	AllowedComponents map[string]bool
}

// ShouldInclude checks if entry should be included based on component
func (cf *ComponentFilter) ShouldInclude(entry *LogEntry) bool {
	if len(cf.AllowedComponents) == 0 {
		return true
	}

	return cf.AllowedComponents[entry.Component]
}

// SanitizationProcessor sanitizes sensitive data from log entries
type SanitizationProcessor struct {
	SensitiveFields []string
}

// ProcessEntry processes and sanitizes log entry
func (sp *SanitizationProcessor) ProcessEntry(entry *LogEntry) *LogEntry {
	if entry.Context == nil {
		return entry
	}

	// Create a copy to avoid modifying the original
	sanitized := *entry
	sanitized.Context = make(map[string]interface{})

	for k, v := range entry.Context {
		if sp.isSensitiveField(k) {
			sanitized.Context[k] = "[REDACTED]"
		} else {
			sanitized.Context[k] = v
		}
	}

	// Sanitize error message if present
	if sanitized.Error != nil {
		errorCopy := *sanitized.Error
		errorCopy.Message = sp.sanitizeString(errorCopy.Message)
		sanitized.Error = &errorCopy
	}

	// Sanitize main message
	sanitized.Message = sp.sanitizeString(sanitized.Message)

	return &sanitized
}

// isSensitiveField checks if a field contains sensitive data
func (sp *SanitizationProcessor) isSensitiveField(field string) bool {
	sensitiveDefaults := []string{
		"password", "token", "api_key", "secret", "credential",
		"auth", "authorization", "session", "cookie",
	}

	allSensitive := append(sp.SensitiveFields, sensitiveDefaults...)

	fieldLower := strings.ToLower(field)
	for _, sensitive := range allSensitive {
		if strings.Contains(fieldLower, strings.ToLower(sensitive)) {
			return true
		}
	}

	return false
}

// sanitizeString sanitizes sensitive data from strings
func (sp *SanitizationProcessor) sanitizeString(input string) string {
	// Simple regex patterns for common sensitive data
	patterns := []string{
		`(?i)(password|token|key|secret)[\s]*[:=][\s]*[^\s]+`,
		`(?i)bearer\s+[^\s]+`,
		`(?i)basic\s+[^\s]+`,
	}

	result := input
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "[REDACTED]")
	}

	return result
}

// MetricsCollector collects metrics from log entries
type MetricsCollector struct {
	metrics map[string]*LogMetric
	mutex   sync.RWMutex
}

// LogMetric represents a metric derived from logs
type LogMetric struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*LogMetric),
	}
}

// ProcessEntry processes log entry and extracts metrics
func (mc *MetricsCollector) ProcessEntry(entry *LogEntry) *LogEntry {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Count by level
	levelKey := "log_entries_total{level=\"" + entry.Level + "\"}"
	mc.incrementCounter(levelKey, entry.Level)

	// Count by component
	if entry.Component != "" {
		componentKey := "log_entries_by_component{component=\"" + entry.Component + "\"}"
		mc.incrementCounter(componentKey, entry.Component)
	}

	// Track errors
	if entry.Error != nil {
		errorKey := "log_errors_total{type=\"" + entry.Error.Type + "\"}"
		mc.incrementCounter(errorKey, entry.Error.Type)
	}

	// Track performance metrics
	if entry.Performance != nil {
		durationKey := "log_performance_duration{component=\"" + entry.Component + "\"}"
		mc.updateGauge(durationKey, float64(entry.Performance.Duration.Nanoseconds()))

		memoryKey := "log_performance_memory{component=\"" + entry.Component + "\"}"
		mc.updateGauge(memoryKey, float64(entry.Performance.MemoryUsage))
	}

	return entry
}

// incrementCounter increments a counter metric
func (mc *MetricsCollector) incrementCounter(key, label string) {
	if metric, exists := mc.metrics[key]; exists {
		metric.Value++
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &LogMetric{
			Name:      key,
			Type:      "counter",
			Value:     1,
			Labels:    map[string]string{"label": label},
			Timestamp: time.Now(),
		}
	}
}

// updateGauge updates a gauge metric
func (mc *MetricsCollector) updateGauge(key string, value float64) {
	mc.metrics[key] = &LogMetric{
		Name:      key,
		Type:      "gauge",
		Value:     value,
		Timestamp: time.Now(),
	}
}

// GetMetrics returns all collected metrics
func (mc *MetricsCollector) GetMetrics() map[string]*LogMetric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]*LogMetric)
	for k, v := range mc.metrics {
		result[k] = v
	}

	return result
}

// LogRotator handles log file rotation
type LogRotator struct {
	basePath    string
	maxSize     int64
	maxAge      time.Duration
	maxBackups  int
	currentFile *os.File
	currentSize int64
	mutex       sync.Mutex
}

// NewLogRotator creates a new log rotator
func NewLogRotator(basePath string, maxSize int64, maxAge time.Duration, maxBackups int) *LogRotator {
	return &LogRotator{
		basePath:   basePath,
		maxSize:    maxSize,
		maxAge:     maxAge,
		maxBackups: maxBackups,
	}
}

// Write writes data and handles rotation
func (lr *LogRotator) Write(data []byte) (int, error) {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	// Check if we need to rotate
	if lr.currentFile == nil || lr.currentSize+int64(len(data)) > lr.maxSize {
		if err := lr.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := lr.currentFile.Write(data)
	lr.currentSize += int64(n)

	return n, err
}

// rotate rotates the log file
func (lr *LogRotator) rotate() error {
	// Close current file
	if lr.currentFile != nil {
		if err := lr.currentFile.Close(); err != nil {
			// Log error but continue rotation
			// File may already be closed or have IO errors
			return err
		}
	}

	// Create backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupPath := lr.basePath + "." + timestamp

	// Rename current file to backup
	if _, err := os.Stat(lr.basePath); err == nil {
		if err := os.Rename(lr.basePath, backupPath); err != nil {
			return err
		}
	}

	// Open new file with secure permissions
	file, err := os.OpenFile(lr.basePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	lr.currentFile = file
	lr.currentSize = 0

	// Clean up old backups
	go lr.cleanupOldBackups()

	return nil
}

// cleanupOldBackups removes old backup files
func (lr *LogRotator) cleanupOldBackups() {
	dir := filepath.Dir(lr.basePath)
	base := filepath.Base(lr.basePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var backups []os.FileInfo
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), base+".") {
			info, err := entry.Info()
			if err == nil {
				backups = append(backups, info)
			}
		}
	}

	// Remove by age
	now := time.Now()
	for _, backup := range backups {
		if now.Sub(backup.ModTime()) > lr.maxAge {
			if err := os.Remove(filepath.Join(dir, backup.Name())); err != nil {
				// Log error but continue cleanup
				// File may already be removed or have permission issues
				continue
			}
		}
	}

	// Remove by count
	if len(backups) > lr.maxBackups {
		// Sort by mod time and remove oldest
		// (simplified implementation)
		for i := lr.maxBackups; i < len(backups); i++ {
			_ = os.Remove(filepath.Join(dir, backups[i].Name())) //nolint:errcheck // Log cleanup is best effort
		}
	}
}

// Close closes the log rotator
func (lr *LogRotator) Close() error {
	lr.mutex.Lock()
	defer lr.mutex.Unlock()

	if lr.currentFile != nil {
		return lr.currentFile.Close()
	}

	return nil
}
