package commit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCommitValidator(t *testing.T) {
	tests := []struct {
		name           string
		config         CommitValidatorConfig
		expectedMaxMsg int
		expectedMaxSub int
		expectedTypes  []string
	}{
		{
			name:           "default config",
			config:         CommitValidatorConfig{},
			expectedMaxMsg: 72,
			expectedMaxSub: 50,
			expectedTypes: []string{
				"feat", "fix", "docs", "style", "refactor",
				"perf", "test", "build", "ci", "chore", "revert",
			},
		},
		{
			name: "custom config",
			config: CommitValidatorConfig{
				MaxMessageLength:    100,
				MaxSubjectLength:    60,
				AllowedTypes:        []string{"feat", "fix"},
				ConventionalCommits: true,
			},
			expectedMaxMsg: 100,
			expectedMaxSub: 60,
			expectedTypes:  []string{"feat", "fix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCommitValidator(tt.config)
			assert.NotNil(t, cv)
			assert.Equal(t, tt.expectedMaxMsg, cv.config.MaxMessageLength)
			assert.Equal(t, tt.expectedMaxSub, cv.config.MaxSubjectLength)
			assert.Equal(t, tt.expectedTypes, cv.config.AllowedTypes)
		})
	}
}

func TestCommitValidator_ValidateMessage(t *testing.T) {
	tests := []struct {
		name           string
		config         CommitValidatorConfig
		message        *ConventionalMessage
		expectedValid  bool
		expectedErrors int
	}{
		{
			name: "valid conventional commit",
			config: CommitValidatorConfig{
				ConventionalCommits: true,
				MaxSubjectLength:    50,
			},
			message: &ConventionalMessage{
				Type:    "feat",
				Scope:   "api",
				Subject: "add user authentication",
				Full:    "feat(api): add user authentication",
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name: "invalid type",
			config: CommitValidatorConfig{
				ConventionalCommits: true,
				AllowedTypes:        []string{"feat", "fix"},
			},
			message: &ConventionalMessage{
				Type:    "invalid",
				Subject: "some change",
				Full:    "invalid: some change",
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name: "missing type",
			config: CommitValidatorConfig{
				ConventionalCommits: true,
			},
			message: &ConventionalMessage{
				Type:    "",
				Subject: "some change",
				Full:    "some change",
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name: "subject too long",
			config: CommitValidatorConfig{
				ConventionalCommits: true,
				MaxSubjectLength:    20,
			},
			message: &ConventionalMessage{
				Type:    "feat",
				Subject: "this is a very long subject that exceeds the limit",
				Full:    "feat: this is a very long subject that exceeds the limit",
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
		{
			name: "valid traditional commit",
			config: CommitValidatorConfig{
				ConventionalCommits: false,
			},
			message: &ConventionalMessage{
				Subject: "Update documentation",
				Full:    "Update documentation",
			},
			expectedValid:  true,
			expectedErrors: 0,
		},
		{
			name: "empty subject",
			config: CommitValidatorConfig{
				ConventionalCommits: false,
			},
			message: &ConventionalMessage{
				Subject: "",
				Full:    "",
			},
			expectedValid:  false,
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCommitValidator(tt.config)
			result := cv.ValidateMessage(context.Background(), tt.message)

			assert.Equal(t, tt.expectedValid, result.Valid)
			assert.Len(t, result.Errors, tt.expectedErrors)
			assert.NotNil(t, result.Score)
		})
	}
}

func TestCommitValidator_ValidateCommitType(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{
		AllowedTypes: []string{"feat", "fix", "docs"},
	})

	tests := []struct {
		name          string
		message       *ConventionalMessage
		expectedValid bool
	}{
		{
			name:          "valid type",
			message:       &ConventionalMessage{Type: "feat"},
			expectedValid: true,
		},
		{
			name:          "invalid type",
			message:       &ConventionalMessage{Type: "invalid"},
			expectedValid: false,
		},
		{
			name:          "missing type",
			message:       &ConventionalMessage{Type: ""},
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateCommitType(tt.message, result)
			assert.Equal(t, tt.expectedValid, result.Valid)
		})
	}
}

func TestCommitValidator_ValidateCommitScope(t *testing.T) {
	tests := []struct {
		name          string
		config        CommitValidatorConfig
		message       *ConventionalMessage
		expectedValid bool
	}{
		{
			name: "valid required scope",
			config: CommitValidatorConfig{
				RequiredScopes: []string{"api", "ui", "db"},
			},
			message:       &ConventionalMessage{Scope: "api"},
			expectedValid: true,
		},
		{
			name: "invalid required scope",
			config: CommitValidatorConfig{
				RequiredScopes: []string{"api", "ui", "db"},
			},
			message:       &ConventionalMessage{Scope: "invalid"},
			expectedValid: false,
		},
		{
			name: "missing required scope",
			config: CommitValidatorConfig{
				RequiredScopes:  []string{"api", "ui", "db"},
				AllowEmptyScope: false,
			},
			message:       &ConventionalMessage{Scope: ""},
			expectedValid: false,
		},
		{
			name: "empty scope allowed",
			config: CommitValidatorConfig{
				RequiredScopes:  []string{"api", "ui", "db"},
				AllowEmptyScope: true,
			},
			message:       &ConventionalMessage{Scope: ""},
			expectedValid: true,
		},
		{
			name: "no scope requirements",
			config: CommitValidatorConfig{
				RequiredScopes: []string{},
			},
			message:       &ConventionalMessage{Scope: "anything"},
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCommitValidator(tt.config)
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateCommitScope(tt.message, result)
			assert.Equal(t, tt.expectedValid, result.Valid)
		})
	}
}

func TestCommitValidator_ValidateSubjectFormat(t *testing.T) {
	tests := []struct {
		name             string
		config           CommitValidatorConfig
		subject          string
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name: "valid subject",
			config: CommitValidatorConfig{
				MaxSubjectLength: 50,
			},
			subject:          "add user authentication",
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name: "subject too long",
			config: CommitValidatorConfig{
				MaxSubjectLength: 20,
			},
			subject:          "this subject is way too long for the limit",
			expectedErrors:   1,
			expectedWarnings: 0,
		},
		{
			name: "subject with period",
			config: CommitValidatorConfig{
				MaxSubjectLength: 50,
			},
			subject:          "add user authentication.",
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name: "capitalization enforced",
			config: CommitValidatorConfig{
				MaxSubjectLength:      50,
				EnforceCapitalization: true,
			},
			subject:          "add user authentication",
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name: "non-imperative mood",
			config: CommitValidatorConfig{
				MaxSubjectLength: 50,
			},
			subject:          "added user authentication",
			expectedErrors:   0,
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCommitValidator(tt.config)
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateSubjectFormat(tt.subject, result)
			assert.Len(t, result.Errors, tt.expectedErrors)
			assert.Len(t, result.Warnings, tt.expectedWarnings)
		})
	}
}

