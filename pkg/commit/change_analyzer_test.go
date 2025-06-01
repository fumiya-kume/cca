package commit

import (
	"context"
	"testing"

	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChangeAnalyzer(t *testing.T) {
	tests := []struct {
		name             string
		config           ChangeAnalyzerConfig
		expectedMaxSize  int
		expectedMinSize  int
		expectedMaxGroup int
	}{
		{
			name:             "default config",
			config:           ChangeAnalyzerConfig{},
			expectedMaxSize:  100,
			expectedMinSize:  1,
			expectedMaxGroup: 20,
		},
		{
			name: "custom config",
			config: ChangeAnalyzerConfig{
				MaxCommitSize: 50,
				MinGroupSize:  3,
				MaxGroupSize:  15,
				AtomicChanges: true,
			},
			expectedMaxSize:  50,
			expectedMinSize:  3,
			expectedMaxGroup: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := NewChangeAnalyzer(tt.config)
			assert.NotNil(t, ca)
			assert.Equal(t, tt.expectedMaxSize, ca.config.MaxCommitSize)
			assert.Equal(t, tt.expectedMinSize, ca.config.MinGroupSize)
			assert.Equal(t, tt.expectedMaxGroup, ca.config.MaxGroupSize)
		})
	}
}

func TestChangeAnalyzer_GroupChanges(t *testing.T) {
	tests := []struct {
		name           string
		config         ChangeAnalyzerConfig
		changes        []FileChange
		analysisResult *analysis.AnalysisResult
		expectedGroups int
		expectedError  bool
	}{
		{
			name:           "empty changes",
			config:         ChangeAnalyzerConfig{},
			changes:        []FileChange{},
			expectedGroups: 0,
			expectedError:  false,
		},
		{
			name:   "atomic changes",
			config: ChangeAnalyzerConfig{AtomicChanges: true},
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
				{Path: "utils.go", Type: ChangeTypeAdd},
			},
			expectedGroups: 2,
			expectedError:  false,
		},
		{
			name:   "group by type",
			config: ChangeAnalyzerConfig{GroupByType: true},
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
				{Path: "utils.go", Type: ChangeTypeModify},
				{Path: "new.go", Type: ChangeTypeAdd},
			},
			expectedGroups: 2, // One for modify, one for add
			expectedError:  false,
		},
		{
			name:   "group by module",
			config: ChangeAnalyzerConfig{GroupByModule: true},
			changes: []FileChange{
				{Path: "pkg/auth/user.go", Type: ChangeTypeModify},
				{Path: "pkg/auth/token.go", Type: ChangeTypeAdd},
				{Path: "cmd/main.go", Type: ChangeTypeModify},
			},
			analysisResult: &analysis.AnalysisResult{},
			expectedGroups: 2, // One for pkg, one for cmd
			expectedError:  false,
		},
		{
			name:   "separate tests and docs",
			config: ChangeAnalyzerConfig{SeparateTests: true, SeparateDocs: true},
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
				{Path: "main_test.go", Type: ChangeTypeModify},
				{Path: "README.md", Type: ChangeTypeModify},
			},
			expectedGroups: 3, // Code, tests, docs
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ca := NewChangeAnalyzer(tt.config)
			groups, err := ca.GroupChanges(context.Background(), tt.changes, tt.analysisResult)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, groups, tt.expectedGroups)
		})
	}
}

func TestChangeAnalyzer_GroupByChangeType(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	changes := []FileChange{
		{Path: "new1.go", Type: ChangeTypeAdd},
		{Path: "new2.go", Type: ChangeTypeAdd},
		{Path: "modified1.go", Type: ChangeTypeModify},
		{Path: "modified2.go", Type: ChangeTypeModify},
		{Path: "deleted.go", Type: ChangeTypeDelete},
	}

	groups := ca.groupByChangeType(changes)
	assert.Len(t, groups, 3) // Add, Modify, Delete

	// Find the add group
	var addGroup *ChangeGroup
	for _, group := range groups {
		if len(group.Changes) == 2 && group.Changes[0].Type == ChangeTypeAdd {
			addGroup = group
			break
		}
	}
	require.NotNil(t, addGroup)
	assert.Contains(t, addGroup.ID, "type_add")
	assert.Len(t, addGroup.Changes, 2)
}

