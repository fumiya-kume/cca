package commit

import (
	"context"
	"testing"

	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/stretchr/testify/assert"
)

func TestNewMessageGenerator(t *testing.T) {
	tests := []struct {
		name           string
		config         MessageGeneratorConfig
		expectedMaxLen int
	}{
		{
			name:           "default config",
			config:         MessageGeneratorConfig{},
			expectedMaxLen: 50,
		},
		{
			name: "custom config",
			config: MessageGeneratorConfig{
				MaxLength: 72,
				Style:     MessageStyleConventional,
			},
			expectedMaxLen: 72,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := NewMessageGenerator(tt.config)
			assert.NotNil(t, mg)
			assert.Equal(t, tt.expectedMaxLen, mg.config.MaxLength)
		})
	}
}

func TestMessageGenerator_GenerateMessage(t *testing.T) {
	tests := []struct {
		name           string
		config         MessageGeneratorConfig
		changes        []FileChange
		analysisResult *analysis.AnalysisResult
		expectedError  bool
		expectedType   string
	}{
		{
			name:          "no changes",
			config:        MessageGeneratorConfig{Style: MessageStyleConventional},
			changes:       []FileChange{},
			expectedError: true,
		},
		{
			name:   "conventional commit",
			config: MessageGeneratorConfig{Style: MessageStyleConventional},
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeAdd},
			},
			analysisResult: &analysis.AnalysisResult{
				ProjectInfo: &analysis.ProjectInfo{
					MainLanguage: "Go",
				},
			},
			expectedError: false,
			expectedType:  "feat",
		},
		{
			name:   "traditional commit",
			config: MessageGeneratorConfig{Style: MessageStyleTraditional},
			changes: []FileChange{
				{Path: "README.md", Type: ChangeTypeModify},
			},
			expectedError: false,
		},
		{
			name: "custom template",
			config: MessageGeneratorConfig{
				Style:    MessageStyleCustom,
				Template: "{{.CommitType}}: {{.MainLanguage}} changes",
			},
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
			},
			analysisResult: &analysis.AnalysisResult{
				ProjectInfo: &analysis.ProjectInfo{
					MainLanguage: "Go",
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := NewMessageGenerator(tt.config)
			message, err := mg.GenerateMessage(context.Background(), tt.changes, tt.analysisResult)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, message)
			assert.NotEmpty(t, message.Full)

			if tt.expectedType != "" {
				assert.Equal(t, tt.expectedType, message.Type)
			}
		})
	}
}

