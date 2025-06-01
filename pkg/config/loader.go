package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading and saving
type Loader struct {
	configPath string
}

// NewLoader creates a new configuration loader
func NewLoader(configPath string) *Loader {
	return &Loader{
		configPath: configPath,
	}
}

// LoadConfig loads configuration from file or returns default config
func (l *Loader) LoadConfig() (*Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// If no specific config path provided, search for config files
	if l.configPath == "" {
		configPath, err := l.findConfigFile()
		if err != nil {
			// No config file found, use defaults with environment overrides
			config.ApplyEnvironmentOverrides()
			return config, nil
		}
		l.configPath = configPath
	}

	// Check if config file exists
	if _, err := os.Stat(l.configPath); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults with environment overrides
		config.ApplyEnvironmentOverrides()
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(l.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", l.configPath, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", l.configPath, err)
	}

	// Apply environment overrides
	config.ApplyEnvironmentOverrides()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to file
func (l *Loader) SaveConfig(config *Config) error {
	if l.configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		l.configPath = filepath.Join(homeDir, ".config", "ccagents", "config.yaml")
	}

	// Ensure directory exists
	configDir := filepath.Dir(l.configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(l.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", l.configPath, err)
	}

	return nil
}

// GetConfigPath returns the current config file path
func (l *Loader) GetConfigPath() string {
	return l.configPath
}

// findConfigFile searches for a configuration file in standard locations
func (l *Loader) findConfigFile() (string, error) {
	paths := GetConfigPaths()

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no configuration file found")
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig(path string) error {
	loader := NewLoader(path)
	config := DefaultConfig()

	return loader.SaveConfig(config)
}

// LoadOrCreateConfig loads configuration or creates default if none exists
func LoadOrCreateConfig(configPath string) (*Config, error) {
	loader := NewLoader(configPath)

	config, err := loader.LoadConfig()
	if err != nil {
		return nil, err
	}

	// If we couldn't find a config file and no specific path was given,
	// create a default config file
	if loader.GetConfigPath() == "" && configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(homeDir, ".config", "ccagents", "config.yaml")
			if err := CreateDefaultConfig(defaultPath); err == nil {
				loader = NewLoader(defaultPath)
				config, err = loader.LoadConfig()
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return config, nil
}