func TestCommitValidator_ValidateBodyFormat(t *testing.T) {
	tests := []struct {
		name             string
		config           CommitValidatorConfig
		message          *ConventionalMessage
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name: "valid body",
			config: CommitValidatorConfig{
				MaxBodyLineLength: 72,
			},
			message: &ConventionalMessage{
				Body: "This is a valid commit body\nwith multiple lines.",
				Full: "feat: add feature\n\nThis is a valid commit body\nwith multiple lines.",
			},
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name: "body line too long",
			config: CommitValidatorConfig{
				MaxBodyLineLength: 30,
			},
			message: &ConventionalMessage{
				Body: "This is a very long line that exceeds the maximum allowed length",
				Full: "feat: add feature\n\nThis is a very long line that exceeds the maximum allowed length",
			},
			expectedErrors:   0,
			expectedWarnings: 1,
		},
		{
			name: "missing blank line",
			config: CommitValidatorConfig{
				MaxBodyLineLength: 72,
			},
			message: &ConventionalMessage{
				Body: "This body has no blank line before it",
				Full: "feat: add feature\nThis body has no blank line before it",
			},
			expectedErrors:   1,
			expectedWarnings: 0,
		},
		{
			name: "required body missing",
			config: CommitValidatorConfig{
				RequireBody: true,
			},
			message: &ConventionalMessage{
				Body: "",
				Full: "feat: add feature",
			},
			expectedErrors:   1,
			expectedWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCommitValidator(tt.config)
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateBodyFormat(tt.message, result)
			assert.Len(t, result.Errors, tt.expectedErrors)
			assert.Len(t, result.Warnings, tt.expectedWarnings)
		})
	}
}

