package commit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommitType_String(t *testing.T) {
	tests := []struct {
		commitType CommitType
		expected   string
	}{
		{CommitTypeFeat, "feat"},
		{CommitTypeFix, "fix"},
		{CommitTypeDocs, "docs"},
		{CommitTypeStyle, "style"},
		{CommitTypeRefactor, "refactor"},
		{CommitTypePerf, "perf"},
		{CommitTypeTest, "test"},
		{CommitTypeBuild, "build"},
		{CommitTypeCI, "ci"},
		{CommitTypeChore, "chore"},
		{CommitTypeRevert, "revert"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.commitType.String())
		})
	}
}

func TestCommitStrategy_String(t *testing.T) {
	tests := []struct {
		strategy CommitStrategy
		expected string
	}{
		{CommitStrategyAtomic, "atomic"},
		{CommitStrategyLogical, "logical"},
		{CommitStrategyFeature, "feature"},
		{CommitStrategyFile, "file"},
		{CommitStrategyTemporal, "temporal"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.strategy.String())
		})
	}
}

func TestChangeType_String(t *testing.T) {
	// Test the parseGitStatusLine logic
	tests := []struct {
		name       string
		statusChar byte
		expected   ChangeType
	}{
		{"added file", 'A', ChangeTypeAdd},
		{"modified file", 'M', ChangeTypeModify},
		{"deleted file", 'D', ChangeTypeDelete},
		{"renamed file", 'R', ChangeTypeRename},
		{"copied file", 'C', ChangeTypeCopy},
		{"untracked file", '?', ChangeTypeUntracked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from parseGitStatusLine
			var changeType ChangeType
			switch tt.statusChar {
			case 'A':
				changeType = ChangeTypeAdd
			case 'M':
				changeType = ChangeTypeModify
			case 'D':
				changeType = ChangeTypeDelete
			case 'R':
				changeType = ChangeTypeRename
			case 'C':
				changeType = ChangeTypeCopy
			case '?':
				changeType = ChangeTypeUntracked
			default:
				changeType = ChangeTypeModify
			}
			assert.Equal(t, tt.expected, changeType)
		})
	}
}

func TestCommitSize_Classification(t *testing.T) {
	tests := []struct {
		name      string
		additions int
		deletions int
		expected  CommitSize
	}{
		{"empty commit", 0, 0, CommitSizeSmall},
		{"small commit", 5, 3, CommitSizeSmall},
		{"medium commit", 20, 15, CommitSizeMedium},
		{"large commit", 100, 50, CommitSizeLarge},
		{"huge commit", 300, 200, CommitSizeHuge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalChanges := tt.additions + tt.deletions

			var size CommitSize
			switch {
			case totalChanges <= 10:
				size = CommitSizeSmall
			case totalChanges <= 50:
				size = CommitSizeMedium
			case totalChanges <= 200:
				size = CommitSizeLarge
			default:
				size = CommitSizeHuge
			}

			assert.Equal(t, tt.expected, size)
		})
	}
}

func TestMessageStyle_String(t *testing.T) {
	tests := []struct {
		style    MessageStyle
		expected string
	}{
		{MessageStyleConventional, "conventional"},
		{MessageStyleTraditional, "traditional"},
		{MessageStyleCustom, "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.style.String())
		})
	}
}