func TestMessageGenerator_DetermineCommitType(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{AutoDetectType: true})

	tests := []struct {
		name     string
		context  *MessageContext
		expected CommitType
	}{
		{
			name: "fix changes",
			context: &MessageContext{
				Fixes: []string{"bug in authentication"},
			},
			expected: CommitTypeFix,
		},
		{
			name: "feature changes",
			context: &MessageContext{
				FilesAdded: []string{"new_feature.go"},
			},
			expected: CommitTypeFeat,
		},
		{
			name: "test changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "main_test.go", Type: ChangeTypeModify},
				},
			},
			expected: CommitTypeTest,
		},
		{
			name: "doc changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "README.md", Type: ChangeTypeModify},
				},
			},
			expected: CommitTypeDocs,
		},
		{
			name: "config changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "config.yaml", Type: ChangeTypeModify},
				},
			},
			expected: CommitTypeChore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.determineCommitType(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_AutoDetectCommitType(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{AutoDetectType: true})

	tests := []struct {
		name     string
		context  *MessageContext
		expected CommitType
	}{
		{
			name: "performance changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "performance/optimizer.go", Type: ChangeTypeModify},
				},
			},
			expected: CommitTypePerf,
		},
		{
			name: "build changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "Makefile", Type: ChangeTypeModify},
				},
			},
			expected: CommitTypeBuild,
		},
		{
			name: "style changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "styles.css", Type: ChangeTypeModify},
				},
			},
			expected: CommitTypeStyle,
		},
		{
			name: "refactor changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "refactor/utils.go", Type: ChangeTypeModify},
				},
			},
			expected: CommitTypeRefactor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.autoDetectCommitType(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_DetermineScope(t *testing.T) {
	tests := []struct {
		name     string
		config   MessageGeneratorConfig
		context  *MessageContext
		expected string
	}{
		{
			name:   "auto detect scope",
			config: MessageGeneratorConfig{AutoDetectScope: true},
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "api/handler.go", Type: ChangeTypeModify},
				},
			},
			expected: "api",
		},
		{
			name: "required scope",
			config: MessageGeneratorConfig{
				RequiredScopes: []string{"frontend", "backend"},
			},
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "backend/service.go", Type: ChangeTypeModify},
				},
			},
			expected: "backend",
		},
		{
			name:   "no scope detection",
			config: MessageGeneratorConfig{},
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "main.go", Type: ChangeTypeModify},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := NewMessageGenerator(tt.config)
			result := mg.determineScope(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_AutoDetectScope(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{AutoDetectScope: true})

	tests := []struct {
		name     string
		context  *MessageContext
		expected string
	}{
		{
			name: "api scope",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "api/routes.go", Type: ChangeTypeModify},
				},
			},
			expected: "api",
		},
		{
			name: "ui scope",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "ui/components.go", Type: ChangeTypeModify},
				},
			},
			expected: "ui",
		},
		{
			name: "auth scope",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "auth/middleware.go", Type: ChangeTypeModify},
				},
			},
			expected: "auth",
		},
		{
			name: "db scope",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "database/models.go", Type: ChangeTypeModify},
				},
			},
			expected: "db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.autoDetectScope(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_IsBreakingChange(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name     string
		context  *MessageContext
		expected bool
	}{
		{
			name: "deleted file",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "api.go", Type: ChangeTypeDelete},
				},
			},
			expected: true,
		},
		{
			name: "api changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "api/handler.go", Type: ChangeTypeModify},
				},
			},
			expected: true,
		},
		{
			name: "large changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "main.go", Type: ChangeTypeModify, Additions: 150, Deletions: 50},
				},
			},
			expected: true,
		},
		{
			name: "normal changes",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "utils.go", Type: ChangeTypeModify, Additions: 10, Deletions: 5},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.isBreakingChange(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_GenerateSubject(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{MaxLength: 50})

	tests := []struct {
		name       string
		context    *MessageContext
		commitType CommitType
		scope      string
		expected   string
	}{
		{
			name: "feature subject",
			context: &MessageContext{
				FilesAdded: []string{"user_service.go"},
				Features:   []string{"user authentication"},
			},
			commitType: CommitTypeFeat,
			scope:      "auth",
			expected:   "add user authentication",
		},
		{
			name: "fix subject",
			context: &MessageContext{
				Fixes: []string{"memory leak in parser"},
			},
			commitType: CommitTypeFix,
			scope:      "",
			expected:   "fix memory leak in parser",
		},
		{
			name: "docs subject",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "README.md", Type: ChangeTypeModify},
				},
			},
			commitType: CommitTypeDocs,
			scope:      "",
			expected:   "update README.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.generateSubject(tt.context, tt.commitType, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_GenerateFeatureSubject(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name     string
		context  *MessageContext
		expected string
	}{
		{
			name: "with features",
			context: &MessageContext{
				Features: []string{"user authentication", "password reset"},
			},
			expected: "add user authentication",
		},
		{
			name: "single file added",
			context: &MessageContext{
				FilesAdded: []string{"auth.go"},
			},
			expected: "add auth.go",
		},
		{
			name: "multiple files added",
			context: &MessageContext{
				FilesAdded: []string{"auth.go", "user.go", "token.go"},
			},
			expected: "add 3 new files",
		},
		{
			name: "no specific features",
			context: &MessageContext{
				Features:   []string{},
				FilesAdded: []string{},
			},
			expected: "add new functionality",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.generateFeatureSubject(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_GenerateFixSubject(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name     string
		context  *MessageContext
		expected string
	}{
		{
			name: "with fixes",
			context: &MessageContext{
				Fixes: []string{"memory leak", "null pointer"},
			},
			expected: "fix memory leak",
		},
		{
			name: "single file change",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "parser.go", Type: ChangeTypeModify},
				},
			},
			expected: "fix issue in parser.go",
		},
		{
			name: "generic fix",
			context: &MessageContext{
				Fixes:   []string{},
				Changes: []FileChange{},
			},
			expected: "fix bugs and issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.generateFixSubject(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_GenerateBody(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	context := &MessageContext{
		FilesAdded:     []string{"new_feature.go", "helper.go"},
		FilesModified:  []string{"main.go"},
		FilesDeleted:   []string{"old_code.go"},
		TotalAdditions: 50,
		TotalDeletions: 20,
	}

	body := mg.generateBody(context)
	assert.Contains(t, body, "Added:")
	assert.Contains(t, body, "Modified:")
	assert.Contains(t, body, "Deleted:")
	assert.Contains(t, body, "Changes: +50 -20")
}