func TestCommitValidator_ValidateBreakingChange(t *testing.T) {
	tests := []struct {
		name             string
		config           CommitValidatorConfig
		message          *ConventionalMessage
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name: "breaking change allowed",
			config: CommitValidatorConfig{
				AllowBreaking: true,
			},
			message: &ConventionalMessage{
				Breaking: true,
				Footer:   "BREAKING CHANGE: Removed old API",
				Full:     "feat!: add new API\n\nBREAKING CHANGE: Removed old API",
			},
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name: "breaking change not allowed",
			config: CommitValidatorConfig{
				AllowBreaking: false,
			},
			message: &ConventionalMessage{
				Breaking: true,
			},
			expectedErrors:   1,
			expectedWarnings: 0,
		},
		{
			name: "breaking change missing documentation",
			config: CommitValidatorConfig{
				AllowBreaking: true,
			},
			message: &ConventionalMessage{
				Breaking: true,
				Footer:   "",
				Full:     "feat: add new feature",
			},
			expectedErrors:   0,
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := NewCommitValidator(tt.config)
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateBreakingChange(tt.message, result)
			assert.Len(t, result.Errors, tt.expectedErrors)
			assert.Len(t, result.Warnings, tt.expectedWarnings)
		})
	}
}

func TestCommitValidator_ValidateMessageContent(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		name             string
		message          *ConventionalMessage
		expectedWarnings int
	}{
		{
			name: "specific subject",
			message: &ConventionalMessage{
				Subject: "add user authentication endpoint",
			},
			expectedWarnings: 0,
		},
		{
			name: "generic subject",
			message: &ConventionalMessage{
				Subject: "fix",
			},
			expectedWarnings: 1,
		},
		{
			name: "wip subject",
			message: &ConventionalMessage{
				Subject: "wip",
			},
			expectedWarnings: 1,
		},
		{
			name: "typo in subject",
			message: &ConventionalMessage{
				Subject: "udpate user model",
			},
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateMessageContent(tt.message, result)
			assert.Len(t, result.Warnings, tt.expectedWarnings)
		})
	}
}

func TestCommitValidator_ValidateCommitSize(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{
		MaxCommitSize: 5,
	})

	tests := []struct {
		name             string
		commit           *PlannedCommit
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name: "normal size commit",
			commit: &PlannedCommit{
				Files: []string{"main.go", "utils.go"},
				Changes: []FileChange{
					{Type: ChangeTypeModify},
					{Type: ChangeTypeAdd},
				},
			},
			expectedErrors:   0,
			expectedWarnings: 1, // mixed-change-types warning
		},
		{
			name: "large commit",
			commit: &PlannedCommit{
				Files: []string{"1.go", "2.go", "3.go", "4.go", "5.go", "6.go"},
				Changes: []FileChange{
					{Type: ChangeTypeModify},
					{Type: ChangeTypeAdd},
				},
			},
			expectedErrors:   1, // commit-too-large (added as error with warning severity)
			expectedWarnings: 1, // mixed-change-types warning
		},
		{
			name: "extremely large commit",
			commit: &PlannedCommit{
				Files: make([]string, 15), // 15 files, > 2 * MaxCommitSize
				Changes: []FileChange{
					{Type: ChangeTypeModify},
					{Type: ChangeTypeAdd},
				},
			},
			expectedErrors:   2, // commit-too-large + commit-extremely-large errors
			expectedWarnings: 1, // mixed-change-types warning
		},
		{
			name: "mixed change types",
			commit: &PlannedCommit{
				Files: []string{"main.go", "utils.go"},
				Changes: []FileChange{
					{Type: ChangeTypeModify},
					{Type: ChangeTypeAdd},
					{Type: ChangeTypeDelete},
				},
			},
			expectedErrors:   0,
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateCommitSize(tt.commit, result)
			assert.Len(t, result.Errors, tt.expectedErrors)
			assert.Len(t, result.Warnings, tt.expectedWarnings)
		})
	}
}

