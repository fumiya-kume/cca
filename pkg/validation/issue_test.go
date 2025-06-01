package validation

import (
	"testing"

	"github.com/fumiya-kume/cca/internal/types"
)

func TestValidateIssueReference(t *testing.T) {
	tests := []struct {
		name     string
		ref      *types.IssueReference
		hasError bool
	}{
		{
			name: "valid reference",
			ref: &types.IssueReference{
				Owner:  "octocat",
				Repo:   "hello-world",
				Number: 123,
				Source: "url",
			},
			hasError: false,
		},
		{
			name:     "nil reference",
			ref:      nil,
			hasError: true,
		},
		{
			name: "invalid issue number",
			ref: &types.IssueReference{
				Owner:  "octocat",
				Repo:   "hello-world",
				Number: 0,
				Source: "url",
			},
			hasError: true,
		},
		{
			name: "invalid owner",
			ref: &types.IssueReference{
				Owner:  "-invalid",
				Repo:   "hello-world",
				Number: 123,
				Source: "url",
			},
			hasError: true,
		},
		{
			name: "invalid repo",
			ref: &types.IssueReference{
				Owner:  "octocat",
				Repo:   ".invalid",
				Number: 123,
				Source: "url",
			},
			hasError: true,
		},
		{
			name: "invalid source",
			ref: &types.IssueReference{
				Owner:  "octocat",
				Repo:   "hello-world",
				Number: 123,
				Source: "invalid",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIssueReference(tt.ref)
			if (err != nil) != tt.hasError {
				t.Errorf("ValidateIssueReference() error = %v, hasError %v", err, tt.hasError)
			}
		})
	}
}

func TestValidateIssueURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		hasError bool
	}{
		{
			name:     "valid GitHub issue URL",
			url:      "https://github.com/octocat/hello-world/issues/123",
			hasError: false,
		},
		{
			name:     "empty URL",
			url:      "",
			hasError: true,
		},
		{
			name:     "invalid URL format",
			url:      "not-a-url",
			hasError: true,
		},
		{
			name:     "non-GitHub URL",
			url:      "https://gitlab.com/user/repo/issues/123",
			hasError: true,
		},
		{
			name:     "invalid GitHub path",
			url:      "https://github.com/octocat/hello-world/pulls/123",
			hasError: true,
		},
		{
			name:     "missing issue number",
			url:      "https://github.com/octocat/hello-world/issues/",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIssueURL(tt.url)
			if (err != nil) != tt.hasError {
				t.Errorf("ValidateIssueURL() error = %v, hasError %v", err, tt.hasError)
			}
		})
	}
}

func TestValidateShorthand(t *testing.T) {
	tests := []struct {
		name      string
		shorthand string
		hasError  bool
	}{
		{
			name:      "valid shorthand",
			shorthand: "octocat/hello-world#123",
			hasError:  false,
		},
		{
			name:      "empty shorthand",
			shorthand: "",
			hasError:  true,
		},
		{
			name:      "invalid format",
			shorthand: "octocat/hello-world-123",
			hasError:  true,
		},
		{
			name:      "invalid owner",
			shorthand: "-invalid/hello-world#123",
			hasError:  true,
		},
		{
			name:      "invalid repo",
			shorthand: "octocat/.invalid#123",
			hasError:  true,
		},
		{
			name:      "invalid issue number",
			shorthand: "octocat/hello-world#0",
			hasError:  true,
		},
		{
			name:      "non-numeric issue number",
			shorthand: "octocat/hello-world#abc",
			hasError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShorthand(tt.shorthand)
			if (err != nil) != tt.hasError {
				t.Errorf("ValidateShorthand() error = %v, hasError %v", err, tt.hasError)
			}
		})
	}
}

