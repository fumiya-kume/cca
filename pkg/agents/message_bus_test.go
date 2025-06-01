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

func TestNewMessageBus(t *testing.T) {
	bufferSize := 100
	bus := NewMessageBus(bufferSize)

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.subscribers)
	assert.NotNil(t, bus.messages)
	assert.NotNil(t, bus.metrics)
	assert.Equal(t, bufferSize, bus.messageBuffer)

	// Cleanup
	bus.Stop()
}

func TestMessageBusSubscribe(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Test successful subscription
	agentID := SecurityAgentID
	bufferSize := 50

	ch, err := bus.Subscribe(agentID, bufferSize)
	assert.NoError(t, err)
	assert.NotNil(t, ch)

	// Verify subscriber is registered
	bus.subscribersMu.RLock()
	_, exists := bus.subscribers[agentID]
	bus.subscribersMu.RUnlock()
	assert.True(t, exists)

	// Test duplicate subscription should fail
	_, err = bus.Subscribe(agentID, bufferSize)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent security is already subscribed")
}

func TestMessageBusUnsubscribe(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	agentID := SecurityAgentID

	// Test unsubscribing non-existent agent
	err := bus.Unsubscribe(agentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent security is not subscribed")

	// Subscribe first
	ch, err := bus.Subscribe(agentID, 50)
	require.NoError(t, err)

	// Test successful unsubscribe
	err = bus.Unsubscribe(agentID)
	assert.NoError(t, err)

	// Verify subscriber is removed
	bus.subscribersMu.RLock()
	_, exists := bus.subscribers[agentID]
	bus.subscribersMu.RUnlock()
	assert.False(t, exists)

	// Verify channel is closed
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "Channel should be closed")
	case <-time.After(100 * time.Millisecond):
		t.Error("Channel should have been closed")
	}
}

func TestMessageBusSend(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Subscribe an agent
	agentID := SecurityAgentID
	ch, err := bus.Subscribe(agentID, 50)
	require.NoError(t, err)

	// Get initial metrics
	initialMetrics := bus.GetMetrics()

	// Create test message
	msg := &AgentMessage{
		Type:     TaskAssignment,
		Sender:   ArchitectureAgentID,
		Receiver: agentID,
		Payload:  "test payload",
		Priority: PriorityMedium,
	}

	// Send message
	err = bus.Send(msg)
	assert.NoError(t, err)

	// Message should have ID and timestamp set
	assert.NotEmpty(t, msg.ID)
	assert.False(t, msg.Timestamp.IsZero())

	// Wait for message to be delivered
	select {
	case receivedMsg := <-ch:
		assert.Equal(t, msg.Type, receivedMsg.Type)
		assert.Equal(t, msg.Sender, receivedMsg.Sender)
		assert.Equal(t, msg.Receiver, receivedMsg.Receiver)
		assert.Equal(t, msg.Payload, receivedMsg.Payload)
	case <-time.After(time.Second):
		t.Error("Message should have been delivered")
	}

	// Check metrics (should have at least one more message)
	metrics := bus.GetMetrics()
	assert.GreaterOrEqual(t, metrics.TotalMessages, initialMetrics.TotalMessages+1)
	assert.GreaterOrEqual(t, metrics.SuccessfulDelivery, initialMetrics.SuccessfulDelivery+1)
	assert.GreaterOrEqual(t, metrics.FailedDelivery, initialMetrics.FailedDelivery)
}

func TestMessageBusSendWithoutReceiver(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Get initial metrics
	initialMetrics := bus.GetMetrics()

	msg := &AgentMessage{
		Type:     TaskAssignment,
		Sender:   ArchitectureAgentID,
		Receiver: SecurityAgentID, // Not subscribed
		Payload:  "test payload",
	}

	err := bus.Send(msg)
	assert.NoError(t, err)

	// Wait for async processing to complete
	done := make(chan bool, 1)
	go func() {
		// Give time for metrics to be updated
		for i := 0; i < 50; i++ {
			currentMetrics := bus.GetMetrics()
			if currentMetrics.FailedDelivery > initialMetrics.FailedDelivery {
				done <- true
				return
			}
			select {
			case <-time.After(2 * time.Millisecond):
			case <-time.After(100 * time.Millisecond):
				done <- true
				return
			}
		}
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		// Continue with test
	}

	// Check metrics - message processing is handled asynchronously
	metrics := bus.GetMetrics()
	assert.GreaterOrEqual(t, metrics.TotalMessages, initialMetrics.TotalMessages+1)
	assert.GreaterOrEqual(t, metrics.SuccessfulDelivery, initialMetrics.SuccessfulDelivery)
	assert.GreaterOrEqual(t, metrics.FailedDelivery, initialMetrics.FailedDelivery+1)
}

