package git

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// RepositoryManager handles local repository operations
type RepositoryManager struct {
	basePath string
}

// LocalRepository represents a local git repository
type LocalRepository struct {
	Owner         string
	Repo          string
	Path          string
	Repository    *git.Repository
	DefaultBranch string
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(basePath string) *RepositoryManager {
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			basePath = "/tmp/ccagents"
		} else {
			basePath = filepath.Join(homeDir, ".ccagents", "repos")
		}
	}

	return &RepositoryManager{
		basePath: basePath,
	}
}

// GetRepository gets or creates a local repository
func (rm *RepositoryManager) GetRepository(owner, repo string) (*LocalRepository, error) {
	if !IsValidGitHubRepository(owner, repo) {
		return nil, errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid repository name").
			WithContext("owner", owner).
			WithContext("repo", repo).
			Build()
	}

	repoPath := filepath.Join(rm.basePath, owner, repo)

	// Check if repository already exists
	if _, err := os.Stat(repoPath); err == nil {
		return rm.openRepository(owner, repo, repoPath)
	}

	// Repository doesn't exist, clone it
	return rm.cloneRepository(owner, repo, repoPath)
}

// openRepository opens an existing local repository
func (rm *RepositoryManager) openRepository(owner, repo, repoPath string) (*LocalRepository, error) {
	gitRepo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to open repository").
			WithCause(err).
			WithContext("path", repoPath).
			Build()
	}

	defaultBranch, err := rm.getDefaultBranch(gitRepo)
	if err != nil {
		return nil, err
	}

	// Update to latest default branch
	if err := rm.updateToLatest(gitRepo, defaultBranch); err != nil {
		return nil, err
	}

	return &LocalRepository{
		Owner:         owner,
		Repo:          repo,
		Path:          repoPath,
		Repository:    gitRepo,
		DefaultBranch: defaultBranch,
	}, nil
}

// cloneRepository clones a remote repository
func (rm *RepositoryManager) cloneRepository(owner, repo, repoPath string) (*LocalRepository, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(repoPath), 0750); err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create repository directory").
			WithCause(err).
			WithContext("path", filepath.Dir(repoPath)).
			Build()
	}

	// Construct GitHub URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	// Clone with sparse checkout for performance
	gitRepo, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
		// Use sparse checkout to save space initially
		SingleBranch: true,
	})

	if err != nil {
		// Clean up on failure
		_ = os.RemoveAll(repoPath) //nolint:errcheck // Cleanup on failure is best effort
		return nil, errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to clone repository").
			WithCause(err).
			WithContext("url", repoURL).
			WithContext("path", repoPath).
			Build()
	}

	defaultBranch, err := rm.getDefaultBranch(gitRepo)
	if err != nil {
		return nil, err
	}

	return &LocalRepository{
		Owner:         owner,
		Repo:          repo,
		Path:          repoPath,
		Repository:    gitRepo,
		DefaultBranch: defaultBranch,
	}, nil
}

// getDefaultBranch determines the default branch of a repository
func (rm *RepositoryManager) getDefaultBranch(repo *git.Repository) (string, error) {
	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return "", errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to get HEAD reference").
			WithCause(err).
			Build()
	}

	// Extract branch name from HEAD reference
	if head.Name().IsBranch() {
		return head.Name().Short(), nil
	}

	// If HEAD is not pointing to a branch, try to get remote HEAD
	remotes, err := repo.Remotes()
	if err != nil || len(remotes) == 0 {
		return branchMain, nil // Default fallback
	}

	// Try to get the default branch from the remote
	remote := remotes[0]
	refList, err := remote.List(&git.ListOptions{})
	if err != nil {
		return branchMain, nil // Default fallback
	}

	// Look for HEAD reference in remote
	for _, ref := range refList {
		if ref.Name() == plumbing.HEAD {
			if ref.Target().IsBranch() {
				return ref.Target().Short(), nil
			}
		}
	}

	// Common default branches
	commonDefaults := []string{"main", "master", "develop"}
	for _, branch := range commonDefaults {
		branchRef := plumbing.NewBranchReferenceName(branch)
		if _, err := repo.Reference(branchRef, true); err == nil {
			return branch, nil
		}
	}

	return "main", nil // Ultimate fallback
}