func TestMessageGenerator_GenerateFooter(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name     string
		context  *MessageContext
		expected string
	}{
		{
			name: "breaking change",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "api.go", Type: ChangeTypeDelete},
				},
			},
			expected: "BREAKING CHANGE: This commit contains breaking changes",
		},
		{
			name: "no breaking change",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "utils.go", Type: ChangeTypeModify},
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.generateFooter(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_BuildFullMessage(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name       string
		commitType string
		scope      string
		subject    string
		body       string
		footer     string
		breaking   bool
		expected   string
	}{
		{
			name:       "simple conventional commit",
			commitType: "feat",
			scope:      "api",
			subject:    "add user endpoint",
			body:       "",
			footer:     "",
			breaking:   false,
			expected:   "feat(api): add user endpoint",
		},
		{
			name:       "breaking change",
			commitType: "feat",
			scope:      "api",
			subject:    "remove old endpoint",
			body:       "",
			footer:     "BREAKING CHANGE: Removed deprecated endpoint",
			breaking:   true,
			expected:   "feat(api)!: remove old endpoint\n\nBREAKING CHANGE: Removed deprecated endpoint",
		},
		{
			name:       "with body",
			commitType: "fix",
			scope:      "",
			subject:    "resolve memory leak",
			body:       "Fixed issue in parser module",
			footer:     "",
			breaking:   false,
			expected:   "fix: resolve memory leak\n\nFixed issue in parser module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.buildFullMessage(tt.commitType, tt.scope, tt.subject, tt.body, tt.footer, tt.breaking)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_GenerateTraditionalMessage(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{Style: MessageStyleTraditional})

	context := &MessageContext{
		Changes: []FileChange{
			{Path: "main.go", Type: ChangeTypeModify},
			{Path: "utils.go", Type: ChangeTypeAdd},
		},
	}

	message, err := mg.generateTraditionalMessage(context)
	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.NotEmpty(t, message.Subject)
	assert.NotEmpty(t, message.Full)
}

func TestMessageGenerator_GenerateTraditionalSubject(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name     string
		context  *MessageContext
		expected string
	}{
		{
			name: "single file added",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "new_file.go", Type: ChangeTypeAdd},
				},
			},
			expected: "Add new_file.go",
		},
		{
			name: "single file modified",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "main.go", Type: ChangeTypeModify},
				},
			},
			expected: "Update main.go",
		},
		{
			name: "single file deleted",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "old_file.go", Type: ChangeTypeDelete},
				},
			},
			expected: "Remove old_file.go",
		},
		{
			name: "multiple files added",
			context: &MessageContext{
				FilesAdded: []string{"file1.go", "file2.go"},
				Changes: []FileChange{
					{Path: "file1.go", Type: ChangeTypeAdd},
					{Path: "file2.go", Type: ChangeTypeAdd},
				},
			},
			expected: "Add 2 files",
		},
		{
			name: "mixed changes",
			context: &MessageContext{
				FilesAdded:    []string{"new.go"},
				FilesModified: []string{"main.go"},
				Changes: []FileChange{
					{Path: "new.go", Type: ChangeTypeAdd},
					{Path: "main.go", Type: ChangeTypeModify},
				},
			},
			expected: "Update 2 files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.generateTraditionalSubject(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_BuildMessageContext(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	changes := []FileChange{
		{Path: "new_feature.go", Type: ChangeTypeAdd, Additions: 100, Deletions: 0},
		{Path: "main.go", Type: ChangeTypeModify, Additions: 20, Deletions: 10},
		{Path: "old_code.go", Type: ChangeTypeDelete, Additions: 0, Deletions: 50},
	}

	analysisResult := &analysis.AnalysisResult{
		ProjectInfo: &analysis.ProjectInfo{
			MainLanguage: "Go",
			ProjectType:  analysis.ProjectTypeLibrary,
		},
	}

	context := mg.buildMessageContext(changes, analysisResult)

	assert.Equal(t, changes, context.Changes)
	assert.Equal(t, analysisResult, context.AnalysisResult)
	assert.Equal(t, []string{"new_feature.go"}, context.FilesAdded)
	assert.Equal(t, []string{"main.go"}, context.FilesModified)
	assert.Equal(t, []string{"old_code.go"}, context.FilesDeleted)
	assert.Equal(t, 120, context.TotalAdditions)
	assert.Equal(t, 60, context.TotalDeletions)
	assert.Equal(t, "Go", context.MainLanguage)
	assert.Equal(t, "library", context.ProjectType)
}

