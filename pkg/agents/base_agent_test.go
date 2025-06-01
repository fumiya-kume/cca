package agents

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{
		Enabled:       true,
		MaxInstances:  1,
		Timeout:       time.Minute,
		RetryAttempts: 3,
		Priority:      PriorityMedium,
	}

	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, SecurityAgentID, agent.GetID())
	assert.Equal(t, StatusOffline, agent.GetStatus())
	assert.NotNil(t, agent.metrics)
	assert.NotNil(t, agent.handlers)
}

func TestNewBaseAgentWithNilMessageBus(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	agent, err := NewBaseAgent(SecurityAgentID, config, nil)
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "message bus cannot be nil")
}

func TestBaseAgentGetters(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(TestingAgentID, config, messageBus)
	require.NoError(t, err)

	// Test GetID
	assert.Equal(t, TestingAgentID, agent.GetID())

	// Test GetStatus
	assert.Equal(t, StatusOffline, agent.GetStatus())

	// Test GetCapabilities (initially empty)
	caps := agent.GetCapabilities()
	assert.Empty(t, caps)
}

func TestBaseAgentSetStatus(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Test setting different statuses
	statuses := []AgentStatus{
		StatusStarting,
		StatusIdle,
		StatusBusy,
		StatusError,
		StatusStopping,
		StatusOffline,
	}

	for _, status := range statuses {
		agent.SetStatus(status)
		assert.Equal(t, status, agent.GetStatus())
	}
}

func TestBaseAgentSetCapabilities(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	capabilities := []string{"security_scan", "vulnerability_detection", "compliance_check"}
	agent.SetCapabilities(capabilities)

	caps := agent.GetCapabilities()
	assert.Len(t, caps, 3)
	assert.Contains(t, caps, "security_scan")
	assert.Contains(t, caps, "vulnerability_detection")
	assert.Contains(t, caps, "compliance_check")

	// Ensure returned slice is a copy
	caps[0] = "modified"
	originalCaps := agent.GetCapabilities()
	assert.NotEqual(t, "modified", originalCaps[0])
}

func TestBaseAgentStartStop(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	ctx := context.Background()

	// Test Start
	err = agent.Start(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StatusIdle, agent.GetStatus())

	// Allow some time for goroutines to start using context
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Wait briefly for startup to complete
	select {
	case <-ctx.Done():
	default:
		// Continue with test
	}

	// Test Stop
	err = agent.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StatusOffline, agent.GetStatus())
}

func TestBaseAgentStartStopTimeout(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	ctx := context.Background()

	// Start agent
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Stop with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err = agent.Stop(ctx)
	assert.NoError(t, err) // Should still succeed even with timeout
	assert.Equal(t, StatusOffline, agent.GetStatus())
}

func TestBaseAgentRegisterHandler(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Custom handler for task assignment
	handlerCalled := false
	handler := func(ctx context.Context, msg *AgentMessage) error {
		handlerCalled = true
		return nil
	}

	agent.RegisterHandler(TaskAssignment, handler)

	// Verify handler is registered
	agent.mu.RLock()
	_, exists := agent.handlers[TaskAssignment]
	agent.mu.RUnlock()
	assert.True(t, exists)

	// Test the handler by processing a message
	msg := &AgentMessage{Type: TaskAssignment}
	ctx := context.Background()
	err = agent.ProcessMessage(ctx, msg)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestBaseAgentProcessMessage(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Register a custom handler
	handlerCalled := false
	var receivedMsg *AgentMessage
	handler := func(ctx context.Context, msg *AgentMessage) error {
		handlerCalled = true
		receivedMsg = msg
		return nil
	}

	agent.RegisterHandler(TaskAssignment, handler)

	// Create test message
	msg := &AgentMessage{
		ID:       "test-msg",
		Type:     TaskAssignment,
		Sender:   ArchitectureAgentID,
		Receiver: SecurityAgentID,
		Payload:  "test payload",
		Priority: PriorityMedium,
	}

	// Process message
	ctx := context.Background()
	err = agent.ProcessMessage(ctx, msg)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, msg, receivedMsg)

	// Check metrics were updated
	metrics := agent.GetMetrics()
	assert.Equal(t, int64(1), metrics.MessagesReceived)
	assert.Equal(t, int64(1), metrics.MessagesProcessed)
	assert.Equal(t, int64(0), metrics.MessagesFailed)
}

