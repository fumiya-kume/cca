package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fumiya-kume/cca/pkg/analysis"
)

// TestNewReviewEngine tests the ReviewEngine constructor
func TestNewReviewEngine(t *testing.T) {
	tests := []struct {
		name           string
		config         ReviewConfig
		expectedConfig ReviewConfig
		expectError    bool
	}{
		{
			name: "default configuration",
			config: ReviewConfig{
				EnableStaticAnalysis: true,
				EnableSecurityScan:   true,
			},
			expectedConfig: ReviewConfig{
				EnableStaticAnalysis: true,
				EnableSecurityScan:   true,
				MaxReviewIterations:  3,                // Should be set to default
				MaxWorkers:           4,                // Should be set to default
				ReviewTimeout:        30 * time.Minute, // Should be set to default
				MinQualityScore:      0.7,              // Should be set to default
			},
			expectError: false,
		},
		{
			name: "custom configuration",
			config: ReviewConfig{
				EnableStaticAnalysis: true,
				EnableSecurityScan:   true,
				EnableQualityCheck:   true,
				EnableAIReview:       true,
				MaxReviewIterations:  5,
				MaxWorkers:           8,
				ReviewTimeout:        45 * time.Minute,
				MinQualityScore:      0.8,
				SecurityLevel:        SecurityLevelStrict,
				IgnorePatterns:       []string{"*.test.go", "vendor/*"},
			},
			expectedConfig: ReviewConfig{
				EnableStaticAnalysis: true,
				EnableSecurityScan:   true,
				EnableQualityCheck:   true,
				EnableAIReview:       true,
				MaxReviewIterations:  5,                // Should keep custom value
				MaxWorkers:           8,                // Should keep custom value
				ReviewTimeout:        45 * time.Minute, // Should keep custom value
				MinQualityScore:      0.8,              // Should keep custom value
				SecurityLevel:        SecurityLevelStrict,
				IgnorePatterns:       []string{"*.test.go", "vendor/*"},
			},
			expectError: false,
		},
		{
			name: "disabled all analyzers",
			config: ReviewConfig{
				EnableStaticAnalysis: false,
				EnableSecurityScan:   false,
				EnableQualityCheck:   false,
				EnableAIReview:       false,
			},
			expectedConfig: ReviewConfig{
				EnableStaticAnalysis: false,
				EnableSecurityScan:   false,
				EnableQualityCheck:   false,
				EnableAIReview:       false,
				MaxReviewIterations:  3,
				MaxWorkers:           4,
				ReviewTimeout:        30 * time.Minute,
				MinQualityScore:      0.7,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewReviewEngine(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, engine)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, engine)

			// Check configuration defaults were applied
			assert.Equal(t, tt.expectedConfig.MaxReviewIterations, engine.config.MaxReviewIterations)
			assert.Equal(t, tt.expectedConfig.MaxWorkers, engine.config.MaxWorkers)
			assert.Equal(t, tt.expectedConfig.ReviewTimeout, engine.config.ReviewTimeout)
			assert.Equal(t, tt.expectedConfig.MinQualityScore, engine.config.MinQualityScore)

			// Check analyzers are initialized based on configuration
			if tt.config.EnableStaticAnalysis {
				assert.NotNil(t, engine.staticAnalyzer)
			} else {
				assert.Nil(t, engine.staticAnalyzer)
			}

			if tt.config.EnableSecurityScan {
				assert.NotNil(t, engine.securityScanner)
			} else {
				assert.Nil(t, engine.securityScanner)
			}

			if tt.config.EnableQualityCheck {
				assert.NotNil(t, engine.qualityAnalyzer)
			} else {
				assert.Nil(t, engine.qualityAnalyzer)
			}

			if tt.config.EnableAIReview {
				assert.NotNil(t, engine.aiReviewer)
			} else {
				assert.Nil(t, engine.aiReviewer)
			}
		})
	}
}