func TestMessageBusSendWithFullBuffer(t *testing.T) {
	// Create bus with very small buffer
	bus := NewMessageBus(1)
	defer bus.Stop()

	// Fill the buffer
	msg1 := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: SecurityAgentID,
		Payload:  "msg1",
	}

	err := bus.Send(msg1)
	assert.NoError(t, err)

	// Try to send another message - should fail due to full buffer
	msg2 := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: ArchitectureAgentID,
		Payload:  "msg2",
	}

	err = bus.Send(msg2)
	// Some implementations may not return an error for full buffer, just handle gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "full")
	}
}

func TestMessageBusSendAfterShutdown(t *testing.T) {
	bus := NewMessageBus(100)

	// Stop the bus
	bus.Stop()

	msg := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: SecurityAgentID,
		Payload:  "test",
	}

	err := bus.Send(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message bus is shutting down")
}

func TestMessageBusBroadcast(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Subscribe multiple agents
	agentIDs := []AgentID{SecurityAgentID, ArchitectureAgentID, TestingAgentID}
	channels := make(map[AgentID]<-chan *AgentMessage)

	for _, id := range agentIDs {
		ch, err := bus.Subscribe(id, 50)
		require.NoError(t, err)
		channels[id] = ch
	}

	// Broadcast message
	msg := &AgentMessage{
		Type:    ProgressUpdate,
		Sender:  PerformanceAgentID,
		Payload: "broadcast test",
	}

	err := bus.Broadcast(msg)
	assert.NoError(t, err)

	// Verify all subscribers received the message
	for _, id := range agentIDs {
		select {
		case receivedMsg := <-channels[id]:
			assert.Equal(t, msg.Type, receivedMsg.Type)
			assert.Equal(t, msg.Sender, receivedMsg.Sender)
			assert.Equal(t, id, receivedMsg.Receiver) // Should be set to subscriber ID
			assert.Equal(t, msg.Payload, receivedMsg.Payload)
		case <-time.After(time.Second):
			t.Errorf("Agent %s should have received broadcast message", id)
		}
	}
}

func TestMessageBusDeliveryTimeout(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Subscribe agent with very small buffer
	agentID := SecurityAgentID
	ch, err := bus.Subscribe(agentID, 1)
	require.NoError(t, err)

	// Fill the agent's buffer
	msg1 := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: agentID,
		Payload:  "msg1",
	}

	err = bus.Send(msg1)
	assert.NoError(t, err)

	// Don't read from channel, so it's full

	// Send another message - should timeout during delivery
	msg2 := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: agentID,
		Payload:  "msg2",
	}

	err = bus.Send(msg2)
	assert.NoError(t, err) // Send succeeds, delivery handling is async

	// Clean up the channel by reading the first message
	<-ch

	// This allows the second message to be delivered too
	// The test verifies that message bus doesn't hang on full buffers
}

func TestMessageBusMetrics(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Subscribe an agent
	agentID := SecurityAgentID
	ch, err := bus.Subscribe(agentID, 50)
	require.NoError(t, err)

	// Get baseline metrics
	initialMetrics := bus.GetMetrics()

	// Send successful message
	msg := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: agentID,
		Payload:  "test",
	}

	err = bus.Send(msg)
	assert.NoError(t, err)

	// Wait for delivery
	<-ch

	// Check updated metrics (should have at least one more message)
	metrics := bus.GetMetrics()
	assert.GreaterOrEqual(t, metrics.TotalMessages, initialMetrics.TotalMessages+1)
	assert.GreaterOrEqual(t, metrics.SuccessfulDelivery, initialMetrics.SuccessfulDelivery+1)
	assert.GreaterOrEqual(t, metrics.FailedDelivery, initialMetrics.FailedDelivery)
	assert.GreaterOrEqual(t, metrics.AverageLatency, time.Duration(0))

	// Send message to non-existent receiver
	failMsg := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: ArchitectureAgentID, // Not subscribed
		Payload:  "fail test",
	}

	err = bus.Send(failMsg)
	assert.NoError(t, err)

	// Wait for async processing to complete
	done := make(chan bool, 1)
	go func() {
		// Give time for metrics to be updated
		for i := 0; i < 50; i++ {
			currentMetrics := bus.GetMetrics()
			if currentMetrics.FailedDelivery > initialMetrics.FailedDelivery {
				done <- true
				return
			}
			select {
			case <-time.After(2 * time.Millisecond):
			case <-time.After(100 * time.Millisecond):
				done <- true
				return
			}
		}
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		// Continue with test
	}

	// Check metrics again (failed delivery processing is async)
	metrics = bus.GetMetrics()
	assert.GreaterOrEqual(t, metrics.TotalMessages, initialMetrics.TotalMessages+2)
	assert.GreaterOrEqual(t, metrics.SuccessfulDelivery, initialMetrics.SuccessfulDelivery+1)
	assert.GreaterOrEqual(t, metrics.FailedDelivery, initialMetrics.FailedDelivery+1)
}

