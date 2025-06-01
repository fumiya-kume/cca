package comments

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGitHubClient is a mock implementation of GitHubClient
type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) ListComments(ctx context.Context, prNumber int) ([]*Comment, error) {
	args := m.Called(ctx, prNumber)
	return args.Get(0).([]*Comment), args.Error(1)
}

func (m *MockGitHubClient) GetComment(ctx context.Context, commentID int) (*Comment, error) {
	args := m.Called(ctx, commentID)
	return args.Get(0).(*Comment), args.Error(1)
}

func (m *MockGitHubClient) CreateComment(ctx context.Context, prNumber int, body string) (*Comment, error) {
	args := m.Called(ctx, prNumber, body)
	return args.Get(0).(*Comment), args.Error(1)
}

func (m *MockGitHubClient) UpdateComment(ctx context.Context, commentID int, body string) (*Comment, error) {
	args := m.Called(ctx, commentID, body)
	return args.Get(0).(*Comment), args.Error(1)
}

func (m *MockGitHubClient) ReplyToComment(ctx context.Context, commentID int, body string) (*Comment, error) {
	args := m.Called(ctx, commentID, body)
	return args.Get(0).(*Comment), args.Error(1)
}

func (m *MockGitHubClient) ResolveComment(ctx context.Context, commentID int) error {
	args := m.Called(ctx, commentID)
	return args.Error(0)
}

func (m *MockGitHubClient) DismissReview(ctx context.Context, prNumber int, reviewID int, message string) error {
	args := m.Called(ctx, prNumber, reviewID, message)
	return args.Error(0)
}

func TestNewCommentHandler(t *testing.T) {
	tests := []struct {
		name           string
		config         CommentHandlerConfig
		expectedConfig CommentHandlerConfig
	}{
		{
			name: "default config values",
			config: CommentHandlerConfig{
				AutoRespond: true,
			},
			expectedConfig: CommentHandlerConfig{
				AutoRespond:       true,
				ResponseDelay:     time.Minute * 5,
				MaxResponseLength: 2000,
				RequiredApprovals: 1,
			},
		},
		{
			name: "custom config values",
			config: CommentHandlerConfig{
				AutoRespond:       false,
				ResponseDelay:     time.Minute * 10,
				MaxResponseLength: 1000,
				RequiredApprovals: 2,
			},
			expectedConfig: CommentHandlerConfig{
				AutoRespond:       false,
				ResponseDelay:     time.Minute * 10,
				MaxResponseLength: 1000,
				RequiredApprovals: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockGitHubClient{}
			handler := NewCommentHandler(tt.config, mockClient)

			assert.NotNil(t, handler)
			assert.NotNil(t, handler.analyzer)
			assert.NotNil(t, handler.responder)
			assert.NotNil(t, handler.monitor)
			assert.Equal(t, tt.expectedConfig.AutoRespond, handler.config.AutoRespond)
			assert.Equal(t, tt.expectedConfig.ResponseDelay, handler.config.ResponseDelay)
			assert.Equal(t, tt.expectedConfig.MaxResponseLength, handler.config.MaxResponseLength)
			assert.Equal(t, tt.expectedConfig.RequiredApprovals, handler.config.RequiredApprovals)
		})
	}
}

