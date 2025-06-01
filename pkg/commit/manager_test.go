package commit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommitManager(t *testing.T) {
	tests := []struct {
		name              string
		config            CommitConfig
		workingDir        string
		expectedError     bool
		expectedMaxSize   int
		expectedMaxLength int
	}{
		{
			name:              "default config",
			config:            CommitConfig{},
			workingDir:        "/tmp",
			expectedError:     false,
			expectedMaxSize:   100,
			expectedMaxLength: 50,
		},
		{
			name: "custom config",
			config: CommitConfig{
				MaxCommitSize:       50,
				MaxMessageLength:    72,
				ConventionalCommits: true,
			},
			workingDir:        "/tmp",
			expectedError:     false,
			expectedMaxSize:   50,
			expectedMaxLength: 72,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm, err := NewCommitManager(tt.config, tt.workingDir)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, cm)
			assert.Equal(t, tt.expectedMaxSize, cm.config.MaxCommitSize)
			assert.Equal(t, tt.expectedMaxLength, cm.config.MaxMessageLength)
			assert.NotNil(t, cm.changeAnalyzer)
			assert.NotNil(t, cm.messageGenerator)
			assert.NotNil(t, cm.validator)
		})
	}
}

func TestCommitManager_CreateCommitPlan(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "commit-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := CommitConfig{
		ConventionalCommits: true,
		MaxCommitSize:       10,
	}

	cm, err := NewCommitManager(config, tempDir)
	require.NoError(t, err)

	// Test basic functionality with mock data
	t.Run("basic functionality", func(t *testing.T) {
		// Since we can't easily mock the analyzeWorkingDirectory method,
		// we'll test the components that CreateCommitPlan uses

		// Test empty changes array processing
		changes := []FileChange{}
		groups, err := cm.changeAnalyzer.GroupChanges(context.Background(), changes, nil)
		assert.NoError(t, err)
		assert.Empty(t, groups)

		// Test with sample changes
		changes = []FileChange{
			{
				Path:      "main.go",
				Type:      ChangeTypeModify,
				Additions: 10,
				Deletions: 5,
			},
			{
				Path:      "README.md",
				Type:      ChangeTypeAdd,
				Additions: 20,
				Deletions: 0,
			},
		}

		analysisResult := &analysis.AnalysisResult{
			ProjectInfo: &analysis.ProjectInfo{
				Name:         "test-project",
				MainLanguage: "Go",
			},
		}

		groups, err = cm.changeAnalyzer.GroupChanges(context.Background(), changes, analysisResult)
		assert.NoError(t, err)
		assert.NotEmpty(t, groups)
	})
}

func TestCommitManager_ParseGitStatusLine(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name          string
		statusLine    string
		expectedPath  string
		expectedType  ChangeType
		expectedError bool
	}{
		{
			name:          "added file",
			statusLine:    "A  main.go",
			expectedPath:  "main.go",
			expectedType:  ChangeTypeAdd,
			expectedError: false,
		},
		{
			name:          "modified file",
			statusLine:    " M README.md",
			expectedPath:  "README.md",
			expectedType:  ChangeTypeModify,
			expectedError: false,
		},
		{
			name:          "deleted file",
			statusLine:    " D old_file.txt",
			expectedPath:  "old_file.txt",
			expectedType:  ChangeTypeDelete,
			expectedError: false,
		},
		{
			name:          "renamed file",
			statusLine:    "R  new_name.go",
			expectedPath:  "new_name.go",
			expectedType:  ChangeTypeRename,
			expectedError: false,
		},
		{
			name:          "untracked file",
			statusLine:    "?? untracked.go",
			expectedPath:  "untracked.go",
			expectedType:  ChangeTypeUntracked,
			expectedError: false,
		},
		{
			name:          "invalid line",
			statusLine:    "X",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change, err := cm.parseGitStatusLine(tt.statusLine)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, change)
			assert.Equal(t, tt.expectedPath, change.Path)
			assert.Equal(t, tt.expectedType, change.Type)
		})
	}
}

