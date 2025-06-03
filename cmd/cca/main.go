package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fumiya-kume/cca/internal"
)

func main() {
	// Check for exactly one argument
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: cca <github-issue-url>")
		fmt.Fprintln(os.Stderr, "Example: cca https://github.com/owner/repo/issues/123")
		os.Exit(1)
	}

	issueURL := os.Args[1]

	// Validate URL
	if !strings.Contains(issueURL, "github.com") || !strings.Contains(issueURL, "/issues/") {
		fmt.Fprintf(os.Stderr, "Error: Invalid GitHub issue URL: %s\n", issueURL)
		fmt.Fprintln(os.Stderr, "URL must contain 'github.com' and '/issues/'")
		os.Exit(1)
	}

	// Create processor and run workflow
	processor := internal.NewProcessor()
	if err := processor.ProcessIssue(issueURL); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Pull request created successfully!")
}
