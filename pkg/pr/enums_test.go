package pr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPRState_String tests the String method for PRState enum
func TestPRState_String(t *testing.T) {
	tests := []struct {
		state    PRState
		expected string
	}{
		{PRStateOpen, "open"},
		{PRStateClosed, "closed"},
		{PRStateMerged, "merged"},
		{PRStateDraft, "draft"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

// TestChangeType_String tests the String method for ChangeType enum
func TestChangeType_String(t *testing.T) {
	tests := []struct {
		changeType ChangeType
		expected   string
	}{
		{ChangeTypeFeature, "feature"},
		{ChangeTypeBugfix, "bugfix"},
		{ChangeTypeHotfix, "hotfix"},
		{ChangeTypeRefactor, "refactor"},
		{ChangeTypeDocumentation, "documentation"},
		{ChangeTypeConfiguration, "configuration"},
		{ChangeTypeTest, "test"},
		{ChangeTypeChore, "chore"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.changeType.String())
		})
	}
}

// TestPriority_String tests the String method for Priority enum
func TestPriority_String(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityMedium, "medium"},
		{PriorityHigh, "high"},
		{PriorityCritical, "critical"},
		{PriorityUrgent, "urgent"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.priority.String())
		})
	}
}

// TestComplexityLevel_String tests the String method for ComplexityLevel enum
func TestComplexityLevel_String(t *testing.T) {
	tests := []struct {
		complexity ComplexityLevel
		expected   string
	}{
		{ComplexityLevelSimple, "simple"},
		{ComplexityLevelModerate, "moderate"},
		{ComplexityLevelComplex, "complex"},
		{ComplexityLevelVeryComplex, "very-complex"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.complexity.String())
		})
	}
}

// TestAutomationLevel_String tests the String method for AutomationLevel enum
func TestAutomationLevel_String(t *testing.T) {
	tests := []struct {
		automation AutomationLevel
		expected   string
	}{
		{AutomationLevelManual, "manual"},
		{AutomationLevelPartial, "partial"},
		{AutomationLevelFull, "full"},
		{AutomationLevelAI, "ai"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.automation.String())
		})
	}
}

// TestCheckState tests CheckState enum values
func TestCheckState(t *testing.T) {
	tests := []struct {
		name  string
		state CheckState
		value int
	}{
		{"pending", CheckStatePending, 0},
		{"queued", CheckStateQueued, 1},
		{"in_progress", CheckStateInProgress, 2},
		{"completed", CheckStateCompleted, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.state))
		})
	}
}

// TestConflictType tests ConflictType enum values
func TestConflictType(t *testing.T) {
	tests := []struct {
		name         string
		conflictType ConflictType
		value        int
	}{
		{"content", ConflictTypeContent, 0},
		{"rename", ConflictTypeRename, 1},
		{"delete", ConflictTypeDelete, 2},
		{"binary", ConflictTypeBinary, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.conflictType))
		})
	}
}

// TestConflictSeverity tests ConflictSeverity enum values
func TestConflictSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity ConflictSeverity
		value    int
	}{
		{"low", ConflictSeverityLow, 0},
		{"medium", ConflictSeverityMedium, 1},
		{"high", ConflictSeverityHigh, 2},
		{"critical", ConflictSeverityCritical, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.severity))
		})
	}
}

// TestReviewState tests ReviewState enum values
func TestReviewState(t *testing.T) {
	tests := []struct {
		name  string
		state ReviewState
		value int
	}{
		{"pending", ReviewStatePending, 0},
		{"approved", ReviewStateApproved, 1},
		{"changes_requested", ReviewStateChangesRequested, 2},
		{"commented", ReviewStateCommented, 3},
		{"dismissed", ReviewStateDismissed, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.state))
		})
	}
}

// TestCommentType tests CommentType enum values
func TestCommentType(t *testing.T) {
	tests := []struct {
		name        string
		commentType CommentType
		value       int
	}{
		{"general", CommentTypeGeneral, 0},
		{"suggestion", CommentTypeSuggestion, 1},
		{"nitpick", CommentTypeNitpick, 2},
		{"blocking", CommentTypeBlocking, 3},
		{"praise", CommentTypePraise, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.commentType))
		})
	}
}

// TestMergeMethod tests MergeMethod enum values
func TestMergeMethod(t *testing.T) {
	tests := []struct {
		name   string
		method MergeMethod
		value  int
	}{
		{"merge", MergeMethodMerge, 0},
		{"squash", MergeMethodSquash, 1},
		{"rebase", MergeMethodRebase, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.method))
		})
	}
}
