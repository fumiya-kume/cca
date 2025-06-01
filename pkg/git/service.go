package git

import (
	"context"
	"fmt"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/fumiya-kume/cca/pkg/errors"
)

// Service provides high-level git operations for the workflow
type Service struct {
	worktreeManager   *WorktreeManager
	repositoryManager *RepositoryManager
}

// WorkflowGitContext contains git context for a workflow execution
type WorkflowGitContext struct {
	Repository       *LocalRepository
	Worktree         *Worktree
	IssueRef         *types.IssueReference
	OriginalBranch   string
	WorkingDirectory string
}

// NewService creates a new git service
func NewService(basePath string) *Service {
	worktreeManager := NewWorktreeManager(basePath)
	repositoryManager := NewRepositoryManager("")
	if basePath != "" {
		repositoryManager = worktreeManager.repositoryManager
	}

	return &Service{
		worktreeManager:   worktreeManager,
		repositoryManager: repositoryManager,
	}
}

// PrepareWorkspace sets up a complete git workspace for issue development
func (s *Service) PrepareWorkspace(ctx context.Context, issueRef *types.IssueReference) (*WorkflowGitContext, error) {
	// Validate input
	if issueRef == nil {
		return nil, errors.NewError(errors.ErrorTypeValidation).
			WithMessage("issue reference cannot be nil").
			Build()
	}

	if !IsValidGitHubRepository(issueRef.Owner, issueRef.Repo) {
		return nil, errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid repository reference").
			WithContext("owner", issueRef.Owner).
			WithContext("repo", issueRef.Repo).
			Build()
	}

	// Get or clone the repository
	repository, err := s.repositoryManager.GetRepository(issueRef.Owner, issueRef.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Validate repository state
	if err := repository.ValidateRepository(); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	// Get current branch for restoration later
	originalBranch := repository.DefaultBranch

	// Create isolated worktree for this issue
	worktree, err := s.worktreeManager.CreateExternalWorktree(issueRef)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	// Validate worktree state
	if err := worktree.ValidateWorktree(); err != nil {
		// Cleanup on validation failure
		_ = s.worktreeManager.CleanupWorktree(worktree) //nolint:errcheck // Cleanup on failure is best effort
		return nil, fmt.Errorf("worktree validation failed: %w", err)
	}

	gitContext := &WorkflowGitContext{
		Repository:       repository,
		Worktree:         worktree,
		IssueRef:         issueRef,
		OriginalBranch:   originalBranch,
		WorkingDirectory: worktree.Path,
	}

	return gitContext, nil
}

// CleanupWorkspace removes the workspace and restores the original state
func (s *Service) CleanupWorkspace(ctx context.Context, gitContext *WorkflowGitContext) error {
	if gitContext == nil {
		return nil
	}

	var cleanupErrors []error

	// Cleanup worktree
	if gitContext.Worktree != nil {
		if err := s.worktreeManager.CleanupWorktree(gitContext.Worktree); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("failed to cleanup worktree: %w", err))
		}
	}

	// Report any cleanup errors
	if len(cleanupErrors) > 0 {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("workspace cleanup encountered errors").
			WithContext("error_count", len(cleanupErrors)).
			WithCause(cleanupErrors[0]).
			Build()
	}

	return nil
}

// ListActiveWorkspaces returns all currently active git workspaces
func (s *Service) ListActiveWorkspaces() ([]*WorkflowGitContext, error) {
	worktrees, err := s.worktreeManager.ListWorktrees()
	if err != nil {
		return nil, err
	}

	var contexts []*WorkflowGitContext
	for _, worktree := range worktrees {
		context := &WorkflowGitContext{
			Worktree:         worktree,
			WorkingDirectory: worktree.Path,
		}

		// Try to get repository info if available
		if worktree.Repository != nil {
			context.Repository = worktree.Repository
		}

		if worktree.IssueRef != nil {
			context.IssueRef = worktree.IssueRef
		}

		contexts = append(contexts, context)
	}

	return contexts, nil
}

// CleanupOldWorkspaces removes workspaces older than the specified duration
func (s *Service) CleanupOldWorkspaces(maxAge time.Duration) error {
	return s.worktreeManager.CleanupOldWorktrees(maxAge)
}

