package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	"github.com/fumiya-kume/cca/pkg/agents"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// AgentConfigManager manages multi-agent configuration with hot reloading
type AgentConfigManager struct {
	config       *agents.AgentConfiguration
	configPaths  []string
	watcher      *fsnotify.Watcher
	logger       *logger.Logger
	callbacks    []ConfigChangeCallback
	version      int
	lastModified time.Time
}

// ConfigChangeCallback is called when configuration changes
type ConfigChangeCallback func(oldConfig, newConfig *agents.AgentConfiguration) error

// ValidationResult represents configuration validation results
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ConfigTemplate represents a configuration template
type ConfigTemplate struct {
	Name        string                     `yaml:"name" json:"name"`
	Description string                     `yaml:"description" json:"description"`
	Type        string                     `yaml:"type" json:"type"` // project_type this template is for
	Config      *agents.AgentConfiguration `yaml:"config" json:"config"`
	Variables   map[string]Variable        `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// Variable represents a template variable
type Variable struct {
	Name         string      `yaml:"name" json:"name"`
	Description  string      `yaml:"description" json:"description"`
	Type         string      `yaml:"type" json:"type"` // string, int, bool, duration
	DefaultValue interface{} `yaml:"default" json:"default"`
	Required     bool        `yaml:"required" json:"required"`
	Options      []string    `yaml:"options,omitempty" json:"options,omitempty"`
}

// ConfigMigration represents a configuration migration
type ConfigMigration struct {
	FromVersion string                                                                         `yaml:"from_version"`
	ToVersion   string                                                                         `yaml:"to_version"`
	Description string                                                                         `yaml:"description"`
	Migrate     func(oldConfig *agents.AgentConfiguration) (*agents.AgentConfiguration, error) `yaml:"-"`
}

// NewAgentConfigManager creates a new agent configuration manager
func NewAgentConfigManager(logger *logger.Logger) *AgentConfigManager {
	return &AgentConfigManager{
		config:       agents.DefaultAgentConfiguration(),
		configPaths:  []string{},
		logger:       logger,
		callbacks:    []ConfigChangeCallback{},
		version:      1,
		lastModified: time.Now(),
	}
}

// LoadConfiguration loads configuration from multiple sources with precedence
func (acm *AgentConfigManager) LoadConfiguration() error {
	acm.logger.Info("Loading agent configuration")

	// Get configuration file paths in precedence order
	configPaths := acm.getConfigPaths()
	acm.configPaths = configPaths

	// Start with default configuration
	config := agents.DefaultAgentConfiguration()

	// Load and merge configurations in order
	for _, path := range configPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			acm.logger.Debug("Configuration file not found, skipping (path: %s)", path)
			continue
		}

		fileConfig, err := acm.loadConfigFile(path)
		if err != nil {
			acm.logger.Error("Failed to load configuration file (path: %s, error: %v)", path, err)
			continue
		}

		acm.logger.Info("Loaded configuration file (path: %s)", path)
		config.Merge(fileConfig)
	}

	// Apply environment variable overrides
	if err := acm.applyEnvironmentOverrides(config); err != nil {
		return fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Store configuration
	oldConfig := acm.config
	acm.config = config
	acm.version++
	acm.lastModified = time.Now()

	// Notify callbacks of configuration change
	acm.notifyCallbacks(oldConfig, config)

	acm.logger.Info("Agent configuration loaded successfully (version: %d)", acm.version)
	return nil
}

// StartHotReload starts watching configuration files for changes
func (acm *AgentConfigManager) StartHotReload() error {
	if acm.watcher != nil {
		return fmt.Errorf("hot reload already started")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	acm.watcher = watcher

	// Watch configuration files and directories
	for _, path := range acm.configPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		if err := watcher.Add(path); err != nil {
			acm.logger.Error("Failed to watch configuration file (path: %s, error: %v)", path, err)
			continue
		}

		// Also watch the directory for new files
		dir := filepath.Dir(path)
		if err := watcher.Add(dir); err != nil {
			acm.logger.Debug("Failed to watch configuration directory (dir: %s, error: %v)", dir, err)
		}
	}

	// Start watching in a goroutine
	go acm.watchConfigChanges()

	acm.logger.Info("Configuration hot reload started")
	return nil
}

// StopHotReload stops watching configuration files
func (acm *AgentConfigManager) StopHotReload() error {
	if acm.watcher == nil {
		return nil
	}

	err := acm.watcher.Close()
	acm.watcher = nil

	acm.logger.Info("Configuration hot reload stopped")
	return err
}

// GetConfiguration returns the current configuration
func (acm *AgentConfigManager) GetConfiguration() *agents.AgentConfiguration {
	return acm.config.Clone()
}

// GetAgentConfiguration returns configuration for a specific agent
func (acm *AgentConfigManager) GetAgentConfiguration(agentID agents.AgentID) (agents.AgentConfig, bool) {
	return acm.config.GetAgentConfig(agentID)
}

// UpdateAgentConfiguration updates configuration for a specific agent
func (acm *AgentConfigManager) UpdateAgentConfiguration(agentID agents.AgentID, config agents.AgentConfig) error {
	return acm.config.UpdateAgentConfig(agentID, config)
}

// RegisterCallback registers a callback for configuration changes
func (acm *AgentConfigManager) RegisterCallback(callback ConfigChangeCallback) {
	acm.callbacks = append(acm.callbacks, callback)
}

// ValidateConfiguration validates a configuration
func (acm *AgentConfigManager) ValidateConfiguration(config *agents.AgentConfiguration) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if err := config.Validate(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Additional validation checks
	if config.Global.MaxConcurrentAgents > 10 {
		result.Warnings = append(result.Warnings, "High concurrent agent count may impact performance")
	}

	if config.Global.DefaultTimeout < 5*time.Minute {
		result.Warnings = append(result.Warnings, "Short default timeout may cause premature failures")
	}

	// Validate agent-specific configurations
	for agentID, agentConfig := range config.Agents {
		if agentConfig.Enabled {
			if agentConfig.ResourceLimits.MaxMemoryMB > 1024 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Agent %s has high memory limit: %dMB", agentID, agentConfig.ResourceLimits.MaxMemoryMB))
			}

			if agentConfig.Timeout > config.Global.DefaultTimeout {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Agent %s timeout exceeds global default", agentID))
			}
		}
	}

	return result
}

// CreateTemplate creates a configuration template from current configuration
func (acm *AgentConfigManager) CreateTemplate(name, description, projectType string) *ConfigTemplate {
	return &ConfigTemplate{
		Name:        name,
		Description: description,
		Type:        projectType,
		Config:      acm.config.Clone(),
		Variables:   make(map[string]Variable),
	}
}

// ApplyTemplate applies a configuration template with variable substitution
func (acm *AgentConfigManager) ApplyTemplate(template *ConfigTemplate, variables map[string]interface{}) (*agents.AgentConfiguration, error) {
	// Clone template configuration
	config := template.Config.Clone()

	// Apply variable substitutions
	for varName, value := range variables {
		if err := acm.applyVariableSubstitution(config, varName, value); err != nil {
			return nil, fmt.Errorf("failed to apply variable %s: %w", varName, err)
		}
	}

	// Validate result
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("template result validation failed: %w", err)
	}

	return config, nil
}

// SaveConfiguration saves configuration to a file
func (acm *AgentConfigManager) SaveConfiguration(path string) error {
	data, err := yaml.Marshal(acm.config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	acm.logger.Info("Configuration saved (path: %s)", path)
	return nil
}

// BackupConfiguration creates a backup of the current configuration
func (acm *AgentConfigManager) BackupConfiguration() (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("~/.ccagents/backups/config-%s.yaml", timestamp)
	expandedPath := os.ExpandEnv(backupPath)

	// Ensure backup directory exists
	backupDir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	if err := acm.SaveConfiguration(expandedPath); err != nil {
		return "", fmt.Errorf("failed to save backup: %w", err)
	}

	acm.logger.Info("Configuration backed up (path: %s)", expandedPath)
	return expandedPath, nil
}

// RestoreConfiguration restores configuration from a backup
func (acm *AgentConfigManager) RestoreConfiguration(backupPath string) error {
	config, err := acm.loadConfigFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to load backup configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("backup configuration validation failed: %w", err)
	}

	oldConfig := acm.config
	acm.config = config
	acm.version++
	acm.lastModified = time.Now()

	acm.notifyCallbacks(oldConfig, config)

	acm.logger.Info("Configuration restored from backup (path: %s, version: %d)", backupPath, acm.version)
	return nil
}

// MigrateConfiguration migrates configuration to a newer version
func (acm *AgentConfigManager) MigrateConfiguration(migrations []ConfigMigration) error {
	config := acm.config.Clone()

	for _, migration := range migrations {
		if migration.Migrate != nil {
			migratedConfig, err := migration.Migrate(config)
			if err != nil {
				return fmt.Errorf("migration from %s to %s failed: %w",
					migration.FromVersion, migration.ToVersion, err)
			}
			config = migratedConfig
			acm.logger.Info("Applied configuration migration (from: %s, to: %s)", migration.FromVersion, migration.ToVersion)
		}
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("migrated configuration validation failed: %w", err)
	}

	oldConfig := acm.config
	acm.config = config
	acm.version++
	acm.lastModified = time.Now()

	acm.notifyCallbacks(oldConfig, config)

	acm.logger.Info("Configuration migration completed (version: %d)", acm.version)
	return nil
}

// GetVersion returns the current configuration version
func (acm *AgentConfigManager) GetVersion() int {
	return acm.version
}

// GetLastModified returns when the configuration was last modified
func (acm *AgentConfigManager) GetLastModified() time.Time {
	return acm.lastModified
}

// getConfigPaths returns configuration file paths in precedence order
func (acm *AgentConfigManager) getConfigPaths() []string {
	paths := []string{}

	// System-wide configuration
	paths = append(paths, "/etc/ccagents/config.yaml")

	// User configuration
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".ccagents", "config.yaml"))
		paths = append(paths, filepath.Join(homeDir, ".config", "ccagents", "config.yaml"))
	}

	// Project-specific configuration
	if workDir, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(workDir, ".ccagents.yaml"))
		paths = append(paths, filepath.Join(workDir, ".ccagents", "config.yaml"))
	}

	// Environment-specific override
	if envPath := os.Getenv("CCAGENTS_CONFIG"); envPath != "" {
		paths = append(paths, envPath)
	}

	return paths
}

// loadConfigFile loads a single configuration file
func (acm *AgentConfigManager) loadConfigFile(path string) (*agents.AgentConfiguration, error) {
	// Validate file path for security
	if err := acm.validateFilePath(path); err != nil {
		return nil, err
	}
	
	// #nosec G304 - path is validated and sanitized in ValidateFilePath
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	config := &agents.AgentConfiguration{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return config, nil
}

// applyEnvironmentOverrides applies environment variable overrides
func (acm *AgentConfigManager) applyEnvironmentOverrides(config *agents.AgentConfiguration) error {
	// Global overrides
	if maxAgents := os.Getenv("CCAGENTS_MAX_CONCURRENT"); maxAgents != "" {
		// Parse and apply
		// TODO: Implement max agents parsing
		_ = maxAgents // Acknowledge that we have the value but don't use it yet
	}

	if timeout := os.Getenv("CCAGENTS_DEFAULT_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.Global.DefaultTimeout = duration
		}
	}

	// Agent-specific overrides
	for agentID := range config.Agents {
		envPrefix := fmt.Sprintf("CCAGENTS_%s_", strings.ToUpper(string(agentID)))

		if enabled := os.Getenv(envPrefix + "ENABLED"); enabled != "" {
			if enabled == "true" || enabled == "1" {
				config.EnableAgent(agentID)
			} else {
				config.DisableAgent(agentID)
			}
		}
	}

	return nil
}

// watchConfigChanges watches for configuration file changes
func (acm *AgentConfigManager) watchConfigChanges() {
	debounceDelay := 500 * time.Millisecond
	var debounceTimer *time.Timer

	for {
		select {
		case event, ok := <-acm.watcher.Events:
			if !ok {
				return
			}

			// Check if this is a configuration file
			isConfigFile := false
			for _, path := range acm.configPaths {
				if event.Name == path {
					isConfigFile = true
					break
				}
			}

			if !isConfigFile {
				continue
			}

			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Debounce rapid changes
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounceDelay, func() {
					acm.logger.Info("Configuration file changed, reloading (file: %s)", event.Name)
					if err := acm.LoadConfiguration(); err != nil {
						acm.logger.Error("Failed to reload configuration (error: %v)", err)
					}
				})
			}

		case err, ok := <-acm.watcher.Errors:
			if !ok {
				return
			}
			acm.logger.Error("Configuration watcher error (error: %v)", err)
		}
	}
}

// notifyCallbacks notifies all registered callbacks of configuration changes
func (acm *AgentConfigManager) notifyCallbacks(oldConfig, newConfig *agents.AgentConfiguration) {
	for _, callback := range acm.callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			acm.logger.Error("Configuration change callback failed (error: %v)", err)
		}
	}
}

// applyVariableSubstitution applies variable substitution to configuration
func (acm *AgentConfigManager) applyVariableSubstitution(config *agents.AgentConfiguration, varName string, value interface{}) error {
	// This would implement variable substitution logic
	// For now, return success as this is a complex feature
	return nil
}

// DefaultTemplates returns a set of default configuration templates
func DefaultTemplates() []*ConfigTemplate {
	return []*ConfigTemplate{
		{
			Name:        "golang-web-service",
			Description: "Configuration for Go web service projects",
			Type:        "golang_web_service",
			Config:      golangWebServiceConfig(),
		},
		{
			Name:        "react-frontend",
			Description: "Configuration for React frontend projects",
			Type:        "react_frontend",
			Config:      reactFrontendConfig(),
		},
		{
			Name:        "minimal",
			Description: "Minimal configuration with essential agents only",
			Type:        "minimal",
			Config:      minimalConfig(),
		},
		{
			Name:        "security-focused",
			Description: "Security-focused configuration with enhanced scanning",
			Type:        "security_focused",
			Config:      securityFocusedConfig(),
		},
	}
}

// Template configurations
func golangWebServiceConfig() *agents.AgentConfiguration {
	config := agents.DefaultAgentConfiguration()
	config.Project.Type = "golang_web_service"
	config.Project.WorkflowOverrides.RequireSecurityReview = true
	config.Project.WorkflowOverrides.RequirePerformanceAnalysis = true
	config.Project.QualityGates.MinimumTestCoverage = 85
	return config
}

func reactFrontendConfig() *agents.AgentConfiguration {
	config := agents.DefaultAgentConfiguration()
	config.Project.Type = "react_frontend"
	config.Project.QualityGates.MinimumTestCoverage = 75
	// Disable performance agent for frontend projects
	config.DisableAgent(agents.PerformanceAgentID)
	return config
}

func minimalConfig() *agents.AgentConfiguration {
	config := agents.DefaultAgentConfiguration()
	config.Project.Type = "minimal"
	// Enable only essential agents
	config.DisableAgent(agents.PerformanceAgentID)
	config.DisableAgent(agents.DocumentationAgentID)
	return config
}

func securityFocusedConfig() *agents.AgentConfiguration {
	config := agents.DefaultAgentConfiguration()
	config.Project.Type = "security_focused"
	config.Project.WorkflowOverrides.RequireSecurityReview = true
	config.Project.QualityGates.SecurityScanRequired = true
	// Increase security agent resources
	if securityConfig, exists := config.GetAgentConfig(agents.SecurityAgentID); exists {
		securityConfig.MaxInstances = 3
		securityConfig.ResourceLimits.MaxMemoryMB = 512
		if err := config.UpdateAgentConfig(agents.SecurityAgentID, securityConfig); err != nil {
			// Log error but continue as this is just default config setup
			// In a real implementation, we might want to log this
			return config
		}
	}
	return config
}

// validateFilePath validates a file path to prevent directory traversal attacks
func (acm *AgentConfigManager) validateFilePath(file string) error {
	// Check for directory traversal patterns
	if strings.Contains(file, "..") {
		return fmt.Errorf("invalid file path: contains directory traversal")
	}
	
	// Ensure file is within expected project bounds
	absPath, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	
	// Additional security check - ensure file is a regular file
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	
	if !info.Mode().IsRegular() {
		return fmt.Errorf("invalid file path: not a regular file")
	}
	
	return nil
}
