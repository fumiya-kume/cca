package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/fumiya-kume/cca/pkg/errors"
)

// WorktreeManager manages git worktrees for isolated development
type WorktreeManager struct {
	repositoryManager *RepositoryManager
	worktreesPath     string
}

// Worktree represents a git worktree instance
type Worktree struct {
	Path       string
	BranchName string
	IssueRef   *types.IssueReference
	CreatedAt  time.Time
	Repository *LocalRepository
}

// NewWorktreeManager creates a worktree manager that creates worktrees within the project
func NewWorktreeManager(projectRoot string) *WorktreeManager {
	ccaPath := filepath.Join(projectRoot, ".cca")
	repoManager := NewRepositoryManager(projectRoot)
	worktreesPath := filepath.Join(ccaPath, "worktrees")

	return &WorktreeManager{
		repositoryManager: repoManager,
		worktreesPath:     worktreesPath,
	}
}

// CreateExternalWorktree creates a new git worktree for external repositories (legacy)
func (wm *WorktreeManager) CreateExternalWorktree(issueRef *types.IssueReference) (*Worktree, error) {
	// Get or clone the repository
	repo, err := wm.repositoryManager.GetRepository(issueRef.Owner, issueRef.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	// Validate repository state
	if err := repo.ValidateRepository(); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	// Generate branch name
	branchName := wm.generateBranchName(issueRef)

	// Generate worktree path
	worktreePath := wm.generateWorktreePath(issueRef)

	// Ensure worktrees directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0750); err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create worktrees directory").
			WithCause(err).
			WithContext("path", filepath.Dir(worktreePath)).
			Build()
	}

	// Create the worktree using native git commands (more reliable than go-git for worktrees)
	if err := wm.createGitWorktree(repo, worktreePath, branchName); err != nil {
		return nil, err
	}

	worktree := &Worktree{
		Path:       worktreePath,
		BranchName: branchName,
		IssueRef:   issueRef,
		CreatedAt:  time.Now(),
		Repository: repo,
	}

	return worktree, nil
}

// CreateProjectWorktree creates a worktree within the project's .cca directory
func (wm *WorktreeManager) CreateProjectWorktree(name, baseBranch string) (*Worktree, error) {
	// Generate branch name with project prefix
	branchName := fmt.Sprintf("cca/%s-%s", name, time.Now().Format("20060102-150405"))

	// Generate worktree path within .cca
	worktreePath := wm.generateProjectWorktreePath(name)

	// Ensure .cca and worktrees directory exists
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0750); err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create .cca/worktrees directory").
			WithCause(err).
			WithContext("path", filepath.Dir(worktreePath)).
			Build()
	}

	// Create worktree using current repository as base
	if err := wm.createProjectGitWorktree(worktreePath, branchName, baseBranch); err != nil {
		return nil, err
	}

	worktree := &Worktree{
		Path:       worktreePath,
		BranchName: branchName,
		CreatedAt:  time.Now(),
	}

	return worktree, nil
}

// CreateAgentWorktree creates a worktree specifically for agent operations
func (wm *WorktreeManager) CreateAgentWorktree(agentName, taskName string) (*Worktree, error) {
	name := fmt.Sprintf("agent-%s-%s", agentName, taskName)
	return wm.CreateProjectWorktree(name, branchHEAD)
}

// CreateExperimentWorktree creates a worktree for experimental changes
func (wm *WorktreeManager) CreateExperimentWorktree(experimentName string) (*Worktree, error) {
	name := fmt.Sprintf("experiment-%s", experimentName)
	return wm.CreateProjectWorktree(name, branchHEAD)
}

// CreateTaskWorktree creates a worktree for a specific development task
func (wm *WorktreeManager) CreateTaskWorktree(taskID string) (*Worktree, error) {
	name := fmt.Sprintf("task-%s", taskID)
	return wm.CreateProjectWorktree(name, branchHEAD)
}

// generateBranchName creates a branch name following conventions
func (wm *WorktreeManager) generateBranchName(issueRef *types.IssueReference) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("ccagents/issue-%d-%s", issueRef.Number, timestamp)
}

// generateWorktreePath creates a path for the worktree
func (wm *WorktreeManager) generateWorktreePath(issueRef *types.IssueReference) string {
	timestamp := time.Now().Format("20060102-150405")
	dirName := fmt.Sprintf("%s-%s-issue-%d-%s",
		issueRef.Owner, issueRef.Repo, issueRef.Number, timestamp)
	return filepath.Join(wm.worktreesPath, dirName)
}