func TestBaseAgentProcessMessageWithNilMessage(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	ctx := context.Background()
	err = agent.ProcessMessage(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message cannot be nil")
}

func TestBaseAgentProcessMessageWithUnknownType(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Create message with unknown type
	msg := &AgentMessage{
		ID:       "test-msg",
		Type:     MessageType("unknown_type"),
		Sender:   ArchitectureAgentID,
		Receiver: SecurityAgentID,
	}

	ctx := context.Background()
	err = agent.ProcessMessage(ctx, msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler for message type")

	// Check metrics were updated for failed message
	metrics := agent.GetMetrics()
	assert.Equal(t, int64(1), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
	assert.Equal(t, int64(1), metrics.MessagesFailed)
}

func TestBaseAgentProcessMessageWithHandlerError(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Register handler that returns error
	expectedError := fmt.Errorf("handler error")
	handler := func(ctx context.Context, msg *AgentMessage) error {
		return expectedError
	}

	agent.RegisterHandler(TaskAssignment, handler)

	// Create test message
	msg := &AgentMessage{
		ID:   "test-msg",
		Type: TaskAssignment,
	}

	ctx := context.Background()
	err = agent.ProcessMessage(ctx, msg)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)

	// Check metrics were updated for failed message
	metrics := agent.GetMetrics()
	assert.Equal(t, int64(1), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
	assert.Equal(t, int64(1), metrics.MessagesFailed)
}

func TestBaseAgentHealthCheck(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	ctx := context.Background()

	// Health check when offline should fail
	err = agent.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent is not healthy")

	// Start agent
	err = agent.Start(ctx)
	require.NoError(t, err)

	// Health check when running should pass
	err = agent.HealthCheck(ctx)
	assert.NoError(t, err)

	// Stop agent
	err = agent.Stop(ctx)
	require.NoError(t, err)

	// Health check when offline should fail again
	err = agent.HealthCheck(ctx)
	assert.Error(t, err)
}

func TestBaseAgentHealthCheckWithStaleProcessing(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	ctx := context.Background()

	// Start agent
	err = agent.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = agent.Stop(ctx) }()

	// Set last process time to be very old
	agent.metrics.mu.Lock()
	agent.metrics.LastProcessTime = time.Now().Add(-10 * time.Minute)
	agent.metrics.mu.Unlock()

	// Health check should fail due to stale processing
	err = agent.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no messages processed in last 5 minutes")
}

func TestBaseAgentSendMessage(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	msg := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: ArchitectureAgentID,
		Payload:  "test",
	}

	err = agent.SendMessage(msg)
	assert.NoError(t, err)
	assert.Equal(t, SecurityAgentID, msg.Sender) // Should be set automatically
}

func TestBaseAgentSendProgressUpdate(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	progress := map[string]interface{}{
		"status":   "scanning",
		"progress": 0.5,
	}

	err = agent.SendProgressUpdate(ArchitectureAgentID, progress)
	assert.NoError(t, err)
}

func TestBaseAgentSendResult(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	result := &AgentResult{
		AgentID: SecurityAgentID,
		Success: true,
		Results: map[string]interface{}{"vulnerabilities": 0},
	}

	err = agent.SendResult(ArchitectureAgentID, result)
	assert.NoError(t, err)
}

func TestBaseAgentSendError(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	testError := fmt.Errorf("test error")
	context := map[string]interface{}{
		"operation": "security_scan",
		"file":      "test.go",
	}

	err = agent.SendError(ArchitectureAgentID, testError, context)
	assert.NoError(t, err)
}

func TestBaseAgentGetMetrics(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Initial metrics
	metrics := agent.GetMetrics()
	assert.Equal(t, int64(0), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
	assert.Equal(t, int64(0), metrics.MessagesFailed)
	assert.Equal(t, time.Duration(0), metrics.TotalProcessTime)
	assert.Equal(t, time.Duration(0), metrics.AverageProcessTime)
	assert.True(t, metrics.LastProcessTime.IsZero())

	// Register a handler and process a message
	handler := func(ctx context.Context, msg *AgentMessage) error {
		// Simulate processing time with CPU-bound work
		for i := 0; i < 1000; i++ {
			_ = i * i * i
		}
		return nil
	}
	agent.RegisterHandler(TaskAssignment, handler)

	msg := &AgentMessage{
		Type: TaskAssignment,
	}

	ctx := context.Background()
	err = agent.ProcessMessage(ctx, msg)
	require.NoError(t, err)

	// Check updated metrics
	metrics = agent.GetMetrics()
	assert.Equal(t, int64(1), metrics.MessagesReceived)
	assert.Equal(t, int64(1), metrics.MessagesProcessed)
	assert.Equal(t, int64(0), metrics.MessagesFailed)
	assert.True(t, metrics.TotalProcessTime > 0)
	assert.True(t, metrics.AverageProcessTime > 0)
	assert.False(t, metrics.LastProcessTime.IsZero())
}

func TestBaseAgentDefaultHandlers(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	ctx := context.Background()

	// Test health check handler
	healthCheckMsg := &AgentMessage{
		ID:       "health-check",
		Type:     HealthCheck,
		Sender:   ArchitectureAgentID,
		Receiver: SecurityAgentID,
		Priority: PriorityMedium,
	}

	err = agent.ProcessMessage(ctx, healthCheckMsg)
	assert.NoError(t, err)

	// Test progress update handler
	progressMsg := &AgentMessage{
		ID:      "progress",
		Type:    ProgressUpdate,
		Payload: map[string]interface{}{"status": "working"},
	}

	err = agent.ProcessMessage(ctx, progressMsg)
	assert.NoError(t, err)
}

func TestBaseAgentConcurrentMessageProcessing(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Register handler that tracks calls
	var mu sync.Mutex
	handlerCalls := 0
	handler := func(ctx context.Context, msg *AgentMessage) error {
		mu.Lock()
		handlerCalls++
		mu.Unlock()
		// Simulate work with CPU-bound operation
		for i := 0; i < 500; i++ {
			_ = i * i * i
		}
		return nil
	}

	agent.RegisterHandler(TaskAssignment, handler)

	ctx := context.Background()

	// Process multiple messages concurrently
	const numMessages = 10
	var wg sync.WaitGroup
	errors := make([]error, numMessages)

	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			msg := &AgentMessage{
				ID:   fmt.Sprintf("msg-%d", index),
				Type: TaskAssignment,
			}
			errors[index] = agent.ProcessMessage(ctx, msg)
		}(i)
	}

	wg.Wait()

	// Check all messages were processed successfully
	for i, err := range errors {
		assert.NoError(t, err, "Message %d failed", i)
	}

	mu.Lock()
	finalHandlerCalls := handlerCalls
	mu.Unlock()
	assert.Equal(t, numMessages, finalHandlerCalls)

	// Check metrics
	metrics := agent.GetMetrics()
	assert.Equal(t, int64(numMessages), metrics.MessagesReceived)
	assert.Equal(t, int64(numMessages), metrics.MessagesProcessed)
	assert.Equal(t, int64(0), metrics.MessagesFailed)
}

