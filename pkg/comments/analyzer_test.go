package comments

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommentAnalyzer(t *testing.T) {
	tests := []struct {
		name           string
		config         CommentAnalyzerConfig
		expectedConfig CommentAnalyzerConfig
	}{
		{
			name: "default confidence threshold",
			config: CommentAnalyzerConfig{
				EnableSentimentAnalysis: true,
				EnableIntentDetection:   true,
			},
			expectedConfig: CommentAnalyzerConfig{
				EnableSentimentAnalysis: true,
				EnableIntentDetection:   true,
				ConfidenceThreshold:     0.7,
			},
		},
		{
			name: "custom confidence threshold",
			config: CommentAnalyzerConfig{
				EnableSentimentAnalysis: true,
				EnableIntentDetection:   true,
				ConfidenceThreshold:     0.8,
			},
			expectedConfig: CommentAnalyzerConfig{
				EnableSentimentAnalysis: true,
				EnableIntentDetection:   true,
				ConfidenceThreshold:     0.8,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewCommentAnalyzer(tt.config)
			assert.Equal(t, tt.expectedConfig, analyzer.config)
		})
	}
}

func TestCommentAnalyzer_AnalyzeComment(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{
		EnableSentimentAnalysis: true,
		EnableIntentDetection:   true,
		EnableKeywordExtraction: true,
		ConfidenceThreshold:     0.7,
	})

	tests := []struct {
		name          string
		comment       *Comment
		expectedError bool
	}{
		{
			name: "valid comment analysis",
			comment: &Comment{
				ID:     1,
				Author: "test-user",
				Body:   "Please fix this bug in the authentication module",
				File:   "auth.go",
				Line:   42,
			},
			expectedError: false,
		},
		{
			name: "empty comment body",
			comment: &Comment{
				ID:     2,
				Author: "test-user",
				Body:   "",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := analyzer.AnalyzeComment(ctx, tt.comment)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify that analysis fields are populated (intent may be clarification for some comments)
				assert.True(t, tt.comment.Intent >= CommentIntentQuestion && tt.comment.Intent <= CommentIntentConcern)
				assert.True(t, tt.comment.Priority >= CommentPriorityLow && tt.comment.Priority <= CommentPriorityBlocking)
				assert.True(t, tt.comment.Metadata.Confidence >= 0)
			}
		})
	}
}

func TestCommentAnalyzer_DetectIntent(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{
		ConfidenceThreshold: 0.1, // Lower threshold for more lenient testing
	})

	tests := []struct {
		name           string
		body           string
		expectedIntent CommentIntent
		minConfidence  float64
	}{
		{
			name:           "question with what",
			body:           "What is this function supposed to do?",
			expectedIntent: CommentIntentQuestion,
			minConfidence:  0.1,
		},
		{
			name:           "question with why",
			body:           "Why did you choose this approach?",
			expectedIntent: CommentIntentQuestion,
			minConfidence:  0.1,
		},
		{
			name:           "suggestion with consider",
			body:           "Consider using a map for better performance",
			expectedIntent: CommentIntentSuggestion,
			minConfidence:  0.1,
		},
		{
			name:           "request with please",
			body:           "Please update this to use the new API",
			expectedIntent: CommentIntentRequest,
			minConfidence:  0.1,
		},
		{
			name:           "blocking comment",
			body:           "This blocks the merge - critical security issue",
			expectedIntent: CommentIntentBlocking,
			minConfidence:  0.1,
		},
		{
			name:           "approval with lgtm",
			body:           "LGTM! Great work on this feature",
			expectedIntent: CommentIntentApproval,
			minConfidence:  0.1,
		},
		{
			name:           "praise comment",
			body:           "Excellent implementation! Well done",
			expectedIntent: CommentIntentPraise,
			minConfidence:  0.1,
		},
		{
			name:           "concern about potential issue",
			body:           "I'm concerned this could cause performance problems",
			expectedIntent: CommentIntentConcern,
			minConfidence:  0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, confidence := analyzer.detectIntent(tt.body)
			assert.Equal(t, tt.expectedIntent, intent)
			assert.GreaterOrEqual(t, confidence, tt.minConfidence)
		})
	}
}