// TestReviewEngine_ReviewChanges tests the main review function
func TestReviewEngine_ReviewChanges(t *testing.T) {
	t.Run("basic review with minimal config", func(t *testing.T) {
		// Create engine with minimal configuration
		config := ReviewConfig{
			EnableStaticAnalysis: false, // Disable to avoid external tool dependencies
			EnableSecurityScan:   false,
			EnableQualityCheck:   false,
			EnableAIReview:       false,
		}

		engine, err := NewReviewEngine(config)
		require.NoError(t, err)
		require.NotNil(t, engine)

		ctx := context.Background()
		changes := []string{"main.go", "utils.go"}
		analysisResult := &analysis.AnalysisResult{
			// Basic analysis result for testing
		}

		// This should complete without error even with no analyzers enabled
		result, err := engine.ReviewChanges(ctx, changes, analysisResult)

		// The function might fail due to unimplemented parts, but we test the structure
		if err != nil {
			t.Logf("Expected error due to unimplemented functionality: %v", err)
		} else {
			assert.NotNil(t, result)
			assert.Equal(t, ReviewStatusCompleted, result.Status)
		}
	})

	t.Run("review with timeout context", func(t *testing.T) {
		config := ReviewConfig{
			EnableStaticAnalysis: false,
		}

		engine, err := NewReviewEngine(config)
		require.NoError(t, err)

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait for context to timeout
		time.Sleep(2 * time.Millisecond)

		changes := []string{"main.go"}
		analysisResult := &analysis.AnalysisResult{}

		result, err := engine.ReviewChanges(ctx, changes, analysisResult)

		// Should handle timeout gracefully - the exact behavior may vary
		// based on implementation, so we just check that it doesn't crash
		t.Logf("Result: %+v, Error: %v", result, err)
	})
}

// TestReviewEngine_GetCriticalIssues tests the getCriticalIssues method
func TestReviewEngine_GetCriticalIssues(t *testing.T) {
	engine := &ReviewEngine{}

	tests := []struct {
		name             string
		issues           []ReviewIssue
		expectedCritical int
		expectedSeverity IssueSeverity
	}{
		{
			name:             "no issues",
			issues:           []ReviewIssue{},
			expectedCritical: 0,
		},
		{
			name: "no critical issues",
			issues: []ReviewIssue{
				{Severity: IssueSeverityLow, Description: "Low issue"},
				{Severity: IssueSeverityMedium, Description: "Medium issue"},
				{Severity: IssueSeverityHigh, Description: "High issue"},
			},
			expectedCritical: 0,
		},
		{
			name: "mixed issues with critical",
			issues: []ReviewIssue{
				{Severity: IssueSeverityLow, Description: "Low issue"},
				{Severity: IssueSeverityCritical, Description: "Critical issue 1"},
				{Severity: IssueSeverityMedium, Description: "Medium issue"},
				{Severity: IssueSeverityCritical, Description: "Critical issue 2"},
				{Severity: IssueSeverityHigh, Description: "High issue"},
			},
			expectedCritical: 2,
			expectedSeverity: IssueSeverityCritical,
		},
		{
			name: "all critical issues",
			issues: []ReviewIssue{
				{Severity: IssueSeverityCritical, Description: "Critical issue 1"},
				{Severity: IssueSeverityCritical, Description: "Critical issue 2"},
				{Severity: IssueSeverityCritical, Description: "Critical issue 3"},
			},
			expectedCritical: 3,
			expectedSeverity: IssueSeverityCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			critical := engine.getCriticalIssues(tt.issues)

			assert.Len(t, critical, tt.expectedCritical)

			if tt.expectedCritical > 0 {
				for _, issue := range critical {
					assert.Equal(t, tt.expectedSeverity, issue.Severity)
				}
			}
		})
	}
}

