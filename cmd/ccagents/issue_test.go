package main

import (
	"testing"

	"github.com/fumiya-kume/cca/internal/types"
)

func TestParseIssueReference(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *types.IssueReference
		hasError bool
	}{
		{
			name:  "full GitHub URL",
			input: "https://github.com/owner/repo/issues/123",
			expected: &types.IssueReference{
				Owner:  "owner",
				Repo:   "repo",
				Number: 123,
				Source: "url",
			},
			hasError: false,
		},
		{
			name:  "full GitHub URL with query params",
			input: "https://github.com/owner/repo/issues/456?tab=comments",
			expected: &types.IssueReference{
				Owner:  "owner",
				Repo:   "repo",
				Number: 456,
				Source: "url",
			},
			hasError: false,
		},
		{
			name:  "shorthand format",
			input: "owner/repo#789",
			expected: &types.IssueReference{
				Owner:  "owner",
				Repo:   "repo",
				Number: 789,
				Source: "shorthand",
			},
			hasError: false,
		},
		{
			name:  "context-aware format in git repo",
			input: "#123",
			expected: &types.IssueReference{
				Owner:  "fumiya-kume", // Will be extracted from current repo
				Repo:   "cca",         // Will be extracted from current repo
				Number: 123,
				Source: "context",
			},
			hasError: false, // Should work when in a git repository
		},
		{
			name:  "full GitHub URL with query params and fragment",
			input: "https://github.com/owner/repo/issues/789?tab=comments#issuecomment-123",
			expected: &types.IssueReference{
				Owner:  "owner",
				Repo:   "repo",
				Number: 789,
				Source: "url",
			},
			hasError: false,
		},
		{
			name:  "shorthand with hyphenated names",
			input: "my-org/my-repo#456",
			expected: &types.IssueReference{
				Owner:  "my-org",
				Repo:   "my-repo",
				Number: 456,
				Source: "shorthand",
			},
			hasError: false,
		},
		{
			name:     "invalid format",
			input:    "invalid-input",
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid number in URL",
			input:    "https://github.com/owner/repo/issues/abc",
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid number in shorthand",
			input:    "owner/repo#abc",
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid number in context",
			input:    "#abc",
			expected: nil,
			hasError: true,
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
			hasError: true,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
			hasError: true,
		},
		{
			name:  "very large issue number",
			input: "https://github.com/owner/repo/issues/999999999",
			expected: &types.IssueReference{
				Owner:  "owner",
				Repo:   "repo",
				Number: 999999999,
				Source: "url",
			},
			hasError: false,
		},
		{
			name:  "shorthand with single character names",
			input: "a/b#1",
			expected: &types.IssueReference{
				Owner:  "a",
				Repo:   "b",
				Number: 1,
				Source: "shorthand",
			},
			hasError: false,
		},
		{
			name:     "URL with trailing slash",
			input:    "https://github.com/owner/repo/issues/123/",
			expected: nil,
			hasError: true,
		},
		{
			name:     "shorthand with missing issue number",
			input:    "owner/repo#",
			expected: nil,
			hasError: true,
		},
		{
			name:  "context reference with multiple digits",
			input: "#123456",
			expected: &types.IssueReference{
				Owner:  "fumiya-kume",
				Repo:   "cca",
				Number: 123456,
				Source: "context",
			},
			hasError: false, // Should work in git repo
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIssueReference(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none for input: %s", tt.input)
				}
				if result != nil {
					t.Errorf("Expected nil result but got: %+v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected result but got nil")
				return
			}

			if result.Owner != tt.expected.Owner {
				t.Errorf("Owner mismatch: got %s, want %s", result.Owner, tt.expected.Owner)
			}

			if result.Repo != tt.expected.Repo {
				t.Errorf("Repo mismatch: got %s, want %s", result.Repo, tt.expected.Repo)
			}

			if result.Number != tt.expected.Number {
				t.Errorf("Number mismatch: got %d, want %d", result.Number, tt.expected.Number)
			}

			if result.Source != tt.expected.Source {
				t.Errorf("Source mismatch: got %s, want %s", result.Source, tt.expected.Source)
			}
		})
	}
}