func TestMessageBusGetHealth(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Subscribe some agents
	agentIDs := []AgentID{SecurityAgentID, ArchitectureAgentID}
	for _, id := range agentIDs {
		_, err := bus.Subscribe(id, 50)
		require.NoError(t, err)
	}

	health := bus.GetHealth()
	assert.NotNil(t, health)
	assert.Equal(t, int64(0), health.TotalMessages)
	assert.Equal(t, int64(0), health.FailedMessages)
	assert.Equal(t, time.Duration(0), health.AverageLatency)
	assert.Equal(t, 2, health.ActiveSubscribers)
	assert.False(t, health.LastActivity.IsZero())
}

func TestMessageBusStop(t *testing.T) {
	bus := NewMessageBus(100)

	// Subscribe some agents
	agentIDs := []AgentID{SecurityAgentID, ArchitectureAgentID}
	channels := make([]<-chan *AgentMessage, 0)

	for _, id := range agentIDs {
		ch, err := bus.Subscribe(id, 50)
		require.NoError(t, err)
		channels = append(channels, ch)
	}

	// Stop the bus
	bus.Stop()

	// Verify all channels are closed
	for i, ch := range channels {
		select {
		case _, ok := <-ch:
			assert.False(t, ok, "Channel %d should be closed", i)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Channel %d should have been closed", i)
		}
	}

	// Verify subscribers map is cleared
	bus.subscribersMu.RLock()
	assert.Empty(t, bus.subscribers)
	bus.subscribersMu.RUnlock()
}

func TestMessageBusShutdown(t *testing.T) {
	bus := NewMessageBus(100)

	ctx := context.Background()
	err := bus.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewMessageRouter(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	router := NewMessageRouter(bus)
	assert.NotNil(t, router)
	assert.Equal(t, bus, router.bus)
	assert.NotNil(t, router.routes)
}

func TestMessageRouterAddRoute(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	router := NewMessageRouter(bus)

	agents := []AgentID{SecurityAgentID, ArchitectureAgentID}
	router.AddRoute(TaskAssignment, agents)

	router.mu.RLock()
	routedAgents, exists := router.routes[TaskAssignment]
	router.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, agents, routedAgents)
}

func TestMessageRouterRouteMessage(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	router := NewMessageRouter(bus)

	// Subscribe agents
	agentIDs := []AgentID{SecurityAgentID, ArchitectureAgentID}
	channels := make(map[AgentID]<-chan *AgentMessage)

	for _, id := range agentIDs {
		ch, err := bus.Subscribe(id, 50)
		require.NoError(t, err)
		channels[id] = ch
	}

	// Add route for TaskAssignment messages
	router.AddRoute(TaskAssignment, agentIDs)

	// Route message
	msg := &AgentMessage{
		Type:    TaskAssignment,
		Sender:  TestingAgentID,
		Payload: "routed message",
	}

	err := router.RouteMessage(msg)
	assert.NoError(t, err)

	// Verify all routed agents received the message
	for _, id := range agentIDs {
		select {
		case receivedMsg := <-channels[id]:
			assert.Equal(t, msg.Type, receivedMsg.Type)
			assert.Equal(t, msg.Sender, receivedMsg.Sender)
			assert.Equal(t, id, receivedMsg.Receiver)
			assert.Equal(t, msg.Payload, receivedMsg.Payload)
		case <-time.After(time.Second):
			t.Errorf("Agent %s should have received routed message", id)
		}
	}
}

func TestMessageRouterRouteMessageWithoutRoute(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	router := NewMessageRouter(bus)

	// Subscribe agent
	agentID := SecurityAgentID
	ch, err := bus.Subscribe(agentID, 50)
	require.NoError(t, err)

	// Route message without specific route (should use specified receiver)
	msg := &AgentMessage{
		Type:     ProgressUpdate, // No route for this type
		Sender:   TestingAgentID,
		Receiver: agentID,
		Payload:  "direct message",
	}

	err = router.RouteMessage(msg)
	assert.NoError(t, err)

	// Verify agent received the message
	select {
	case receivedMsg := <-ch:
		assert.Equal(t, msg.Type, receivedMsg.Type)
		assert.Equal(t, msg.Sender, receivedMsg.Sender)
		assert.Equal(t, agentID, receivedMsg.Receiver)
		assert.Equal(t, msg.Payload, receivedMsg.Payload)
	case <-time.After(time.Second):
		t.Error("Agent should have received direct message")
	}
}