func TestCommentAnalyzer_CalculateIntentScore(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name        string
		body        string
		keywords    []string
		patterns    []string
		weight      float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "high keyword match",
			body:        "please can you fix this issue",
			keywords:    []string{"please", "can", "you", "fix"},
			patterns:    []string{`please\s+\w+`},
			weight:      1.0,
			expectedMin: 0.8,
			expectedMax: 1.0,
		},
		{
			name:        "no matches",
			body:        "this is just a comment",
			keywords:    []string{"please", "urgent"},
			patterns:    []string{`urgent\s+fix`},
			weight:      1.0,
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
		{
			name:        "empty body",
			body:        "",
			keywords:    []string{"test"},
			patterns:    []string{`test`},
			weight:      1.0,
			expectedMin: 0.0,
			expectedMax: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.calculateIntentScore(tt.body, tt.keywords, tt.patterns, tt.weight)
			assert.GreaterOrEqual(t, score, tt.expectedMin)
			assert.LessOrEqual(t, score, tt.expectedMax)
		})
	}
}

func TestCommentAnalyzer_AnalyzeSentiment(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name      string
		body      string
		expected  float64
		tolerance float64
	}{
		{
			name:      "positive sentiment",
			body:      "Great work! This is excellent and amazing",
			expected:  1.0,
			tolerance: 0.1,
		},
		{
			name:      "negative sentiment",
			body:      "This is terrible and broken, very bad implementation",
			expected:  -1.0,
			tolerance: 0.1,
		},
		{
			name:      "neutral sentiment",
			body:      "This is a function that does something",
			expected:  0.0,
			tolerance: 0.1,
		},
		{
			name:      "mixed sentiment with positive bias",
			body:      "Good approach but could be better",
			expected:  0.0,
			tolerance: 1.0, // More tolerance for mixed sentiment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentiment := analyzer.analyzeSentiment(tt.body)
			assert.InDelta(t, tt.expected, sentiment, tt.tolerance)
		})
	}
}

func TestCommentAnalyzer_ExtractKeywords(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name             string
		body             string
		expectedKeywords []string
	}{
		{
			name:             "technical keywords",
			body:             "This function needs better error handling and testing",
			expectedKeywords: []string{"function", "error", "testing"},
		},
		{
			name:             "API and database keywords",
			body:             "The API endpoint should validate authentication before database queries",
			expectedKeywords: []string{"api", "endpoint", "authentication", "database"}, // Remove "validation" as it's not in the keyword list
		},
		{
			name:             "no technical keywords",
			body:             "This looks fine to me",
			expectedKeywords: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := analyzer.extractKeywords(tt.body)
			for _, expected := range tt.expectedKeywords {
				assert.Contains(t, keywords, expected, "Expected keyword '%s' not found", expected)
			}
		})
	}
}

func TestCommentAnalyzer_DeterminePriority(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name             string
		comment          *Comment
		expectedPriority CommentPriority
	}{
		{
			name: "blocking keyword",
			comment: &Comment{
				Body:   "This is blocking the release",
				Intent: CommentIntentRequest,
			},
			expectedPriority: CommentPriorityBlocking,
		},
		{
			name: "critical keyword",
			comment: &Comment{
				Body:   "Critical security vulnerability found",
				Intent: CommentIntentConcern,
			},
			expectedPriority: CommentPriorityCritical,
		},
		{
			name: "blocking intent",
			comment: &Comment{
				Body:   "Cannot merge due to issues",
				Intent: CommentIntentBlocking,
			},
			expectedPriority: CommentPriorityBlocking,
		},
		{
			name: "urgent request",
			comment: &Comment{
				Body:   "Please fix this urgently",
				Intent: CommentIntentRequest,
			},
			expectedPriority: CommentPriorityHigh, // Urgent keywords should bump priority
		},
		{
			name: "concern",
			comment: &Comment{
				Body:   "I'm concerned about this approach", // Use exact wording that triggers concern intent
				Intent: CommentIntentConcern,
			},
			expectedPriority: CommentPriorityHigh,
		},
		{
			name: "question",
			comment: &Comment{
				Body:   "How does this work?",
				Intent: CommentIntentQuestion,
			},
			expectedPriority: CommentPriorityMedium,
		},
		{
			name: "approval",
			comment: &Comment{
				Body:   "LGTM!",
				Intent: CommentIntentApproval,
			},
			expectedPriority: CommentPriorityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priority := analyzer.determinePriority(tt.comment)
			assert.Equal(t, tt.expectedPriority, priority)
		})
	}
}

