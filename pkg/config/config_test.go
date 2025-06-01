package config

import (
	"os"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)

// Test constants
const (
	invalidValue = "invalid"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}

	if config.Claude.Command != "claude" {
		t.Errorf("Expected claude command, got %s", config.Claude.Command)
	}

	if config.Claude.MaxInstances != 3 {
		t.Errorf("Expected 3 max instances, got %d", config.Claude.MaxInstances)
	}

	if config.Claude.Timeout != 5*time.Minute {
		t.Errorf("Expected 5 minute timeout, got %v", config.Claude.Timeout)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test valid config
	config := DefaultConfig()
	if err := config.Validate(); err != nil {
		t.Errorf("Default config should be valid, got error: %v", err)
	}

	// Test invalid claude command
	config = DefaultConfig()
	config.Claude.Command = ""
	if err := config.Validate(); err == nil {
		t.Error("Expected validation error for empty claude command")
	}

	// Test invalid max instances
	config = DefaultConfig()
	config.Claude.MaxInstances = 0
	if err := config.Validate(); err == nil {
		t.Error("Expected validation error for invalid max instances")
	}

	// Test invalid timeout
	config = DefaultConfig()
	config.Claude.Timeout = 0
	if err := config.Validate(); err == nil {
		t.Error("Expected validation error for invalid timeout")
	}

	// Test invalid commit style
	config = DefaultConfig()
	config.Workflow.CommitStyle = invalidValue
	if err := config.Validate(); err == nil {
		t.Error("Expected validation error for invalid commit style")
	}

	// Test invalid log level
	config = DefaultConfig()
	config.Logging.Level = invalidValue
	if err := config.Validate(); err == nil {
		t.Error("Expected validation error for invalid log level")
	}
}

func TestEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("CCAGENTS_CLAUDE_COMMAND", "custom-claude")
	_ = os.Setenv("CCAGENTS_LOG_LEVEL", "debug")
	_ = os.Setenv("CCAGENTS_DEBUG", "true")
	defer func() {
		_ = os.Unsetenv("CCAGENTS_CLAUDE_COMMAND")
		_ = os.Unsetenv("CCAGENTS_LOG_LEVEL")
		_ = os.Unsetenv("CCAGENTS_DEBUG")
	}()

	config := DefaultConfig()
	config.ApplyEnvironmentOverrides()

	if config.Claude.Command != "custom-claude" {
		t.Errorf("Expected custom-claude command, got %s", config.Claude.Command)
	}

	if config.Logging.Level != "debug" {
		t.Errorf("Expected debug log level, got %s", config.Logging.Level)
	}

	if !config.UI.VerboseOutput {
		t.Error("Expected verbose output to be enabled in debug mode")
	}
}

func TestGetConfigPaths(t *testing.T) {
	// Test with environment variable override
	customPath := "/custom/path/config.yaml"
	_ = os.Setenv("CCAGENTS_CONFIG", customPath)
	defer func() { _ = os.Unsetenv("CCAGENTS_CONFIG") }()

	paths := GetConfigPaths()
	if len(paths) == 0 {
		t.Fatal("Expected at least one config path")
	}

	if paths[0] != customPath {
		t.Errorf("Expected first path to be custom path %s, got %s", customPath, paths[0])
	}
}

func TestToLoggerConfig(t *testing.T) {
	config := DefaultConfig()
	config.Logging.Level = "debug"
	config.Logging.File = "/tmp/test.log"

	loggerConfig := config.ToLoggerConfig()

	if loggerConfig.Level != logger.LevelDebug {
		t.Errorf("Expected debug level, got %v", loggerConfig.Level)
	}

	if loggerConfig.LogFile != "/tmp/test.log" {
		t.Errorf("Expected log file /tmp/test.log, got %s", loggerConfig.LogFile)
	}

	if !loggerConfig.Debug {
		t.Error("Expected debug mode to be enabled")
	}

	if loggerConfig.Prefix != "ccagents" {
		t.Errorf("Expected prefix ccagents, got %s", loggerConfig.Prefix)
	}
}

func TestConfigValidationLevels(t *testing.T) {
	config := DefaultConfig()

	// Test basic validation
	validator := NewConfigValidator(ValidationLevelBasic)
	result := validator.ValidateConfig(config)

	if result.HasErrors() {
		for _, err := range result.Errors {
			t.Logf("Validation error: %s", err.Error())
		}
		t.Errorf("Basic validation should pass for default config, got %d errors", len(result.Errors))
	}

	// Test strict validation
	validator = NewConfigValidator(ValidationLevelStrict)
	result = validator.ValidateConfig(config)

	// Strict validation might have warnings but should not have errors for default config
	if result.HasErrors() {
		t.Errorf("Strict validation should pass for default config, got %d errors", len(result.Errors))
	}

	// Test complete validation
	validator = NewConfigValidator(ValidationLevelComplete)
	result = validator.ValidateConfig(config)

	// Complete validation might have warnings but should not have errors for default config
	if result.HasErrors() {
		t.Errorf("Complete validation should pass for default config, got %d errors", len(result.Errors))
	}
}

func TestConfigValidationErrors(t *testing.T) {
	validator := NewConfigValidator(ValidationLevelStrict)

	// Test empty version
	config := DefaultConfig()
	config.Version = ""
	result := validator.ValidateConfig(config)
	if !result.HasErrors() {
		t.Error("Expected validation error for empty version")
	}

	// Test invalid version format
	config = DefaultConfig()
	config.Version = invalidValue
	result = validator.ValidateConfig(config)
	if !result.HasErrors() {
		t.Error("Expected validation error for invalid version format")
	}

	// Test invalid GitHub label
	config = DefaultConfig()
	config.GitHub.DefaultLabels = []string{""}
	result = validator.ValidateConfig(config)
	if !result.HasErrors() {
		t.Error("Expected validation error for empty GitHub label")
	}

	// Test invalid theme
	config = DefaultConfig()
	config.UI.Theme = invalidValue
	result = validator.ValidateConfig(config)
	if !result.HasErrors() {
		t.Error("Expected validation error for invalid theme")
	}

	// Test invalid GitHub username
	config = DefaultConfig()
	config.GitHub.Reviewers = []string{"invalid-user-name-with-invalid-chars!@#"}
	result = validator.ValidateConfig(config)
	if !result.HasErrors() {
		t.Error("Expected validation error for invalid GitHub username")
	}
}
