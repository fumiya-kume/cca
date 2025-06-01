package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/fumiya-kume/cca/pkg/git"
	"github.com/fumiya-kume/cca/pkg/validation"
	"github.com/fumiya-kume/cca/pkg/workflow"
	"github.com/spf13/cobra"
)

// issueCmd represents the issue command
var issueCmd = &cobra.Command{
	Use:   "issue [github-issue-url]",
	Short: "Process a GitHub issue and create a pull request",
	Long: `Process a GitHub issue and create a pull request using AI automation.

Supported URL formats:
  - Full URL: https://github.com/owner/repo/issues/123
  - Shorthand: owner/repo#123
  - Context-aware: #123 (when run from within a git repository)

Examples:
  ccagents issue https://github.com/myorg/myrepo/issues/42
  ccagents issue myorg/myrepo#42
  ccagents issue #42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Sanitize and validate input length
		input := validation.SanitizeInput(args[0])
		if err := validation.ValidateInputLength(input, 2000, "issue reference"); err != nil {
			return err
		}

		// Parse the issue reference
		issueRef, err := parseIssueReference(input)
		if err != nil {
			return fmt.Errorf("failed to parse issue reference: %w", err)
		}

		// Validate the parsed reference
		if err := validation.ValidateIssueReference(issueRef); err != nil {
			return fmt.Errorf("invalid issue reference: %w", err)
		}

		if verbose {
			fmt.Printf("Parsed issue reference: %+v\n", issueRef)
		}

		fmt.Printf("🔍 Processing issue #%d from %s/%s\n", issueRef.Number, issueRef.Owner, issueRef.Repo)
		
		// Validate Claude setup before starting workflow
		fmt.Print("🔧 Validating Claude Code setup... ")

		// Get command flags
		draft, err := cmd.Flags().GetBool("draft")
		if err != nil {
			return fmt.Errorf("failed to get draft flag: %w", err)
		}
		labels, err := cmd.Flags().GetStringSlice("labels")
		if err != nil {
			return fmt.Errorf("failed to get labels flag: %w", err)
		}
		assignees, err := cmd.Flags().GetStringSlice("assignees")
		if err != nil {
			return fmt.Errorf("failed to get assignees flag: %w", err)
		}
		base, err := cmd.Flags().GetString("base")
		if err != nil {
			return fmt.Errorf("failed to get base flag: %w", err)
		}
		autoMerge, err := cmd.Flags().GetBool("auto-merge")
		if err != nil {
			return fmt.Errorf("failed to get auto-merge flag: %w", err)
		}
		skipReview, err := cmd.Flags().GetBool("skip-review")
		if err != nil {
			return fmt.Errorf("failed to get skip-review flag: %w", err)
		}
		timeoutMinutes, err := cmd.Flags().GetInt("timeout")
		if err != nil {
			return fmt.Errorf("failed to get timeout flag: %w", err)
		}

		// Create workflow options
		options := &workflow.IssueWorkflowOptions{
			Draft:       draft,
			Labels:      labels,
			Assignees:   assignees,
			Base:        base,
			AutoMerge:   autoMerge,
			SkipReview:  skipReview,
			SkipTesting: false, // Could add flag for this
		}

		if timeoutMinutes > 0 {
			options.Timeout = time.Duration(timeoutMinutes) * time.Minute
		}

		// Create workflow service with default configuration
		config := workflow.DefaultIssueWorkflowConfig()
		service, err := workflow.NewIssueWorkflowService(config)
		if err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}
		defer func() {
			if err := service.Close(); err != nil {
				fmt.Printf("Warning: failed to close service: %v\n", err)
			}
		}()

		// Create context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start workflow
		fmt.Println("\n🚀 Starting automated issue workflow...")
		
		if verbose {
			fmt.Printf("Debug: Workflow options: %+v\n", options)
			fmt.Printf("Debug: Using timeout: %v\n", options.Timeout)
		}
		
		// Process the issue
		fmt.Println("📋 Stage 1/5: Initializing workflow...")
		instance, err := service.ProcessIssue(ctx, issueRef, options)
		if err != nil {
			fmt.Printf("❌ Workflow failed: %v\n", err)
			return fmt.Errorf("workflow failed: %w", err)
		}
		
		if verbose {
			fmt.Printf("Debug: Workflow completed with status: %v\n", instance.Status)
		}

		// Display results
		fmt.Printf("\n✅ Workflow completed successfully!\n")
		if instance.PullRequestURL != "" {
			fmt.Printf("📄 Pull Request: %s\n", instance.PullRequestURL)
		}
		if instance.Worktree != nil && instance.Worktree.BranchName != "" {
			fmt.Printf("🌿 Branch: %s\n", instance.Worktree.BranchName)
		}
		
		duration := instance.EndTime.Sub(instance.StartTime)
		fmt.Printf("⏱️  Duration: %v\n", duration.Round(time.Second))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(issueCmd)

	// Add issue-specific flags
	issueCmd.Flags().BoolP("draft", "d", false, "create pull request as draft")
	issueCmd.Flags().StringSliceP("labels", "l", []string{}, "additional labels for the pull request")
	issueCmd.Flags().StringSliceP("assignees", "a", []string{}, "assignees for the pull request")
	issueCmd.Flags().StringP("base", "b", "", "base branch for the pull request")
	issueCmd.Flags().BoolP("auto-merge", "m", false, "enable auto-merge when checks pass")
	issueCmd.Flags().BoolP("skip-review", "s", false, "skip automated code review")
	issueCmd.Flags().IntP("timeout", "t", 0, "timeout in minutes (0 for no timeout)")
}

// parseIssueReference parses various GitHub issue reference formats
func parseIssueReference(input string) (*types.IssueReference, error) {
	input = strings.TrimSpace(input)

	// Full GitHub URL pattern
	fullURLPattern := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/issues/(\d+)(?:\?.*)?(?:#.*)?$`)
	if matches := fullURLPattern.FindStringSubmatch(input); matches != nil {
		number, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("invalid issue number: %s", matches[3])
		}
		return &types.IssueReference{
			Owner:  matches[1],
			Repo:   matches[2],
			Number: number,
			Source: "url",
		}, nil
	}

	// Shorthand pattern: owner/repo#123
	shorthandPattern := regexp.MustCompile(`^([^/\s]+)/([^/\s#]+)#(\d+)$`)
	if matches := shorthandPattern.FindStringSubmatch(input); matches != nil {
		number, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, fmt.Errorf("invalid issue number: %s", matches[3])
		}
		return &types.IssueReference{
			Owner:  matches[1],
			Repo:   matches[2],
			Number: number,
			Source: "shorthand",
		}, nil
	}

	// Context-aware pattern: #123
	contextPattern := regexp.MustCompile(`^#(\d+)$`)
	if matches := contextPattern.FindStringSubmatch(input); matches != nil {
		number, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid issue number: %s", matches[1])
		}

		// Extract owner/repo from git context
		owner, repo, err := getGitContext()
		if err != nil {
			return nil, fmt.Errorf("context-aware parsing requires being in a git repository: %w", err)
		}

		return &types.IssueReference{
			Owner:  owner,
			Repo:   repo,
			Number: number,
			Source: "context",
		}, nil
	}

	return nil, fmt.Errorf("invalid issue reference format: %s", input)
}

// getGitContext extracts owner and repo from the current git repository
func getGitContext() (string, string, error) {
	repoInfo, err := git.GetRepositoryContext()
	if err != nil {
		return "", "", err
	}

	if !repoInfo.IsGitRepo {
		return "", "", fmt.Errorf("not in a git repository")
	}

	if repoInfo.Owner == "" || repoInfo.Repo == "" {
		return "", "", fmt.Errorf("could not determine GitHub repository from git context")
	}

	return repoInfo.Owner, repoInfo.Repo, nil
}