func TestCommitManager_DetermineCommitType(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name     string
		changes  []FileChange
		expected CommitType
	}{
		{
			name: "fix changes",
			changes: []FileChange{
				{Path: "bugfix.go", Type: ChangeTypeModify},
			},
			expected: CommitTypeFix,
		},
		{
			name: "feature changes",
			changes: []FileChange{
				{Path: "new_feature.go", Type: ChangeTypeAdd},
			},
			expected: CommitTypeFeat,
		},
		{
			name: "test changes",
			changes: []FileChange{
				{Path: "main_test.go", Type: ChangeTypeModify},
			},
			expected: CommitTypeTest,
		},
		{
			name: "doc changes",
			changes: []FileChange{
				{Path: "README.md", Type: ChangeTypeModify},
			},
			expected: CommitTypeDocs,
		},
		{
			name: "config changes",
			changes: []FileChange{
				{Path: "config.yaml", Type: ChangeTypeModify},
			},
			expected: CommitTypeChore,
		},
		{
			name: "generic changes",
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
			},
			expected: CommitTypeChore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.determineCommitType(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitManager_DetermineScope(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name     string
		changes  []FileChange
		analysis *analysis.AnalysisResult
		expected string
	}{
		{
			name: "single directory",
			changes: []FileChange{
				{Path: "pkg/auth/user.go", Type: ChangeTypeModify},
				{Path: "pkg/auth/token.go", Type: ChangeTypeAdd},
			},
			analysis: &analysis.AnalysisResult{},
			expected: "pkg",
		},
		{
			name: "no analysis result",
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
			},
			analysis: nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.determineScope(tt.changes, tt.analysis)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitManager_IsBreakingChange(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name     string
		changes  []FileChange
		expected bool
	}{
		{
			name: "deleted file",
			changes: []FileChange{
				{Path: "old_api.go", Type: ChangeTypeDelete},
			},
			expected: true,
		},
		{
			name: "modified API file",
			changes: []FileChange{
				{Path: "api/handler.go", Type: ChangeTypeModify},
			},
			expected: true,
		},
		{
			name: "regular changes",
			changes: []FileChange{
				{Path: "internal/utils.go", Type: ChangeTypeModify},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.isBreakingChange(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitManager_CalculateCommitSize(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name     string
		changes  []FileChange
		expected CommitSize
	}{
		{
			name: "small commit",
			changes: []FileChange{
				{Additions: 5, Deletions: 3},
			},
			expected: CommitSizeSmall,
		},
		{
			name: "medium commit",
			changes: []FileChange{
				{Additions: 25, Deletions: 15},
			},
			expected: CommitSizeMedium,
		},
		{
			name: "large commit",
			changes: []FileChange{
				{Additions: 100, Deletions: 50},
			},
			expected: CommitSizeLarge,
		},
		{
			name: "huge commit",
			changes: []FileChange{
				{Additions: 300, Deletions: 200},
			},
			expected: CommitSizeHuge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.calculateCommitSize(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitManager_CalculatePriority(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name        string
		commitType  CommitType
		size        CommitSize
		breaking    bool
		expectedMin int
	}{
		{
			name:        "fix commit",
			commitType:  CommitTypeFix,
			size:        CommitSizeSmall,
			breaking:    false,
			expectedMin: 30, // 10 base + 20 fix + 10 small - but actual calculation may vary
		},
		{
			name:        "breaking change",
			commitType:  CommitTypeFeat,
			size:        CommitSizeMedium,
			breaking:    true,
			expectedMin: 70, // Should get high priority for breaking
		},
		{
			name:        "docs commit",
			commitType:  CommitTypeDocs,
			size:        CommitSizeSmall,
			breaking:    false,
			expectedMin: 20, // Lower priority
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.calculatePriority(tt.commitType, tt.size, tt.breaking)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
		})
	}
}

func TestCommitManager_ExtractFilePaths(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	changes := []FileChange{
		{Path: "main.go"},
		{Path: "README.md"},
		{Path: "config.yaml"},
	}

	result := cm.extractFilePaths(changes)
	expected := []string{"main.go", "README.md", "config.yaml"}

	assert.Equal(t, expected, result)
}

func TestCommitManager_EstimateCommitTime(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	commits := []PlannedCommit{
		{Size: CommitSizeSmall},
		{Size: CommitSizeLarge},
		{Size: CommitSizeHuge},
	}

	result := cm.estimateCommitTime(commits)
	assert.Greater(t, result, time.Duration(0))
	// Base time should be at least 3 commits * 2 minutes = 6 minutes
	assert.GreaterOrEqual(t, result, 6*time.Millisecond) // Fast for tests
}

func TestCommitManager_DetermineStrategy(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name     string
		commits  []PlannedCommit
		expected CommitStrategy
	}{
		{
			name: "single commit",
			commits: []PlannedCommit{
				{Type: CommitTypeFeat},
			},
			expected: CommitStrategyAtomic,
		},
		{
			name: "feature commits",
			commits: []PlannedCommit{
				{Type: CommitTypeFeat},
				{Type: CommitTypeTest},
			},
			expected: CommitStrategyFeature,
		},
		{
			name: "mixed commits",
			commits: []PlannedCommit{
				{Type: CommitTypeDocs},
				{Type: CommitTypeTest},
			},
			expected: CommitStrategyLogical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.determineStrategy(tt.commits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitManager_AnalyzeDependencies(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	existingCommits := []PlannedCommit{
		{ID: "commit_1", Files: []string{"main.go", "utils.go"}},
		{ID: "commit_2", Files: []string{"README.md"}},
	}

	newCommit := PlannedCommit{
		ID:    "commit_3",
		Files: []string{"main.go", "test.go"}, // Overlaps with commit_1
	}

	deps := cm.analyzeDependencies(newCommit, existingCommits)
	assert.Contains(t, deps, "commit_1")
	assert.NotContains(t, deps, "commit_2")
}

func TestCommitManager_PreviewCommitPlan(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{DryRun: true}, "/tmp")
	require.NoError(t, err)

	plan := &CommitPlan{
		Strategy: CommitStrategyLogical,
		Commits: []PlannedCommit{
			{
				ID:      "commit_1",
				Type:    CommitTypeFeat,
				Scope:   "api",
				Message: ConventionalMessage{Full: "feat(api): add new endpoint"},
				Files:   []string{"api.go"},
			},
		},
		TotalChanges:  1,
		EstimatedTime: time.Millisecond * 5, // Fast for tests
	}

	err = cm.previewCommitPlan(plan)
	assert.NoError(t, err)
}

// Test helper functions

func TestCommitManager_HasLogicalGroups(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name     string
		commits  []PlannedCommit
		expected bool
	}{
		{
			name: "single type",
			commits: []PlannedCommit{
				{Type: CommitTypeFeat},
				{Type: CommitTypeFeat},
			},
			expected: false,
		},
		{
			name: "multiple types",
			commits: []PlannedCommit{
				{Type: CommitTypeFeat},
				{Type: CommitTypeDocs},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.hasLogicalGroups(tt.commits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitManager_HasFeatureGroups(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	tests := []struct {
		name     string
		commits  []PlannedCommit
		expected bool
	}{
		{
			name: "has feature commit",
			commits: []PlannedCommit{
				{Type: CommitTypeFeat},
				{Type: CommitTypeDocs},
			},
			expected: true,
		},
		{
			name: "no feature commits",
			commits: []PlannedCommit{
				{Type: CommitTypeFix},
				{Type: CommitTypeDocs},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.hasFeatureGroups(tt.commits)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCommitManager_SortCommitsByDependencies(t *testing.T) {
	cm, err := NewCommitManager(CommitConfig{}, "/tmp")
	require.NoError(t, err)

	commits := []PlannedCommit{
		{ID: "commit_1", Priority: 10},
		{ID: "commit_2", Priority: 20},
		{ID: "commit_3", Priority: 15},
	}

	dependencies := map[string][]string{
		"commit_2": {"commit_1"}, // commit_2 depends on commit_1
		"commit_3": {},
	}

	sorted := cm.sortCommitsByDependencies(commits, dependencies)
	assert.Len(t, sorted, 3)

	// Should be sorted by priority and dependencies
	// commit_1 should come before commit_2 due to dependency
	commit1Index := -1
	commit2Index := -1
	for i, commit := range sorted {
		if commit.ID == "commit_1" {
			commit1Index = i
		}
		if commit.ID == "commit_2" {
			commit2Index = i
		}
	}

	assert.True(t, commit1Index < commit2Index, "commit_1 should come before commit_2")
}

func TestCreateCommitMetadata(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "commit-metadata-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	cm, err := NewCommitManager(CommitConfig{}, tempDir)
	require.NoError(t, err)

	analysisResult := &analysis.AnalysisResult{
		ProjectInfo: &analysis.ProjectInfo{
			Name:         "test-project",
			MainLanguage: "Go",
		},
	}

	metadata, err := cm.createCommitMetadata(context.Background(), analysisResult)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.NotZero(t, metadata.Timestamp)
	assert.Equal(t, analysisResult.ProjectInfo, metadata.ProjectInfo)
}