func TestChangeAnalyzer_GroupByModule(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	changes := []FileChange{
		{Path: "pkg/auth/user.go", Type: ChangeTypeModify},
		{Path: "pkg/auth/token.go", Type: ChangeTypeAdd},
		{Path: "cmd/server/main.go", Type: ChangeTypeModify},
		{Path: "internal/utils.go", Type: ChangeTypeModify},
	}

	analysisResult := &analysis.AnalysisResult{}
	groups := ca.groupByModule(changes, analysisResult)
	assert.Len(t, groups, 3) // pkg, cmd, internal

	// Verify group structure
	for _, group := range groups {
		assert.NotEmpty(t, group.ID)
		assert.NotEmpty(t, group.Changes)
		assert.Contains(t, group.ID, "module_")
	}
}

func TestChangeAnalyzer_GroupByFeature(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	changes := []FileChange{
		{Path: "auth/user.go", Type: ChangeTypeModify},
		{Path: "auth/middleware.go", Type: ChangeTypeAdd},
		{Path: "api/handler.go", Type: ChangeTypeModify},
		{Path: "ui/components.go", Type: ChangeTypeAdd},
	}

	analysisResult := &analysis.AnalysisResult{}
	groups := ca.groupByFeature(changes, analysisResult)
	assert.GreaterOrEqual(t, len(groups), 1)

	// Verify group structure
	for _, group := range groups {
		assert.NotEmpty(t, group.ID)
		assert.NotEmpty(t, group.Changes)
		assert.Contains(t, group.ID, "feature_")
	}
}

func TestChangeAnalyzer_CreateLogicalGroups(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{
		SeparateTests: true,
		SeparateDocs:  true,
	})

	changes := []FileChange{
		{Path: "main.go", Type: ChangeTypeModify},
		{Path: "utils.go", Type: ChangeTypeAdd},
		{Path: "main_test.go", Type: ChangeTypeModify},
		{Path: "utils_test.go", Type: ChangeTypeAdd},
		{Path: "README.md", Type: ChangeTypeModify},
		{Path: "config.yaml", Type: ChangeTypeModify},
	}

	analysisResult := &analysis.AnalysisResult{}
	groups := ca.createLogicalGroups(changes, analysisResult)

	// Should have separate groups for code, tests, docs, and config
	assert.GreaterOrEqual(t, len(groups), 3)

	// Find test group
	testGroupFound := false
	docGroupFound := false
	configGroupFound := false

	for _, group := range groups {
		if group.Type == GroupTypeTest {
			testGroupFound = true
			assert.Len(t, group.Changes, 2) // Two test files
		}
		if group.Type == GroupTypeDocs {
			docGroupFound = true
			assert.Len(t, group.Changes, 1) // One doc file
		}
		if group.Type == GroupTypeConfig {
			configGroupFound = true
			assert.Len(t, group.Changes, 1) // One config file
		}
	}

	assert.True(t, testGroupFound)
	assert.True(t, docGroupFound)
	assert.True(t, configGroupFound)
}