// Test file type detection helpers

func TestMessageGenerator_IsTestFile(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"main_test.go", true},
		{"test_utils.go", true},
		{"spec_helper.js", true},
		{"api.test.js", true},
		{"component.spec.js", true},
		{"main.go", false},
		{"utils.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := mg.isTestFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_IsDocFile(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"README.md", true},
		{"CHANGELOG.rst", true},
		{"docs/api.md", true},
		{"manual.txt", true},
		{"main.go", false},
		{"config.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := mg.isDocFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_IsConfigFile(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"config.yaml", true},
		{"settings.json", true},
		{"app.toml", true},
		{"setup.ini", true},
		{".env", true},
		{"Dockerfile", true},
		{"main.go", false},
		{"utils.py", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := mg.isConfigFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_IsBuildFile(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"Makefile", true},
		{"package.json", true},
		{"go.mod", true},
		{"Cargo.toml", true},
		{"webpack.config.js", true},
		{"build.gradle", true},
		{"main.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := mg.isBuildFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_IsStyleFile(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"styles.css", true},
		{"main.scss", true},
		{"theme.sass", true},
		{"layout.less", true},
		{"main.go", false},
		{"script.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := mg.isStyleFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_LimitFiles(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name     string
		files    []string
		limit    int
		expected []string
	}{
		{
			name:     "within limit",
			files:    []string{"main.go", "utils.go"},
			limit:    3,
			expected: []string{"main.go", "utils.go"},
		},
		{
			name:     "exceeds limit",
			files:    []string{"path/to/main.go", "path/to/utils.go", "path/to/config.go", "path/to/extra.go"},
			limit:    2,
			expected: []string{"main.go", "utils.go"},
		},
		{
			name:     "empty files",
			files:    []string{},
			limit:    3,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.limitFiles(tt.files, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_Deduplicate(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name     string
		items    []string
		expected []string
	}{
		{
			name:     "with duplicates",
			items:    []string{"feature", "fix", "feature", "docs", "fix"},
			expected: []string{"feature", "fix", "docs"},
		},
		{
			name:     "no duplicates",
			items:    []string{"feature", "fix", "docs"},
			expected: []string{"feature", "fix", "docs"},
		},
		{
			name:     "with empty strings",
			items:    []string{"feature", "", "fix", "", "docs"},
			expected: []string{"feature", "fix", "docs"},
		},
		{
			name:     "empty input",
			items:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mg.deduplicate(tt.items)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessageGenerator_DetectFeatures(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	changes := []FileChange{
		{Path: "user_auth.go", Type: ChangeTypeAdd},
		{Path: "password_reset.go", Type: ChangeTypeAdd},
		{Path: "main.go", Type: ChangeTypeModify},
	}

	features := mg.detectFeatures(changes)
	assert.Contains(t, features, "user auth")
	assert.Contains(t, features, "password reset")
	assert.NotContains(t, features, "main")
}

func TestMessageGenerator_DetectFixes(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	changes := []FileChange{
		{Path: "fix_memory_leak.go", Type: ChangeTypeModify},
		{Path: "bug_parser.go", Type: ChangeTypeModify},
		{Path: "main.go", Type: ChangeTypeModify},
	}

	fixes := mg.detectFixes(changes)
	assert.Contains(t, fixes, "memory leak")
	assert.Contains(t, fixes, "parser")
	assert.NotContains(t, fixes, "main")
}

func TestMessageGenerator_ParseGeneratedMessage(t *testing.T) {
	mg := NewMessageGenerator(MessageGeneratorConfig{})

	tests := []struct {
		name             string
		fullMessage      string
		expectedType     string
		expectedScope    string
		expectedSubject  string
		expectedBreaking bool
	}{
		{
			name:             "conventional commit",
			fullMessage:      "feat(api): add user authentication",
			expectedType:     "feat",
			expectedScope:    "api",
			expectedSubject:  "add user authentication",
			expectedBreaking: false,
		},
		{
			name:             "breaking change",
			fullMessage:      "feat(api)!: remove old endpoint",
			expectedType:     "feat",
			expectedScope:    "api",
			expectedSubject:  "remove old endpoint",
			expectedBreaking: true,
		},
		{
			name:             "no scope",
			fullMessage:      "fix: resolve memory leak",
			expectedType:     "fix",
			expectedScope:    "",
			expectedSubject:  "resolve memory leak",
			expectedBreaking: false,
		},
		{
			name:             "traditional commit",
			fullMessage:      "Update README documentation",
			expectedType:     "",
			expectedScope:    "",
			expectedSubject:  "Update README documentation",
			expectedBreaking: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := &ConventionalMessage{Full: tt.fullMessage}
			mg.parseGeneratedMessage(message)

			assert.Equal(t, tt.expectedType, message.Type)
			assert.Equal(t, tt.expectedScope, message.Scope)
			assert.Equal(t, tt.expectedSubject, message.Subject)
			assert.Equal(t, tt.expectedBreaking, message.Breaking)
		})
	}
}

func TestMessageGenerator_GenerateCustomMessage(t *testing.T) {
	tests := []struct {
		name          string
		template      string
		context       *MessageContext
		expectedError bool
		expectedFull  string
	}{
		{
			name:     "valid template",
			template: "{{.MainLanguage}}: updated {{len .Changes}} files",
			context: &MessageContext{
				MainLanguage: "Go",
				Changes:      []FileChange{{}, {}},
			},
			expectedError: false,
			expectedFull:  "Go: updated 2 files",
		},
		{
			name:          "invalid template",
			template:      "{{.InvalidField",
			context:       &MessageContext{},
			expectedError: true,
		},
		{
			name:     "empty template falls back to conventional",
			template: "",
			context: &MessageContext{
				Changes: []FileChange{
					{Path: "main.go", Type: ChangeTypeModify},
				},
			},
			expectedError: false,
			// Should fallback to conventional message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := NewMessageGenerator(MessageGeneratorConfig{
				Style:    MessageStyleCustom,
				Template: tt.template,
			})

			message, err := mg.generateCustomMessage(tt.context)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, message)

			if tt.expectedFull != "" {
				assert.Equal(t, tt.expectedFull, message.Full)
			}
		})
	}
}
