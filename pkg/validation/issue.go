// Package validation provides validation utilities and issue management for ccAgents
package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/fumiya-kume/cca/pkg/errors"
)

// ValidateIssueReference validates an issue reference for completeness and correctness
func ValidateIssueReference(ref *types.IssueReference) error {
	if ref == nil {
		return errors.ValidationError("issue reference cannot be nil")
	}

	// Validate issue number
	if ref.Number <= 0 {
		return errors.ValidationError("issue number must be positive")
	}

	// Validate owner
	if err := validateGitHubUsername(ref.Owner); err != nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid repository owner").
			WithCause(err).
			WithContext("owner", ref.Owner).
			WithSuggestion("Use a valid GitHub username").
			Build()
	}

	// Validate repository name
	if err := validateGitHubRepository(ref.Repo); err != nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid repository name").
			WithCause(err).
			WithContext("repo", ref.Repo).
			WithSuggestion("Use a valid GitHub repository name").
			Build()
	}

	// Validate source
	validSources := map[string]bool{
		"url":       true,
		"shorthand": true,
		"context":   true,
	}
	if !validSources[ref.Source] {
		return errors.ValidationError(fmt.Sprintf("invalid source type: %s", ref.Source))
	}

	return nil
}

// ValidateIssueURL validates a GitHub issue URL
func ValidateIssueURL(urlStr string) error {
	if urlStr == "" {
		return errors.ValidationError("URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid URL format").
			WithCause(err).
			WithContext("url", urlStr).
			WithSuggestion("Provide a valid GitHub issue URL").
			Build()
	}

	// Check if it's a GitHub URL
	if parsedURL.Host != "github.com" {
		return errors.ValidationError("URL must be from github.com")
	}

	// Check URL path structure
	pathPattern := regexp.MustCompile(`^/([^/]+)/([^/]+)/issues/(\d+)$`)
	if !pathPattern.MatchString(parsedURL.Path) {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("URL must be a GitHub issue URL").
			WithContext("url", urlStr).
			WithSuggestion("Use format: https://github.com/owner/repo/issues/123").
			Build()
	}

	return nil
}

// ValidateShorthand validates a shorthand issue reference (owner/repo#123)
func ValidateShorthand(shorthand string) error {
	if shorthand == "" {
		return errors.ValidationError("shorthand cannot be empty")
	}

	pattern := regexp.MustCompile(`^([^/\s]+)/([^/\s#]+)#(\d+)$`)
	matches := pattern.FindStringSubmatch(shorthand)
	if matches == nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid shorthand format").
			WithContext("shorthand", shorthand).
			WithSuggestion("Use format: owner/repo#123").
			Build()
	}

	owner, repo, numberStr := matches[1], matches[2], matches[3]

	// Validate owner
	if err := validateGitHubUsername(owner); err != nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid owner in shorthand").
			WithCause(err).
			WithContext("owner", owner).
			Build()
	}

	// Validate repository
	if err := validateGitHubRepository(repo); err != nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid repository in shorthand").
			WithCause(err).
			WithContext("repo", repo).
			Build()
	}

	// Validate issue number
	number, err := strconv.Atoi(numberStr)
	if err != nil || number <= 0 {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid issue number in shorthand").
			WithContext("number", numberStr).
			WithSuggestion("Issue number must be a positive integer").
			Build()
	}

	return nil
}

// ValidateContextReference validates a context-aware reference (#123)
func ValidateContextReference(contextRef string) error {
	if contextRef == "" {
		return errors.ValidationError("context reference cannot be empty")
	}

	pattern := regexp.MustCompile(`^#(\d+)$`)
	matches := pattern.FindStringSubmatch(contextRef)
	if matches == nil {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid context reference format").
			WithContext("reference", contextRef).
			WithSuggestion("Use format: #123").
			Build()
	}

	// Validate issue number
	number, err := strconv.Atoi(matches[1])
	if err != nil || number <= 0 {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessage("invalid issue number in context reference").
			WithContext("number", matches[1]).
			WithSuggestion("Issue number must be a positive integer").
			Build()
	}

	return nil
}

// validateGitHubUsername validates a GitHub username
func validateGitHubUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) > 39 {
		return fmt.Errorf("username too long (max 39 characters)")
	}

	// GitHub username rules:
	// - Can contain alphanumeric characters and hyphens
	// - Cannot start or end with a hyphen
	// - Cannot have consecutive hyphens
	pattern := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	if !pattern.MatchString(username) {
		return fmt.Errorf("invalid username format")
	}

	// Check for consecutive hyphens
	if strings.Contains(username, "--") {
		return fmt.Errorf("username cannot contain consecutive hyphens")
	}

	return nil
}

// validateGitHubRepository validates a GitHub repository name
func validateGitHubRepository(repoName string) error {
	if repoName == "" {
		return fmt.Errorf("repository name cannot be empty")
	}

	if len(repoName) > 100 {
		return fmt.Errorf("repository name too long (max 100 characters)")
	}

	// GitHub repository name rules:
	// - Can contain alphanumeric characters, hyphens, underscores, and periods
	// - Cannot start with a period
	// - Cannot end with .git
	if strings.HasPrefix(repoName, ".") {
		return fmt.Errorf("repository name cannot start with a period")
	}

	if strings.HasSuffix(repoName, ".git") {
		return fmt.Errorf("repository name cannot end with .git")
	}

	pattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !pattern.MatchString(repoName) {
		return fmt.Errorf("invalid repository name format")
	}

	return nil
}

// SanitizeInput sanitizes user input to prevent injection attacks
func SanitizeInput(input string) string {
	// Remove control characters and excessive whitespace
	input = strings.TrimSpace(input)

	// Remove null bytes and other control characters
	sanitized := strings.Map(func(r rune) rune {
		if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return -1 // Remove the character
		}
		return r
	}, input)

	return sanitized
}

// ValidateInputLength validates that input doesn't exceed reasonable limits
func ValidateInputLength(input string, maxLength int, fieldName string) error {
	if len(input) > maxLength {
		return errors.NewError(errors.ErrorTypeValidation).
			WithMessagef("%s exceeds maximum length of %d characters", fieldName, maxLength).
			WithContext("field", fieldName).
			WithContext("length", len(input)).
			WithContext("max_length", maxLength).
			WithSuggestion(fmt.Sprintf("Limit %s to %d characters or less", fieldName, maxLength)).
			Build()
	}
	return nil
}
