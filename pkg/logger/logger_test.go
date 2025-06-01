package logger

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		hasErr bool
	}{
		{
			name: "basic configuration",
			config: Config{
				Level:     LevelInfo,
				Debug:     false,
				Timestamp: true,
				Prefix:    "test",
			},
			hasErr: false,
		},
		{
			name: "with log file",
			config: Config{
				Level:     LevelDebug,
				LogFile:   filepath.Join(t.TempDir(), "test.log"),
				Debug:     true,
				Timestamp: false,
				Prefix:    "testapp",
			},
			hasErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.hasErr {
				t.Errorf("New() error = %v, hasErr %v", err, tt.hasErr)
				return
			}
			if !tt.hasErr {
				if logger == nil {
					t.Error("New() returned nil logger")
					return
				}
				if logger.level != tt.config.Level {
					t.Errorf("New() level = %v, want %v", logger.level, tt.config.Level)
				}
				if logger.prefix != tt.config.Prefix {
					t.Errorf("New() prefix = %v, want %v", logger.prefix, tt.config.Prefix)
				}
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:     LevelInfo,
		writers:   []io.Writer{&buf},
		timestamp: false,
		prefix:    "test",
	}

	// Debug should not be logged (below threshold)
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message was logged when level is Info")
	}

	// Info should be logged
	logger.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info message was not logged")
	}
	if !strings.Contains(buf.String(), "[INFO]") {
		t.Error("Info level was not included in log")
	}

	buf.Reset()

	// Warn should be logged
	logger.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("Warn message was not logged")
	}
	if !strings.Contains(buf.String(), "[WARN]") {
		t.Error("Warn level was not included in log")
	}

	buf.Reset()

	// Error should be logged
	logger.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error message was not logged")
	}
	if !strings.Contains(buf.String(), "[ERROR]") {
		t.Error("Error level was not included in log")
	}
}

func TestLoggerWithPrefix(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:     LevelInfo,
		writers:   []io.Writer{&buf},
		timestamp: false,
		prefix:    "main",
	}

	subLogger := logger.WithPrefix("sub")
	subLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "[main:sub]") {
		t.Errorf("Expected prefix '[main:sub]' in output: %s", output)
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:     LevelInfo,
		writers:   []io.Writer{&buf},
		timestamp: false,
	}

	// Debug should not be logged initially
	logger.Debug("debug message 1")
	if buf.Len() > 0 {
		t.Error("Debug message was logged when level is Info")
	}

	// Change level to Debug
	logger.SetLevel(LevelDebug)
	logger.Debug("debug message 2")
	if !strings.Contains(buf.String(), "debug message 2") {
		t.Error("Debug message was not logged after changing level")
	}
}

func TestSetDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:     LevelInfo,
		writers:   []io.Writer{&buf},
		timestamp: false,
	}

	// Enable debug mode
	logger.SetDebug(true)
	logger.Debug("debug message")

	if !strings.Contains(buf.String(), "debug message") {
		t.Error("Debug message was not logged after enabling debug mode")
	}
	if logger.level != LevelDebug {
		t.Error("Level was not changed to Debug when SetDebug(true) was called")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test that global functions work
	var buf bytes.Buffer
	originalLogger := GetGlobalLogger()

	// Create a test logger
	testLogger := &Logger{
		level:     LevelInfo,
		writers:   []io.Writer{&buf},
		timestamp: false,
		prefix:    "global",
	}

	SetGlobalLogger(testLogger)
	Info("global test message")

	if !strings.Contains(buf.String(), "global test message") {
		t.Error("Global Info function did not use the set global logger")
	}

	// Restore original logger
	SetGlobalLogger(originalLogger)
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
