package pr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestPRMetadata tests PRMetadata functionality
func TestPRMetadata(t *testing.T) {
	metadata := PRMetadata{
		IssueReferences:     []string{"#123", "#456"},
		ChangeType:          ChangeTypeFeature,
		Priority:            PriorityHigh,
		EstimatedReviewTime: 30 * time.Minute,
		Complexity:          ComplexityLevelModerate,
		TestCoverage:        85.5,
		GeneratedBy:         "ccagents",
		AutomationLevel:     AutomationLevelPartial,
		Tags:                []string{"backend", "api"},
	}

	assert.Equal(t, 2, len(metadata.IssueReferences))
	assert.Equal(t, ChangeTypeFeature, metadata.ChangeType)
	assert.Equal(t, PriorityHigh, metadata.Priority)
	assert.Equal(t, 30*time.Minute, metadata.EstimatedReviewTime)
	assert.Equal(t, ComplexityLevelModerate, metadata.Complexity)
	assert.Equal(t, 85.5, metadata.TestCoverage)
	assert.Equal(t, "ccagents", metadata.GeneratedBy)
	assert.Equal(t, AutomationLevelPartial, metadata.AutomationLevel)
	assert.Equal(t, []string{"backend", "api"}, metadata.Tags)
}

// TestCheckStatus tests CheckStatus functionality
func TestCheckStatus(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Minute)

	check := CheckStatus{
		Name:        "ci/build",
		Status:      CheckStateCompleted,
		Conclusion:  "success",
		DetailsURL:  "https://github.com/example/repo/actions/runs/123",
		StartedAt:   startTime,
		CompletedAt: &endTime,
		Output: CheckOutput{
			Title:   "Build Successful",
			Summary: "All tests passed",
			Text:    "Build completed successfully with 100% test coverage",
		},
	}

	assert.Equal(t, "ci/build", check.Name)
	assert.Equal(t, CheckStateCompleted, check.Status)
	assert.Equal(t, "success", check.Conclusion)
	assert.Equal(t, startTime, check.StartedAt)
	assert.Equal(t, &endTime, check.CompletedAt)
	assert.Equal(t, "Build Successful", check.Output.Title)
	assert.Equal(t, "All tests passed", check.Output.Summary)
	assert.Equal(t, "Build completed successfully with 100% test coverage", check.Output.Text)
}

// TestCheckOutput tests CheckOutput functionality
func TestCheckOutput(t *testing.T) {
	output := CheckOutput{
		Title:   "Lint Check",
		Summary: "Found 3 issues",
		Text:    "Detailed lint output with line numbers and suggestions",
	}

	assert.Equal(t, "Lint Check", output.Title)
	assert.Equal(t, "Found 3 issues", output.Summary)
	assert.Equal(t, "Detailed lint output with line numbers and suggestions", output.Text)
}

// TestConflictInfo tests ConflictInfo functionality
func TestConflictInfo(t *testing.T) {
	conflict := ConflictInfo{
		File:        "src/main.go",
		Lines:       []int{15, 16, 17},
		Type:        ConflictTypeContent,
		Severity:    ConflictSeverityMedium,
		Suggestion:  "Consider using the updated API method",
		AutoFixable: true,
	}

	assert.Equal(t, "src/main.go", conflict.File)
	assert.Equal(t, []int{15, 16, 17}, conflict.Lines)
	assert.Equal(t, ConflictTypeContent, conflict.Type)
	assert.Equal(t, ConflictSeverityMedium, conflict.Severity)
	assert.Equal(t, "Consider using the updated API method", conflict.Suggestion)
	assert.True(t, conflict.AutoFixable)
}

// TestReviewStatus tests ReviewStatus functionality
func TestReviewStatus(t *testing.T) {
	submittedAt := time.Now()

	review := ReviewStatus{
		Reviewer:    "reviewer@example.com",
		State:       ReviewStateApproved,
		SubmittedAt: submittedAt,
		Comments: []ReviewComment{
			{
				ID:       1,
				File:     "src/api.go",
				Line:     42,
				Body:     "Great implementation!",
				Type:     CommentTypePraise,
				Resolved: false,
			},
			{
				ID:       2,
				File:     "src/handler.go",
				Line:     15,
				Body:     "Consider adding error handling here",
				Type:     CommentTypeSuggestion,
				Resolved: true,
			},
		},
	}

	assert.Equal(t, "reviewer@example.com", review.Reviewer)
	assert.Equal(t, ReviewStateApproved, review.State)
	assert.Equal(t, submittedAt, review.SubmittedAt)
	assert.Equal(t, 2, len(review.Comments))
	assert.Equal(t, CommentTypePraise, review.Comments[0].Type)
	assert.Equal(t, CommentTypeSuggestion, review.Comments[1].Type)
	assert.False(t, review.Comments[0].Resolved)
	assert.True(t, review.Comments[1].Resolved)
}

