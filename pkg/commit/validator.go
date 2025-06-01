package commit

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Status constants
const (
	statusUnknown = "unknown"
)

// CommitValidator validates commit plans and messages
type CommitValidator struct {
	config CommitValidatorConfig
}

// CommitValidatorConfig configures the commit validator
type CommitValidatorConfig struct {
	ConventionalCommits   bool
	MaxMessageLength      int
	MaxSubjectLength      int
	MaxBodyLineLength     int
	RequiredScopes        []string
	AllowedTypes          []string
	AllowBreaking         bool
	RequireBody           bool
	RequireFooter         bool
	CustomRules           []ValidationRule
	EnforceCapitalization bool
	AllowEmptyScope       bool
	MaxCommitSize         int
	ValidateFilePatterns  bool
}

// ValidationRule represents a custom validation rule
type ValidationRule struct {
	ID          string
	Name        string
	Description string
	Pattern     string
	Message     string
	Severity    ErrorSeverity
	Scope       ValidationScope
}

// ValidationScope defines where a validation rule applies
type ValidationScope int

const (
	ValidationScopeSubject ValidationScope = iota
	ValidationScopeBody
	ValidationScopeFooter
	ValidationScopeFull
	ValidationScopeFiles
)

// NewCommitValidator creates a new commit validator
func NewCommitValidator(config CommitValidatorConfig) *CommitValidator {
	// Set defaults
	if config.MaxMessageLength == 0 {
		config.MaxMessageLength = 72
	}
	if config.MaxSubjectLength == 0 {
		config.MaxSubjectLength = 50
	}
	if config.MaxBodyLineLength == 0 {
		config.MaxBodyLineLength = 72
	}
	if config.MaxCommitSize == 0 {
		config.MaxCommitSize = 100
	}

	// Set default allowed types if not specified
	if len(config.AllowedTypes) == 0 {
		config.AllowedTypes = []string{
			"feat", "fix", "docs", "style", "refactor",
			"perf", "test", "build", "ci", "chore", "revert",
		}
	}

	return &CommitValidator{
		config: config,
	}
}

// ValidatePlan validates a commit plan
func (cv *CommitValidator) ValidatePlan(ctx context.Context, plan *CommitPlan) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []ValidationError{},
		Warnings:    []ValidationWarning{},
		Suggestions: []string{},
		Score:       1.0,
	}

	// Validate individual commits
	for _, commit := range plan.Commits {
		commitResult := cv.ValidateCommit(ctx, &commit)

		// Merge results
		result.Errors = append(result.Errors, commitResult.Errors...)
		result.Warnings = append(result.Warnings, commitResult.Warnings...)
		result.Suggestions = append(result.Suggestions, commitResult.Suggestions...)

		if !commitResult.Valid {
			result.Valid = false
		}
	}

	// Validate plan-level constraints
	cv.validatePlanConstraints(plan, result)

	// Calculate overall score
	result.Score = cv.calculateValidationScore(result)

	return result, nil
}

// ValidateCommit validates a single commit
func (cv *CommitValidator) ValidateCommit(ctx context.Context, commit *PlannedCommit) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []ValidationError{},
		Warnings:    []ValidationWarning{},
		Suggestions: []string{},
		Score:       1.0,
	}

	// Validate commit message
	cv.validateMessage(&commit.Message, result)

	// Validate commit size
	cv.validateCommitSize(commit, result)

	// Validate file patterns
	if cv.config.ValidateFilePatterns {
		cv.validateFilePatterns(commit, result)
	}

	// Apply custom rules
	cv.applyCustomRules(commit, result)

	// Calculate commit score
	result.Score = cv.calculateValidationScore(result)

	return result
}

// ValidateMessage validates a commit message
func (cv *CommitValidator) ValidateMessage(ctx context.Context, message *ConventionalMessage) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      []ValidationError{},
		Warnings:    []ValidationWarning{},
		Suggestions: []string{},
		Score:       1.0,
	}

	cv.validateMessage(message, result)
	result.Score = cv.calculateValidationScore(result)

	return result
}

// validateMessage validates the commit message structure and content
func (cv *CommitValidator) validateMessage(message *ConventionalMessage, result *ValidationResult) {
	if cv.config.ConventionalCommits {
		cv.validateConventionalMessage(message, result)
	} else {
		cv.validateTraditionalMessage(message, result)
	}

	// Common validations
	cv.validateMessageLength(message, result)
	cv.validateMessageContent(message, result)
}

