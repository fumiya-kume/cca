// Package git provides Git repository management and operations for ccAgents
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fumiya-kume/cca/pkg/errors"
)

// Branch constants
const (
	branchMain = "main"
	branchHEAD = "HEAD"
)

// RepositoryInfo contains information about the current git repository
type RepositoryInfo struct {
	Owner      string
	Repo       string
	RemoteURL  string
	Branch     string
	IsGitRepo  bool
	WorkingDir string
}

// GetRepositoryContext extracts repository information from the current directory
func GetRepositoryContext() (*RepositoryInfo, error) {
	info := &RepositoryInfo{
		WorkingDir: getCurrentWorkingDir(),
	}

	// Check if we're in a git repository
	if !isGitRepository() {
		info.IsGitRepo = false
		return info, nil
	}

	info.IsGitRepo = true

	// Get the remote URL
	remoteURL, err := getRemoteURL()
	if err != nil {
		return info, errors.GitError("get remote URL", err)
	}
	info.RemoteURL = remoteURL

	// Parse owner and repo from remote URL
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return info, errors.GitError("parse GitHub URL", err)
	}
	info.Owner = owner
	info.Repo = repo

	// Get current branch
	branch, err := getCurrentBranch()
	if err != nil {
		return info, errors.GitError("get current branch", err)
	}
	info.Branch = branch

	return info, nil
}

// getCurrentWorkingDir returns the current working directory
func getCurrentWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

// isGitRepository checks if the current directory is within a git repository
func isGitRepository() bool {
	// Look for .git directory in current or parent directories
	dir, err := os.Getwd()
	if err != nil {
		return false
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if stat, err := os.Stat(gitDir); err == nil {
			// Check if it's a directory or a file (for git worktrees)
			return stat.IsDir() || stat.Mode().IsRegular()
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	return false
}

// getRemoteURL gets the remote URL for the current repository
func getRemoteURL() (string, error) {
	// Try 'origin' first, then fall back to the first remote
	remotes := []string{"origin", ""}

	for _, remote := range remotes {
		var cmd *exec.Cmd
		if remote == "" {
			// Get the first remote
			// #nosec G204 - git remote command with no arguments
			cmd = exec.Command("git", "remote")
		} else {
			// #nosec G204 - git remote get-url with validated remote name
			cmd = exec.Command("git", "remote", "get-url", remote)
		}

		output, err := cmd.Output()
		if err != nil {
			continue
		}

		result := strings.TrimSpace(string(output))
		if remote == "" {
			// We got a list of remotes, try the first one
			lines := strings.Split(result, "\n")
			if len(lines) > 0 && lines[0] != "" {
				// #nosec G204 - git remote get-url with validated remote name
				cmd = exec.Command("git", "remote", "get-url", lines[0])
				output, err = cmd.Output()
				if err != nil {
					continue
				}
				result = strings.TrimSpace(string(output))
			}
		}

		if result != "" {
			return result, nil
		}
	}

	return "", fmt.Errorf("no remote URL found")
}

// parseGitHubURL extracts owner and repository name from a GitHub URL
func parseGitHubURL(url string) (string, string, error) {
	// GitHub URL patterns
	patterns := []*regexp.Regexp{
		// HTTPS: https://github.com/owner/repo.git
		regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`),
		// SSH: git@github.com:owner/repo.git
		regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?/?$`),
		// SSH alternative: ssh://git@github.com/owner/repo.git
		regexp.MustCompile(`^ssh://git@github\.com/([^/]+)/([^/]+?)(?:\.git)?/?$`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(url)
		if len(matches) == 3 {
			return matches[1], matches[2], nil
		}
	}

	return "", "", fmt.Errorf("not a valid GitHub URL: %s", url)
}

// getCurrentBranch gets the current git branch
func getCurrentBranch() (string, error) {
	// #nosec G204 - git rev-parse command with fixed arguments
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(string(output))
	if branch == branchHEAD {
		// We're in detached HEAD state, try to get the branch from symbolic-ref
		// #nosec G204 - git symbolic-ref command with fixed arguments
		cmd = exec.Command("git", "symbolic-ref", "--short", "HEAD")
		output, err = cmd.Output()
		if err == nil {
			branch = strings.TrimSpace(string(output))
		}
	}

	return branch, nil
}

// IsValidGitHubRepository checks if the given owner/repo combination is valid
func IsValidGitHubRepository(owner, repo string) bool {
	if owner == "" || repo == "" {
		return false
	}

	// Basic validation for GitHub usernames and repository names
	validName := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$`)

	if !validName.MatchString(owner) || !validName.MatchString(repo) {
		return false
	}

	// Additional length constraints (GitHub's actual limits)
	if len(owner) > 39 || len(repo) > 100 {
		return false
	}

	return true
}

// GetDefaultBranch attempts to determine the default branch for a repository
func GetDefaultBranch(owner, repo string) (string, error) {
	// Common default branches to try
	defaultBranches := []string{branchMain, "master", "develop"}

	for _, branch := range defaultBranches {
		// Try to check if the branch exists (this would require GitHub API)
		// For now, return the first common one
		return branch, nil
	}

	return branchMain, nil // Default to 'main' as GitHub's current default
}
