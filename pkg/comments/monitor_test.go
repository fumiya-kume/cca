package comments

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewCommentMonitor(t *testing.T) {
	tests := []struct {
		name           string
		config         CommentMonitorConfig
		expectedConfig CommentMonitorConfig
	}{
		{
			name: "default config values",
			config: CommentMonitorConfig{
				TrackResolution: true,
			},
			expectedConfig: CommentMonitorConfig{
				PollInterval:      time.Minute * 2,
				TrackResolution:   true,
				MaxTrackingTime:   time.Hour * 24 * 7,
				ResolutionTimeout: time.Hour * 24,
				RetryAttempts:     3,
			},
		},
		{
			name: "custom config values",
			config: CommentMonitorConfig{
				PollInterval:      time.Minute * 5,
				TrackResolution:   false,
				MaxTrackingTime:   time.Hour * 48,
				ResolutionTimeout: time.Hour * 12,
				RetryAttempts:     5,
			},
			expectedConfig: CommentMonitorConfig{
				PollInterval:      time.Minute * 5,
				TrackResolution:   false,
				MaxTrackingTime:   time.Hour * 48,
				ResolutionTimeout: time.Hour * 12,
				RetryAttempts:     5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewCommentMonitor(tt.config)

			assert.NotNil(t, monitor)
			assert.Equal(t, tt.expectedConfig.PollInterval, monitor.config.PollInterval)
			assert.Equal(t, tt.expectedConfig.TrackResolution, monitor.config.TrackResolution)
			assert.Equal(t, tt.expectedConfig.MaxTrackingTime, monitor.config.MaxTrackingTime)
			assert.Equal(t, tt.expectedConfig.ResolutionTimeout, monitor.config.ResolutionTimeout)
			assert.Equal(t, tt.expectedConfig.RetryAttempts, monitor.config.RetryAttempts)
			assert.False(t, monitor.isMonitoring)
			assert.Empty(t, monitor.tracking)
			assert.Empty(t, monitor.callbacks)
		})
	}
}

func TestCommentMonitor_TrackComment(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{})

	comment := &Comment{
		ID:     1,
		Author: "test-user",
		Body:   "Test comment",
		Status: CommentStatusPending,
	}

	// Track the comment
	monitor.TrackComment(comment)

	// Verify tracking
	assert.Len(t, monitor.tracking, 1)
	tracker, exists := monitor.tracking[comment.ID]
	require.True(t, exists)
	assert.Equal(t, comment, tracker.Comment)
	assert.Equal(t, TrackingStatusActive, tracker.Status)
	assert.True(t, time.Since(tracker.FirstSeen) < time.Second)
	assert.Empty(t, tracker.Escalations)
	assert.Empty(t, tracker.Notifications)
	assert.Equal(t, 0, tracker.ResponseCount)
}

func TestCommentMonitor_UpdateCommentStatus(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{})

	comment := &Comment{
		ID:     1,
		Author: "test-user",
		Body:   "Test comment",
		Status: CommentStatusPending,
	}

	// Track the comment first
	monitor.TrackComment(comment)

	tests := []struct {
		name           string
		status         CommentStatus
		expectedStatus TrackingStatus
		expectResolved bool
	}{
		{
			name:           "resolve comment",
			status:         CommentStatusResolved,
			expectedStatus: TrackingStatusResolved,
			expectResolved: true,
		},
		{
			name:           "escalate comment",
			status:         CommentStatusEscalated,
			expectedStatus: TrackingStatusEscalated,
			expectResolved: false,
		},
		{
			name:           "dismiss comment",
			status:         CommentStatusDismissed,
			expectedStatus: TrackingStatusIgnored,
			expectResolved: false,
		},
		{
			name:           "acknowledge comment",
			status:         CommentStatusAcknowledged,
			expectedStatus: TrackingStatusActive,
			expectResolved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset comment status and tracking
			comment.Status = CommentStatusPending
			monitor.tracking[comment.ID].Status = TrackingStatusActive
			monitor.tracking[comment.ID].Resolution = nil

			monitor.UpdateCommentStatus(comment.ID, tt.status)

			tracker := monitor.tracking[comment.ID]
			assert.Equal(t, tt.status, tracker.Comment.Status)
			assert.Equal(t, tt.expectedStatus, tracker.Status)
			assert.Greater(t, tracker.Metrics.InteractionCount, 0)

			if tt.expectResolved {
				assert.NotNil(t, tracker.Resolution)
				assert.Equal(t, "ccagents-ai", tracker.Resolution.ResolvedBy)
				assert.Equal(t, "automatic", tracker.Resolution.Method)
			} else {
				// Only check if it was previously nil (for non-resolved tests)
				if tt.name != "resolve comment" {
					assert.Nil(t, tracker.Resolution)
				}
			}
		})
	}
}