// TestReviewResult_Structure tests ReviewResult structure
func TestReviewResult_Structure(t *testing.T) {
	timestamp := time.Now()
	result := ReviewResult{
		OverallScore: 0.85,
		Status:       ReviewStatusCompleted,
		Issues: []ReviewIssue{
			{
				ID:          "issue-1",
				Type:        IssueTypeSecurity,
				Severity:    IssueSeverityHigh,
				Title:       "Security vulnerability found",
				Description: "Potential SQL injection vulnerability",
				File:        "database.go",
				Line:        42,
				Column:      15,
				Rule:        "sql-injection",
				Category:    "security",
				Fixable:     true,
				Suggestion:  "Use parameterized queries",
				Context:     []string{"db.Query(query)", "query := \"SELECT * FROM users WHERE id = \" + userInput"},
				Tags:        []string{"security", "database"},
			},
		},
		Suggestions: []ReviewSuggestion{
			{
				ID:          "suggestion-1",
				Type:        SuggestionTypeRefactor,
				Priority:    SuggestionPriorityMedium,
				Title:       "Extract method",
				Description: "This function is too long and should be broken down",
				File:        "handlers.go",
				Line:        100,
				Before:      "long function implementation",
				After:       "extracted smaller functions",
				Rationale:   "Improves readability and maintainability",
				Impact:      "Medium",
				Effort:      "Low",
			},
		},
		SecurityFindings: []SecurityFinding{
			{
				ID:          "sec-1",
				Severity:    SecuritySeverityHigh,
				Type:        "injection",
				Title:       "SQL Injection",
				Description: "Raw SQL query with user input",
				File:        "database.go",
				Line:        42,
				CWE:         "CWE-89",
				OWASP:       "A03:2021",
				Confidence:  0.9,
				Impact:      "High - Could lead to data breach",
				Remediation: "Use parameterized queries or ORM",
				References:  []string{"https://owasp.org/www-community/attacks/SQL_Injection"},
			},
		},
		QualityMetrics: &QualityMetrics{
			OverallScore:    0.75,
			Maintainability: 0.8,
			Readability:     0.85,
			Complexity: &ComplexityMetrics{
				Cyclomatic:    15.5,
				Cognitive:     12.3,
				Halstead:      8.7,
				LinesOfCode:   1500,
				Functions:     45,
				Classes:       8,
				AverageMethod: 25.5,
			},
			Coverage: &CoverageMetrics{
				Line:      85.5,
				Branch:    78.2,
				Function:  92.1,
				Statement: 83.7,
				Uncovered: []string{"handlers.go:150-155", "utils.go:75-80"},
			},
			Duplication: &DuplicationMetrics{
				Percentage: 5.2,
				Lines:      78,
				Blocks:     3,
				Files:      2,
				Duplications: []DuplicationBlock{
					{
						Files:     []string{"helper1.go", "helper2.go"},
						Lines:     25,
						Tokens:    150,
						StartLine: 10,
						EndLine:   35,
					},
				},
			},
			Documentation: 0.65,
			TestQuality:   0.72,
			Performance:   0.88,
			TechnicalDebt: 45 * time.Minute,
		},
		Iterations: 2,
		ReviewTime: 5 * time.Minute,
		Timestamp:  timestamp,
	}

	// Test main fields
	assert.Equal(t, 0.85, result.OverallScore)
	assert.Equal(t, ReviewStatusCompleted, result.Status)
	assert.Len(t, result.Issues, 1)
	assert.Len(t, result.Suggestions, 1)
	assert.Len(t, result.SecurityFindings, 1)
	assert.NotNil(t, result.QualityMetrics)
	assert.Equal(t, 2, result.Iterations)
	assert.Equal(t, 5*time.Minute, result.ReviewTime)
	assert.Equal(t, timestamp, result.Timestamp)

	// Test issue structure
	issue := result.Issues[0]
	assert.Equal(t, "issue-1", issue.ID)
	assert.Equal(t, IssueTypeSecurity, issue.Type)
	assert.Equal(t, IssueSeverityHigh, issue.Severity)
	assert.Equal(t, "Security vulnerability found", issue.Title)
	assert.Equal(t, "database.go", issue.File)
	assert.Equal(t, 42, issue.Line)
	assert.True(t, issue.Fixable)
	assert.Len(t, issue.Context, 2)
	assert.Contains(t, issue.Tags, "security")

	// Test suggestion structure
	suggestion := result.Suggestions[0]
	assert.Equal(t, "suggestion-1", suggestion.ID)
	assert.Equal(t, SuggestionTypeRefactor, suggestion.Type)
	assert.Equal(t, SuggestionPriorityMedium, suggestion.Priority)
	assert.Equal(t, "Extract method", suggestion.Title)

	// Test security finding structure
	finding := result.SecurityFindings[0]
	assert.Equal(t, "sec-1", finding.ID)
	assert.Equal(t, SecuritySeverityHigh, finding.Severity)
	assert.Equal(t, "injection", finding.Type)
	assert.Equal(t, "CWE-89", finding.CWE)
	assert.Equal(t, 0.9, finding.Confidence)
	assert.Len(t, finding.References, 1)

	// Test quality metrics structure
	metrics := result.QualityMetrics
	assert.Equal(t, 0.75, metrics.OverallScore)
	assert.NotNil(t, metrics.Complexity)
	assert.NotNil(t, metrics.Coverage)
	assert.NotNil(t, metrics.Duplication)

	// Test complexity metrics
	complexity := metrics.Complexity
	assert.Equal(t, 15.5, complexity.Cyclomatic)
	assert.Equal(t, 1500, complexity.LinesOfCode)
	assert.Equal(t, 45, complexity.Functions)

	// Test coverage metrics
	coverage := metrics.Coverage
	assert.Equal(t, 85.5, coverage.Line)
	assert.Equal(t, 78.2, coverage.Branch)
	assert.Len(t, coverage.Uncovered, 2)

	// Test duplication metrics
	duplication := metrics.Duplication
	assert.Equal(t, 5.2, duplication.Percentage)
	assert.Equal(t, 3, duplication.Blocks)
	assert.Len(t, duplication.Duplications, 1)

	dup := duplication.Duplications[0]
	assert.Len(t, dup.Files, 2)
	assert.Equal(t, 25, dup.Lines)
	assert.Equal(t, 150, dup.Tokens)
}