// generateProjectWorktreePath creates a path for project-local worktrees
func (wm *WorktreeManager) generateProjectWorktreePath(name string) string {
	timestamp := time.Now().Format("20060102-150405")
	dirName := fmt.Sprintf("%s-%s", name, timestamp)
	return filepath.Join(wm.worktreesPath, dirName)
}

// createGitWorktree creates a git worktree using native git commands
func (wm *WorktreeManager) createGitWorktree(repo *LocalRepository, worktreePath, branchName string) error {
	// Change to repository directory for git commands
	originalDir, err := os.Getwd()
	if err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to get current directory").
			WithCause(err).
			Build()
	}
	defer func() { _ = os.Chdir(originalDir) }() //nolint:errcheck // Directory restoration is best effort

	if err := os.Chdir(repo.Path); err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to change to repository directory").
			WithCause(err).
			WithContext("path", repo.Path).
			Build()
	}

	// Create worktree with new branch based on default branch
	// #nosec G204 - git worktree command with validated parameters
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, repo.DefaultBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up on failure
		_ = os.RemoveAll(worktreePath) //nolint:errcheck // Cleanup on failure is best effort
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to create git worktree").
			WithCause(err).
			WithContext("command", cmd.String()).
			WithContext("output", string(output)).
			WithContext("branch", branchName).
			WithContext("path", worktreePath).
			Build()
	}

	return nil
}

// createProjectGitWorktree creates a git worktree within the current project
func (wm *WorktreeManager) createProjectGitWorktree(worktreePath, branchName, baseBranch string) error {
	// Get current working directory (should be the git repository)
	currentDir, err := os.Getwd()
	if err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to get current directory").
			WithCause(err).
			Build()
	}

	// Find the git root directory
	gitRoot, err := wm.findGitRoot(currentDir)
	if err != nil {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("not in a git repository").
			WithCause(err).
			WithContext("current_dir", currentDir).
			Build()
	}

	// Change to git root for worktree operations
	originalDir := currentDir
	defer func() { _ = os.Chdir(originalDir) }() //nolint:errcheck // Directory restoration is best effort

	if err := os.Chdir(gitRoot); err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to change to git root directory").
			WithCause(err).
			WithContext("git_root", gitRoot).
			Build()
	}

	// Resolve the base branch (default to current branch if "HEAD")
	if baseBranch == branchHEAD {
		baseBranch, err = wm.getCurrentBranch()
		if err != nil {
			return errors.NewError(errors.ErrorTypeGit).
				WithMessage("failed to get current branch").
				WithCause(err).
				Build()
		}
	}

	// Create worktree with new branch based on specified branch
	// #nosec G204 - git worktree command with validated parameters
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up on failure
		_ = os.RemoveAll(worktreePath) //nolint:errcheck // Cleanup on failure is best effort
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to create project git worktree").
			WithCause(err).
			WithContext("command", cmd.String()).
			WithContext("output", string(output)).
			WithContext("branch", branchName).
			WithContext("base_branch", baseBranch).
			WithContext("path", worktreePath).
			Build()
	}

	return nil
}

// findGitRoot finds the root directory of the git repository
func (wm *WorktreeManager) findGitRoot(startPath string) (string, error) {
	path := startPath
	for {
		gitDir := filepath.Join(path, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return path, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			// Reached filesystem root
			break
		}
		path = parent
	}

	return "", fmt.Errorf("not in a git repository")
}

// getCurrentBranch gets the current branch name
func (wm *WorktreeManager) getCurrentBranch() (string, error) {
	// #nosec G204 - git rev-parse command with fixed arguments
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(string(output))
	if branch == branchHEAD {
		// Detached HEAD state, get commit hash instead
		cmd = exec.Command("git", "rev-parse", "HEAD")
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	}

	return branch, nil
}

// ListWorktrees lists all active worktrees
func (wm *WorktreeManager) ListWorktrees() ([]*Worktree, error) {
	var worktrees []*Worktree

	if _, err := os.Stat(wm.worktreesPath); os.IsNotExist(err) {
		return worktrees, nil
	}

	entries, err := os.ReadDir(wm.worktreesPath)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to read worktrees directory").
			WithCause(err).
			WithContext("path", wm.worktreesPath).
			Build()
	}

	for _, entry := range entries {
		if entry.IsDir() {
			worktreePath := filepath.Join(wm.worktreesPath, entry.Name())

			// Try to parse worktree info from directory name
			worktree, err := wm.parseWorktreeFromPath(worktreePath)
			if err != nil {
				// Skip invalid worktrees
				continue
			}

			worktrees = append(worktrees, worktree)
		}
	}

	return worktrees, nil
}