// validateConventionalMessage validates conventional commit format
func (cv *CommitValidator) validateConventionalMessage(message *ConventionalMessage, result *ValidationResult) {
	// Validate type
	cv.validateCommitType(message, result)

	// Validate scope
	cv.validateCommitScope(message, result)

	// Validate subject
	cv.validateSubject(message, result)

	// Validate body format
	cv.validateBodyFormat(message, result)

	// Validate footer format
	cv.validateFooterFormat(message, result)

	// Validate breaking change format
	cv.validateBreakingChange(message, result)
}

// validateTraditionalMessage validates traditional commit message format
func (cv *CommitValidator) validateTraditionalMessage(message *ConventionalMessage, result *ValidationResult) {
	// Validate subject exists and is reasonable
	if message.Subject == "" {
		cv.addError(result, "empty-subject", "Commit subject cannot be empty", ErrorSeverityError)
		return
	}

	// Validate subject format
	cv.validateSubjectFormat(message.Subject, result)
}

// validateCommitType validates the commit type
func (cv *CommitValidator) validateCommitType(message *ConventionalMessage, result *ValidationResult) {
	if message.Type == "" {
		cv.addError(result, "missing-type", "Commit type is required", ErrorSeverityError)
		return
	}

	// Check if type is allowed
	allowed := false
	for _, allowedType := range cv.config.AllowedTypes {
		if message.Type == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		cv.addError(result, "invalid-type",
			fmt.Sprintf("Invalid commit type '%s'. Allowed types: %s",
				message.Type, strings.Join(cv.config.AllowedTypes, ", ")),
			ErrorSeverityError)
	}
}

// validateCommitScope validates the commit scope
func (cv *CommitValidator) validateCommitScope(message *ConventionalMessage, result *ValidationResult) {
	// Check if scope is required
	if len(cv.config.RequiredScopes) > 0 {
		if message.Scope == "" && !cv.config.AllowEmptyScope {
			cv.addError(result, "missing-scope",
				fmt.Sprintf("Scope is required. Valid scopes: %s",
					strings.Join(cv.config.RequiredScopes, ", ")),
				ErrorSeverityError)
			return
		}

		// Validate scope is in allowed list
		if message.Scope != "" {
			allowed := false
			for _, requiredScope := range cv.config.RequiredScopes {
				if message.Scope == requiredScope {
					allowed = true
					break
				}
			}

			if !allowed {
				cv.addError(result, "invalid-scope",
					fmt.Sprintf("Invalid scope '%s'. Valid scopes: %s",
						message.Scope, strings.Join(cv.config.RequiredScopes, ", ")),
					ErrorSeverityError)
			}
		}
	}

	// Validate scope format
	if message.Scope != "" {
		if !cv.isValidScopeFormat(message.Scope) {
			cv.addError(result, "invalid-scope-format",
				"Scope should contain only lowercase letters, numbers, and hyphens",
				ErrorSeverityWarning)
		}
	}
}

// validateSubject validates the commit subject
func (cv *CommitValidator) validateSubject(message *ConventionalMessage, result *ValidationResult) {
	if message.Subject == "" {
		cv.addError(result, "empty-subject", "Commit subject cannot be empty", ErrorSeverityError)
		return
	}

	cv.validateSubjectFormat(message.Subject, result)
}

// validateSubjectFormat validates the subject format
func (cv *CommitValidator) validateSubjectFormat(subject string, result *ValidationResult) {
	// Check length
	if utf8.RuneCountInString(subject) > cv.config.MaxSubjectLength {
		cv.addError(result, "subject-too-long",
			fmt.Sprintf("Subject line too long (%d > %d)",
				utf8.RuneCountInString(subject), cv.config.MaxSubjectLength),
			ErrorSeverityError)
	}

	// Check capitalization
	if cv.config.EnforceCapitalization {
		if len(subject) > 0 && !cv.isCapitalized(subject) {
			cv.addWarning(result, "subject-capitalization",
				"Subject should start with a capital letter",
				"Capitalize the first letter of the subject")
		}
	}

	// Check for trailing period
	if strings.HasSuffix(subject, ".") {
		cv.addWarning(result, "subject-period",
			"Subject should not end with a period",
			"Remove the trailing period")
	}

	// Check for imperative mood (basic heuristics)
	cv.validateImperativeMood(subject, result)
}

