package github

import (
	"context"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/google/go-github/v60/github"
)

// GitHubClient defines the interface for GitHub operations
type GitHubClient interface {
	ValidateAccess(ctx context.Context, owner, repo string) error
	GetIssue(ctx context.Context, owner, repo string, number int) (*types.IssueData, error)
	CreatePR(ctx context.Context, owner, repo string, pr *types.PRData) (*types.PRData, error)
	GetDefaultBranch(ctx context.Context, owner, repo string) (string, error)
	CheckIssueExists(ctx context.Context, owner, repo string, number int) error
	IsAuthenticated(ctx context.Context) error
	GetRateLimit(ctx context.Context) (*github.RateLimits, error)
	GetIssueUsingCLI(owner, repo string, number int) (*types.IssueData, error)
}

// Ensure our Client implements the interface
var _ GitHubClient = (*Client)(nil)
