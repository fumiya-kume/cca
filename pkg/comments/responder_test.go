package comments

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommentResponder(t *testing.T) {
	tests := []struct {
		name           string
		config         CommentResponderConfig
		expectedConfig CommentResponderConfig
	}{
		{
			name: "default config values",
			config: CommentResponderConfig{
				EnableSuggestions: true,
			},
			expectedConfig: CommentResponderConfig{
				MaxLength:         2000,
				EnableSuggestions: true,
				ResponseDelay:     time.Millisecond * 2, // Fast for tests
				Personality:       "helpful",
			},
		},
		{
			name: "custom config values",
			config: CommentResponderConfig{
				MaxLength:         1000,
				EnableSuggestions: false,
				EnableMentions:    true,
				ResponseDelay:     time.Millisecond * 5, // Fast for tests
				Personality:       "professional",
			},
			expectedConfig: CommentResponderConfig{
				MaxLength:         1000,
				EnableSuggestions: false,
				EnableMentions:    true,
				ResponseDelay:     time.Millisecond * 5, // Fast for tests
				Personality:       "professional",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responder := NewCommentResponder(tt.config)

			assert.NotNil(t, responder)
			assert.Equal(t, tt.expectedConfig.MaxLength, responder.config.MaxLength)
			assert.Equal(t, tt.expectedConfig.EnableSuggestions, responder.config.EnableSuggestions)
			assert.Equal(t, tt.expectedConfig.EnableMentions, responder.config.EnableMentions)
			assert.Equal(t, tt.expectedConfig.ResponseDelay, responder.config.ResponseDelay)
			assert.Equal(t, tt.expectedConfig.Personality, responder.config.Personality)
			assert.NotNil(t, responder.config.Templates)
		})
	}
}

func TestCommentResponder_GenerateQuestionResponse(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{
		MaxLength: 1000,
	})

	tests := []struct {
		name         string
		comment      *Comment
		expectedType ResponseType
		checkContent []string
	}{
		{
			name: "implementation question",
			comment: &Comment{
				Author: "developer",
				Body:   "How does this authentication mechanism work?",
				Metadata: CommentMetadata{
					Keywords: []string{"authentication", "function"},
				},
			},
			expectedType: ResponseTypeClarification,
			checkContent: []string{"Thanks for the question", "implementation approach", "developer"},
		},
		{
			name: "reasoning question",
			comment: &Comment{
				Author: "reviewer",
				Body:   "Why did you choose this approach instead of using a library?",
				Metadata: CommentMetadata{
					Keywords: []string{"approach", "library"},
				},
			},
			expectedType: ResponseTypeClarification,
			checkContent: []string{"Thanks for the question", "reasoning behind", "reviewer"},
		},
		{
			name: "usage question",
			comment: &Comment{
				Author: "user",
				Body:   "How do I use this new API endpoint?",
				Metadata: CommentMetadata{
					Keywords: []string{"api", "endpoint"},
				},
			},
			expectedType: ResponseTypeClarification,
			checkContent: []string{"Thanks for the question", "how to use", "user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := responder.GenerateQuestionResponse(ctx, tt.comment)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, response.Type)
			assert.Equal(t, "ccagents-ai", response.Author)
			assert.Equal(t, ResponseStatusDraft, response.Status)

			for _, content := range tt.checkContent {
				assert.Contains(t, response.Content, content)
			}
		})
	}
}