func TestCommentHandler_HandleComments(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:   false, // Disable auto-respond for testing
		IgnoreUsers:   []string{"bot-user"},
		ResponseDelay: time.Millisecond * 100,
	}

	tests := []struct {
		name        string
		prNumber    int
		comments    []*Comment
		expectError bool
	}{
		{
			name:     "handle valid comments",
			prNumber: 123,
			comments: []*Comment{
				{
					ID:        1,
					Author:    "reviewer1",
					Body:      "Please fix this bug",
					CreatedAt: time.Now(),
					Intent:    CommentIntentRequest,
					Priority:  CommentPriorityMedium,
					Status:    CommentStatusPending,
				},
				{
					ID:        2,
					Author:    "reviewer2",
					Body:      "Why did you choose this approach?",
					CreatedAt: time.Now(),
					Intent:    CommentIntentQuestion,
					Priority:  CommentPriorityMedium,
					Status:    CommentStatusPending,
				},
			},
			expectError: false,
		},
		{
			name:     "ignore bot comments",
			prNumber: 124,
			comments: []*Comment{
				{
					ID:        3,
					Author:    "bot-user",
					Body:      "This is a bot comment",
					CreatedAt: time.Now(),
				},
				{
					ID:        4,
					Author:    "github-bot",
					Body:      "Another bot comment",
					CreatedAt: time.Now(),
				},
			},
			expectError: false,
		},
		{
			name:        "handle empty comment list",
			prNumber:    125,
			comments:    []*Comment{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockGitHubClient{}
			handler := NewCommentHandler(config, mockClient)

			// Setup mock expectations
			mockClient.On("ListComments", mock.Anything, tt.prNumber).Return(tt.comments, nil)

			ctx := context.Background()
			err := handler.HandleComments(ctx, tt.prNumber, mockClient)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestCommentHandler_ShouldIgnoreComment(t *testing.T) {
	config := CommentHandlerConfig{
		IgnoreUsers: []string{"ignored-user", "bot-account"},
	}
	handler := NewCommentHandler(config, &MockGitHubClient{})

	tests := []struct {
		name         string
		comment      *Comment
		shouldIgnore bool
	}{
		{
			name: "ignore user in ignore list",
			comment: &Comment{
				Author:    "ignored-user",
				CreatedAt: time.Now(),
			},
			shouldIgnore: true,
		},
		{
			name: "ignore bot users",
			comment: &Comment{
				Author:    "github-bot",
				CreatedAt: time.Now(),
			},
			shouldIgnore: true,
		},
		{
			name: "ignore old comments",
			comment: &Comment{
				Author:    "regular-user",
				CreatedAt: time.Now().Add(-time.Hour * 24 * 8), // 8 days old
			},
			shouldIgnore: true,
		},
		{
			name: "don't ignore regular comments",
			comment: &Comment{
				Author:    "regular-user",
				CreatedAt: time.Now(),
			},
			shouldIgnore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldIgnore := handler.shouldIgnoreComment(tt.comment)
			assert.Equal(t, tt.shouldIgnore, shouldIgnore)
		})
	}
}

func TestCommentHandler_HandleQuestion(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:   true,
		ResponseDelay: time.Millisecond * 10,
	}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	comment := &Comment{
		ID:     1,
		Author: "reviewer",
		Body:   "How does this function work?",
		Intent: CommentIntentQuestion,
	}

	// Setup mock expectations
	mockClient.On("ReplyToComment", mock.Anything, comment.ID, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := handler.handleQuestion(ctx, comment, mockClient)

	assert.NoError(t, err)
	assert.NotEmpty(t, comment.Responses)
	mockClient.AssertExpectations(t)
}

func TestCommentHandler_HandleSuggestion(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:   true,
		ResponseDelay: time.Millisecond * 10,
	}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	comment := &Comment{
		ID:     2,
		Author: "reviewer",
		Body:   "Consider using a map instead of a slice for better performance",
		Intent: CommentIntentSuggestion,
		File:   "handler.go",
		Line:   42,
	}

	// Setup mock expectations
	mockClient.On("ReplyToComment", mock.Anything, comment.ID, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := handler.handleSuggestion(ctx, comment, mockClient)

	assert.NoError(t, err)
	assert.NotEmpty(t, comment.Responses)
	mockClient.AssertExpectations(t)
}

func TestCommentHandler_HandleBlockingComment(t *testing.T) {
	config := CommentHandlerConfig{}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	comment := &Comment{
		ID:       3,
		Author:   "security-reviewer",
		Body:     "This introduces a critical security vulnerability that blocks the merge",
		Intent:   CommentIntentBlocking,
		Priority: CommentPriorityBlocking,
	}

	// Setup mock expectations
	mockClient.On("ReplyToComment", mock.Anything, comment.ID, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := handler.handleBlockingComment(ctx, comment, mockClient)

	assert.NoError(t, err)
	assert.Equal(t, CommentPriorityBlocking, comment.Priority)
	assert.NotEmpty(t, comment.Responses)
	mockClient.AssertExpectations(t)
}

func TestCommentHandler_HandleApproval(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:   true,
		ResponseDelay: time.Millisecond * 10,
	}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	comment := &Comment{
		ID:     4,
		Author: "approver",
		Body:   "LGTM! Great work",
		Intent: CommentIntentApproval,
	}

	// Setup mock expectations
	mockClient.On("ReplyToComment", mock.Anything, comment.ID, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := handler.handleApproval(ctx, comment, mockClient)

	assert.NoError(t, err)
	assert.Equal(t, CommentStatusResolved, comment.Status)
	assert.NotNil(t, comment.ResolvedAt)
	mockClient.AssertExpectations(t)
}

func TestCommentHandler_HandleRequest(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:   true,
		ResponseDelay: time.Millisecond * 10,
	}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	comment := &Comment{
		ID:     5,
		Author: "reviewer",
		Body:   "Please add tests for this function",
		Intent: CommentIntentRequest,
		File:   "service.go",
	}

	// Setup mock expectations
	mockClient.On("ReplyToComment", mock.Anything, comment.ID, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := handler.handleRequest(ctx, comment, mockClient)

	assert.NoError(t, err)
	assert.Equal(t, CommentStatusResolved, comment.Status)
	assert.NotNil(t, comment.ResolvedAt)
	mockClient.AssertExpectations(t)
}

func TestCommentHandler_HandlePraise(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:   true,
		ResponseDelay: time.Millisecond * 10,
	}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	comment := &Comment{
		ID:     6,
		Author: "reviewer",
		Body:   "Excellent implementation! Well done",
		Intent: CommentIntentPraise,
	}

	// Setup mock expectations
	mockClient.On("ReplyToComment", mock.Anything, comment.ID, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := handler.handlePraise(ctx, comment, mockClient)

	assert.NoError(t, err)
	assert.Equal(t, CommentStatusResolved, comment.Status)
	assert.NotNil(t, comment.ResolvedAt)
	mockClient.AssertExpectations(t)
}

func TestCommentHandler_HandleGenericComment(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:   true,
		ResponseDelay: time.Millisecond * 10,
	}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	comment := &Comment{
		ID:     7,
		Author: "reviewer",
		Body:   "This is unclear",
		Intent: CommentIntentClarification,
	}

	// Setup mock expectations
	mockClient.On("ReplyToComment", mock.Anything, comment.ID, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := handler.handleGenericComment(ctx, comment, mockClient)

	assert.NoError(t, err)
	assert.NotEmpty(t, comment.Responses)
	mockClient.AssertExpectations(t)
}

func TestCommentHandler_MatchesEscalationRule(t *testing.T) {
	handler := NewCommentHandler(CommentHandlerConfig{}, &MockGitHubClient{})

	tests := []struct {
		name     string
		comment  *Comment
		rule     EscalationRule
		expected bool
	}{
		{
			name: "blocking rule matches",
			comment: &Comment{
				Priority: CommentPriorityBlocking,
			},
			rule: EscalationRule{
				Condition: "blocking",
			},
			expected: true,
		},
		{
			name: "critical rule matches",
			comment: &Comment{
				Priority: CommentPriorityCritical,
			},
			rule: EscalationRule{
				Condition: "critical",
			},
			expected: true,
		},
		{
			name: "unresolved time rule matches",
			comment: &Comment{
				CreatedAt: time.Now().Add(-time.Hour * 2),
			},
			rule: EscalationRule{
				Condition: "unresolved_time",
				Delay:     time.Hour,
			},
			expected: true,
		},
		{
			name: "unresolved time rule doesn't match",
			comment: &Comment{
				CreatedAt: time.Now().Add(-time.Minute * 30),
			},
			rule: EscalationRule{
				Condition: "unresolved_time",
				Delay:     time.Hour,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := handler.matchesEscalationRule(tt.comment, tt.rule)
			assert.Equal(t, tt.expected, matches)
		})
	}
}

// Test enum string methods
func TestCommentType_String(t *testing.T) {
	tests := []struct {
		commentType CommentType
		expected    string
	}{
		{CommentTypeReview, "review"},
		{CommentTypeInline, "inline"},
		{CommentTypeGeneral, "general"},
		{CommentTypeApproval, "approval"},
		{CommentTypeDismissal, "dismissal"},
		{CommentTypeRequest, "request"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.commentType.String())
		})
	}
}

func TestCommentIntent_String(t *testing.T) {
	tests := []struct {
		intent   CommentIntent
		expected string
	}{
		{CommentIntentQuestion, "question"},
		{CommentIntentSuggestion, "suggestion"},
		{CommentIntentRequest, "request"},
		{CommentIntentApproval, "approval"},
		{CommentIntentBlocking, "blocking"},
		{CommentIntentPraise, "praise"},
		{CommentIntentClarification, "clarification"},
		{CommentIntentConcern, "concern"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.intent.String())
		})
	}
}