func TestCommentMonitor_GetTrackingStats(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{})

	// Add various tracking entries
	comments := []*Comment{
		{ID: 1, Status: CommentStatusPending},
		{ID: 2, Status: CommentStatusPending},
		{ID: 3, Status: CommentStatusResolved},
		{ID: 4, Status: CommentStatusEscalated},
	}

	for _, comment := range comments {
		monitor.TrackComment(comment)
	}

	// Update statuses
	monitor.UpdateCommentStatus(2, CommentStatusAcknowledged) // Still active
	monitor.UpdateCommentStatus(3, CommentStatusResolved)
	monitor.UpdateCommentStatus(4, CommentStatusEscalated)

	stats := monitor.GetTrackingStats()

	assert.Equal(t, 4, stats["total_tracked"])
	assert.Equal(t, 2, stats["active"])    // Comments 1 and 2
	assert.Equal(t, 1, stats["resolved"])  // Comment 3
	assert.Equal(t, 1, stats["escalated"]) // Comment 4
	assert.False(t, stats["is_monitoring"].(bool))

	// Check that average times are present for resolved comments
	_, hasAvgResolution := stats["avg_resolution_time"]
	_, hasAvgResponse := stats["avg_response_time"]
	assert.True(t, hasAvgResolution)
	assert.True(t, hasAvgResponse)
}

func TestCommentMonitor_StartStopMonitoring(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{
		PollInterval: time.Millisecond * 100,
	})

	mockClient := &MockGitHubClient{}
	prNumber := 123

	// Setup mock to return empty comments to avoid actual monitoring work
	mockClient.On("ListComments", mock.Anything, prNumber).Return([]*Comment{}, nil).Maybe()

	// Test callback function
	callback := func(ctx context.Context, comment *Comment, client GitHubClient) error {
		return nil
	}

	ctx := context.Background()

	// Start monitoring
	err := monitor.StartMonitoring(ctx, prNumber, mockClient, callback)
	assert.NoError(t, err)
	assert.True(t, monitor.isMonitoring)

	// Stop monitoring
	monitor.StopMonitoring(prNumber)

	assert.False(t, monitor.isMonitoring)
	assert.Empty(t, monitor.stopChannels)
	assert.Empty(t, monitor.callbacks)
}

func TestCommentMonitor_StopAllMonitoring(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{
		PollInterval: time.Millisecond * 100,
	})

	mockClient := &MockGitHubClient{}
	callback := func(ctx context.Context, comment *Comment, client GitHubClient) error {
		return nil
	}

	// Setup mock
	mockClient.On("ListComments", mock.Anything, mock.AnythingOfType("int")).Return([]*Comment{}, nil).Maybe()

	ctx := context.Background()

	// Start monitoring multiple PRs
	err1 := monitor.StartMonitoring(ctx, 123, mockClient, callback)
	err2 := monitor.StartMonitoring(ctx, 124, mockClient, callback)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.True(t, monitor.isMonitoring)
	assert.Len(t, monitor.stopChannels, 2)

	// Stop all monitoring
	monitor.StopAllMonitoring()

	assert.False(t, monitor.isMonitoring)
	assert.Empty(t, monitor.stopChannels)
	assert.Empty(t, monitor.callbacks)
}

