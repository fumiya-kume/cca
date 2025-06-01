package git

import (
	"testing"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
		hasError      bool
	}{
		{
			name:          "HTTPS URL",
			url:           "https://github.com/octocat/hello-world.git",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			hasError:      false,
		},
		{
			name:          "HTTPS URL without .git",
			url:           "https://github.com/octocat/hello-world",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			hasError:      false,
		},
		{
			name:          "SSH URL",
			url:           "git@github.com:octocat/hello-world.git",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			hasError:      false,
		},
		{
			name:          "SSH URL without .git",
			url:           "git@github.com:octocat/hello-world",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			hasError:      false,
		},
		{
			name:          "SSH alternative URL",
			url:           "ssh://git@github.com/octocat/hello-world.git",
			expectedOwner: "octocat",
			expectedRepo:  "hello-world",
			hasError:      false,
		},
		{
			name:     "invalid URL",
			url:      "https://gitlab.com/user/repo.git",
			hasError: true,
		},
		{
			name:     "malformed URL",
			url:      "not-a-url",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubURL(tt.url)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none for URL: %s", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if owner != tt.expectedOwner {
				t.Errorf("Expected owner %s, got %s", tt.expectedOwner, owner)
			}

			if repo != tt.expectedRepo {
				t.Errorf("Expected repo %s, got %s", tt.expectedRepo, repo)
			}
		})
	}
}

func TestIsValidGitHubRepository(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		expected bool
	}{
		{
			name:     "valid repository",
			owner:    "octocat",
			repo:     "hello-world",
			expected: true,
		},
		{
			name:     "valid repository with numbers",
			owner:    "user123",
			repo:     "repo456",
			expected: true,
		},
		{
			name:     "empty owner",
			owner:    "",
			repo:     "repo",
			expected: false,
		},
		{
			name:     "empty repo",
			owner:    "owner",
			repo:     "",
			expected: false,
		},
		{
			name:     "invalid owner with special chars",
			owner:    "user@domain",
			repo:     "repo",
			expected: false,
		},
		{
			name:     "invalid repo with special chars",
			owner:    "owner",
			repo:     "repo@name",
			expected: false,
		},
		{
			name:     "owner too long",
			owner:    "this-username-is-way-too-long-for-github-which-has-limits",
			repo:     "repo",
			expected: false,
		},
		{
			name:     "repo too long",
			owner:    "owner",
			repo:     "this-repository-name-is-way-too-long-for-github-repositories-which-have-a-maximum-length-limit-set-and-exceeds-100-chars",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidGitHubRepository(tt.owner, tt.repo)
			if result != tt.expected {
				t.Errorf("IsValidGitHubRepository(%s, %s) = %v, expected %v",
					tt.owner, tt.repo, result, tt.expected)
			}
		})
	}
}

func TestGetDefaultBranch(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		expectedBranch string
		hasError       bool
	}{
		{
			name:           "valid repository",
			owner:          "octocat",
			repo:           "hello-world",
			expectedBranch: "main",
			hasError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, err := GetDefaultBranch(tt.owner, tt.repo)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if branch != tt.expectedBranch {
				t.Errorf("Expected branch %s, got %s", tt.expectedBranch, branch)
			}
		})
	}
}

// Note: Tests for GetRepositoryContext, isGitRepository, getRemoteURL, and getCurrentBranch
// would require actual git repositories or extensive mocking, so they are omitted here.
// In a real project, these would use test fixtures or dependency injection for testing.