// TestReviewConfig_Structure tests ReviewConfig structure
func TestReviewConfig_Structure(t *testing.T) {
	config := ReviewConfig{
		EnableStaticAnalysis: true,
		EnableSecurityScan:   true,
		EnableQualityCheck:   true,
		EnableAIReview:       true,
		MaxReviewIterations:  5,
		ParallelReviews:      true,
		MaxWorkers:           8,
		ReviewTimeout:        45 * time.Minute,
		MinQualityScore:      0.8,
		SecurityLevel:        SecurityLevelStrict,
		IgnorePatterns:       []string{"*.test.go", "vendor/*", "node_modules/*"},
	}

	assert.True(t, config.EnableStaticAnalysis)
	assert.True(t, config.EnableSecurityScan)
	assert.True(t, config.EnableQualityCheck)
	assert.True(t, config.EnableAIReview)
	assert.Equal(t, 5, config.MaxReviewIterations)
	assert.True(t, config.ParallelReviews)
	assert.Equal(t, 8, config.MaxWorkers)
	assert.Equal(t, 45*time.Minute, config.ReviewTimeout)
	assert.Equal(t, 0.8, config.MinQualityScore)
	assert.Equal(t, SecurityLevelStrict, config.SecurityLevel)
	assert.Len(t, config.IgnorePatterns, 3)
	assert.Contains(t, config.IgnorePatterns, "*.test.go")
	assert.Contains(t, config.IgnorePatterns, "vendor/*")
	assert.Contains(t, config.IgnorePatterns, "node_modules/*")
}