func TestCommentMonitor_CheckTimeouts(t *testing.T) {
	config := CommentMonitorConfig{
		MaxTrackingTime:   time.Hour,
		ResolutionTimeout: time.Minute * 30,
		TrackResolution:   true,
	}
	monitor := NewCommentMonitor(config)

	// Add comments with different ages
	oldComment := &Comment{
		ID:     1,
		Status: CommentStatusPending,
	}
	recentComment := &Comment{
		ID:     2,
		Status: CommentStatusPending,
	}
	unresolvedComment := &Comment{
		ID:     3,
		Status: CommentStatusAcknowledged,
	}

	monitor.TrackComment(oldComment)
	monitor.TrackComment(recentComment)
	monitor.TrackComment(unresolvedComment)

	// Manually set creation times to simulate age
	monitor.tracking[oldComment.ID].FirstSeen = time.Now().Add(-time.Hour * 2)           // Expired
	monitor.tracking[recentComment.ID].FirstSeen = time.Now().Add(-time.Minute * 10)     // Recent
	monitor.tracking[unresolvedComment.ID].FirstSeen = time.Now().Add(-time.Minute * 45) // Unresolved timeout but not expired

	// Run timeout check
	monitor.checkTimeouts()

	// Verify timeout handling
	assert.Equal(t, TrackingStatusExpired, monitor.tracking[oldComment.ID].Status)
	assert.Equal(t, TrackingStatusActive, monitor.tracking[recentComment.ID].Status)
	assert.Equal(t, TrackingStatusActive, monitor.tracking[unresolvedComment.ID].Status) // Still active but timed out
}

func TestCommentMonitor_ShouldEscalateComment(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{})

	tests := []struct {
		name           string
		tracker        *CommentTracker
		rule           EscalationRule
		shouldEscalate bool
	}{
		{
			name: "blocking comment should escalate",
			tracker: &CommentTracker{
				Comment:     &Comment{Priority: CommentPriorityBlocking},
				FirstSeen:   time.Now(),
				Escalations: []EscalationEvent{},
			},
			rule: EscalationRule{
				Name:      "blocking-rule",
				Condition: "blocking",
			},
			shouldEscalate: true,
		},
		{
			name: "critical comment should escalate",
			tracker: &CommentTracker{
				Comment:     &Comment{Priority: CommentPriorityCritical},
				FirstSeen:   time.Now(),
				Escalations: []EscalationEvent{},
			},
			rule: EscalationRule{
				Name:      "critical-rule",
				Condition: "critical",
			},
			shouldEscalate: true,
		},
		{
			name: "unresolved time rule should escalate",
			tracker: &CommentTracker{
				Comment:     &Comment{Priority: CommentPriorityMedium},
				FirstSeen:   time.Now().Add(-time.Hour * 2),
				Escalations: []EscalationEvent{},
			},
			rule: EscalationRule{
				Name:      "time-rule",
				Condition: "unresolved_time",
				Delay:     time.Hour,
			},
			shouldEscalate: true,
		},
		{
			name: "no response rule should escalate",
			tracker: &CommentTracker{
				Comment:       &Comment{Priority: CommentPriorityMedium},
				FirstSeen:     time.Now().Add(-time.Hour),
				ResponseCount: 0,
				Escalations:   []EscalationEvent{},
			},
			rule: EscalationRule{
				Name:      "no-response-rule",
				Condition: "no_response",
				Delay:     time.Minute * 30,
			},
			shouldEscalate: true,
		},
		{
			name: "multiple interactions rule should escalate",
			tracker: &CommentTracker{
				Comment:     &Comment{Priority: CommentPriorityMedium},
				FirstSeen:   time.Now(),
				Metrics:     TrackingMetrics{InteractionCount: 5},
				Escalations: []EscalationEvent{},
			},
			rule: EscalationRule{
				Name:      "interaction-rule",
				Condition: "multiple_interactions",
				Threshold: 3,
			},
			shouldEscalate: true,
		},
		{
			name: "already escalated should not escalate again",
			tracker: &CommentTracker{
				Comment:   &Comment{Priority: CommentPriorityBlocking},
				FirstSeen: time.Now(),
				Escalations: []EscalationEvent{
					{Rule: EscalationRule{Name: "blocking-rule"}},
				},
			},
			rule: EscalationRule{
				Name:      "blocking-rule",
				Condition: "blocking",
			},
			shouldEscalate: false,
		},
		{
			name: "time not reached should not escalate",
			tracker: &CommentTracker{
				Comment:     &Comment{Priority: CommentPriorityMedium},
				FirstSeen:   time.Now().Add(-time.Minute * 15),
				Escalations: []EscalationEvent{},
			},
			rule: EscalationRule{
				Name:      "time-rule",
				Condition: "unresolved_time",
				Delay:     time.Hour,
			},
			shouldEscalate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldEscalate := monitor.shouldEscalateComment(tt.tracker, tt.rule)
			assert.Equal(t, tt.shouldEscalate, shouldEscalate)
		})
	}
}

