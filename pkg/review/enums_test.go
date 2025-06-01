package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReviewStatus_String tests the String method for ReviewStatus enum
func TestReviewStatus_String(t *testing.T) {
	tests := []struct {
		status   ReviewStatus
		expected string
	}{
		{ReviewStatusPending, "pending"},
		{ReviewStatusInProgress, "in_progress"},
		{ReviewStatusCompleted, "completed"},
		{ReviewStatusFailed, "failed"},
		{ReviewStatusPartial, "partial"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

// TestIssueType_String tests the String method for IssueType enum
func TestIssueType_String(t *testing.T) {
	tests := []struct {
		issueType IssueType
		expected  string
	}{
		{IssueTypeSyntax, "syntax"},
		{IssueTypeLogic, "logic"},
		{IssueTypeSecurity, "security"},
		{IssueTypePerformance, "performance"},
		{IssueTypeStyle, "style"},
		{IssueTypeMaintainability, "maintainability"},
		{IssueTypeDocumentation, "documentation"},
		{IssueTypeTesting, "testing"},
		{IssueTypeAccessibility, "accessibility"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.issueType.String())
		})
	}
}

// TestIssueSeverity_String tests the String method for IssueSeverity enum
func TestIssueSeverity_String(t *testing.T) {
	tests := []struct {
		severity IssueSeverity
		expected string
	}{
		{IssueSeverityInfo, "info"},
		{IssueSeverityLow, "low"},
		{IssueSeverityMedium, "medium"},
		{IssueSeverityHigh, "high"},
		{IssueSeverityCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

// TestSuggestionType tests SuggestionType enum values
func TestSuggestionType_Enum(t *testing.T) {
	tests := []struct {
		name           string
		suggestionType SuggestionType
		value          int
	}{
		{"refactor", SuggestionTypeRefactor, 0},
		{"optimization", SuggestionTypeOptimization, 1},
		{"architecture", SuggestionTypeArchitecture, 2},
		{"security", SuggestionTypeSecurity, 3},
		{"style", SuggestionTypeStyle, 4},
		{"documentation", SuggestionTypeDocumentation, 5},
		{"testing", SuggestionTypeTesting, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.suggestionType))
		})
	}
}

// TestSuggestionPriority tests SuggestionPriority enum values
func TestSuggestionPriority_Enum(t *testing.T) {
	tests := []struct {
		name     string
		priority SuggestionPriority
		value    int
	}{
		{"low", SuggestionPriorityLow, 0},
		{"medium", SuggestionPriorityMedium, 1},
		{"high", SuggestionPriorityHigh, 2},
		{"critical", SuggestionPriorityCritical, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.priority))
		})
	}
}

// TestSecuritySeverity tests SecuritySeverity enum values
func TestSecuritySeverity_Enum(t *testing.T) {
	tests := []struct {
		name     string
		severity SecuritySeverity
		value    int
	}{
		{"info", SecuritySeverityInfo, 0},
		{"low", SecuritySeverityLow, 1},
		{"medium", SecuritySeverityMedium, 2},
		{"high", SecuritySeverityHigh, 3},
		{"critical", SecuritySeverityCritical, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.severity))
		})
	}
}

// TestSecurityLevel tests SecurityLevel enum values
func TestSecurityLevel_Enum(t *testing.T) {
	tests := []struct {
		name  string
		level SecurityLevel
		value int
	}{
		{"basic", SecurityLevelBasic, 0},
		{"standard", SecurityLevelStandard, 1},
		{"strict", SecurityLevelStrict, 2},
		{"pentesting", SecurityLevelPentesting, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.level))
		})
	}
}

// TestReviewStatus tests ReviewStatus enum values
func TestReviewStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ReviewStatus
		value  int
	}{
		{"pending", ReviewStatusPending, 0},
		{"in_progress", ReviewStatusInProgress, 1},
		{"completed", ReviewStatusCompleted, 2},
		{"failed", ReviewStatusFailed, 3},
		{"partial", ReviewStatusPartial, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.status))
		})
	}
}

// TestIssueType tests IssueType enum values
func TestIssueType(t *testing.T) {
	tests := []struct {
		name      string
		issueType IssueType
		value     int
	}{
		{"syntax", IssueTypeSyntax, 0},
		{"logic", IssueTypeLogic, 1},
		{"security", IssueTypeSecurity, 2},
		{"performance", IssueTypePerformance, 3},
		{"style", IssueTypeStyle, 4},
		{"maintainability", IssueTypeMaintainability, 5},
		{"documentation", IssueTypeDocumentation, 6},
		{"testing", IssueTypeTesting, 7},
		{"accessibility", IssueTypeAccessibility, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.issueType))
		})
	}
}

// TestIssueSeverity tests IssueSeverity enum values
func TestIssueSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity IssueSeverity
		value    int
	}{
		{"info", IssueSeverityInfo, 0},
		{"low", IssueSeverityLow, 1},
		{"medium", IssueSeverityMedium, 2},
		{"high", IssueSeverityHigh, 3},
		{"critical", IssueSeverityCritical, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, int(tt.severity))
		})
	}
}