func TestBaseAgentProcessMessagesIntegration(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	ctx := context.Background()

	// Start the agent
	err = agent.Start(ctx)
	require.NoError(t, err)
	defer func() { _ = agent.Stop(ctx) }()

	// Register handler to track received messages
	var mu sync.Mutex
	receivedMessages := make([]*AgentMessage, 0)
	handler := func(ctx context.Context, msg *AgentMessage) error {
		mu.Lock()
		receivedMessages = append(receivedMessages, msg)
		mu.Unlock()
		return nil
	}

	agent.RegisterHandler(TaskAssignment, handler)

	// Send messages through the message bus
	for i := 0; i < 5; i++ {
		msg := &AgentMessage{
			ID:       fmt.Sprintf("msg-%d", i),
			Type:     TaskAssignment,
			Sender:   ArchitectureAgentID,
			Receiver: SecurityAgentID,
			Payload:  fmt.Sprintf("payload-%d", i),
		}

		err = messageBus.Send(msg)
		require.NoError(t, err)
	}

	// Wait for messages to be processed using ticker
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mu.Lock()
			count := len(receivedMessages)
			mu.Unlock()
			if count >= 5 {
				goto messagesProcessed
			}
		case <-ctx.Done():
			goto messagesProcessed
		}
	}
messagesProcessed:

	// Check received messages
	mu.Lock()
	assert.Len(t, receivedMessages, 5)
	mu.Unlock()

	// Verify agent status changed during processing
	// Note: Status changes are fast, so we mainly verify the agent can handle the load
	assert.Contains(t, []AgentStatus{StatusIdle, StatusBusy}, agent.GetStatus())
}

func TestAgentMetricsThreadSafety(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Register a simple handler
	agent.RegisterHandler(TaskAssignment, func(ctx context.Context, msg *AgentMessage) error {
		return nil
	})

	ctx := context.Background()

	// Concurrently process messages and read metrics
	var wg sync.WaitGroup
	const numGoroutines = 10
	const messagesPerGoroutine = 10

	// Message processing goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := &AgentMessage{
					Type: TaskAssignment,
				}
				_ = agent.ProcessMessage(ctx, msg)
			}
		}()
	}

	// Metrics reading goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				agent.GetMetrics()
				// Brief yield to prevent tight loop with CPU work
				if j%10 == 0 {
					for k := 0; k < 10; k++ {
						_ = k * k
					}
				}
			}
		}()
	}

	wg.Wait()

	// Final verification
	metrics := agent.GetMetrics()
	assert.Equal(t, int64(numGoroutines*messagesPerGoroutine), metrics.MessagesReceived)
	assert.Equal(t, int64(numGoroutines*messagesPerGoroutine), metrics.MessagesProcessed)
}

func TestBaseAgentEdgeCases(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	config := AgentConfig{Enabled: true}
	agent, err := NewBaseAgent(SecurityAgentID, config, messageBus)
	require.NoError(t, err)

	// Test setting empty capabilities
	agent.SetCapabilities([]string{})
	caps := agent.GetCapabilities()
	assert.Empty(t, caps)

	// Test setting nil capabilities
	agent.SetCapabilities(nil)
	caps = agent.GetCapabilities()
	assert.Empty(t, caps)

	// Test multiple start/stop cycles
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		err = agent.Start(ctx)
		assert.NoError(t, err)
		err = agent.Stop(ctx)
		assert.NoError(t, err)
	}
}
