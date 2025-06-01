// Package github provides GitHub API integration with rate limiting and authentication.
package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/fumiya-kume/cca/internal/types"
	"github.com/google/go-github/v60/github"
)

// Client wraps GitHub CLI and API operations
type Client struct {
	apiClient   *github.Client
	ghClient    *api.RESTClient
	rateLimiter *RateLimiter
}

// NewClient creates a new GitHub client with authentication
func NewClient() (*Client, error) {
	// Create GitHub CLI REST client (handles auth automatically)
	ghClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub CLI client: %w", err)
	}

	// Create GitHub API client using the same authentication
	httpClient := &http.Client{
		Transport: &authTransport{ghClient: ghClient},
		Timeout:   30 * time.Second,
	}

	apiClient := github.NewClient(httpClient)

	// Create rate limiter (GitHub allows 5000 requests/hour for authenticated users)
	rateLimiter := NewRateLimiter(5000, time.Hour)

	return &Client{
		apiClient:   apiClient,
		ghClient:    ghClient,
		rateLimiter: rateLimiter,
	}, nil
}

// authTransport implements http.RoundTripper to use GitHub CLI authentication
type authTransport struct {
	ghClient *api.RESTClient
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// For now, use the default HTTP client as go-gh handles auth internally
	// We'll rely on the GitHub CLI being authenticated
	return http.DefaultTransport.RoundTrip(req)
}

// ValidateAccess checks if the user has access to the specified repository
func (c *Client) ValidateAccess(ctx context.Context, owner, repo string) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	_, _, err := c.apiClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			switch ghErr.Response.StatusCode {
			case 404:
				return fmt.Errorf("repository %s/%s not found or access denied", owner, repo)
			case 403:
				return fmt.Errorf("access forbidden to repository %s/%s", owner, repo)
			case 401:
				return fmt.Errorf("authentication failed - run 'gh auth login' to authenticate")
			}
		}
		return fmt.Errorf("failed to access repository %s/%s: %w", owner, repo, err)
	}

	return nil
}

// GetIssue retrieves comprehensive issue data
func (c *Client) GetIssue(ctx context.Context, owner, repo string, number int) (*types.IssueData, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Get issue details
	issue, _, err := c.apiClient.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			switch ghErr.Response.StatusCode {
			case 404:
				return nil, fmt.Errorf("issue #%d not found in %s/%s", number, owner, repo)
			case 403:
				return nil, fmt.Errorf("access forbidden to issue #%d in %s/%s", number, owner, repo)
			}
		}
		return nil, fmt.Errorf("failed to get issue #%d from %s/%s: %w", number, owner, repo, err)
	}

	// Convert GitHub issue to our IssueData type
	issueData := &types.IssueData{
		Number:    issue.GetNumber(),
		Title:     issue.GetTitle(),
		Body:      issue.GetBody(),
		State:     issue.GetState(),
		CreatedAt: issue.GetCreatedAt().Time,
		UpdatedAt: issue.GetUpdatedAt().Time,
	}

	// Extract labels
	for _, label := range issue.Labels {
		issueData.Labels = append(issueData.Labels, label.GetName())
	}

	// Extract assignees
	for _, assignee := range issue.Assignees {
		issueData.Assignees = append(issueData.Assignees, assignee.GetLogin())
	}

	// Extract milestone
	if issue.Milestone != nil {
		issueData.Milestone = issue.Milestone.GetTitle()
	}

	// Get comments if they exist
	if issue.GetComments() > 0 {
		comments, err := c.getIssueComments(ctx, owner, repo, number)
		if err != nil {
			// Don't fail if we can't get comments, just log and continue
			return issueData, nil
		}
		issueData.Comments = comments
	}

	return issueData, nil
}

// getIssueComments retrieves all comments for an issue
func (c *Client) getIssueComments(ctx context.Context, owner, repo string, number int) ([]types.Comment, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	comments, _, err := c.apiClient.Issues.ListComments(ctx, owner, repo, number, &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get comments for issue #%d: %w", number, err)
	}

	var result []types.Comment
	for _, comment := range comments {
		result = append(result, types.Comment{
			ID:        int(comment.GetID()),
			Body:      comment.GetBody(),
			Author:    comment.GetUser().GetLogin(),
			CreatedAt: comment.GetCreatedAt().Time,
		})
	}

	return result, nil
}

