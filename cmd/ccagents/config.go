package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fumiya-kume/cca/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ccAgents configuration",
	Long: `Manage ccAgents configuration settings.

Configuration files are searched in the following order:
  1. $CCAGENTS_CONFIG (if set)
  2. ./.ccagents.yaml
  3. ~/.ccagents.yaml
  4. ~/.config/ccagents/config.yaml

Use subcommands to view, edit, or validate configuration.`,
}

// configShowCmd shows the current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Display the current configuration settings, including defaults and overrides.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadOrCreateConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Convert to YAML for display
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}

		fmt.Print(string(data))
		return nil
	},
}

// configValidateCmd validates the configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long:  "Validate the configuration file for syntax and semantic errors.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadOrCreateConfig(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Println("Configuration is valid ✓")
		return nil
	},
}

// configInitCmd initializes a new configuration file
var configInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize a new configuration file",
	Long: `Initialize a new configuration file with default settings.

If no path is provided, creates config in ~/.config/ccagents/config.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var configPath string
		if len(args) > 0 {
			configPath = args[0]
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			configPath = filepath.Join(homeDir, ".config", "ccagents", "config.yaml")
		}

		// Check if file already exists
		if _, err := os.Stat(configPath); err == nil {
			overwrite, err := cmd.Flags().GetBool("force")
			if err != nil {
				return fmt.Errorf("failed to get force flag: %w", err)
			}
			if !overwrite {
				return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
			}
		}

		if err := config.CreateDefaultConfig(configPath); err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}

		fmt.Printf("Configuration file created at: %s\n", configPath)
		return nil
	},
}

// configPathCmd shows the path to the configuration file
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Long:  "Display the path to the configuration file that would be used.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgFile != "" {
			fmt.Println(cfgFile)
			return nil
		}

		// Search for config file
		paths := config.GetConfigPaths()
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				fmt.Println(path)
				return nil
			}
		}

		// No config file found, show where it would be created
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		defaultPath := filepath.Join(homeDir, ".config", "ccagents", "config.yaml")
		fmt.Printf("%s (would be created)\n", defaultPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configPathCmd)

	// Add flags
	configInitCmd.Flags().BoolP("force", "f", false, "overwrite existing configuration file")
}