// updateToLatest updates the repository to the latest commit of the default branch
func (rm *RepositoryManager) updateToLatest(repo *git.Repository, defaultBranch string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to get worktree").
			WithCause(err).
			Build()
	}

	// Fetch latest changes
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})

	// It's OK if there's nothing to fetch
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to fetch latest changes").
			WithCause(err).
			Build()
	}

	// Checkout default branch
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(defaultBranch),
	})

	if err != nil {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to checkout default branch").
			WithCause(err).
			WithContext("branch", defaultBranch).
			Build()
	}

	// Pull latest changes
	err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
	})

	// It's OK if there's nothing to pull
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("failed to pull latest changes").
			WithCause(err).
			Build()
	}

	return nil
}

// GetRepositoryPath returns the path where a repository would be stored
func (rm *RepositoryManager) GetRepositoryPath(owner, repo string) string {
	return filepath.Join(rm.basePath, owner, repo)
}

// ListRepositories lists all locally cached repositories
func (rm *RepositoryManager) ListRepositories() ([]string, error) {
	var repos []string

	if _, err := os.Stat(rm.basePath); os.IsNotExist(err) {
		return repos, nil
	}

	err := filepath.Walk(rm.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors and continue
		}

		// Check if this is a git repository
		if info.IsDir() && info.Name() == ".git" {
			repoPath := filepath.Dir(path)
			relPath, err := filepath.Rel(rm.basePath, repoPath)
			if err == nil {
				repos = append(repos, relPath)
			}
		}

		return nil
	})

	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to list repositories").
			WithCause(err).
			Build()
	}

	return repos, nil
}

// CleanupRepository removes a repository from local cache
func (rm *RepositoryManager) CleanupRepository(owner, repo string) error {
	repoPath := rm.GetRepositoryPath(owner, repo)

	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil // Already doesn't exist
	}

	err := os.RemoveAll(repoPath)
	if err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to remove repository").
			WithCause(err).
			WithContext("path", repoPath).
			Build()
	}

	return nil
}

// GetRepositoryInfo returns basic information about a local repository
func (lr *LocalRepository) GetRepositoryInfo() map[string]interface{} {
	return map[string]interface{}{
		"owner":          lr.Owner,
		"repo":           lr.Repo,
		"path":           lr.Path,
		"default_branch": lr.DefaultBranch,
		"full_name":      fmt.Sprintf("%s/%s", lr.Owner, lr.Repo),
	}
}

// GetRemoteURL returns the remote URL for the repository
func (lr *LocalRepository) GetRemoteURL() (string, error) {
	remotes, err := lr.Repository.Remotes()
	if err != nil || len(remotes) == 0 {
		return "", errors.NewError(errors.ErrorTypeGit).
			WithMessage("no remotes found").
			WithCause(err).
			Build()
	}

	// Get origin remote or first available
	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			urls := remote.Config().URLs
			if len(urls) > 0 {
				return urls[0], nil
			}
		}
	}

	// Fallback to first remote
	urls := remotes[0].Config().URLs
	if len(urls) > 0 {
		return urls[0], nil
	}

	return "", errors.NewError(errors.ErrorTypeGit).
		WithMessage("no remote URLs found").
		Build()
}

// ValidateRepository checks if the local repository is in a good state
func (lr *LocalRepository) ValidateRepository() error {
	// Check if repository is accessible
	_, err := lr.Repository.Head()
	if err != nil {
		return errors.NewError(errors.ErrorTypeGit).
			WithMessage("repository is in invalid state").
			WithCause(err).
			Build()
	}

	// Check if working directory exists
	if _, err := os.Stat(lr.Path); os.IsNotExist(err) {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("repository path does not exist").
			WithContext("path", lr.Path).
			Build()
	}

	return nil
}