func TestCommitValidator_ValidateFilePatterns(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{
		ValidateFilePatterns: true,
	})

	tests := []struct {
		name             string
		commit           *PlannedCommit
		expectedWarnings int
	}{
		{
			name: "clean separation",
			commit: &PlannedCommit{
				Files: []string{"main.go", "utils.go"},
			},
			expectedWarnings: 0,
		},
		{
			name: "mixed test and prod",
			commit: &PlannedCommit{
				Files: []string{"main.go", "main_test.go"},
			},
			expectedWarnings: 1,
		},
		{
			name: "mixed config and code",
			commit: &PlannedCommit{
				Files: []string{"main.go", "config.yaml"},
			},
			expectedWarnings: 1,
		},
		{
			name: "only tests",
			commit: &PlannedCommit{
				Files: []string{"main_test.go", "utils_test.go"},
			},
			expectedWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.validateFilePatterns(tt.commit, result)
			assert.Len(t, result.Warnings, tt.expectedWarnings)
		})
	}
}

func TestCommitValidator_ValidatePlan(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{
		ConventionalCommits: true,
		MaxCommitSize:       10,
	})

	plan := &CommitPlan{
		Commits: []PlannedCommit{
			{
				ID:   "commit_1",
				Type: CommitTypeFeat,
				Message: ConventionalMessage{
					Type:    "feat",
					Subject: "add user authentication",
					Full:    "feat: add user authentication",
				},
				Files: []string{"auth.go"},
				Changes: []FileChange{
					{Path: "auth.go", Type: ChangeTypeAdd},
				},
			},
			{
				ID:   "commit_2",
				Type: CommitTypeTest,
				Message: ConventionalMessage{
					Type:    "test",
					Subject: "add auth tests",
					Full:    "test: add auth tests",
				},
				Files: []string{"auth_test.go"},
				Changes: []FileChange{
					{Path: "auth_test.go", Type: ChangeTypeAdd},
				},
			},
		},
		Dependencies: map[string][]string{
			"commit_2": {"commit_1"}, // Tests depend on code
		},
	}

	result, err := cv.ValidatePlan(context.Background(), plan)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.NotZero(t, result.Score)
}

func TestCommitValidator_ValidateCommit(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{
		ConventionalCommits: true,
		MaxCommitSize:       5,
	})

	commit := &PlannedCommit{
		ID:   "commit_1",
		Type: CommitTypeFeat,
		Message: ConventionalMessage{
			Type:    "feat",
			Subject: "add user authentication",
			Full:    "feat: add user authentication",
		},
		Files: []string{"auth.go", "middleware.go"},
		Changes: []FileChange{
			{Path: "auth.go", Type: ChangeTypeAdd},
			{Path: "middleware.go", Type: ChangeTypeAdd},
		},
	}

	result := cv.ValidateCommit(context.Background(), commit)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.NotZero(t, result.Score)
}

func TestCommitValidator_ApplyCustomRules(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{
		CustomRules: []ValidationRule{
			{
				ID:       "no-wip",
				Pattern:  "(?i)\\bwip\\b",
				Message:  "WIP commits not allowed",
				Severity: ErrorSeverityError,
				Scope:    ValidationScopeSubject,
			},
			{
				ID:       "no-temp-files",
				Pattern:  "temp|tmp",
				Message:  "Temporary files should not be committed",
				Severity: ErrorSeverityWarning,
				Scope:    ValidationScopeFiles,
			},
		},
	})

	tests := []struct {
		name           string
		commit         *PlannedCommit
		expectedErrors int
	}{
		{
			name: "valid commit",
			commit: &PlannedCommit{
				Message: ConventionalMessage{Subject: "add user feature"},
				Files:   []string{"user.go", "auth.go"},
			},
			expectedErrors: 0,
		},
		{
			name: "wip commit",
			commit: &PlannedCommit{
				Message: ConventionalMessage{Subject: "WIP: working on feature"},
				Files:   []string{"feature.go"},
			},
			expectedErrors: 1,
		},
		{
			name: "temp files",
			commit: &PlannedCommit{
				Message: ConventionalMessage{Subject: "add feature"},
				Files:   []string{"feature.go", "temp_debug.go"},
			},
			expectedErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ValidationResult{
				Valid:    true,
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			}

			cv.applyCustomRules(tt.commit, result)
			assert.Len(t, result.Errors, tt.expectedErrors)
		})
	}
}