// TestReviewComment tests ReviewComment functionality
func TestReviewComment(t *testing.T) {
	comment := ReviewComment{
		ID:       123,
		File:     "src/utils.go",
		Line:     25,
		Body:     "This could be optimized for better performance",
		Type:     CommentTypeSuggestion,
		Resolved: false,
	}

	assert.Equal(t, 123, comment.ID)
	assert.Equal(t, "src/utils.go", comment.File)
	assert.Equal(t, 25, comment.Line)
	assert.Equal(t, "This could be optimized for better performance", comment.Body)
	assert.Equal(t, CommentTypeSuggestion, comment.Type)
	assert.False(t, comment.Resolved)
}

// TestPRFilters tests PRFilters functionality
func TestPRFilters(t *testing.T) {
	filters := PRFilters{
		State:  PRStateOpen,
		Author: "developer@example.com",
		Labels: []string{"bug", "priority-high"},
		Branch: "feature/new-api",
	}

	assert.Equal(t, PRStateOpen, filters.State)
	assert.Equal(t, "developer@example.com", filters.Author)
	assert.Equal(t, []string{"bug", "priority-high"}, filters.Labels)
	assert.Equal(t, "feature/new-api", filters.Branch)
}

// TestNotificationSettings tests NotificationSettings functionality
func TestNotificationSettings(t *testing.T) {
	settings := NotificationSettings{
		OnCreate:   true,
		OnUpdate:   false,
		OnReview:   true,
		OnFailure:  true,
		OnMerge:    true,
		Recipients: []string{"team@example.com", "lead@example.com"},
		Channels:   []string{"#dev", "#alerts"},
	}

	assert.True(t, settings.OnCreate)
	assert.False(t, settings.OnUpdate)
	assert.True(t, settings.OnReview)
	assert.True(t, settings.OnFailure)
	assert.True(t, settings.OnMerge)
	assert.Equal(t, 2, len(settings.Recipients))
	assert.Equal(t, 2, len(settings.Channels))
	assert.Contains(t, settings.Recipients, "team@example.com")
	assert.Contains(t, settings.Recipients, "lead@example.com")
	assert.Contains(t, settings.Channels, "#dev")
	assert.Contains(t, settings.Channels, "#alerts")
}

// TestPRConfig tests PRConfig functionality
func TestPRConfig(t *testing.T) {
	config := PRConfig{
		AutoCreate:             true,
		AutoMerge:              false,
		RequireReviews:         true,
		MinReviewers:           2,
		AutoLabeling:           true,
		AutoAssignment:         true,
		ChecksRequired:         []string{"build", "test", "lint"},
		BranchProtection:       true,
		ConflictResolution:     ConflictStrategyManual,
		FailureRetries:         3,
		Template:               "default",
		CustomLabels:           map[string]string{"bug": "🐛 bug", "feature": "✨ enhancement"},
		DraftMode:              false,
		AutoUpdateBranch:       true,
		SquashOnMerge:          true,
		DeleteBranchAfterMerge: true,
	}

	assert.True(t, config.AutoCreate)
	assert.False(t, config.AutoMerge)
	assert.True(t, config.RequireReviews)
	assert.Equal(t, 2, config.MinReviewers)
	assert.True(t, config.AutoLabeling)
	assert.True(t, config.AutoAssignment)
	assert.Equal(t, []string{"build", "test", "lint"}, config.ChecksRequired)
	assert.True(t, config.BranchProtection)
	assert.Equal(t, ConflictStrategyManual, config.ConflictResolution)
	assert.Equal(t, 3, config.FailureRetries)
	assert.Equal(t, "default", config.Template)
	assert.Equal(t, 2, len(config.CustomLabels))
	assert.Equal(t, "🐛 bug", config.CustomLabels["bug"])
	assert.Equal(t, "✨ enhancement", config.CustomLabels["feature"])
	assert.False(t, config.DraftMode)
	assert.True(t, config.AutoUpdateBranch)
	assert.True(t, config.SquashOnMerge)
	assert.True(t, config.DeleteBranchAfterMerge)
}