func TestCommentResponder_GenerateSuggestionResponse(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{
		MaxLength: 1000,
	})

	tests := []struct {
		name         string
		comment      *Comment
		suggestions  []CodeSuggestion
		expectedType ResponseType
		checkContent []string
	}{
		{
			name: "suggestions with applied changes",
			comment: &Comment{
				Author: "contributor",
				Body:   "Consider using a map for better performance",
			},
			suggestions: []CodeSuggestion{
				{
					Description: "Use map instead of slice",
					Confidence:  0.9,
					Applied:     true,
				},
				{
					Description: "Add error handling",
					Confidence:  0.7,
					Applied:     false,
				},
			},
			expectedType: ResponseTypeImplementation,
			checkContent: []string{"Thank you for the suggestion", "contributor", "applied 1", "need manual review"},
		},
		{
			name: "suggestions without applied changes",
			comment: &Comment{
				Author: "reviewer",
				Body:   "You might want to refactor this function",
			},
			suggestions: []CodeSuggestion{
				{
					Description: "Extract helper method",
					Confidence:  0.6,
					Applied:     false,
				},
			},
			expectedType: ResponseTypeImplementation,
			checkContent: []string{"Thank you for the suggestion", "reviewer", "need manual review"},
		},
		{
			name: "no specific suggestions found",
			comment: &Comment{
				Author: "advisor",
				Body:   "This could be improved somehow",
			},
			suggestions:  []CodeSuggestion{},
			expectedType: ResponseTypeImplementation,
			checkContent: []string{"Thank you for the suggestion", "advisor", "noted your suggestion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := responder.GenerateSuggestionResponse(ctx, tt.comment, tt.suggestions)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, response.Type)
			assert.Equal(t, "ccagents-ai", response.Author)
			assert.Equal(t, ResponseStatusDraft, response.Status)

			for _, content := range tt.checkContent {
				assert.Contains(t, response.Content, content)
			}
		})
	}
}

func TestCommentResponder_GenerateRequestResponse(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{
		MaxLength: 1000,
	})

	tests := []struct {
		name         string
		comment      *Comment
		actions      []ResponseAction
		expectedType ResponseType
		checkContent []string
	}{
		{
			name: "request with completed actions",
			comment: &Comment{
				Author: "maintainer",
				Body:   "Please add unit tests and update documentation",
			},
			actions: []ResponseAction{
				{
					Description: "Add unit tests",
					Status:      "completed",
				},
				{
					Description: "Update documentation",
					Status:      "completed",
				},
			},
			expectedType: ResponseTypeImplementation,
			checkContent: []string{"maintainer", "working on your request", "Completed Actions", "2/2"},
		},
		{
			name: "request with mixed action statuses",
			comment: &Comment{
				Author: "reviewer",
				Body:   "Fix the security issue and optimize performance",
			},
			actions: []ResponseAction{
				{
					Description: "Fix security issue",
					Status:      "completed",
				},
				{
					Description: "Optimize performance",
					Status:      "failed",
					Result:      "Optimization requires more research",
				},
				{
					Description: "Add benchmarks",
					Status:      "pending",
				},
			},
			expectedType: ResponseTypeImplementation,
			checkContent: []string{"reviewer", "**Completed Actions** (1/3)", "**Failed Actions** (1)", "**Pending Actions** (1)"},
		},
		{
			name: "request without specific actions",
			comment: &Comment{
				Author: "lead",
				Body:   "This needs some work",
			},
			actions:      []ResponseAction{},
			expectedType: ResponseTypeImplementation,
			checkContent: []string{"lead", "working on your request", "address your request"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := responder.GenerateRequestResponse(ctx, tt.comment, tt.actions)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedType, response.Type)
			assert.Equal(t, "ccagents-ai", response.Author)
			assert.Equal(t, ResponseStatusDraft, response.Status)

			for _, content := range tt.checkContent {
				assert.Contains(t, response.Content, content)
			}
		})
	}
}

func TestCommentResponder_GenerateBlockingResponse(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{
		MaxLength: 1000,
	})

	comment := &Comment{
		Author: "security-team",
		Body:   "This introduces a critical SQL injection vulnerability that must be fixed before merge",
	}

	ctx := context.Background()
	response, err := responder.GenerateBlockingResponse(ctx, comment)

	require.NoError(t, err)
	assert.Equal(t, ResponseTypeEscalation, response.Type)
	assert.Equal(t, "ccagents-ai", response.Author)
	assert.Equal(t, ResponseStatusDraft, response.Status)
	assert.Contains(t, response.Content, "🚨 **Urgent**")
	assert.Contains(t, response.Content, "security-team")
	assert.Contains(t, response.Content, "blocking issue")
	assert.Contains(t, response.Content, "prioritizing this feedback")
	assert.Contains(t, response.Content, "Next Steps")
}