// validateBodyFormat validates the body format
func (cv *CommitValidator) validateBodyFormat(message *ConventionalMessage, result *ValidationResult) {
	if message.Body == "" {
		if cv.config.RequireBody {
			cv.addError(result, "missing-body", "Commit body is required", ErrorSeverityError)
		}
		return
	}

	// Validate body line lengths
	lines := strings.Split(message.Body, "\n")
	for i, line := range lines {
		if utf8.RuneCountInString(line) > cv.config.MaxBodyLineLength {
			cv.addWarning(result, "body-line-too-long",
				fmt.Sprintf("Body line %d is too long (%d > %d)",
					i+1, utf8.RuneCountInString(line), cv.config.MaxBodyLineLength),
				"Wrap long lines in the commit body")
		}
	}

	// Check for blank line after subject
	fullLines := strings.Split(message.Full, "\n")
	if len(fullLines) > 1 && fullLines[1] != "" {
		cv.addError(result, "missing-blank-line",
			"Blank line required between subject and body",
			ErrorSeverityError)
	}
}

// validateFooterFormat validates the footer format
func (cv *CommitValidator) validateFooterFormat(message *ConventionalMessage, result *ValidationResult) {
	if message.Footer == "" {
		if cv.config.RequireFooter {
			cv.addError(result, "missing-footer", "Commit footer is required", ErrorSeverityError)
		}
		return
	}

	// Validate footer format (Git trailers or breaking change format)
	cv.validateFooterTrailers(message.Footer, result)
}

// validateBreakingChange validates breaking change format
func (cv *CommitValidator) validateBreakingChange(message *ConventionalMessage, result *ValidationResult) {
	if message.Breaking {
		if !cv.config.AllowBreaking {
			cv.addError(result, "breaking-not-allowed",
				"Breaking changes are not allowed",
				ErrorSeverityCritical)
			return
		}

		// Check if breaking change is properly documented
		hasBreakingChangeFooter := strings.Contains(message.Footer, "BREAKING CHANGE:")
		hasBreakingChangeMarker := strings.Contains(message.Full, "!")

		if !hasBreakingChangeFooter && !hasBreakingChangeMarker {
			cv.addWarning(result, "breaking-change-documentation",
				"Breaking change should be documented in footer with 'BREAKING CHANGE:'",
				"Add breaking change documentation")
		}
	}
}

// validateMessageLength validates overall message length
func (cv *CommitValidator) validateMessageLength(message *ConventionalMessage, result *ValidationResult) {
	if utf8.RuneCountInString(message.Full) > cv.config.MaxMessageLength {
		cv.addWarning(result, "message-too-long",
			fmt.Sprintf("Commit message is very long (%d > %d)",
				utf8.RuneCountInString(message.Full), cv.config.MaxMessageLength),
			"Consider shortening the commit message")
	}
}

// validateMessageContent validates message content quality
func (cv *CommitValidator) validateMessageContent(message *ConventionalMessage, result *ValidationResult) {
	// Check for common anti-patterns
	subject := strings.ToLower(message.Subject)

	// Generic messages
	genericPatterns := []string{
		"fix", "update", "change", "modify", "wip", "work in progress",
		"tmp", "temp", "temporary", "test", "debug", "refactor",
	}

	for _, pattern := range genericPatterns {
		if subject == pattern {
			cv.addWarning(result, "generic-subject",
				"Subject is too generic",
				"Be more specific about what was changed")
			break
		}
	}

	// Check for typos in common words
	cv.validateCommonTypos(message.Subject, result)
}

// validateCommitSize validates the size of the commit
func (cv *CommitValidator) validateCommitSize(commit *PlannedCommit, result *ValidationResult) {
	if len(commit.Files) > cv.config.MaxCommitSize {
		cv.addError(result, "commit-too-large",
			fmt.Sprintf("Commit contains too many files (%d > %d)",
				len(commit.Files), cv.config.MaxCommitSize),
			ErrorSeverityWarning)
	}

	// Check for very large commits
	if len(commit.Files) > cv.config.MaxCommitSize*2 {
		cv.addError(result, "commit-extremely-large",
			"Commit is extremely large and should be split",
			ErrorSeverityError)
	}

	// Suggest atomic commits for mixed change types
	if cv.hasMixedChangeTypes(commit) {
		cv.addWarning(result, "mixed-change-types",
			"Commit contains mixed change types",
			"Consider splitting into separate commits")
	}
}