func TestCommitValidator_HasCyclicDependency(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		name         string
		dependencies map[string][]string
		startNode    string
		expected     bool
	}{
		{
			name: "no cycle",
			dependencies: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {},
			},
			startNode: "A",
			expected:  false,
		},
		{
			name: "simple cycle",
			dependencies: map[string][]string{
				"A": {"B"},
				"B": {"A"},
			},
			startNode: "A",
			expected:  true,
		},
		{
			name: "complex cycle",
			dependencies: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {"A"},
			},
			startNode: "A",
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visited := make(map[string]bool)
			recStack := make(map[string]bool)
			result := cv.hasCyclicDependency(tt.startNode, tt.dependencies, visited, recStack)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test helper methods

func TestCommitValidator_IsValidScopeFormat(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		scope    string
		expected bool
	}{
		{"api", true},
		{"user-auth", true},
		{"db2", true},
		{"API", false},       // Uppercase not allowed
		{"user_auth", false}, // Underscore not allowed
		{"123", false},       // Cannot start with number
		{"", false},          // Empty scope
		{"api!", false},      // Special characters not allowed
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			result := cv.isValidScopeFormat(tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitValidator_IsCapitalized(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		text     string
		expected bool
	}{
		{"Add feature", true},
		{"add feature", false},
		{"API update", true},
		{"123 test", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := cv.isCapitalized(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitValidator_IsTestFile(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"main_test.go", true},
		{"utils_test.go", true},
		{"api.test.js", true},
		{"component.spec.js", true},
		{"test_helper.py", true},
		{"spec_utils.rb", true},
		{"main.go", false},
		{"utils.js", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := cv.isTestFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitValidator_IsConfigFile(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"config.yaml", true},
		{"settings.json", true},
		{"app.toml", true},
		{"setup.ini", true},
		{"myapp.conf", true},
		{".env", true},
		{"Dockerfile", true},
		{"main.go", false},
		{"utils.py", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := cv.isConfigFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitValidator_IsDocFile(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"README.md", true},
		{"CHANGELOG.rst", true},
		{"manual.txt", true},
		{"docs/api.md", true},
		{"documentation/guide.md", true},
		{"main.go", false},
		{"config.json", false},
		{"test.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := cv.isDocFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitValidator_HasMixedChangeTypes(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		name     string
		commit   *PlannedCommit
		expected bool
	}{
		{
			name: "single change type",
			commit: &PlannedCommit{
				Changes: []FileChange{
					{Type: ChangeTypeModify},
					{Type: ChangeTypeModify},
				},
			},
			expected: false,
		},
		{
			name: "mixed change types",
			commit: &PlannedCommit{
				Changes: []FileChange{
					{Type: ChangeTypeModify},
					{Type: ChangeTypeAdd},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cv.hasMixedChangeTypes(tt.commit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitValidator_CalculateValidationScore(t *testing.T) {
	cv := NewCommitValidator(CommitValidatorConfig{})

	tests := []struct {
		name     string
		result   *ValidationResult
		expected float64
	}{
		{
			name: "perfect score",
			result: &ValidationResult{
				Errors:   []ValidationError{},
				Warnings: []ValidationWarning{},
			},
			expected: 1.0,
		},
		{
			name: "with warnings",
			result: &ValidationResult{
				Errors: []ValidationError{},
				Warnings: []ValidationWarning{
					{Type: "test-warning"},
					{Type: "another-warning"},
				},
			},
			expected: 0.9, // 1.0 - (2 * 0.05)
		},
		{
			name: "with errors",
			result: &ValidationResult{
				Errors: []ValidationError{
					{Severity: ErrorSeverityError},
					{Severity: ErrorSeverityWarning},
				},
				Warnings: []ValidationWarning{},
			},
			expected: 0.6, // 1.0 - 0.3 - 0.1
		},
		{
			name: "critical error",
			result: &ValidationResult{
				Errors: []ValidationError{
					{Severity: ErrorSeverityCritical},
				},
				Warnings: []ValidationWarning{},
			},
			expected: 0.5, // 1.0 - 0.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cv.calculateValidationScore(tt.result)
			assert.InDelta(t, tt.expected, result, 0.0001) // Allow small floating point precision differences
		})
	}
}

// Test ValidationScope string method

func TestValidationScope_String(t *testing.T) {
	tests := []struct {
		scope    ValidationScope
		expected string
	}{
		{ValidationScopeSubject, "subject"},
		{ValidationScopeBody, "body"},
		{ValidationScopeFooter, "footer"},
		{ValidationScopeFull, "full"},
		{ValidationScopeFiles, "files"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.scope.String())
		})
	}
}