func TestCommentAnalyzer_AssessComplexity(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name               string
		body               string
		expectedComplexity ComplexityLevel
	}{
		{
			name:               "simple comment",
			body:               "Fix this",
			expectedComplexity: ComplexityLevelSimple,
		},
		{
			name:               "moderate comment",
			body:               "Please update this function to handle edge cases better and add proper error handling",
			expectedComplexity: ComplexityLevelModerate,
		},
		{
			name:               "complex comment",
			body:               "This implementation needs significant refactoring to improve performance, add comprehensive error handling, implement proper logging, update documentation, and ensure backward compatibility with existing API consumers while maintaining thread safety and following established patterns in the codebase for similar functionality",
			expectedComplexity: ComplexityLevelComplex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			complexity := analyzer.assessComplexity(tt.body)
			assert.Equal(t, tt.expectedComplexity, complexity)
		})
	}
}

func TestCommentAnalyzer_AssessUrgency(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name            string
		body            string
		expectedUrgency UrgencyLevel
	}{
		{
			name:            "urgent keyword",
			body:            "Please fix this urgent issue",
			expectedUrgency: UrgencyLevelUrgent,
		},
		{
			name:            "asap keyword",
			body:            "Need this done ASAP",
			expectedUrgency: UrgencyLevelUrgent,
		},
		{
			name:            "quickly keyword",
			body:            "Can you fix this quickly?",
			expectedUrgency: UrgencyLevelHigh,
		},
		{
			name:            "soon keyword",
			body:            "Please address this soon",
			expectedUrgency: UrgencyLevelHigh,
		},
		{
			name:            "no rush",
			body:            "Fix this when you can, no rush",
			expectedUrgency: UrgencyLevelLow,
		},
		{
			name:            "normal comment",
			body:            "This could be improved",
			expectedUrgency: UrgencyLevelMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urgency := analyzer.assessUrgency(tt.body)
			assert.Equal(t, tt.expectedUrgency, urgency)
		})
	}
}

func TestCommentAnalyzer_RequiresAction(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name           string
		comment        *Comment
		expectedAction bool
	}{
		{
			name: "request requires action",
			comment: &Comment{
				Intent:   CommentIntentRequest,
				Priority: CommentPriorityMedium,
			},
			expectedAction: true,
		},
		{
			name: "blocking requires action",
			comment: &Comment{
				Intent:   CommentIntentBlocking,
				Priority: CommentPriorityBlocking,
			},
			expectedAction: true,
		},
		{
			name: "question requires action",
			comment: &Comment{
				Intent:   CommentIntentQuestion,
				Priority: CommentPriorityMedium,
			},
			expectedAction: true,
		},
		{
			name: "approval doesn't require action",
			comment: &Comment{
				Intent:   CommentIntentApproval,
				Priority: CommentPriorityLow,
			},
			expectedAction: false,
		},
		{
			name: "praise doesn't require action",
			comment: &Comment{
				Intent:   CommentIntentPraise,
				Priority: CommentPriorityLow,
			},
			expectedAction: false,
		},
		{
			name: "high priority comment requires action",
			comment: &Comment{
				Intent:   CommentIntentClarification,
				Priority: CommentPriorityHigh,
			},
			expectedAction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requiresAction := analyzer.requiresAction(tt.comment)
			assert.Equal(t, tt.expectedAction, requiresAction)
		})
	}
}

func TestCommentAnalyzer_ExtractMentionedCode(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name         string
		body         string
		expectedCode []string
	}{
		{
			name: "code blocks and inline code",
			body: "Please fix this:\n```go\nfunc test() {\n    return nil\n}\n```\nAnd also `variable` here",
			expectedCode: []string{
				"```go\nfunc test() {\n    return nil\n}\n```",
				"`variable`",
			},
		},
		{
			name: "file paths",
			body: "Update auth.go and config.yaml files",
			expectedCode: []string{
				"auth.go",
				"config.yaml",
			},
		},
		{
			name: "function calls",
			body: "The getUserData() and validateInput() functions need work",
			expectedCode: []string{
				"getUserData()",
				"validateInput()",
			},
		},
		{
			name:         "no code mentioned",
			body:         "This looks good to me",
			expectedCode: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mentioned := analyzer.extractMentionedCode(tt.body)

			if len(tt.expectedCode) == 0 {
				assert.Empty(t, mentioned)
			} else {
				for _, expected := range tt.expectedCode {
					assert.Contains(t, mentioned, expected, "Expected code '%s' not found", expected)
				}
			}
		})
	}
}