// validateFilePatterns validates file patterns in the commit
func (cv *CommitValidator) validateFilePatterns(commit *PlannedCommit, result *ValidationResult) {
	// Check for test files mixed with production code
	hasTests := false
	hasProdCode := false

	for _, file := range commit.Files {
		if cv.isTestFile(file) {
			hasTests = true
		} else {
			hasProdCode = true
		}
	}

	if hasTests && hasProdCode {
		cv.addWarning(result, "mixed-test-prod",
			"Commit mixes test files with production code",
			"Consider separating test changes from production code changes")
	}

	// Check for configuration files mixed with code
	hasConfig := false
	hasCode := false

	for _, file := range commit.Files {
		if cv.isConfigFile(file) {
			hasConfig = true
		} else if !cv.isTestFile(file) && !cv.isDocFile(file) {
			hasCode = true
		}
	}

	if hasConfig && hasCode {
		cv.addWarning(result, "mixed-config-code",
			"Commit mixes configuration files with code",
			"Consider separating configuration changes")
	}
}

// validatePlanConstraints validates plan-level constraints
func (cv *CommitValidator) validatePlanConstraints(plan *CommitPlan, result *ValidationResult) {
	// Check for dependency violations
	cv.validateDependencies(plan, result)

	// Check for optimal commit ordering
	cv.validateCommitOrdering(plan, result)
}

// validateDependencies validates commit dependencies
func (cv *CommitValidator) validateDependencies(plan *CommitPlan, result *ValidationResult) {
	// Build dependency graph and check for cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, commit := range plan.Commits {
		if cv.hasCyclicDependency(commit.ID, plan.Dependencies, visited, recStack) {
			cv.addError(result, "cyclic-dependency",
				fmt.Sprintf("Cyclic dependency detected involving commit %s", commit.ID),
				ErrorSeverityError)
		}
	}
}

// validateCommitOrdering validates the ordering of commits
func (cv *CommitValidator) validateCommitOrdering(plan *CommitPlan, result *ValidationResult) {
	// Check if breaking changes come last
	for i, commit := range plan.Commits {
		if commit.Breaking && i < len(plan.Commits)-1 {
			cv.addWarning(result, "breaking-change-ordering",
				"Breaking changes should typically come last",
				"Consider reordering commits to put breaking changes at the end")
			break
		}
	}

	// Check if tests come after related code changes
	cv.validateTestOrdering(plan, result)
}

// validateTestOrdering validates that tests come after related code
func (cv *CommitValidator) validateTestOrdering(plan *CommitPlan, result *ValidationResult) {
	// This is a simplified check - in practice, this would be more sophisticated
	for i, commit := range plan.Commits {
		if commit.Type == CommitTypeTest && i == 0 {
			cv.addWarning(result, "test-before-code",
				"Test commit appears before code changes",
				"Consider putting test changes after related code changes")
		}
	}
}

// applyCustomRules applies custom validation rules
func (cv *CommitValidator) applyCustomRules(commit *PlannedCommit, result *ValidationResult) {
	for _, rule := range cv.config.CustomRules {
		cv.applyCustomRule(rule, commit, result)
	}
}

// applyCustomRule applies a single custom validation rule
func (cv *CommitValidator) applyCustomRule(rule ValidationRule, commit *PlannedCommit, result *ValidationResult) {
	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return // Skip invalid patterns
	}

	var content string
	switch rule.Scope {
	case ValidationScopeSubject:
		content = commit.Message.Subject
	case ValidationScopeBody:
		content = commit.Message.Body
	case ValidationScopeFooter:
		content = commit.Message.Footer
	case ValidationScopeFull:
		content = commit.Message.Full
	case ValidationScopeFiles:
		content = strings.Join(commit.Files, "\n")
	}

	if pattern.MatchString(content) {
		cv.addError(result, rule.ID, rule.Message, rule.Severity)
	}
}

// Helper methods

func (cv *CommitValidator) addError(result *ValidationResult, errorType, message string, severity ErrorSeverity) {
	result.Valid = false
	result.Errors = append(result.Errors, ValidationError{
		Type:     errorType,
		Message:  message,
		Severity: severity,
	})
}

func (cv *CommitValidator) addWarning(result *ValidationResult, warningType, message, suggestion string) {
	result.Warnings = append(result.Warnings, ValidationWarning{
		Type:       warningType,
		Message:    message,
		Suggestion: suggestion,
	})
}