// parseWorktreeFromPath attempts to extract worktree information from the path
func (wm *WorktreeManager) parseWorktreeFromPath(worktreePath string) (*Worktree, error) {
	dirName := filepath.Base(worktreePath)

	// Expected format: owner-repo-issue-number-timestamp
	parts := strings.Split(dirName, "-")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid worktree directory format: %s", dirName)
	}

	// This is a simplified parser - in practice, you might want more robust parsing
	// For now, just create a basic worktree object
	return &Worktree{
		Path:      worktreePath,
		CreatedAt: time.Now(), // We'd need to get actual creation time
	}, nil
}

// CleanupWorktree removes a worktree and its branch
func (wm *WorktreeManager) CleanupWorktree(worktree *Worktree) error {
	if worktree.Repository == nil {
		return wm.cleanupWorktreeByPath(worktree.Path)
	}

	// Change to repository directory
	originalDir, err := os.Getwd()
	if err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to get current directory").
			WithCause(err).
			Build()
	}
	defer func() { _ = os.Chdir(originalDir) }() //nolint:errcheck // Directory restoration is best effort

	if err := os.Chdir(worktree.Repository.Path); err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to change to repository directory").
			WithCause(err).
			WithContext("path", worktree.Repository.Path).
			Build()
	}

	// Remove worktree
	// #nosec G204 - git worktree remove with validated path
	cmd := exec.Command("git", "worktree", "remove", worktree.Path, "--force")
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try manual cleanup if git command fails
		_ = os.RemoveAll(worktree.Path) //nolint:errcheck // Manual cleanup fallback is best effort

		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to remove git worktree").
			WithCause(err).
			WithContext("command", cmd.String()).
			WithContext("output", string(output)).
			WithContext("path", worktree.Path).
			Build()
	}

	// Delete the branch if it exists
	if worktree.BranchName != "" {
		// #nosec G204 - git branch delete with validated branch name
		cmd = exec.Command("git", "branch", "-D", worktree.BranchName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Branch deletion failure is not critical
			// Log it but don't fail the entire cleanup
			fmt.Printf("Warning: failed to delete branch %s: %s\n", worktree.BranchName, string(output))
		}
	}

	return nil
}

// cleanupWorktreeByPath removes a worktree directory directly
func (wm *WorktreeManager) cleanupWorktreeByPath(worktreePath string) error {
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return nil // Already doesn't exist
	}

	err := os.RemoveAll(worktreePath)
	if err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to remove worktree directory").
			WithCause(err).
			WithContext("path", worktreePath).
			Build()
	}

	return nil
}

// CleanupOldWorktrees removes worktrees older than the specified duration
func (wm *WorktreeManager) CleanupOldWorktrees(maxAge time.Duration) error {
	worktrees, err := wm.ListWorktrees()
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	var errors []error

	for _, worktree := range worktrees {
		if worktree.CreatedAt.Before(cutoff) {
			if err := wm.CleanupWorktree(worktree); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to cleanup %d worktrees", len(errors))
	}

	return nil
}

// GetWorktreeInfo returns information about a worktree
func (w *Worktree) GetWorktreeInfo() map[string]interface{} {
	info := map[string]interface{}{
		"path":        w.Path,
		"branch_name": w.BranchName,
		"created_at":  w.CreatedAt,
	}

	if w.IssueRef != nil {
		info["issue_number"] = w.IssueRef.Number
		info["owner"] = w.IssueRef.Owner
		info["repo"] = w.IssueRef.Repo
	}

	if w.Repository != nil {
		info["repository"] = w.Repository.GetRepositoryInfo()
	}

	return info
}

// ValidateWorktree checks if the worktree is in a valid state
func (w *Worktree) ValidateWorktree() error {
	// Check if worktree directory exists
	if _, err := os.Stat(w.Path); os.IsNotExist(err) {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("worktree directory does not exist").
			WithContext("path", w.Path).
			Build()
	}

	// Check if it's a valid git directory
	gitDir := filepath.Join(w.Path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("worktree is not a valid git directory").
			WithContext("path", w.Path).
			Build()
	}

	return nil
}

// GetCurrentBranch returns the current branch in the worktree
func (w *Worktree) GetCurrentBranch() (string, error) {
	originalDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Chdir(originalDir) }() //nolint:errcheck // Directory restoration is best effort

	if err := os.Chdir(w.Path); err != nil {
		return "", err
	}

	// #nosec G204 - git rev-parse command with fixed arguments
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
