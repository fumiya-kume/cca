package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// version is set by build flags
	version = "dev"
	// buildDate is set by build flags
	buildDate = "unknown"
	// gitCommit is set by build flags
	gitCommit = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display version, build date, and system information for ccAgents.",
	Run: func(cmd *cobra.Command, args []string) {
		showDetailed, err := cmd.Flags().GetBool("detailed")
		if err != nil {
			showDetailed = false
		}
		showShort, err := cmd.Flags().GetBool("short")
		if err != nil {
			showShort = false
		}

		if showShort {
			fmt.Printf("%s\n", version)
		} else if showDetailed {
			fmt.Printf("ccAgents version %s\n", version)
			fmt.Printf("Build date: %s\n", buildDate)
			fmt.Printf("Git commit: %s\n", gitCommit)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		} else {
			fmt.Printf("ccAgents version %s\n", version)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolP("detailed", "d", false, "show detailed version information")
	versionCmd.Flags().BoolP("short", "s", false, "show only version number")
}