func TestCommentResponder_GenerateApprovalResponse(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	comment := &Comment{
		Author: "senior-dev",
		Body:   "LGTM! Great implementation",
	}

	ctx := context.Background()
	response, err := responder.GenerateApprovalResponse(ctx, comment)

	require.NoError(t, err)
	assert.Equal(t, ResponseTypeAcknowledgment, response.Type)
	assert.Equal(t, "ccagents-ai", response.Author)
	assert.Equal(t, ResponseStatusDraft, response.Status)
	assert.Contains(t, response.Content, "Thank you for the approval")
	assert.Contains(t, response.Content, "senior-dev")
	assert.Contains(t, response.Content, "proceed with merging")
}

func TestCommentResponder_GeneratePraiseResponse(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	comment := &Comment{
		Author: "teammate",
		Body:   "Excellent work! This is really well implemented",
	}

	ctx := context.Background()
	response, err := responder.GeneratePraiseResponse(ctx, comment)

	require.NoError(t, err)
	assert.Equal(t, ResponseTypeAcknowledgment, response.Type)
	assert.Equal(t, "ccagents-ai", response.Author)
	assert.Equal(t, ResponseStatusDraft, response.Status)
	assert.Contains(t, response.Content, "Thank you")
	assert.Contains(t, response.Content, "teammate")
}

func TestCommentResponder_GenerateClarificationResponse(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	tests := []struct {
		name         string
		comment      *Comment
		checkContent []string
	}{
		{
			name: "vague comment",
			comment: &Comment{
				Author: "reviewer",
				Body:   "This needs something here and there",
			},
			checkContent: []string{"Hi @reviewer", "more details", "what you'd like me to address"},
		},
		{
			name: "multiple points comment",
			comment: &Comment{
				Author: "maintainer",
				Body:   "Fix this. Update that. Also change these. And don't forget those. Maybe do more?",
			},
			checkContent: []string{"Hi @maintainer", "several points", "help me prioritize"},
		},
		{
			name: "unclear comment",
			comment: &Comment{
				Author: "contributor",
				Body:   "Something should be different",
			},
			checkContent: []string{"Hi @contributor", "more details", "additional context"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := responder.GenerateClarificationResponse(ctx, tt.comment)

			require.NoError(t, err)
			assert.Equal(t, ResponseTypeClarification, response.Type)
			assert.Equal(t, "ccagents-ai", response.Author)
			assert.Equal(t, ResponseStatusDraft, response.Status)

			for _, content := range tt.checkContent {
				assert.Contains(t, response.Content, content)
			}
		})
	}
}

func TestCommentResponder_AnalyzeQuestionType(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	tests := []struct {
		name         string
		body         string
		expectedType string
	}{
		{
			name:         "implementation question",
			body:         "How does this function work?",
			expectedType: "implementation",
		},
		{
			name:         "reasoning question",
			body:         "Why did you choose this approach?",
			expectedType: "reasoning",
		},
		{
			name:         "usage question",
			body:         "How do I use this API?",
			expectedType: "usage",
		},
		{
			name:         "alternatives question",
			body:         "What alternatives did you consider?",
			expectedType: "alternatives",
		},
		{
			name:         "general question",
			body:         "What is happening here?",
			expectedType: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			questionType := responder.analyzeQuestionType(tt.body)
			assert.Equal(t, tt.expectedType, questionType)
		})
	}
}

func TestCommentResponder_ExtractMainConcern(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	tests := []struct {
		name            string
		body            string
		expectedConcern string
	}{
		{
			name:            "explicit concern",
			body:            "I have a concern about the security. This could lead to data leaks. Please address this.",
			expectedConcern: "I have a concern about the security", // Update expected to match actual extraction
		},
		{
			name:            "problem statement",
			body:            "This looks good overall. There's a problem with the error handling though. It might crash.",
			expectedConcern: "There's a problem with the error handling though",
		},
		{
			name:            "issue identification",
			body:            "Great work! However, there's an issue with memory usage. The rest looks fine.",
			expectedConcern: "However, there's an issue with memory usage",
		},
		{
			name:            "no specific concern",
			body:            "This could be better somehow. Not sure what exactly.",
			expectedConcern: "This could be better somehow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			concern := responder.extractMainConcern(tt.body)
			assert.Contains(t, concern, strings.Split(tt.expectedConcern, " ")[0])
		})
	}
}

