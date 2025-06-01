// Package logger provides logging utilities and structured logging functionality for ccAgents
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Level represents the logging level
type Level int

// LoggerInterface defines the logging interface
type LoggerInterface interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger represents a structured logger implementation
type Logger struct {
	level     Level
	writers   []io.Writer
	prefix    string
	debugMode bool
	timestamp bool
}

// Config holds logger configuration
type Config struct {
	Level     Level
	LogFile   string
	Debug     bool
	Timestamp bool
	Prefix    string
}

// New creates a new logger with the given configuration
func New(config Config) (*Logger, error) {
	writers := []io.Writer{}

	// Don't write to stdout during tests
	if !testing.Testing() {
		writers = append(writers, os.Stdout)
	}

	logger := &Logger{
		level:     config.Level,
		prefix:    config.Prefix,
		debugMode: config.Debug,
		timestamp: config.Timestamp,
		writers:   writers,
	}

	// Add file writer if log file is specified
	if config.LogFile != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(config.LogFile)
		if err := os.MkdirAll(logDir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
		}

		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", config.LogFile, err)
		}

		logger.writers = append(logger.writers, file)
	}

	return logger, nil
}

// NewDefault creates a logger with default settings
func NewDefault() *Logger {
	logger, _ := New(Config{ //nolint:errcheck // Default logger creation should not fail with valid config
		Level:     LevelInfo,
		Debug:     false,
		Timestamp: true,
		Prefix:    "ccagents",
	})
	return logger
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetDebug enables or disables debug mode
func (l *Logger) SetDebug(debug bool) {
	l.debugMode = debug
	if debug {
		l.level = LevelDebug
	}
}

// log writes a log message at the specified level
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	message := fmt.Sprintf(format, args...)

	var parts []string

	// Add timestamp if enabled
	if l.timestamp {
		parts = append(parts, time.Now().Format("2006-01-02 15:04:05"))
	}

	// Add level
	parts = append(parts, fmt.Sprintf("[%s]", level.String()))

	// Add prefix if set
	if l.prefix != "" {
		parts = append(parts, fmt.Sprintf("[%s]", l.prefix))
	}

	// Add message
	parts = append(parts, message)

	logLine := strings.Join(parts, " ") + "\n"

	// Write to all configured writers
	for _, writer := range l.writers {
		_, _ = writer.Write([]byte(logLine)) //nolint:errcheck // Logging output errors are not critical
	}
}

// GetLogger returns a default logger instance
func GetLogger() *Logger {
	return NewDefault()
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// WithPrefix creates a new logger with an additional prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	newLogger := *l
	if l.prefix != "" {
		newLogger.prefix = l.prefix + ":" + prefix
	} else {
		newLogger.prefix = prefix
	}
	return &newLogger
}

// Global logger instance
var globalLogger = NewDefault()

// Debug logs a debug message using the global logger
func Debug(format string, args ...interface{}) {
	globalLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	globalLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	globalLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	globalLogger.Error(format, args...)
}

func SetLevel(level Level) {
	globalLogger.SetLevel(level)
}

func SetDebug(debug bool) {
	globalLogger.SetDebug(debug)
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}
