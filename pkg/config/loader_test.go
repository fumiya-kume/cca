package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// Test constants
const (
	testVersion = "1.0"
)

func createTempDir(t *testing.T, prefix string) string {
	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})
	return tempDir
}

func createTempFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)

	// Create directory if it doesn't exist
	// #nosec G301 - 0755 is acceptable for test directories in temporary location
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("Failed to create directory for temp file: %v", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return filePath
}

func TestLoader(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")
	configPath := filepath.Join(tempDir, "config.yaml")

	loader := NewLoader(configPath)

	// Test loading non-existent config (should return defaults)
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error loading non-existent config, got: %v", err)
	}

	if config == nil {
		t.Fatal("Expected default config, got nil")
	}

	if config.Version != testVersion {
		t.Errorf("Expected default version 1.0, got %s", config.Version)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")
	configPath := filepath.Join(tempDir, "config.yaml")

	loader := NewLoader(configPath)

	// Create a test configuration
	originalConfig := DefaultConfig()
	originalConfig.Claude.Command = "test-claude"
	originalConfig.Claude.MaxInstances = 5
	originalConfig.UI.Theme = "light"

	// Save configuration
	err := loader.SaveConfig(originalConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load configuration
	loadedConfig, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config matches original
	if loadedConfig.Claude.Command != originalConfig.Claude.Command {
		t.Errorf("Expected claude command %s, got %s", originalConfig.Claude.Command, loadedConfig.Claude.Command)
	}

	if loadedConfig.Claude.MaxInstances != originalConfig.Claude.MaxInstances {
		t.Errorf("Expected max instances %d, got %d", originalConfig.Claude.MaxInstances, loadedConfig.Claude.MaxInstances)
	}

	if loadedConfig.UI.Theme != originalConfig.UI.Theme {
		t.Errorf("Expected theme %s, got %s", originalConfig.UI.Theme, loadedConfig.UI.Theme)
	}
}

func TestInvalidYAMLConfig(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create invalid YAML file
	invalidYAML := "invalid: yaml: content:\n  - missing:"
	createTempFile(t, tempDir, "config.yaml", invalidYAML)

	loader := NewLoader(configPath)

	// Should return error for invalid YAML
	_, err := loader.LoadConfig()
	if err == nil {
		t.Error("Expected error for invalid YAML config")
	}
}

func TestInvalidConfigValidation(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create config with invalid values
	invalidConfig := map[string]interface{}{
		"version": testVersion,
		"claude": map[string]interface{}{
			"command":       "", // Invalid: empty command
			"max_instances": 0,  // Invalid: zero instances
		},
	}

	data, err := yaml.Marshal(invalidConfig)
	if err != nil {
		t.Fatalf("Failed to marshal invalid config: %v", err)
	}

	createTempFile(t, tempDir, "config.yaml", string(data))

	loader := NewLoader(configPath)

	// Should return error for invalid config
	_, err = loader.LoadConfig()
	if err == nil {
		t.Error("Expected validation error for invalid config")
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")
	configPath := filepath.Join(tempDir, "config.yaml")

	err := CreateDefaultConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to create default config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Default config file was not created")
	}

	// Load and verify it's valid
	loader := NewLoader(configPath)
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load created default config: %v", err)
	}

	if config.Version != testVersion {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}
}

func TestLoadOrCreateConfig(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")
	configPath := filepath.Join(tempDir, "config.yaml")

	// Test loading non-existent config (should create default)
	config, err := LoadOrCreateConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load or create config: %v", err)
	}

	if config == nil {
		t.Fatal("Expected config, got nil")
	}

	if config.Version != testVersion {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}
}

func TestFindConfigFile(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")

	// Create a config file in a standard location
	createTempFile(t, tempDir, ".ccagents.yaml", "version: 1.0")

	// Change working directory to temp dir
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()
	_ = os.Chdir(tempDir)

	loader := NewLoader("")
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to find and load config: %v", err)
	}

	if config.Version != testVersion {
		t.Errorf("Expected version 1.0, got %s", config.Version)
	}
}

func TestLoaderGetConfigPath(t *testing.T) {
	configPath := "/test/path/config.yaml"
	loader := NewLoader(configPath)

	if loader.GetConfigPath() != configPath {
		t.Errorf("Expected config path %s, got %s", configPath, loader.GetConfigPath())
	}
}

func TestEnvironmentConfigPath(t *testing.T) {
	tempDir := createTempDir(t, "config-test-")
	configPath := filepath.Join(tempDir, "custom-config.yaml")

	// Create config file
	createTempFile(t, tempDir, "custom-config.yaml", `
version: testVersion
claude:
  command: "env-claude"
`)

	// Set environment variable
	_ = os.Setenv("CCAGENTS_CONFIG", configPath)
	defer func() { _ = os.Unsetenv("CCAGENTS_CONFIG") }()

	// Verify it's in the paths
	paths := GetConfigPaths()
	if len(paths) == 0 || paths[0] != configPath {
		t.Errorf("Expected first path to be %s, got %v", configPath, paths)
	}

	// Load config using environment path
	loader := NewLoader("")
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config from environment path: %v", err)
	}

	if config.Claude.Command != "env-claude" {
		t.Errorf("Expected claude command from env config, got %s", config.Claude.Command)
	}
}