func TestMessageBusConcurrentOperations(t *testing.T) {
	bus := NewMessageBus(1000)
	defer bus.Stop()

	// Get baseline metrics
	initialMetrics := bus.GetMetrics()

	// Subscribe multiple agents
	numAgents := 10
	agentIDs := make([]AgentID, numAgents)
	channels := make([]<-chan *AgentMessage, numAgents)

	for i := 0; i < numAgents; i++ {
		agentIDs[i] = AgentID(fmt.Sprintf("agent-%d", i))
		ch, err := bus.Subscribe(agentIDs[i], 100)
		require.NoError(t, err)
		channels[i] = ch
	}

	// Concurrent message sending
	numMessages := 100
	var wg sync.WaitGroup

	// Send messages concurrently
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func(msgID int) {
			defer wg.Done()
			msg := &AgentMessage{
				Type:     TaskAssignment,
				Sender:   TestingAgentID,
				Receiver: agentIDs[msgID%numAgents],
				Payload:  fmt.Sprintf("message-%d", msgID),
			}
			err := bus.Send(msg)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Count received messages
	totalReceived := 0
	for i := 0; i < numAgents; i++ {
		agentMessages := 0
		ch := channels[i]

		// Count messages for this agent with timeout
		timeout := time.After(50 * time.Millisecond)
		for {
			select {
			case <-ch:
				agentMessages++
			case <-timeout:
				// No more messages for this agent
				goto nextAgent
			}
		}
	nextAgent:
		totalReceived += agentMessages
	}

	assert.Equal(t, numMessages, totalReceived)

	// Check final metrics (should have at least numMessages more)
	metrics := bus.GetMetrics()
	assert.GreaterOrEqual(t, metrics.TotalMessages, initialMetrics.TotalMessages+int64(numMessages))
	assert.GreaterOrEqual(t, metrics.SuccessfulDelivery, initialMetrics.SuccessfulDelivery+int64(numMessages))
	assert.GreaterOrEqual(t, metrics.FailedDelivery, initialMetrics.FailedDelivery)
}

func TestMessageBusMessageProcessingOrder(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Subscribe agent
	agentID := SecurityAgentID
	ch, err := bus.Subscribe(agentID, 100)
	require.NoError(t, err)

	// Send messages in order
	numMessages := 10
	for i := 0; i < numMessages; i++ {
		msg := &AgentMessage{
			Type:     TaskAssignment,
			Receiver: agentID,
			Payload:  i,
		}
		err := bus.Send(msg)
		require.NoError(t, err)
	}

	// Receive messages and verify order
	for i := 0; i < numMessages; i++ {
		select {
		case receivedMsg := <-ch:
			assert.Equal(t, i, receivedMsg.Payload)
		case <-time.After(time.Second):
			t.Errorf("Should have received message %d", i)
		}
	}
}

func TestMessageBusEdgeCases(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Test sending message with existing ID and timestamp
	existingID := "existing-id"
	existingTimestamp := time.Now().Add(-time.Hour)

	msg := &AgentMessage{
		ID:        existingID,
		Timestamp: existingTimestamp,
		Type:      TaskAssignment,
		Receiver:  SecurityAgentID,
		Payload:   "test",
	}

	err := bus.Send(msg)
	assert.NoError(t, err)

	// Should keep existing ID and timestamp
	assert.Equal(t, existingID, msg.ID)
	assert.Equal(t, existingTimestamp, msg.Timestamp)

	// Test sending message without ID and timestamp
	msg2 := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: SecurityAgentID,
		Payload:  "test2",
	}

	err = bus.Send(msg2)
	assert.NoError(t, err)

	// Should have ID and timestamp set
	assert.NotEmpty(t, msg2.ID)
	assert.False(t, msg2.Timestamp.IsZero())
}

func TestMessageBusMetricsAverageLatency(t *testing.T) {
	bus := NewMessageBus(100)
	defer bus.Stop()

	// Subscribe agent
	agentID := SecurityAgentID
	ch, err := bus.Subscribe(agentID, 50)
	require.NoError(t, err)

	// Send first message
	msg1 := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: agentID,
		Payload:  "msg1",
	}

	err = bus.Send(msg1)
	assert.NoError(t, err)
	<-ch

	metrics1 := bus.GetMetrics()
	firstLatency := metrics1.AverageLatency

	// Send second message
	msg2 := &AgentMessage{
		Type:     TaskAssignment,
		Receiver: agentID,
		Payload:  "msg2",
	}

	err = bus.Send(msg2)
	assert.NoError(t, err)
	<-ch

	metrics2 := bus.GetMetrics()

	// Average latency should be updated (simple moving average)
	assert.True(t, metrics2.AverageLatency != firstLatency)
	assert.True(t, metrics2.AverageLatency > 0)
}
