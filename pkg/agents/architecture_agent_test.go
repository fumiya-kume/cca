package agents

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArchitectureAgent(t *testing.T) {
	config := AgentConfig{
		Enabled:      true,
		MaxInstances: 1,
		Timeout:      time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, ArchitectureAgentID, agent.GetID())
	assert.Equal(t, StatusOffline, agent.GetStatus())
}

func TestArchitectureAgent_Lifecycle(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	// Test starting the agent
	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, StatusIdle, agent.GetStatus())

	// Test health check
	err = agent.HealthCheck(ctx)
	require.NoError(t, err)

	// Test stopping the agent
	err = agent.Stop(ctx)
	require.NoError(t, err)
}

func TestArchitectureAgent_ProcessMessage_TaskAssignment(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test task assignment message
	message := &AgentMessage{
		ID:       "msg-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: ArchitectureAgentID,
		Payload: map[string]interface{}{
			"action":      "analyze",
			"task_type":   "architecture_analysis",
			"target_path": "./testdata",
			"parameters": map[string]interface{}{
				"deep_analysis": true,
				"check_solid":   true,
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestArchitectureAgent_ProcessMessage_CollaborationRequest(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test collaboration request
	message := &AgentMessage{
		ID:       "collab-1",
		Type:     CollaborationRequest,
		Sender:   SecurityAgentID,
		Receiver: ArchitectureAgentID,
		Payload: &AgentCollaboration{
			CollaborationType: ExpertiseConsultation,
			Context: CollaborationContext{
				Purpose: "Need architecture guidance for security fixes",
				SharedData: map[string]interface{}{
					"security_issues": []string{"insecure-dependency"},
					"target_files":    []string{"./go.mod", "./go.sum"},
				},
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestArchitectureAgent_GetCapabilities(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	capabilities := agent.GetCapabilities()
	assert.NotEmpty(t, capabilities)

	// Should have core architecture capabilities
	assert.Contains(t, capabilities, "complexity_analysis")
	assert.Contains(t, capabilities, "design_principle_validation")
	assert.Contains(t, capabilities, "code_smell_detection")
	assert.Contains(t, capabilities, "refactoring_suggestions")
	assert.Contains(t, capabilities, "technical_debt_assessment")
	assert.Contains(t, capabilities, "architecture_validation")
}

func TestArchitectureAgent_Metrics(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	// Get initial metrics
	metrics := agent.GetMetrics()
	// metrics is a value type, not a pointer, so we don't need to check for nil
	assert.Equal(t, int64(0), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

// Test data structures

func TestArchitectureAnalysis_Structure(t *testing.T) {
	analysis := &ArchitectureAnalysis{
		Analyzer:  "complexity",
		Timestamp: time.Now(),
		Issues: []ArchitectureIssue{
			{
				ID:          "ARCH-001",
				Type:        "complexity",
				Severity:    "high",
				Description: "High cyclomatic complexity",
				File:        "example.go",
				Location:    "func Calculate()",
				Principle:   "single_responsibility",
				Impact:      "maintainability",
				Fixable:     true,
				Refactoring: "extract_method",
			},
		},
		Metrics: ArchitectureMetrics{
			CyclomaticComplexity: map[string]int{
				"Calculate": 15,
				"Process":   8,
			},
			CognitiveComplexity: map[string]int{
				"Calculate": 12,
				"Process":   5,
			},
			MethodLength: map[string]int{
				"Calculate": 45,
				"Process":   20,
			},
		},
		Suggestions: []RefactoringSuggestion{
			{
				Type:        "extract_method",
				Description: "Extract complex calculation logic",
				Target:      "func Calculate()",
				Reason:      "High complexity detected",
				Impact:      "Improved maintainability",
				Automated:   true,
				Complexity:  "medium",
			},
		},
	}

	assert.Equal(t, "complexity", analysis.Analyzer)
	assert.Len(t, analysis.Issues, 1)
	assert.Len(t, analysis.Suggestions, 1)

	issue := analysis.Issues[0]
	assert.Equal(t, "ARCH-001", issue.ID)
	assert.Equal(t, "complexity", issue.Type)
	assert.Equal(t, "high", issue.Severity)
	assert.True(t, issue.Fixable)

	metrics := analysis.Metrics
	assert.Equal(t, 15, metrics.CyclomaticComplexity["Calculate"])
	assert.Equal(t, 8, metrics.CyclomaticComplexity["Process"])

	suggestion := analysis.Suggestions[0]
	assert.Equal(t, "extract_method", suggestion.Type)
	assert.Equal(t, "Extract complex calculation logic", suggestion.Description)
	assert.True(t, suggestion.Automated)
}

func TestArchitectureIssue_Structure(t *testing.T) {
	issue := ArchitectureIssue{
		ID:          "ARCH-002",
		Type:        "solid_violation",
		Severity:    "medium",
		Description: "Single Responsibility Principle violation",
		File:        "user_service.go",
		Location:    "type UserService struct",
		Principle:   "single_responsibility",
		Impact:      "maintainability",
		Fixable:     true,
		Refactoring: "split_class",
	}

	assert.Equal(t, "ARCH-002", issue.ID)
	assert.Equal(t, "solid_violation", issue.Type)
	assert.Equal(t, "medium", issue.Severity)
	assert.Equal(t, "Single Responsibility Principle violation", issue.Description)
	assert.Equal(t, "user_service.go", issue.File)
	assert.Equal(t, "type UserService struct", issue.Location)
	assert.Equal(t, "single_responsibility", issue.Principle)
	assert.Equal(t, "maintainability", issue.Impact)
	assert.True(t, issue.Fixable)
	assert.Equal(t, "split_class", issue.Refactoring)
}

func TestArchitectureMetrics_Structure(t *testing.T) {
	metrics := ArchitectureMetrics{
		CyclomaticComplexity: map[string]int{
			"func1": 10,
			"func2": 5,
		},
		CognitiveComplexity: map[string]int{
			"func1": 8,
			"func2": 3,
		},
		MethodLength: map[string]int{
			"func1": 50,
			"func2": 20,
		},
		ClassSize: map[string]int{
			"UserService": 200,
			"AuthService": 150,
		},
		DependencyCount: map[string]int{
			"UserService": 5,
			"AuthService": 3,
		},
		Coupling: map[string]float64{
			"UserService": 0.7,
			"AuthService": 0.4,
		},
	}

	assert.Equal(t, 10, metrics.CyclomaticComplexity["func1"])
	assert.Equal(t, 5, metrics.CyclomaticComplexity["func2"])
	assert.Equal(t, 8, metrics.CognitiveComplexity["func1"])
	assert.Equal(t, 200, metrics.ClassSize["UserService"])
	assert.Equal(t, 5, metrics.DependencyCount["UserService"])
	assert.Equal(t, 0.7, metrics.Coupling["UserService"])
}

func TestRefactoringSuggestion_Structure(t *testing.T) {
	suggestion := RefactoringSuggestion{
		ID:          "REF-001",
		Type:        "extract_method",
		Target:      "func ValidateUser()",
		Description: "Extract complex validation logic into separate method",
		Reason:      "Method is too complex",
		Impact:      "Improved readability and testability",
		Automated:   true,
		Complexity:  "low",
	}

	assert.Equal(t, "REF-001", suggestion.ID)
	assert.Equal(t, "extract_method", suggestion.Type)
	assert.Equal(t, "func ValidateUser()", suggestion.Target)
	assert.Equal(t, "Extract complex validation logic into separate method", suggestion.Description)
	assert.Equal(t, "Method is too complex", suggestion.Reason)
	assert.Equal(t, "Improved readability and testability", suggestion.Impact)
	assert.True(t, suggestion.Automated)
	assert.Equal(t, "low", suggestion.Complexity)
}

// Test error conditions

func TestArchitectureAgent_ErrorHandling(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test invalid message type
	message := &AgentMessage{
		ID:       "invalid-msg",
		Type:     "invalid_type",
		Sender:   "test-agent",
		Receiver: ArchitectureAgentID,
		Payload:  map[string]interface{}{},
	}

	_ = agent.ProcessMessage(context.Background(), message)
	// Should handle gracefully rather than crashing
	// The actual behavior depends on the implementation
	// but it shouldn't panic
	assert.NotPanics(t, func() {
		_ = agent.ProcessMessage(context.Background(), message)
	})
}

func TestArchitectureAgent_MessageHandling(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test different message types the agent should handle
	messageTypes := []MessageType{
		TaskAssignment,
		CollaborationRequest,
		HealthCheck,
	}

	for _, msgType := range messageTypes {
		message := &AgentMessage{
			ID:       "test-msg",
			Type:     msgType,
			Sender:   "test-agent",
			Receiver: ArchitectureAgentID,
			Payload: map[string]interface{}{
				"test": "data",
			},
			Timestamp: time.Now(),
			Priority:  PriorityMedium,
		}

		// Should not panic when processing known message types
		assert.NotPanics(t, func() {
			_ = agent.ProcessMessage(context.Background(), message)
		})
	}
}

func TestArchitectureAgent_AgentConfiguration(t *testing.T) {
	// Test with different configurations
	configs := []AgentConfig{
		{
			Enabled:       true,
			MaxInstances:  1,
			Timeout:       time.Second * 30,
			RetryAttempts: 3,
			Priority:      PriorityHigh,
		},
		{
			Enabled:       true,
			MaxInstances:  2,
			Timeout:       time.Minute * 5,
			RetryAttempts: 5,
			Priority:      PriorityMedium,
		},
	}

	for i, config := range configs {
		messageBus := NewMessageBus(100)

		agent, err := NewArchitectureAgent(config, messageBus)
		require.NoError(t, err, "Config %d should create agent successfully", i)
		assert.NotNil(t, agent, "Agent %d should not be nil", i)
		assert.Equal(t, ArchitectureAgentID, agent.GetID(), "Agent %d should have correct ID", i)

		messageBus.Stop()
	}
}

func TestArchitectureAgent_ConcurrentMessageProcessing(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewArchitectureAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Process multiple messages concurrently
	done := make(chan bool, 3)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		go func(id int) {
			message := &AgentMessage{
				ID:       "concurrent-msg",
				Type:     TaskAssignment,
				Sender:   "test-agent",
				Receiver: ArchitectureAgentID,
				Payload: map[string]interface{}{
					"task_type": "architecture_analysis",
				},
				Timestamp: time.Now(),
				Priority:  PriorityMedium,
			}

			// Should handle concurrent processing without panic
			assert.NotPanics(t, func() {
				_ = agent.ProcessMessage(ctx, message)
			})
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(time.Millisecond * 100):
			t.Fatal("Concurrent message processing timed out")
		}
	}
}