func (cv *CommitValidator) calculateValidationScore(result *ValidationResult) float64 {
	score := 1.0

	// Deduct points for errors and warnings
	for _, error := range result.Errors {
		switch error.Severity {
		case ErrorSeverityCritical:
			score -= 0.5
		case ErrorSeverityError:
			score -= 0.3
		case ErrorSeverityWarning:
			score -= 0.1
		}
	}

	for range result.Warnings {
		score -= 0.05
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (cv *CommitValidator) isValidScopeFormat(scope string) bool {
	pattern := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	return pattern.MatchString(scope)
}

func (cv *CommitValidator) isCapitalized(text string) bool {
	if len(text) == 0 {
		return false
	}

	firstRune := []rune(text)[0]
	return firstRune >= 'A' && firstRune <= 'Z'
}

func (cv *CommitValidator) validateImperativeMood(subject string, result *ValidationResult) {
	// Basic check for imperative mood
	words := strings.Fields(strings.ToLower(subject))
	if len(words) > 0 {
		firstWord := words[0]

		// Common non-imperative patterns
		nonImperativePatterns := []string{
			"added", "adding", "adds",
			"fixed", "fixing", "fixes",
			"updated", "updating", "updates",
			"changed", "changing", "changes",
			"removed", "removing", "removes",
		}

		for _, pattern := range nonImperativePatterns {
			if firstWord == pattern {
				cv.addWarning(result, "non-imperative",
					"Subject should use imperative mood",
					"Use imperative mood (e.g., 'add' instead of 'added')")
				break
			}
		}
	}
}

func (cv *CommitValidator) validateFooterTrailers(footer string, result *ValidationResult) {
	lines := strings.Split(footer, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for valid trailer format (Key: Value or Key #Value)
		if !strings.Contains(line, ":") && !strings.Contains(line, "#") {
			cv.addWarning(result, "invalid-footer-format",
				"Footer should use Git trailer format (Key: Value)",
				"Format footer as 'Key: Value'")
		}
	}
}

func (cv *CommitValidator) validateCommonTypos(text string, result *ValidationResult) {
	// nolint:misspell // These are intentional misspellings for typo detection
	commonTypos := map[string]string{
		"teh":    "the",
		"adn":    "and",
		"nad":    "and",
		"udpate": "update",
		"upate":  "update",
		"chnage": "change",
		"chagne": "change",
	}

	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		if correction, exists := commonTypos[word]; exists {
			cv.addWarning(result, "possible-typo",
				fmt.Sprintf("Possible typo: '%s' (did you mean '%s'?)", word, correction),
				fmt.Sprintf("Check spelling of '%s'", word))
		}
	}
}

func (cv *CommitValidator) hasMixedChangeTypes(commit *PlannedCommit) bool {
	changeTypes := make(map[ChangeType]bool)
	for _, change := range commit.Changes {
		changeTypes[change.Type] = true
	}
	return len(changeTypes) > 1
}

func (cv *CommitValidator) hasCyclicDependency(commitID string, dependencies map[string][]string, visited, recStack map[string]bool) bool {
	visited[commitID] = true
	recStack[commitID] = true

	for _, dep := range dependencies[commitID] {
		if !visited[dep] {
			if cv.hasCyclicDependency(dep, dependencies, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	recStack[commitID] = false
	return false
}

func (cv *CommitValidator) isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "test") ||
		strings.Contains(lower, "spec") ||
		strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.js")
}

func (cv *CommitValidator) isConfigFile(path string) bool {
	lower := strings.ToLower(path)
	configExts := []string{".json", ".yaml", ".yml", ".toml", ".ini", ".conf"}

	for _, ext := range configExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	return strings.Contains(lower, "config") ||
		strings.Contains(lower, ".env") ||
		strings.Contains(lower, "dockerfile")
}

func (cv *CommitValidator) isDocFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") ||
		strings.HasSuffix(lower, ".rst") ||
		strings.HasSuffix(lower, ".txt") ||
		strings.Contains(lower, "readme") ||
		strings.Contains(lower, "doc")
}

// String methods for enums

func (vs ValidationScope) String() string {
	switch vs {
	case ValidationScopeSubject:
		return "subject"
	case ValidationScopeBody:
		return "body"
	case ValidationScopeFooter:
		return "footer"
	case ValidationScopeFull:
		return "full"
	case ValidationScopeFiles:
		return "files"
	default:
		return statusUnknown
	}
}
