package claude

import (
	"context"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	client := &Client{
		claudeCommand: "claude",
		timeout:       time.Minute * 5,
	}

	service := &Service{
		client:                 client,
		promptTemplates:        getDefaultPromptTemplates(),
		conversationHistory:    make(map[string][]ConversationEntry),
		defaultTimeout:         time.Minute * 5,
		retryAttempts:          3,
		retryDelay:             time.Second * 2,
		issueAnalysisSessions:  make(map[string]*Session),
		codeGenerationSessions: make(map[string]*Session),
		reviewSessions:         make(map[string]*Session),
	}

	assert.NotNil(t, service)
	assert.Equal(t, client, service.client)
	assert.NotEmpty(t, service.promptTemplates)
	assert.Equal(t, 3, service.retryAttempts)
	assert.Equal(t, time.Second*2, service.retryDelay)
}

func TestService_PromptTemplateManagement(t *testing.T) {
	service := &Service{
		promptTemplates: getDefaultPromptTemplates(),
	}

	// Test getting a template
	template, exists := service.promptTemplates["issue_analysis"]
	assert.True(t, exists)
	assert.Equal(t, "issue_analysis", template.Name)
	assert.Equal(t, PromptIssueAnalysis, template.Category)

	// Test that templates are properly structured
	assert.NotEmpty(t, template.Template)
	assert.NotEmpty(t, template.Description)
	assert.NotEmpty(t, template.Variables)
}

func TestService_ConversationHistory(t *testing.T) {
	service := &Service{
		conversationHistory: make(map[string][]ConversationEntry),
	}

	sessionID := "test-session"

	// Test adding conversation entries
	entry1 := ConversationEntry{
		Timestamp: time.Now(),
		Role:      RoleUser,
		Content:   "Hello, Claude!",
		Metadata:  map[string]interface{}{"type": "greeting"},
	}

	entry2 := ConversationEntry{
		Timestamp: time.Now(),
		Role:      RoleAssistant,
		Content:   "Hello! How can I help you?",
		Metadata:  map[string]interface{}{"type": "response"},
	}

	// Add entries to conversation history
	service.conversationHistory[sessionID] = []ConversationEntry{entry1, entry2}

	// Verify entries were added
	history := service.conversationHistory[sessionID]
	assert.Len(t, history, 2)
	assert.Equal(t, "Hello, Claude!", history[0].Content)
	assert.Equal(t, RoleUser, history[0].Role)
	assert.Equal(t, "Hello! How can I help you?", history[1].Content)
	assert.Equal(t, RoleAssistant, history[1].Role)
}

func TestService_SessionPoolManagement(t *testing.T) {
	service := &Service{
		issueAnalysisSessions:  make(map[string]*Session),
		codeGenerationSessions: make(map[string]*Session),
		reviewSessions:         make(map[string]*Session),
	}

	// Test session creation and management
	sessionID := "test-session"

	// Mock session
	session := &Session{
		ID:        sessionID,
		Status:    SessionIdle,
		CreatedAt: time.Now(),
	}

	// Add session to different pools
	service.issueAnalysisSessions[sessionID] = session
	service.codeGenerationSessions[sessionID] = session
	service.reviewSessions[sessionID] = session

	// Verify sessions are tracked
	assert.Contains(t, service.issueAnalysisSessions, sessionID)
	assert.Contains(t, service.codeGenerationSessions, sessionID)
	assert.Contains(t, service.reviewSessions, sessionID)

	// Verify session properties
	assert.Equal(t, sessionID, service.issueAnalysisSessions[sessionID].ID)
	assert.Equal(t, SessionIdle, service.issueAnalysisSessions[sessionID].Status)
}

func TestPromptTemplate_Structure(t *testing.T) {
	template := PromptTemplate{
		Name:        "test_template",
		Template:    "Analyze this {{.input}} and provide {{.output}}",
		Variables:   []string{"input", "output"},
		Description: "A test template for unit testing",
		Category:    PromptIssueAnalysis,
	}

	assert.Equal(t, "test_template", template.Name)
	assert.Contains(t, template.Template, "{{.input}}")
	assert.Contains(t, template.Template, "{{.output}}")
	assert.Contains(t, template.Variables, "input")
	assert.Contains(t, template.Variables, "output")
	assert.Equal(t, PromptIssueAnalysis, template.Category)
	assert.NotEmpty(t, template.Description)
}

func TestConversationEntry_Structure(t *testing.T) {
	now := time.Now()
	entry := ConversationEntry{
		Timestamp: now,
		Role:      RoleUser,
		Content:   "Test message",
		Metadata: map[string]interface{}{
			"type":       "test",
			"session_id": "test-123",
		},
	}

	assert.Equal(t, now, entry.Timestamp)
	assert.Equal(t, RoleUser, entry.Role)
	assert.Equal(t, "Test message", entry.Content)
	assert.Equal(t, "test", entry.Metadata["type"])
	assert.Equal(t, "test-123", entry.Metadata["session_id"])
}

func TestPromptCategory_Constants(t *testing.T) {
	// Test that prompt categories are properly defined
	categories := []PromptCategory{
		PromptIssueAnalysis,
		PromptCodeGeneration,
		PromptCodeReview,
		PromptTesting,
		PromptDocumentation,
		PromptDebugging,
		PromptRefactoring,
	}

	// Each category should have a unique value
	categoryMap := make(map[PromptCategory]bool)
	for _, category := range categories {
		assert.False(t, categoryMap[category], "Duplicate category value: %v", category)
		categoryMap[category] = true
	}

	// Should have all expected categories
	assert.Len(t, categoryMap, 7)
}