func TestCommentMonitor_EscalateComment(t *testing.T) {
	config := CommentMonitorConfig{
		NotificationConfig: NotificationConfig{
			Enabled: true,
		},
	}
	monitor := NewCommentMonitor(config)

	mockClient := &MockGitHubClient{}
	prNumber := 123

	comment := &Comment{
		ID:       1,
		Author:   "reviewer",
		Body:     "This is a blocking issue",
		Priority: CommentPriorityBlocking,
	}

	tracker := &CommentTracker{
		Comment:     comment,
		FirstSeen:   time.Now(),
		Status:      TrackingStatusActive,
		Escalations: []EscalationEvent{},
		Metrics:     TrackingMetrics{},
	}

	rule := EscalationRule{
		Name:       "blocking-escalation",
		Condition:  "blocking",
		Action:     "comment",
		Recipients: []string{"@team-lead", "@security-team"},
	}

	// Setup mock for escalation comment creation
	mockClient.On("CreateComment", mock.Anything, prNumber, mock.AnythingOfType("string")).Return(&Comment{}, nil)

	ctx := context.Background()
	err := monitor.escalateComment(ctx, prNumber, tracker, rule, mockClient)

	assert.NoError(t, err)
	assert.Equal(t, TrackingStatusEscalated, tracker.Status)
	assert.Equal(t, CommentStatusEscalated, tracker.Comment.Status)
	assert.Len(t, tracker.Escalations, 1)
	assert.Equal(t, 1, tracker.Metrics.EscalationCount)

	escalation := tracker.Escalations[0]
	assert.Equal(t, rule.Name, escalation.Rule.Name)
	assert.Equal(t, "posted", escalation.Status)
	assert.Contains(t, escalation.Recipients, "@team-lead")
	assert.Contains(t, escalation.Recipients, "@security-team")

	mockClient.AssertExpectations(t)
}

func TestCommentMonitor_RecordResponseTime(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{})

	originalComment := &Comment{
		ID:     1,
		Author: "reviewer",
		Body:   "Original comment",
	}

	aiResponse := &Comment{
		ID:     2,
		Author: "ccagents-ai",
		Body:   "AI response",
	}

	// Track the original comment
	monitor.TrackComment(originalComment)

	// Record response time (no need to simulate time passing)
	monitor.recordResponseTime(aiResponse)

	// Should not create new tracking entry for AI response
	assert.Len(t, monitor.tracking, 1)

	// Now test with AI response to tracked comment
	// Reset the tracking to simulate the AI responding to the tracked comment
	monitor.tracking[originalComment.ID].Metrics.TimeToFirstResponse = 0
	monitor.tracking[originalComment.ID].ResponseCount = 0

	// Simulate AI responding to the original comment by using same ID
	aiResponse.ID = originalComment.ID
	monitor.recordResponseTime(aiResponse)

	tracker := monitor.tracking[originalComment.ID]
	assert.Greater(t, tracker.Metrics.TimeToFirstResponse, time.Duration(0))
	assert.Equal(t, 1, tracker.ResponseCount)
}

func TestCommentMonitor_GenerateEscalationComment(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{})

	comment := &Comment{
		ID:     1,
		Author: "reviewer",
		Body:   "This is a critical security issue that needs immediate attention",
	}

	rule := EscalationRule{
		Name:       "security-escalation",
		Condition:  "critical",
		Recipients: []string{"@security-team", "@team-lead"},
	}

	escalationMsg := monitor.generateEscalationComment(comment, rule)

	assert.Contains(t, escalationMsg, "🚨 **Escalation**")
	assert.Contains(t, escalationMsg, comment.Author)
	assert.Contains(t, escalationMsg, comment.Body)
	assert.Contains(t, escalationMsg, rule.Name)
	assert.Contains(t, escalationMsg, rule.Condition)
	assert.Contains(t, escalationMsg, "@security-team")
	assert.Contains(t, escalationMsg, "@team-lead")
}

