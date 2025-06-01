// Package config provides configuration management and settings for ccAgents
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)

// Log level constants
const (
	logLevelDebug = "debug"
)

// ValidationLevel represents the level of configuration validation
type ValidationLevel int

const (
	ValidationLevelBasic ValidationLevel = iota
	ValidationLevelStrict
	ValidationLevelComplete
)

// ConfigValidator validates configuration
type ConfigValidator struct {
	level ValidationLevel
}

// ConfigValidationResult contains validation results
type ConfigValidationResult struct {
	Errors   []error
	Warnings []string
}

// HasErrors returns true if there are validation errors
func (cvr *ConfigValidationResult) HasErrors() bool {
	return len(cvr.Errors) > 0
}

// NewConfigValidator creates a new config validator
func NewConfigValidator(level ValidationLevel) *ConfigValidator {
	return &ConfigValidator{level: level}
}

// ValidateConfig validates a configuration
func (cv *ConfigValidator) ValidateConfig(config *Config) *ConfigValidationResult {
	result := &ConfigValidationResult{
		Errors:   []error{},
		Warnings: []string{},
	}

	if err := cv.validateBasicConfig(config); err != nil {
		result.Errors = append(result.Errors, err)
		return result
	}

	cv.validateVersion(config, result)
	cv.validateGitHubConfig(config, result)
	cv.validateUIConfig(config, result)
	cv.validateStrictLevel(config, result)
	cv.validateCompleteLevel(config, result)

	return result
}

// validateBasicConfig performs basic null checks
func (cv *ConfigValidator) validateBasicConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	return nil
}

// validateVersion validates the configuration version
func (cv *ConfigValidator) validateVersion(config *Config, result *ConfigValidationResult) {
	if config.Version == "" {
		result.Errors = append(result.Errors, fmt.Errorf("version cannot be empty"))
		return
	}

	validVersions := map[string]bool{"1.0": true, "2.0": true}
	if !validVersions[config.Version] {
		result.Errors = append(result.Errors, fmt.Errorf("invalid version format: %s", config.Version))
	}
}

// validateGitHubConfig validates GitHub-related configuration
func (cv *ConfigValidator) validateGitHubConfig(config *Config, result *ConfigValidationResult) {
	cv.validateGitHubLabels(config, result)
	cv.validateGitHubReviewers(config, result)
}

// validateGitHubLabels validates GitHub default labels
func (cv *ConfigValidator) validateGitHubLabels(config *Config, result *ConfigValidationResult) {
	for _, label := range config.GitHub.DefaultLabels {
		if label == "" {
			result.Errors = append(result.Errors, fmt.Errorf("GitHub label cannot be empty"))
			return
		}
	}
}

// validateGitHubReviewers validates GitHub reviewer usernames
func (cv *ConfigValidator) validateGitHubReviewers(config *Config, result *ConfigValidationResult) {
	for _, reviewer := range config.GitHub.Reviewers {
		if err := cv.validateUsername(reviewer); err != nil {
			result.Errors = append(result.Errors, err)
			return
		}
	}
}

// validateUsername validates a GitHub username
func (cv *ConfigValidator) validateUsername(username string) error {
	if username == "" || len(username) > 39 {
		return fmt.Errorf("invalid GitHub username: %s", username)
	}

	invalidChars := "!@#$%^&*()=+[]{}|\\:;\"'<>?,"
	for _, char := range invalidChars {
		if contains(username, string(char)) {
			return fmt.Errorf("invalid GitHub username: %s", username)
		}
	}

	return nil
}

// validateUIConfig validates UI configuration
func (cv *ConfigValidator) validateUIConfig(config *Config, result *ConfigValidationResult) {
	validThemes := map[string]bool{"dark": true, "light": true, "auto": true}
	if !validThemes[config.UI.Theme] {
		result.Errors = append(result.Errors, fmt.Errorf("invalid theme: %s", config.UI.Theme))
	}
}

// validateStrictLevel performs strict-level validation
func (cv *ConfigValidator) validateStrictLevel(config *Config, result *ConfigValidationResult) {
	if cv.level < ValidationLevelStrict {
		return
	}

	if config.Claude.Command == "" {
		result.Warnings = append(result.Warnings, "Claude command not specified")
	}
}

