package main

import (
	"fmt"
	"os"

	"github.com/fumiya-kume/cca/pkg/config"
	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	debug   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ccagents",
	Short: "AI-powered GitHub issue to PR automation",
	Long: `ccAgents transforms GitHub issues into merged pull requests using Claude Code as an AI sub-agent.

ccAgents automates the entire development workflow:
- Parses GitHub issues and extracts requirements
- Creates isolated git worktrees
- Executes Claude Code with project context
- Generates atomic commits with conventional messages
- Creates and manages pull requests
- Handles code review and iteration`,
	Version: "0.1.0-dev",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ccagents.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug output")
	rootCmd.PersistentFlags().String("log-level", "", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-file", "", "log file path")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().String("theme", "", "UI theme (dark, light)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load configuration
	cfg, err := config.LoadOrCreateConfig(cfgFile)
	if err != nil {
		if debug {
			fmt.Printf("Warning: Failed to load config: %v\n", err)
		}
		// Use default config
		cfg = config.DefaultConfig()
	}

	// Apply command line flag overrides
	if debug {
		cfg.Logging.Level = "debug"
		cfg.UI.VerboseOutput = true
	}

	// Apply other flag overrides
	logLevel, err := rootCmd.PersistentFlags().GetString("log-level")
	if err != nil {
		logLevel = ""
	}
	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}
	logFile, err := rootCmd.PersistentFlags().GetString("log-file")
	if err != nil {
		logFile = ""
	}
	if logFile != "" {
		cfg.Logging.File = logFile
	}
	theme, err := rootCmd.PersistentFlags().GetString("theme")
	if err != nil {
		theme = ""
	}
	if theme != "" {
		cfg.UI.Theme = theme
	}
	if verbose {
		cfg.UI.VerboseOutput = true
	}

	// Initialize logger with configuration
	loggerConfig := cfg.ToLoggerConfig()
	globalLogger, err := logger.New(loggerConfig)
	if err != nil {
		if debug {
			fmt.Printf("Warning: Failed to initialize logger: %v\n", err)
		}
		globalLogger = logger.NewDefault()
	}
	logger.SetGlobalLogger(globalLogger)

}