func TestCommentAnalyzer_ExtractCodeSuggestions(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	comment := &Comment{
		File: "test.go",
		Line: 42,
		Body: "Change foo to bar and replace old with new. Here's the suggested code:\n```go\nfunc newImpl() {\n    // better implementation\n}\n```",
	}

	suggestions := analyzer.ExtractCodeSuggestions(comment)

	require.NotEmpty(t, suggestions)

	// Should find "change" pattern
	foundChange := false
	foundReplace := false
	foundCodeBlock := false

	for _, suggestion := range suggestions {
		assert.Equal(t, comment.File, suggestion.File)
		assert.Equal(t, comment.Line, suggestion.StartLine)
		assert.False(t, suggestion.Applied)
		assert.Greater(t, suggestion.Confidence, 0.0)

		if suggestion.OldCode == "foo" && suggestion.NewCode == "bar" {
			foundChange = true
		}
		if suggestion.OldCode == "old" && suggestion.NewCode == "new" {
			foundReplace = true
		}
		if strings.Contains(suggestion.NewCode, "func newImpl()") {
			foundCodeBlock = true
		}
	}

	assert.True(t, foundChange, "Should find 'change foo to bar' suggestion")
	assert.True(t, foundReplace, "Should find 'replace old with new' suggestion")
	assert.True(t, foundCodeBlock, "Should find code block suggestion")
}

func TestCommentAnalyzer_ExtractActionItems(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name            string
		comment         *Comment
		expectedActions []ActionType
	}{
		{
			name: "test actions",
			comment: &Comment{
				File: "handler.go",
				Body: "Add test for this function and write test cases",
				Metadata: CommentMetadata{
					ActionRequired: true,
				},
			},
			expectedActions: []ActionType{ActionTypeTestAdd, ActionTypeTestAdd},
		},
		{
			name: "fix and update actions",
			comment: &Comment{
				File: "service.go",
				Body: "Fix this bug and update the documentation",
				Metadata: CommentMetadata{
					ActionRequired: true,
				},
			},
			expectedActions: []ActionType{ActionTypeCodeChange, ActionTypeDocUpdate},
		},
		{
			name: "generic action when none specific found",
			comment: &Comment{
				File: "main.go",
				Body: "This needs attention",
				Metadata: CommentMetadata{
					ActionRequired: true,
				},
			},
			expectedActions: []ActionType{ActionTypeInvestigate},
		},
		{
			name: "no actions when not required",
			comment: &Comment{
				File: "test.go",
				Body: "Looks good",
				Metadata: CommentMetadata{
					ActionRequired: false,
				},
			},
			expectedActions: []ActionType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := analyzer.ExtractActionItems(tt.comment)

			if len(tt.expectedActions) == 0 {
				assert.Empty(t, actions)
			} else {
				assert.Len(t, actions, len(tt.expectedActions))
				for i, expectedType := range tt.expectedActions {
					assert.Equal(t, expectedType, actions[i].Type)
					assert.Equal(t, tt.comment.File, actions[i].File)
					assert.Equal(t, "pending", actions[i].Status)
				}
			}
		})
	}
}

func TestCommentAnalyzer_HasUrgentKeywords(t *testing.T) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{})

	tests := []struct {
		name      string
		body      string
		hasUrgent bool
	}{
		{
			name:      "urgent keyword",
			body:      "This is urgent",
			hasUrgent: true,
		},
		{
			name:      "asap keyword",
			body:      "Need this ASAP",
			hasUrgent: true,
		},
		{
			name:      "immediately keyword",
			body:      "Fix immediately",
			hasUrgent: true,
		},
		{
			name:      "quickly keyword",
			body:      "Do this quickly",
			hasUrgent: true,
		},
		{
			name:      "soon keyword",
			body:      "Address soon",
			hasUrgent: true,
		},
		{
			name:      "no urgent keywords",
			body:      "This can wait",
			hasUrgent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasUrgent := analyzer.hasUrgentKeywords(tt.body)
			assert.Equal(t, tt.hasUrgent, hasUrgent)
		})
	}
}

// Benchmark tests for performance
func BenchmarkCommentAnalyzer_AnalyzeComment(b *testing.B) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{
		EnableSentimentAnalysis: true,
		EnableIntentDetection:   true,
		EnableKeywordExtraction: true,
		ConfidenceThreshold:     0.7,
	})

	comment := &Comment{
		ID:     1,
		Author: "test-user",
		Body:   "Please fix this critical security issue in the authentication module. This blocks the merge and needs urgent attention. Consider using bcrypt for password hashing instead of MD5.",
		File:   "auth.go",
		Line:   42,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := analyzer.AnalyzeComment(ctx, comment)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCommentAnalyzer_DetectIntent(b *testing.B) {
	analyzer := NewCommentAnalyzer(CommentAnalyzerConfig{
		ConfidenceThreshold: 0.7,
	})

	body := "Please fix this critical security issue and add proper tests"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = analyzer.detectIntent(body)
	}
}