// TestPullRequest tests PullRequest struct functionality
func TestPullRequest(t *testing.T) {
	createdAt := time.Now()
	updatedAt := createdAt.Add(1 * time.Hour)
	mergedAt := updatedAt.Add(30 * time.Minute)

	pr := PullRequest{
		ID:          123,
		Number:      456,
		Title:       "Add user authentication feature",
		Description: "This PR adds user authentication with JWT tokens",
		State:       PRStateOpen,
		Branch:      "feature/auth",
		BaseBranch:  "main",
		Author:      "developer@example.com",
		Reviewers:   []string{"reviewer1@example.com", "reviewer2@example.com"},
		Labels:      []string{"feature", "security"},
		Assignees:   []string{"maintainer@example.com"},
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		MergedAt:    &mergedAt,
		Metadata: PRMetadata{
			ChangeType:      ChangeTypeFeature,
			Priority:        PriorityHigh,
			Complexity:      ComplexityLevelModerate,
			AutomationLevel: AutomationLevelPartial,
		},
	}

	assert.Equal(t, 123, pr.ID)
	assert.Equal(t, 456, pr.Number)
	assert.Equal(t, "Add user authentication feature", pr.Title)
	assert.Equal(t, "This PR adds user authentication with JWT tokens", pr.Description)
	assert.Equal(t, PRStateOpen, pr.State)
	assert.Equal(t, "feature/auth", pr.Branch)
	assert.Equal(t, "main", pr.BaseBranch)
	assert.Equal(t, "developer@example.com", pr.Author)
	assert.Equal(t, 2, len(pr.Reviewers))
	assert.Contains(t, pr.Reviewers, "reviewer1@example.com")
	assert.Contains(t, pr.Reviewers, "reviewer2@example.com")
	assert.Equal(t, 2, len(pr.Labels))
	assert.Contains(t, pr.Labels, "feature")
	assert.Contains(t, pr.Labels, "security")
	assert.Equal(t, 1, len(pr.Assignees))
	assert.Contains(t, pr.Assignees, "maintainer@example.com")
	assert.Equal(t, createdAt, pr.CreatedAt)
	assert.Equal(t, updatedAt, pr.UpdatedAt)
	assert.Equal(t, &mergedAt, pr.MergedAt)
	assert.Equal(t, ChangeTypeFeature, pr.Metadata.ChangeType)
	assert.Equal(t, PriorityHigh, pr.Metadata.Priority)
}

// TestPRTemplate tests PRTemplate functionality
func TestPRTemplate(t *testing.T) {
	template := PRTemplate{
		Name:        "feature_template",
		Title:       "Feature: {{.FeatureName}}",
		Description: "## Summary\n{{.Summary}}\n\n## Changes\n{{.Changes}}",
		Variables: map[string]string{
			"FeatureName": "User Authentication",
			"Summary":     "Adds JWT-based user authentication",
			"Changes":     "- Added auth middleware\n- Added JWT utilities",
		},
		Sections: []TemplateSection{
			{
				Name:     "summary",
				Title:    "Summary",
				Content:  "{{.Summary}}",
				Required: true,
				Order:    1,
			},
			{
				Name:     "changes",
				Title:    "Changes",
				Content:  "{{.Changes}}",
				Required: true,
				Order:    2,
			},
		},
		Conditions: []TemplateCondition{
			{
				Field:    "change_type",
				Operator: "equals",
				Value:    "feature",
			},
		},
	}

	assert.Equal(t, "feature_template", template.Name)
	assert.Equal(t, "Feature: {{.FeatureName}}", template.Title)
	assert.Contains(t, template.Description, "## Summary")
	assert.Contains(t, template.Description, "## Changes")
	assert.Equal(t, 3, len(template.Variables))
	assert.Equal(t, "User Authentication", template.Variables["FeatureName"])
	assert.Equal(t, 2, len(template.Sections))
	assert.Equal(t, "summary", template.Sections[0].Name)
	assert.True(t, template.Sections[0].Required)
	assert.Equal(t, 1, template.Sections[0].Order)
	assert.Equal(t, 1, len(template.Conditions))
	assert.Equal(t, "change_type", template.Conditions[0].Field)
	assert.Equal(t, "equals", template.Conditions[0].Operator)
	assert.Equal(t, "feature", template.Conditions[0].Value)
}

// TestTemplateSection tests TemplateSection functionality
func TestTemplateSection(t *testing.T) {
	section := TemplateSection{
		Name:     "testing",
		Title:    "Testing Plan",
		Content:  "Please describe your testing approach",
		Required: true,
		Order:    3,
	}

	assert.Equal(t, "testing", section.Name)
	assert.Equal(t, "Testing Plan", section.Title)
	assert.Equal(t, "Please describe your testing approach", section.Content)
	assert.True(t, section.Required)
	assert.Equal(t, 3, section.Order)
}

// TestTemplateCondition tests TemplateCondition functionality
func TestTemplateCondition(t *testing.T) {
	condition := TemplateCondition{
		Field:    "complexity",
		Operator: "greater_than",
		Value:    "simple",
	}

	assert.Equal(t, "complexity", condition.Field)
	assert.Equal(t, "greater_than", condition.Operator)
	assert.Equal(t, "simple", condition.Value)
}