func TestCommentMonitor_CheckForNewComments(t *testing.T) {
	monitor := NewCommentMonitor(CommentMonitorConfig{})
	mockClient := &MockGitHubClient{}
	prNumber := 123

	// Mock comments returned by the API
	comments := []*Comment{
		{
			ID:        1,
			Author:    "reviewer1",
			Body:      "First comment",
			CreatedAt: time.Now().Add(-time.Hour),
		},
		{
			ID:        2,
			Author:    "reviewer2",
			Body:      "Second comment",
			CreatedAt: time.Now(),
		},
	}

	mockClient.On("ListComments", mock.Anything, prNumber).Return(comments, nil)

	callbackCount := 0
	callback := func(ctx context.Context, comment *Comment, client GitHubClient) error {
		callbackCount++
		return nil
	}

	ctx := context.Background()
	since := time.Now().Add(-time.Minute * 30) // Only comments newer than 30 minutes

	err := monitor.checkForNewComments(ctx, prNumber, mockClient, callback, since)

	assert.NoError(t, err)
	assert.Equal(t, 1, callbackCount)  // Only the second comment should trigger callback
	assert.Len(t, monitor.tracking, 1) // Only the second comment should be tracked

	mockClient.AssertExpectations(t)
}

func TestCommentMonitor_SendNotification(t *testing.T) {
	tests := []struct {
		name          string
		config        CommentMonitorConfig
		expectedError bool
	}{
		{
			name: "notifications enabled",
			config: CommentMonitorConfig{
				NotificationConfig: NotificationConfig{
					Enabled: true,
				},
			},
			expectedError: false,
		},
		{
			name: "notifications disabled",
			config: CommentMonitorConfig{
				NotificationConfig: NotificationConfig{
					Enabled: false,
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewCommentMonitor(tt.config)

			comment := &Comment{
				ID:     1,
				Author: "reviewer",
				Body:   "Test comment",
			}

			rule := EscalationRule{
				Name:       "test-rule",
				Recipients: []string{"@team"},
			}

			err := monitor.sendNotification(comment, rule)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test TrackingStatus String method
func TestTrackingStatus_String(t *testing.T) {
	tests := []struct {
		status   TrackingStatus
		expected string
	}{
		{TrackingStatusActive, "active"},
		{TrackingStatusResolved, "resolved"},
		{TrackingStatusEscalated, "escalated"},
		{TrackingStatusExpired, "expired"},
		{TrackingStatusIgnored, "ignored"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

// Integration test for monitoring workflow
func TestCommentMonitor_MonitoringWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := CommentMonitorConfig{
		PollInterval:    time.Millisecond * 50,
		TrackResolution: true,
		EscalationRules: []EscalationRule{
			{
				Name:       "quick-escalation",
				Condition:  "blocking",
				Action:     "notify",
				Recipients: []string{"@team"},
			},
		},
	}

	monitor := NewCommentMonitor(config)
	mockClient := &MockGitHubClient{}
	prNumber := 123

	// Setup mock to return a blocking comment
	blockingComment := &Comment{
		ID:        1,
		Author:    "security-reviewer",
		Body:      "This introduces a security vulnerability - blocking merge",
		CreatedAt: time.Now(),
		Priority:  CommentPriorityBlocking,
	}

	mockClient.On("ListComments", mock.Anything, prNumber).Return([]*Comment{blockingComment}, nil)

	callbackExecuted := false
	done := make(chan bool, 1)
	callback := func(ctx context.Context, comment *Comment, client GitHubClient) error {
		callbackExecuted = true
		// Mark comment as escalated to test escalation handling
		monitor.UpdateCommentStatus(comment.ID, CommentStatusEscalated)
		done <- true
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// Start monitoring
	err := monitor.StartMonitoring(ctx, prNumber, mockClient, callback)
	assert.NoError(t, err)

	// Wait for callback to be executed
	select {
	case <-done:
		// Continue with test
	case <-time.After(time.Second):
		t.Fatal("Callback was not executed within timeout")
	}

	// Stop monitoring
	monitor.StopMonitoring(prNumber)

	// Verify callback was executed
	assert.True(t, callbackExecuted)

	// Verify comment is being tracked
	assert.Len(t, monitor.tracking, 1)
	tracker, exists := monitor.tracking[blockingComment.ID]
	require.True(t, exists)
	assert.Equal(t, TrackingStatusEscalated, tracker.Status)

	mockClient.AssertExpectations(t)
}