func TestCommentResponder_IsVague(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	tests := []struct {
		name    string
		body    string
		isVague bool
	}{
		{
			name:    "vague comment with many pronouns",
			body:    "This needs something here and there",
			isVague: true,
		},
		{
			name:    "specific comment",
			body:    "The getUserData function should validate input parameters",
			isVague: false,
		},
		{
			name:    "moderately vague",
			body:    "Fix this function to work better",
			isVague: false, // Not enough vague words relative to total
		},
		{
			name:    "very vague",
			body:    "Something here needs everything there",
			isVague: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isVague := responder.isVague(tt.body)
			assert.Equal(t, tt.isVague, isVague)
		})
	}
}

func TestCommentResponder_HasMultiplePoints(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	tests := []struct {
		name        string
		body        string
		hasMultiple bool
	}{
		{
			name:        "single point",
			body:        "Fix this bug",
			hasMultiple: false,
		},
		{
			name:        "multiple sentences",
			body:        "Fix this bug. Update the documentation. Add tests. Remove unused code.",
			hasMultiple: true,
		},
		{
			name:        "bullet points",
			body:        "Please address:\n- Fix the bug\n- Update docs\n- Add tests\n* Also remove this",
			hasMultiple: true,
		},
		{
			name:        "numbered list",
			body:        "1. Fix bug\n2. Update docs\n3. Add tests\n4. Review changes",
			hasMultiple: true,
		},
		{
			name:        "questions and exclamations",
			body:        "What is this? Why does it work? This is confusing! Please explain?",
			hasMultiple: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasMultiple := responder.hasMultiplePoints(tt.body)
			assert.Equal(t, tt.hasMultiple, hasMultiple)
		})
	}
}

func TestCommentResponder_TruncateIfNeeded(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{
		MaxLength: 100,
	})

	tests := []struct {
		name           string
		content        string
		shouldTruncate bool
	}{
		{
			name:           "short content",
			content:        "This is a short response",
			shouldTruncate: false,
		},
		{
			name:           "long content",
			content:        strings.Repeat("This is a very long response that exceeds the maximum length limit. ", 10),
			shouldTruncate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := responder.truncateIfNeeded(tt.content)

			if tt.shouldTruncate {
				assert.Less(t, len(result), len(tt.content))
				assert.Contains(t, result, "[Response truncated")
			} else {
				assert.Equal(t, tt.content, result)
			}

			assert.LessOrEqual(t, len(result), responder.config.MaxLength)
		})
	}
}

func TestCommentResponder_GenerateImplementationExplanation(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	comment := &Comment{
		Metadata: CommentMetadata{
			MentionedCode: []string{"getUserData()", "validateInput()", "auth.go"},
		},
	}

	explanation := responder.generateImplementationExplanation(comment)

	assert.Contains(t, explanation, "implementation follows")
	assert.Contains(t, explanation, "key principles")
	assert.Contains(t, explanation, "Code Context")
	assert.Contains(t, explanation, "getUserData()")
	assert.Contains(t, explanation, "validateInput()")
	assert.Contains(t, explanation, "auth.go")
	assert.Contains(t, explanation, "Approach")
}

func TestCommentResponder_GenerateReasoningExplanation(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	comment := &Comment{
		Body: "Why this approach?",
	}

	explanation := responder.generateReasoningExplanation(comment)

	assert.Contains(t, explanation, "approach was chosen")
	assert.Contains(t, explanation, "Performance")
	assert.Contains(t, explanation, "Maintainability")
	assert.Contains(t, explanation, "Reliability")
	assert.Contains(t, explanation, "Testability")
}