func TestChangeAnalyzer_DetermineGroupType(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	tests := []struct {
		name     string
		changes  []FileChange
		expected GroupType
	}{
		{
			name: "test files",
			changes: []FileChange{
				{Path: "main_test.go", Type: ChangeTypeModify},
				{Path: "utils_test.go", Type: ChangeTypeAdd},
			},
			expected: GroupTypeTest,
		},
		{
			name: "doc files",
			changes: []FileChange{
				{Path: "README.md", Type: ChangeTypeModify},
				{Path: "docs/api.md", Type: ChangeTypeAdd},
			},
			expected: GroupTypeDocs,
		},
		{
			name: "config files",
			changes: []FileChange{
				{Path: "config.yaml", Type: ChangeTypeModify},
				{Path: "settings.json", Type: ChangeTypeAdd},
			},
			expected: GroupTypeConfig,
		},
		{
			name: "fix files",
			changes: []FileChange{
				{Path: "fix_bug.go", Type: ChangeTypeModify},
			},
			expected: GroupTypeFix,
		},
		{
			name: "feature files",
			changes: []FileChange{
				{Path: "new_feature.go", Type: ChangeTypeAdd},
			},
			expected: GroupTypeFeature,
		},
		{
			name: "mixed files",
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
				{Path: "utils.go", Type: ChangeTypeModify},
			},
			expected: GroupTypeMixed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ca.determineGroupType(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChangeAnalyzer_GenerateDescription(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	tests := []struct {
		name     string
		changes  []FileChange
		expected string
	}{
		{
			name: "single file",
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
			},
			expected: "Update main.go",
		},
		{
			name: "test files",
			changes: []FileChange{
				{Path: "main_test.go", Type: ChangeTypeModify},
				{Path: "utils_test.go", Type: ChangeTypeAdd},
			},
			expected: "Update tests (2 files)",
		},
		{
			name: "doc files",
			changes: []FileChange{
				{Path: "README.md", Type: ChangeTypeModify},
				{Path: "CHANGELOG.md", Type: ChangeTypeAdd},
			},
			expected: "Update documentation (2 files)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ca.generateDescription(tt.changes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChangeAnalyzer_CalculateGroupScore(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{MaxCommitSize: 20})

	tests := []struct {
		name        string
		changes     []FileChange
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "empty changes",
			changes:     []FileChange{},
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
		{
			name: "same file type",
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
				{Path: "utils.go", Type: ChangeTypeModify},
			},
			expectedMin: 1.4, // Base 1.0 + bonuses, allowing for variation
			expectedMax: 1.7,
		},
		{
			name: "mixed file types",
			changes: []FileChange{
				{Path: "main.go", Type: ChangeTypeModify},
				{Path: "README.md", Type: ChangeTypeAdd},
				{Path: "config.yaml", Type: ChangeTypeDelete},
			},
			expectedMin: 1.0, // At least base score
			expectedMax: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ca.calculateGroupScore(tt.changes)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
			assert.LessOrEqual(t, result, tt.expectedMax)
		})
	}
}

func TestChangeAnalyzer_ExtractModule(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "nested path",
			path:     "pkg/auth/user.go",
			expected: "pkg",
		},
		{
			name:     "root file",
			path:     "main.go",
			expected: "root",
		},
		{
			name:     "single directory",
			path:     "cmd/main.go",
			expected: "cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ca.extractModule(tt.path, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChangeAnalyzer_DetectFeature(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	tests := []struct {
		name     string
		change   FileChange
		expected string
	}{
		{
			name:     "auth feature",
			change:   FileChange{Path: "auth/middleware.go"},
			expected: "authentication",
		},
		{
			name:     "user feature",
			change:   FileChange{Path: "user/profile.go"},
			expected: "user_management",
		},
		{
			name:     "api feature",
			change:   FileChange{Path: "api/handler.go"},
			expected: "api",
		},
		{
			name:     "ui feature",
			change:   FileChange{Path: "ui/components.go"},
			expected: "ui",
		},
		{
			name:     "db feature",
			change:   FileChange{Path: "database/models.go"},
			expected: "database",
		},
		{
			name:     "config feature",
			change:   FileChange{Path: "config/settings.go"},
			expected: "configuration",
		},
		{
			name:     "core feature",
			change:   FileChange{Path: "main.go"},
			expected: "core",
		},
		{
			name:     "nested directory",
			change:   FileChange{Path: "pkg/utils/helper.go"},
			expected: "pkg_utils",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ca.detectFeature(tt.change, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChangeAnalyzer_IsTestFile(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

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
			result := ca.isTestFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChangeAnalyzer_IsDocFile(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

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
			result := ca.isDocFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChangeAnalyzer_IsConfigFile(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{})

	tests := []struct {
		path     string
		expected bool
	}{
		{"config.yaml", true},
		{"settings.json", true},
		{"app.toml", true},
		{"setup.ini", true},
		{"myapp.conf", true},
		{"database.config", true},
		{".env", true},
		{"Dockerfile", true},
		{"Makefile", true},
		{"main.go", false},
		{"utils.py", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ca.isConfigFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChangeAnalyzer_MergeSmallGroups(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{MinGroupSize: 2})

	groups := []*ChangeGroup{
		{
			ID:   "small1",
			Type: GroupTypeTest,
			Changes: []FileChange{
				{Path: "test1.go"},
			},
			Description: "Test group 1",
		},
		{
			ID:   "small2",
			Type: GroupTypeTest,
			Changes: []FileChange{
				{Path: "test2.go"},
			},
			Description: "Test group 2",
		},
		{
			ID:   "large",
			Type: GroupTypeFeature,
			Changes: []FileChange{
				{Path: "feature1.go"},
				{Path: "feature2.go"},
			},
			Description: "Feature group",
		},
	}

	refined := ca.mergeSmallGroups(groups)
	assert.Len(t, refined, 2) // small groups merged, large group remains

	// Find the merged test group
	var mergedTestGroup *ChangeGroup
	for _, group := range refined {
		if group.Type == GroupTypeTest {
			mergedTestGroup = group
			break
		}
	}

	require.NotNil(t, mergedTestGroup)
	assert.Len(t, mergedTestGroup.Changes, 2)
	assert.Contains(t, mergedTestGroup.ID, "merged_test")
	assert.Contains(t, mergedTestGroup.Description, "Test group 1; Test group 2")
}

func TestChangeAnalyzer_SplitLargeGroups(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{MaxGroupSize: 2})

	groups := []*ChangeGroup{
		{
			ID:   "small",
			Type: GroupTypeTest,
			Changes: []FileChange{
				{Path: "test1.go"},
			},
		},
		{
			ID:   "large",
			Type: GroupTypeFeature,
			Changes: []FileChange{
				{Path: "feature1.go"},
				{Path: "feature2.go"},
				{Path: "feature3.go"},
				{Path: "feature4.go"},
			},
		},
	}

	refined := ca.splitLargeGroups(groups)
	assert.Greater(t, len(refined), 2) // Large group should be split

	// Check that no group exceeds max size
	for _, group := range refined {
		assert.LessOrEqual(t, len(group.Changes), ca.config.MaxGroupSize)
	}
}

func TestChangeAnalyzer_SplitGroup(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{MaxGroupSize: 2})

	group := &ChangeGroup{
		ID:   "large_group",
		Type: GroupTypeFeature,
		Changes: []FileChange{
			{Path: "feature1.go"},
			{Path: "feature2.go"},
			{Path: "feature3.py"},
			{Path: "feature4.py"},
			{Path: "feature5.js"},
		},
	}

	subGroups := ca.splitGroup(group)
	assert.GreaterOrEqual(t, len(subGroups), 3) // Should create multiple subgroups

	// Check that each subgroup respects max size
	for _, subGroup := range subGroups {
		assert.LessOrEqual(t, len(subGroup.Changes), ca.config.MaxGroupSize)
		assert.Contains(t, subGroup.ID, "large_group_split_")
	}

	// Check that all original changes are preserved
	totalChanges := 0
	for _, subGroup := range subGroups {
		totalChanges += len(subGroup.Changes)
	}
	assert.Equal(t, len(group.Changes), totalChanges)
}

func TestChangeAnalyzer_AdjustGroupSizes(t *testing.T) {
	ca := NewChangeAnalyzer(ChangeAnalyzerConfig{MaxCommitSize: 3})

	groups := []*ChangeGroup{
		{
			ID:   "normal",
			Type: GroupTypeTest,
			Changes: []FileChange{
				{Path: "test1.go"},
				{Path: "test2.go"},
			},
		},
		{
			ID:   "oversized",
			Type: GroupTypeFeature,
			Changes: []FileChange{
				{Path: "feature1.go"},
				{Path: "feature2.go"},
				{Path: "feature3.go"},
				{Path: "feature4.go"},
				{Path: "feature5.go"},
			},
		},
	}

	adjusted := ca.adjustGroupSizes(groups)
	assert.Greater(t, len(adjusted), len(groups)) // Oversized group should be split

	// Check that no group exceeds max commit size
	for _, group := range adjusted {
		assert.LessOrEqual(t, len(group.Changes), ca.config.MaxCommitSize)
	}
}

// Test enum string methods

func TestGroupType_String(t *testing.T) {
	tests := []struct {
		groupType GroupType
		expected  string
	}{
		{GroupTypeFeature, "feature"},
		{GroupTypeFix, "fix"},
		{GroupTypeRefactor, "refactor"},
		{GroupTypeTest, "test"},
		{GroupTypeDocs, "docs"},
		{GroupTypeConfig, "config"},
		{GroupTypeStyle, "style"},
		{GroupTypeMixed, "mixed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.groupType.String())
		})
	}
}

func TestChangeAnalyzer_ChangeType_String(t *testing.T) {
	tests := []struct {
		changeType ChangeType
		expected   string
	}{
		{ChangeTypeAdd, "add"},
		{ChangeTypeModify, "modify"},
		{ChangeTypeDelete, "delete"},
		{ChangeTypeRename, "rename"},
		{ChangeTypeCopy, "copy"},
		{ChangeTypeUntracked, "untracked"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.changeType.String())
		})
	}
}
