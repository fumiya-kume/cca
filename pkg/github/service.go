package github

import (
	"context"
	"fmt"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/fumiya-kume/cca/pkg/errors"
)

// Service provides high-level GitHub operations
type Service struct {
	client GitHubClient
}

// NewService creates a new GitHub service
func NewService() (*Service, error) {
	client, err := NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return &Service{
		client: client,
	}, nil
}

// ValidateIssueAccess validates that the user can access the specified issue
func (s *Service) ValidateIssueAccess(ctx context.Context, issueRef *types.IssueReference) error {
	// First validate repository access
	if err := s.client.ValidateAccess(ctx, issueRef.Owner, issueRef.Repo); err != nil {
		return fmt.Errorf("repository access validation failed: %w", err)
	}

	// Then check if the specific issue exists and is accessible
	if err := s.client.CheckIssueExists(ctx, issueRef.Owner, issueRef.Repo, issueRef.Number); err != nil {
		return fmt.Errorf("issue access validation failed: %w", err)
	}

	return nil
}

// RetrieveIssueData fetches comprehensive issue data with error handling and retries
func (s *Service) RetrieveIssueData(ctx context.Context, issueRef *types.IssueReference) (*types.IssueData, error) {
	var issueData *types.IssueData
	var err error

	// Retry with exponential backoff for network issues
	retryErr := errors.RetryWithExponentialBackoff(ctx, 3, func() error {
		issueData, err = s.client.GetIssue(ctx, issueRef.Owner, issueRef.Repo, issueRef.Number)
		return err
	})

	if retryErr != nil {
		// If API method fails, try CLI method as fallback
		issueData, err = s.client.GetIssueUsingCLI(issueRef.Owner, issueRef.Repo, issueRef.Number)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve issue data using both API and CLI methods: %w", err)
		}
	}

	// Validate that we got valid issue data
	if issueData == nil {
		return nil, fmt.Errorf("received nil issue data")
	}

	if issueData.Number != issueRef.Number {
		return nil, fmt.Errorf("issue number mismatch: expected %d, got %d", issueRef.Number, issueData.Number)
	}

	if issueData.State == "closed" {
		return nil, fmt.Errorf("issue #%d is closed", issueData.Number)
	}

	return issueData, nil
}

// CreatePullRequest creates a new pull request with the given data
func (s *Service) CreatePullRequest(ctx context.Context, owner, repo string, prData *types.PRData) (*types.PRData, error) {
	// Validate that the base branch exists
	defaultBranch, err := s.client.GetDefaultBranch(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get default branch: %w", err)
	}

	// Use default branch if base branch is not specified
	if prData.BaseBranch == "" {
		prData.BaseBranch = defaultBranch
	}

	// Create the pull request with retry
	var createdPR *types.PRData
	retryErr := errors.RetryWithExponentialBackoff(ctx, 3, func() error {
		createdPR, err = s.client.CreatePR(ctx, owner, repo, prData)
		return err
	})

	if retryErr != nil {
		return nil, fmt.Errorf("failed to create pull request after retries: %w", retryErr)
	}

	return createdPR, nil
}

// CheckAuthentication verifies that the user is properly authenticated
func (s *Service) CheckAuthentication(ctx context.Context) error {
	return s.client.IsAuthenticated(ctx)
}

// GetRepositoryInfo retrieves basic repository information
func (s *Service) GetRepositoryInfo(ctx context.Context, owner, repo string) (map[string]interface{}, error) {
	// Check repository access
	if err := s.client.ValidateAccess(ctx, owner, repo); err != nil {
		return nil, err
	}

	// Get default branch
	defaultBranch, err := s.client.GetDefaultBranch(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	return map[string]interface{}{
		"owner":          owner,
		"repo":           repo,
		"default_branch": defaultBranch,
		"full_name":      fmt.Sprintf("%s/%s", owner, repo),
	}, nil
}

// GetRateLimitStatus returns the current GitHub API rate limit status
func (s *Service) GetRateLimitStatus(ctx context.Context) (map[string]interface{}, error) {
	rateLimits, err := s.client.GetRateLimit(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"core": map[string]interface{}{
			"limit":     rateLimits.Core.Limit,
			"remaining": rateLimits.Core.Remaining,
			"reset":     rateLimits.Core.Reset.Time,
		},
		"search": map[string]interface{}{
			"limit":     rateLimits.Search.Limit,
			"remaining": rateLimits.Search.Remaining,
			"reset":     rateLimits.Search.Reset.Time,
		},
	}, nil
}

// ValidateWorkflowRequirements checks if all requirements for the workflow are met
func (s *Service) ValidateWorkflowRequirements(ctx context.Context, issueRef *types.IssueReference) error {
	// Check authentication first
	if err := s.CheckAuthentication(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Validate issue access
	if err := s.ValidateIssueAccess(ctx, issueRef); err != nil {
		return fmt.Errorf("issue access validation failed: %w", err)
	}

	// Check rate limit status
	rateLimitInfo, err := s.GetRateLimitStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}

	// Warn if rate limit is low
	if core, ok := rateLimitInfo["core"].(map[string]interface{}); ok {
		if remaining, ok := core["remaining"].(int); ok && remaining < 100 {
			return fmt.Errorf("GitHub API rate limit is low (remaining: %d). Please wait before proceeding", remaining)
		}
	}

	return nil
}

// Close closes the GitHub service and cleans up resources
func (s *Service) Close() error {
	// Currently no cleanup needed, but we can add connection pooling cleanup here later
	return nil
}