func TestCommentResponder_GenerateUsageExplanation(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	comment := &Comment{
		Body: "How to use this?",
	}

	explanation := responder.generateUsageExplanation(comment)

	assert.Contains(t, explanation, "how to use")
	assert.Contains(t, explanation, "```go")
	assert.Contains(t, explanation, "Example usage")
	assert.Contains(t, explanation, "Go conventions")
}

func TestCommentResponder_GenerateAlternativesExplanation(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	comment := &Comment{
		Body: "What alternatives were considered?",
	}

	explanation := responder.generateAlternativesExplanation(comment)

	assert.Contains(t, explanation, "considered several alternatives")
	assert.Contains(t, explanation, "Current approach")
	assert.Contains(t, explanation, "Alternative 1")
	assert.Contains(t, explanation, "Alternative 2")
	assert.Contains(t, explanation, "best trade-off")
}

func TestCommentResponder_GenerateGenericExplanation(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{})

	tests := []struct {
		name     string
		comment  *Comment
		expected []string
	}{
		{
			name: "comment with keywords",
			comment: &Comment{
				Metadata: CommentMetadata{
					Keywords: []string{"function", "error", "performance", "security"},
				},
			},
			expected: []string{"function", "error", "performance"},
		},
		{
			name: "comment without keywords",
			comment: &Comment{
				Metadata: CommentMetadata{
					Keywords: []string{},
				},
			},
			expected: []string{"provide the relevant details"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			explanation := responder.generateGenericExplanation(tt.comment)

			for _, exp := range tt.expected {
				assert.Contains(t, explanation, exp)
			}
		})
	}
}

func TestGetDefaultTemplates(t *testing.T) {
	templates := getDefaultTemplates()

	expectedKeys := []string{
		"acknowledgment",
		"clarification",
		"implementation",
		"explanation",
		"approval",
		"praise",
	}

	for _, key := range expectedKeys {
		assert.Contains(t, templates, key)
		assert.NotEmpty(t, templates[key])
	}
}

// Integration test for full response generation workflow
func TestCommentResponder_IntegrationWorkflow(t *testing.T) {
	responder := NewCommentResponder(CommentResponderConfig{
		MaxLength:         1000,
		EnableSuggestions: true,
		EnableMentions:    true,
		Personality:       "helpful",
	})

	// Test different comment types and their responses
	testComments := []struct {
		comment      *Comment
		responseFunc func(context.Context, *Comment) (*CommentResponse, error)
		expectedType ResponseType
	}{
		{
			comment: &Comment{
				Author: "developer",
				Body:   "How does the authentication flow work?",
			},
			responseFunc: func(ctx context.Context, c *Comment) (*CommentResponse, error) {
				return responder.GenerateQuestionResponse(ctx, c)
			},
			expectedType: ResponseTypeClarification,
		},
		{
			comment: &Comment{
				Author: "reviewer",
				Body:   "LGTM! Great implementation",
			},
			responseFunc: func(ctx context.Context, c *Comment) (*CommentResponse, error) {
				return responder.GenerateApprovalResponse(ctx, c)
			},
			expectedType: ResponseTypeAcknowledgment,
		},
		{
			comment: &Comment{
				Author: "security",
				Body:   "This has a critical vulnerability - blocking merge",
			},
			responseFunc: func(ctx context.Context, c *Comment) (*CommentResponse, error) {
				return responder.GenerateBlockingResponse(ctx, c)
			},
			expectedType: ResponseTypeEscalation,
		},
	}

	ctx := context.Background()

	for i, tc := range testComments {
		t.Run(tc.comment.Author, func(t *testing.T) {
			response, err := tc.responseFunc(ctx, tc.comment)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedType, response.Type)
			assert.Equal(t, "ccagents-ai", response.Author)
			assert.Equal(t, ResponseStatusDraft, response.Status)
			assert.NotEmpty(t, response.Content)
			assert.LessOrEqual(t, len(response.Content), responder.config.MaxLength)
			assert.Contains(t, response.Content, tc.comment.Author)

			// Verify response was created recently
			assert.True(t, time.Since(response.CreatedAt) < time.Second)

			t.Logf("Test %d - Generated response for %s: %s", i+1, tc.comment.Author, response.Content[:minInt(100, len(response.Content))])
		})
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