// validateCompleteLevel performs complete-level validation
func (cv *ConfigValidator) validateCompleteLevel(config *Config, result *ConfigValidationResult) {
	if cv.level < ValidationLevelComplete {
		return
	}

	if config.Workflow.ParallelTasks {
		result.Warnings = append(result.Warnings, "Workflow parallel tasks enabled")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Config represents the application configuration
type Config struct {
	Version string `yaml:"version"`

	Claude ClaudeConfig `yaml:"claude"`
	GitHub GitHubConfig `yaml:"github"`
	UI     UIConfig     `yaml:"ui"`

	Workflow    WorkflowConfig    `yaml:"workflow"`
	Development DevelopmentConfig `yaml:"development"`
	Logging     LoggingConfig     `yaml:"logging"`
}

// ClaudeConfig holds Claude Code integration settings
type ClaudeConfig struct {
	Command      string        `yaml:"command"`
	Timeout      time.Duration `yaml:"timeout"`
	MaxInstances int           `yaml:"max_instances"`
	AutoStart    bool          `yaml:"auto_start"`
	Args         []string      `yaml:"args"`
}

// GitHubConfig holds GitHub integration settings
type GitHubConfig struct {
	DefaultLabels []string `yaml:"default_labels"`
	DraftPR       bool     `yaml:"draft_pr"`
	AutoMerge     bool     `yaml:"auto_merge"`
	Reviewers     []string `yaml:"reviewers"`
	Assignees     []string `yaml:"assignees"`
}

// UIConfig holds user interface settings
type UIConfig struct {
	Theme           string      `yaml:"theme"`
	ShowTimestamps  bool        `yaml:"show_timestamps"`
	VerboseOutput   bool        `yaml:"verbose_output"`
	ViewportBuffer  int         `yaml:"viewport_buffer"`
	RefreshInterval int         `yaml:"refresh_interval"`
	Sound           SoundConfig `yaml:"sound"`
}

// SoundConfig holds sound notification settings
type SoundConfig struct {
	Enabled               bool `yaml:"enabled"`
	ConfirmationSound     bool `yaml:"confirmation_sound"`
	SuccessSound          bool `yaml:"success_sound"`
	ErrorSound            bool `yaml:"error_sound"`
	WorkflowCompleteSound bool `yaml:"workflow_complete_sound"`
}

// WorkflowConfig holds workflow execution settings
type WorkflowConfig struct {
	AutoReview          bool   `yaml:"auto_review"`
	MaxReviewIterations int    `yaml:"max_review_iterations"`
	CommitStyle         string `yaml:"commit_style"`
	ParallelTasks       bool   `yaml:"parallel_tasks"`
	SkipCI              bool   `yaml:"skip_ci"`
}

// DevelopmentConfig holds development and debugging settings
type DevelopmentConfig struct {
	WorktreeBase  string `yaml:"worktree_base"`
	KeepWorktrees bool   `yaml:"keep_worktrees"`
	MaxWorktrees  int    `yaml:"max_worktrees"`
	CleanupAge    string `yaml:"cleanup_age"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	Format     string `yaml:"format"`
	Rotation   bool   `yaml:"rotation"`
	MaxSize    int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Use current directory as fallback if home directory cannot be determined
		homeDir = "."
	}

	return &Config{
		Version: "1.0",

		Claude: ClaudeConfig{
			Command:      "claude",
			Timeout:      5 * time.Minute,
			MaxInstances: 3,
			AutoStart:    true,
			Args:         []string{"--no-confirmation"},
		},

		GitHub: GitHubConfig{
			DefaultLabels: []string{"ccagents-generated"},
			DraftPR:       true,
			AutoMerge:     false,
			Reviewers:     []string{},
			Assignees:     []string{},
		},

		UI: UIConfig{
			Theme:           "dark",
			ShowTimestamps:  true,
			VerboseOutput:   false,
			ViewportBuffer:  10000,
			RefreshInterval: 100,
			Sound: SoundConfig{
				Enabled:               true,
				ConfirmationSound:     true,
				SuccessSound:          true,
				ErrorSound:            true,
				WorkflowCompleteSound: true,
			},
		},

		Workflow: WorkflowConfig{
			AutoReview:          true,
			MaxReviewIterations: 3,
			CommitStyle:         "conventional",
			ParallelTasks:       true,
			SkipCI:              false,
		},

		Development: DevelopmentConfig{
			WorktreeBase:  filepath.Join(homeDir, ".ccagents", "worktrees"),
			KeepWorktrees: false,
			MaxWorktrees:  10,
			CleanupAge:    "168h", // 7 days
		},

		Logging: LoggingConfig{
			Level:      "info",
			File:       filepath.Join(homeDir, ".ccagents", "logs", "ccagents.log"),
			Format:     "text",
			Rotation:   true,
			MaxSize:    100, // MB
			MaxAge:     30,  // days
			MaxBackups: 5,
		},
	}
}

// GetConfigPaths returns the list of configuration file paths to check
func GetConfigPaths() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Use current directory as fallback if home directory cannot be determined
		homeDir = "."
	}

	paths := []string{
		".ccagents.yaml",
		".ccagents.yml",
		filepath.Join(homeDir, ".ccagents.yaml"),
		filepath.Join(homeDir, ".ccagents.yml"),
		filepath.Join(homeDir, ".config", "ccagents", "config.yaml"),
		filepath.Join(homeDir, ".config", "ccagents", "config.yml"),
	}

	// Add environment variable override
	if envPath := os.Getenv("CCAGENTS_CONFIG"); envPath != "" {
		paths = append([]string{envPath}, paths...)
	}

	return paths
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Claude configuration
	if c.Claude.Command == "" {
		return fmt.Errorf("claude.command cannot be empty")
	}
	if c.Claude.MaxInstances < 1 {
		return fmt.Errorf("claude.max_instances must be at least 1")
	}
	if c.Claude.Timeout < time.Second {
		return fmt.Errorf("claude.timeout must be at least 1 second")
	}

	// Validate Workflow configuration
	if c.Workflow.MaxReviewIterations < 1 {
		return fmt.Errorf("workflow.max_review_iterations must be at least 1")
	}
	if c.Workflow.CommitStyle != "conventional" && c.Workflow.CommitStyle != "simple" {
		return fmt.Errorf("workflow.commit_style must be 'conventional' or 'simple'")
	}

	// Validate UI configuration
	if c.UI.ViewportBuffer < 100 {
		return fmt.Errorf("ui.viewport_buffer must be at least 100")
	}
	if c.UI.RefreshInterval < 10 {
		return fmt.Errorf("ui.refresh_interval must be at least 10ms")
	}

	// Validate Development configuration
	if c.Development.MaxWorktrees < 1 {
		return fmt.Errorf("development.max_worktrees must be at least 1")
	}

	// Validate Logging configuration
	validLevels := map[string]bool{
		logLevelDebug: true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}

	return nil
}

// ApplyEnvironmentOverrides applies environment variable overrides to the configuration
func (c *Config) ApplyEnvironmentOverrides() {
	// Claude overrides
	if cmd := os.Getenv("CCAGENTS_CLAUDE_COMMAND"); cmd != "" {
		c.Claude.Command = cmd
	}

	// GitHub overrides
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		// Token will be used by GitHub CLI
		_ = token // Acknowledge that we have the token
	}

	// Logging overrides
	if level := os.Getenv("CCAGENTS_LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
	if file := os.Getenv("CCAGENTS_LOG_FILE"); file != "" {
		c.Logging.File = file
	}

	// Debug mode override
	if os.Getenv("CCAGENTS_DEBUG") == "true" {
		c.Logging.Level = logLevelDebug
		c.UI.VerboseOutput = true
	}
}

// ToLoggerConfig converts the logging configuration to logger.Config
func (c *Config) ToLoggerConfig() logger.Config {
	var level logger.Level
	switch c.Logging.Level {
	case logLevelDebug:
		level = logger.LevelDebug
	case "info":
		level = logger.LevelInfo
	case "warn":
		level = logger.LevelWarn
	case "error":
		level = logger.LevelError
	default:
		level = logger.LevelInfo
	}

	return logger.Config{
		Level:     level,
		LogFile:   c.Logging.File,
		Debug:     c.Logging.Level == logLevelDebug,
		Timestamp: true,
		Prefix:    "ccagents",
	}
}