func TestCommentPriority_String(t *testing.T) {
	tests := []struct {
		priority CommentPriority
		expected string
	}{
		{CommentPriorityLow, "low"},
		{CommentPriorityMedium, "medium"},
		{CommentPriorityHigh, "high"},
		{CommentPriorityCritical, "critical"},
		{CommentPriorityBlocking, "blocking"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.priority.String())
		})
	}
}

func TestCommentStatus_String(t *testing.T) {
	tests := []struct {
		status   CommentStatus
		expected string
	}{
		{CommentStatusPending, "pending"},
		{CommentStatusAcknowledged, "acknowledged"},
		{CommentStatusInProgress, "in_progress"},
		{CommentStatusResolved, "resolved"},
		{CommentStatusDismissed, "dismissed"},
		{CommentStatusEscalated, "escalated"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

// Integration test for full comment handling workflow
func TestCommentHandler_IntegrationWorkflow(t *testing.T) {
	config := CommentHandlerConfig{
		AutoRespond:       true,
		ResponseDelay:     time.Millisecond * 10,
		MaxResponseLength: 1000,
		EnableMentions:    true,
		RequiredApprovals: 2,
	}

	mockClient := &MockGitHubClient{}
	handler := NewCommentHandler(config, mockClient)

	// Create test comments with different intents
	comments := []*Comment{
		{
			ID:        1,
			Author:    "reviewer1",
			Body:      "Please fix this security issue - it's blocking",
			CreatedAt: time.Now(),
			File:      "auth.go",
			Line:      25,
		},
		{
			ID:        2,
			Author:    "reviewer2",
			Body:      "Why did you choose this implementation?",
			CreatedAt: time.Now(),
			File:      "service.go",
			Line:      42,
		},
		{
			ID:        3,
			Author:    "approver",
			Body:      "LGTM! Great work",
			CreatedAt: time.Now(),
		},
	}

	// Setup mock expectations for listing comments
	mockClient.On("ListComments", mock.Anything, 123).Return(comments, nil)

	// Setup mock expectations for replies
	mockClient.On("ReplyToComment", mock.Anything, mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(&Comment{}, nil).Times(3)

	ctx := context.Background()
	err := handler.HandleComments(ctx, 123, mockClient)

	require.NoError(t, err)

	// Verify comments were analyzed and have proper intent/priority
	for _, comment := range comments {
		assert.True(t, comment.Intent >= CommentIntentQuestion && comment.Intent <= CommentIntentConcern, "Comment should have valid intent")
		assert.True(t, comment.Metadata.Confidence >= 0, "Comment should have confidence score")
	}

	mockClient.AssertExpectations(t)
}