func TestValidateContextReference(t *testing.T) {
	tests := []struct {
		name       string
		contextRef string
		hasError   bool
	}{
		{
			name:       "valid context reference",
			contextRef: "#123",
			hasError:   false,
		},
		{
			name:       "empty context reference",
			contextRef: "",
			hasError:   true,
		},
		{
			name:       "invalid format",
			contextRef: "123",
			hasError:   true,
		},
		{
			name:       "invalid issue number",
			contextRef: "#0",
			hasError:   true,
		},
		{
			name:       "non-numeric issue number",
			contextRef: "#abc",
			hasError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContextReference(tt.contextRef)
			if (err != nil) != tt.hasError {
				t.Errorf("ValidateContextReference() error = %v, hasError %v", err, tt.hasError)
			}
		})
	}
}

func TestValidateGitHubUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		hasError bool
	}{
		{
			name:     "valid username",
			username: "octocat",
			hasError: false,
		},
		{
			name:     "valid username with hyphen",
			username: "hello-world",
			hasError: false,
		},
		{
			name:     "valid username with numbers",
			username: "user123",
			hasError: false,
		},
		{
			name:     "empty username",
			username: "",
			hasError: true,
		},
		{
			name:     "username starting with hyphen",
			username: "-invalid",
			hasError: true,
		},
		{
			name:     "username ending with hyphen",
			username: "invalid-",
			hasError: true,
		},
		{
			name:     "username with consecutive hyphens",
			username: "hello--world",
			hasError: true,
		},
		{
			name:     "username too long",
			username: "this-username-is-way-too-long-for-github",
			hasError: true,
		},
		{
			name:     "username with special characters",
			username: "user@domain",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitHubUsername(tt.username)
			if (err != nil) != tt.hasError {
				t.Errorf("validateGitHubUsername() error = %v, hasError %v", err, tt.hasError)
			}
		})
	}
}

func TestValidateGitHubRepository(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		hasError bool
	}{
		{
			name:     "valid repository name",
			repoName: "hello-world",
			hasError: false,
		},
		{
			name:     "valid repository with underscore",
			repoName: "hello_world",
			hasError: false,
		},
		{
			name:     "valid repository with period",
			repoName: "hello.world",
			hasError: false,
		},
		{
			name:     "empty repository name",
			repoName: "",
			hasError: true,
		},
		{
			name:     "repository starting with period",
			repoName: ".invalid",
			hasError: true,
		},
		{
			name:     "repository ending with .git",
			repoName: "repo.git",
			hasError: true,
		},
		{
			name:     "repository name too long",
			repoName: "this-repository-name-is-way-too-long-for-github-repositories-which-have-a-maximum-length-limit-and-exceeds-100-chars",
			hasError: true,
		},
		{
			name:     "repository with invalid characters",
			repoName: "repo@name",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitHubRepository(tt.repoName)
			if (err != nil) != tt.hasError {
				t.Errorf("validateGitHubRepository() error = %v, hasError %v", err, tt.hasError)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal input",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "input with leading/trailing spaces",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "input with null bytes",
			input:    "hello\x00world",
			expected: "helloworld",
		},
		{
			name:     "input with control characters",
			input:    "hello\x01\x02world",
			expected: "helloworld",
		},
		{
			name:     "input with tabs and newlines",
			input:    "hello\tworld\n",
			expected: "hello\tworld", // Trailing newline gets trimmed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestValidateInputLength(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		fieldName string
		hasError  bool
	}{
		{
			name:      "valid length",
			input:     "hello",
			maxLength: 10,
			fieldName: "test",
			hasError:  false,
		},
		{
			name:      "exact max length",
			input:     "hello",
			maxLength: 5,
			fieldName: "test",
			hasError:  false,
		},
		{
			name:      "exceeds max length",
			input:     "hello world",
			maxLength: 5,
			fieldName: "test",
			hasError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputLength(tt.input, tt.maxLength, tt.fieldName)
			if (err != nil) != tt.hasError {
				t.Errorf("ValidateInputLength() error = %v, hasError %v", err, tt.hasError)
			}
		})
	}
}