// ValidateWorkspace checks if a workspace is in a valid state
func (s *Service) ValidateWorkspace(gitContext *WorkflowGitContext) error {
	if gitContext == nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("git context cannot be nil").
			Build()
	}

	// Validate repository
	if gitContext.Repository != nil {
		if err := gitContext.Repository.ValidateRepository(); err != nil {
			return fmt.Errorf("repository validation failed: %w", err)
		}
	}

	// Validate worktree
	if gitContext.Worktree != nil {
		if err := gitContext.Worktree.ValidateWorktree(); err != nil {
			return fmt.Errorf("worktree validation failed: %w", err)
		}
	}

	return nil
}

// GetWorkspaceInfo returns detailed information about a workspace
func (s *Service) GetWorkspaceInfo(gitContext *WorkflowGitContext) map[string]interface{} {
	if gitContext == nil {
		return map[string]interface{}{"error": "no git context"}
	}

	info := map[string]interface{}{
		"working_directory": gitContext.WorkingDirectory,
		"original_branch":   gitContext.OriginalBranch,
	}

	if gitContext.Repository != nil {
		info["repository"] = gitContext.Repository.GetRepositoryInfo()
	}

	if gitContext.Worktree != nil {
		info["worktree"] = gitContext.Worktree.GetWorktreeInfo()
	}

	if gitContext.IssueRef != nil {
		info["issue"] = map[string]interface{}{
			"owner":  gitContext.IssueRef.Owner,
			"repo":   gitContext.IssueRef.Repo,
			"number": gitContext.IssueRef.Number,
			"source": gitContext.IssueRef.Source,
		}
	}

	return info
}

// SwitchToWorkspace changes the working directory to the workspace
func (s *Service) SwitchToWorkspace(gitContext *WorkflowGitContext) (func() error, error) {
	if gitContext == nil || gitContext.WorkingDirectory == "" {
		return nil, errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid workspace context").
			Build()
	}

	// This would be used by the workflow orchestrator to execute commands
	// in the workspace directory. The actual directory switching would be
	// handled by the command execution context.

	// Return a cleanup function that could restore original directory
	return func() error {
		// In practice, this would restore the original working directory
		// For now, it's a no-op since directory switching is handled
		// at the command execution level
		return nil
	}, nil
}

// GetRepositoryManager returns the repository manager for advanced operations
func (s *Service) GetRepositoryManager() *RepositoryManager {
	return s.repositoryManager
}

// GetWorktreeManager returns the worktree manager for advanced operations
func (s *Service) GetWorktreeManager() *WorktreeManager {
	return s.worktreeManager
}

// CreateBranchInWorkspace creates a new branch in the workspace
func (s *Service) CreateBranchInWorkspace(gitContext *WorkflowGitContext, branchName string) error {
	if gitContext == nil || gitContext.Worktree == nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid workspace for branch creation").
			Build()
	}

	// The branch is typically created during worktree creation
	// This method could be used for additional branch operations
	// For now, we'll validate that the branch exists

	currentBranch, err := gitContext.Worktree.GetCurrentBranch()
	if err != nil {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to get current branch").
			WithCause(err).
			Build()
	}

	if currentBranch != branchName && currentBranch != gitContext.Worktree.BranchName {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("workspace is not on expected branch").
			WithContext("current", currentBranch).
			WithContext("expected", branchName).
			Build()
	}

	return nil
}

// GetWorkspaceStatus returns the git status of the workspace
func (s *Service) GetWorkspaceStatus(gitContext *WorkflowGitContext) (map[string]interface{}, error) {
	if gitContext == nil || gitContext.Worktree == nil {
		return nil, errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid workspace for status check").
			Build()
	}

	status := map[string]interface{}{
		"path":   gitContext.Worktree.Path,
		"branch": gitContext.Worktree.BranchName,
		"valid":  true,
	}

	// Get current branch
	if currentBranch, err := gitContext.Worktree.GetCurrentBranch(); err == nil {
		status["current_branch"] = currentBranch
	}

	// Validate workspace
	if err := s.ValidateWorkspace(gitContext); err != nil {
		status["valid"] = false
		status["error"] = err.Error()
	}

	return status, nil
}