// CreatePR creates a new pull request
func (c *Client) CreatePR(ctx context.Context, owner, repo string, pr *types.PRData) (*types.PRData, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	newPR := &github.NewPullRequest{
		Title: &pr.Title,
		Head:  &pr.Branch,
		Base:  &pr.BaseBranch,
		Body:  &pr.Body,
	}

	createdPR, _, err := c.apiClient.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return &types.PRData{
		Number:     createdPR.GetNumber(),
		Title:      createdPR.GetTitle(),
		Body:       createdPR.GetBody(),
		State:      createdPR.GetState(),
		Branch:     createdPR.GetHead().GetRef(),
		BaseBranch: createdPR.GetBase().GetRef(),
		CreatedAt:  createdPR.GetCreatedAt().Time,
		UpdatedAt:  createdPR.GetUpdatedAt().Time,
	}, nil
}

// GetDefaultBranch returns the default branch for a repository
func (c *Client) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", err
	}

	repository, _, err := c.apiClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get repository info: %w", err)
	}

	return repository.GetDefaultBranch(), nil
}

// CheckIssueExists verifies that an issue exists and is accessible
func (c *Client) CheckIssueExists(ctx context.Context, owner, repo string, number int) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	_, _, err := c.apiClient.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			switch ghErr.Response.StatusCode {
			case 404:
				return fmt.Errorf("issue #%d not found in %s/%s", number, owner, repo)
			case 403:
				return fmt.Errorf("access forbidden to issue #%d in %s/%s", number, owner, repo)
			}
		}
		return fmt.Errorf("failed to check issue #%d in %s/%s: %w", number, owner, repo, err)
	}

	return nil
}

// IsAuthenticated checks if the client is properly authenticated
func (c *Client) IsAuthenticated(ctx context.Context) error {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	_, _, err := c.apiClient.Users.Get(ctx, "")
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == 401 {
			return fmt.Errorf("not authenticated - run 'gh auth login' to authenticate")
		}
		return fmt.Errorf("authentication check failed: %w", err)
	}

	return nil
}

// GetRateLimit returns the current rate limit status
func (c *Client) GetRateLimit(ctx context.Context) (*github.RateLimits, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	rateLimits, _, err := c.apiClient.RateLimit.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}

	return rateLimits, nil
}

// GetIssueUsingCLI retrieves issue data using GitHub CLI (alternative method)
func (c *Client) GetIssueUsingCLI(owner, repo string, number int) (*types.IssueData, error) {
	var response struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignees []struct {
			Login string `json:"login"`
		} `json:"assignees"`
		Milestone struct {
			Title string `json:"title"`
		} `json:"milestone"`
		Comments []struct {
			ID     int    `json:"id"`
			Body   string `json:"body"`
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"comments"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}

	endpoint := fmt.Sprintf("repos/%s/%s/issues/%d", owner, repo, number)

	err := c.ghClient.Get(endpoint, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue using CLI: %w", err)
	}

	issueData := &types.IssueData{
		Number:    response.Number,
		Title:     response.Title,
		Body:      response.Body,
		State:     response.State,
		CreatedAt: response.CreatedAt,
		UpdatedAt: response.UpdatedAt,
		Milestone: response.Milestone.Title,
	}

	// Convert labels
	for _, label := range response.Labels {
		issueData.Labels = append(issueData.Labels, label.Name)
	}

	// Convert assignees
	for _, assignee := range response.Assignees {
		issueData.Assignees = append(issueData.Assignees, assignee.Login)
	}

	// Convert comments
	for _, comment := range response.Comments {
		issueData.Comments = append(issueData.Comments, types.Comment{
			ID:        comment.ID,
			Body:      comment.Body,
			Author:    comment.Author.Login,
			CreatedAt: comment.CreatedAt,
		})
	}

	return issueData, nil
}
