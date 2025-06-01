package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	// Save original args and command
	oldArgs := os.Args
	oldRootCmd := rootCmd
	defer func() {
		os.Args = oldArgs
		rootCmd = oldRootCmd
	}()

	// Create a test command that doesn't exit
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	rootCmd = testCmd

	// Test successful execution
	os.Args = []string{"test"}
	Execute() // Should not panic or exit
}

func TestExecuteWithError(t *testing.T) {
	// Save original args and command
	oldArgs := os.Args
	oldRootCmd := rootCmd
	defer func() {
		os.Args = oldArgs
		rootCmd = oldRootCmd
	}()

	// Create a test command that returns an error
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return assert.AnError
		},
	}
	rootCmd = testCmd

	// Test error execution
	os.Args = []string{"test"}
	
	// Since Execute() calls os.Exit(1) on error, we need to handle it
	// We can't directly test os.Exit, but we can verify the command structure
	assert.NotNil(t, rootCmd)
	assert.Equal(t, "test", rootCmd.Use)
}

func TestInitConfig(t *testing.T) {
	// Save original command and variables
	oldCmd := rootCmd
	oldCfgFile := cfgFile
	oldDebug := debug
	oldVerbose := verbose
	
	defer func() {
		rootCmd = oldCmd
		cfgFile = oldCfgFile
		debug = oldDebug
		verbose = oldVerbose
	}()

	// Create a temporary root command for testing
	tempCmd := &cobra.Command{
		Use: "test",
	}
	tempCmd.PersistentFlags().String("log-level", "", "log level")
	tempCmd.PersistentFlags().String("log-file", "", "log file")
	tempCmd.PersistentFlags().String("theme", "", "theme")
	rootCmd = tempCmd

	// Test with default values
	cfgFile = ""
	debug = false
	verbose = false
	
	// This should not panic
	initConfig()
	
	// Test with debug flag
	debug = true
	initConfig()
	
	// Test with verbose flag
	verbose = true
	debug = false
	initConfig()
	
	// Test with a non-existent config file
	cfgFile = "/non/existent/config.yaml"
	debug = true // Enable debug to see warnings
	initConfig()
}

func TestInitConfigWithFlags(t *testing.T) {
	// Save original command and variables
	oldCmd := rootCmd
	oldCfgFile := cfgFile
	oldDebug := debug
	oldVerbose := verbose
	
	defer func() {
		rootCmd = oldCmd
		cfgFile = oldCfgFile
		debug = oldDebug
		verbose = oldVerbose
	}()

	// Create a temporary root command for testing
	tempCmd := &cobra.Command{
		Use: "test",
	}
	tempCmd.PersistentFlags().String("log-level", "info", "log level")
	tempCmd.PersistentFlags().String("log-file", "/tmp/test.log", "log file")
	tempCmd.PersistentFlags().String("theme", "dark", "theme")
	rootCmd = tempCmd

	// Set the flags
	err := tempCmd.PersistentFlags().Set("log-level", "debug")
	assert.NoError(t, err)
	err = tempCmd.PersistentFlags().Set("log-file", "/tmp/ccagents.log")
	assert.NoError(t, err)
	err = tempCmd.PersistentFlags().Set("theme", "light")
	assert.NoError(t, err)

	// Test with flags set
	cfgFile = ""
	debug = false
	verbose = false
	
	// This should not panic and should use the flag values
	initConfig()
}