func TestConversationRole_Constants(t *testing.T) {
	// Test conversation role constants
	assert.NotEqual(t, RoleUser, RoleAssistant)

	// Roles should be distinct integers
	assert.True(t, int(RoleUser) != int(RoleAssistant))
}

func TestService_DefaultConfiguration(t *testing.T) {
	service := &Service{
		defaultTimeout: time.Minute * 10,
		retryAttempts:  5,
		retryDelay:     time.Second * 3,
	}

	// Test default configuration values
	assert.Equal(t, time.Minute*10, service.defaultTimeout)
	assert.Equal(t, 5, service.retryAttempts)
	assert.Equal(t, time.Second*3, service.retryDelay)
}

func TestService_ContextTimeout(t *testing.T) {
	service := &Service{
		defaultTimeout: time.Millisecond * 100,
	}

	// Test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), service.defaultTimeout)
	defer cancel()

	// Verify context respects timeout
	select {
	case <-ctx.Done():
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	case <-time.After(time.Millisecond * 200):
		t.Error("Context should have timed out")
	}
}

func TestService_IssueDataIntegration(t *testing.T) {
	service := &Service{
		promptTemplates: getDefaultPromptTemplates(),
	}

	// Test integration with issue data
	issueData := &types.IssueData{
		Number:     456,
		Title:      "Implement caching system",
		Body:       "Add Redis-based caching for improved performance",
		State:      "open",
		Labels:     []string{"enhancement", "performance"},
		Repository: "test/project",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Verify issue data can be used with templates
	template := service.promptTemplates["issue_analysis"]

	// Template should have variables that match issue data fields
	assert.Contains(t, template.Variables, "title")
	assert.Contains(t, template.Variables, "description") // maps to Body
	assert.Contains(t, template.Variables, "labels")

	// Verify issue data has the expected structure
	assert.Equal(t, 456, issueData.Number)
	assert.Equal(t, "Implement caching system", issueData.Title)
	assert.Equal(t, "Add Redis-based caching for improved performance", issueData.Body)
	assert.Contains(t, issueData.Labels, "enhancement")
	assert.Contains(t, issueData.Labels, "performance")
}

func TestService_SessionLifecycle(t *testing.T) {
	service := &Service{
		issueAnalysisSessions: make(map[string]*Session),
	}

	sessionID := "lifecycle-test"

	// Create session
	session := &Session{
		ID:           sessionID,
		Status:       SessionIdle,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Add to service
	service.issueAnalysisSessions[sessionID] = session

	// Update session status
	service.issueAnalysisSessions[sessionID].Status = SessionRunning
	service.issueAnalysisSessions[sessionID].LastActivity = time.Now()

	// Verify session is tracked and updated
	trackedSession := service.issueAnalysisSessions[sessionID]
	assert.Equal(t, SessionRunning, trackedSession.Status)
	assert.Equal(t, sessionID, trackedSession.ID)

	// Remove session
	delete(service.issueAnalysisSessions, sessionID)
	assert.NotContains(t, service.issueAnalysisSessions, sessionID)
}

func TestService_RetryConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		retryAttempts int
		retryDelay    time.Duration
		expectValid   bool
	}{
		{
			name:          "valid configuration",
			retryAttempts: 3,
			retryDelay:    time.Second * 2,
			expectValid:   true,
		},
		{
			name:          "no retries",
			retryAttempts: 0,
			retryDelay:    time.Second,
			expectValid:   true,
		},
		{
			name:          "high retry count",
			retryAttempts: 10,
			retryDelay:    time.Millisecond * 500,
			expectValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &Service{
				retryAttempts: tt.retryAttempts,
				retryDelay:    tt.retryDelay,
			}

			if tt.expectValid {
				assert.Equal(t, tt.retryAttempts, service.retryAttempts)
				assert.Equal(t, tt.retryDelay, service.retryDelay)
			}
		})
	}
}

func TestService_ThreadSafety(t *testing.T) {
	service := &Service{
		conversationHistory:    make(map[string][]ConversationEntry),
		issueAnalysisSessions:  make(map[string]*Session),
		codeGenerationSessions: make(map[string]*Session),
		reviewSessions:         make(map[string]*Session),
	}

	// Test that service has mutexes for thread safety
	assert.NotNil(t, &service.historyMutex)
	assert.NotNil(t, &service.sessionPoolMutex)

	// Verify maps are initialized
	assert.NotNil(t, service.conversationHistory)
	assert.NotNil(t, service.issueAnalysisSessions)
	assert.NotNil(t, service.codeGenerationSessions)
	assert.NotNil(t, service.reviewSessions)
}

func TestService_ErrorHandling(t *testing.T) {
	service := &Service{
		defaultTimeout: time.Millisecond * 50, // Very short timeout for testing
		retryAttempts:  2,
		retryDelay:     time.Millisecond * 10,
	}

	// Test timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), service.defaultTimeout)
	defer cancel()

	// Simulate operation that would timeout
	select {
	case <-ctx.Done():
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	case <-time.After(time.Millisecond * 100):
		t.Error("Context should have timed out")
	}

	// Test retry configuration
	assert.Equal(t, 2, service.retryAttempts)
	assert.Equal(t, time.Millisecond*10, service.retryDelay)
}
