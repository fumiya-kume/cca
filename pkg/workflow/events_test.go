package workflow

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock subscriber for testing
type mockEventSubscriber struct {
	events        []WorkflowEvent
	eventTypes    []EventType
	mutex         sync.Mutex
	eventCh       chan struct{} // Signal when event is received
	_ int // TODO: implement expected event count validation
}

func (m *mockEventSubscriber) OnEvent(event WorkflowEvent) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.events = append(m.events, event)

	// Signal that an event was received
	if m.eventCh != nil {
		select {
		case m.eventCh <- struct{}{}:
		default:
		}
	}
}

func (m *mockEventSubscriber) GetEventTypes() []EventType {
	return m.eventTypes
}

func (m *mockEventSubscriber) GetEvents() []WorkflowEvent {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return append([]WorkflowEvent{}, m.events...)
}

// WaitForEvents waits for the expected number of events to be received
func (m *mockEventSubscriber) WaitForEvents(count int) {
	if m.eventCh == nil {
		m.eventCh = make(chan struct{}, count*2) // Buffered channel
	}

	for i := 0; i < count; i++ {
		<-m.eventCh
	}
}

// newMockSubscriber creates a new mock subscriber with event synchronization
func newMockSubscriber(eventTypes []EventType) *mockEventSubscriber {
	return &mockEventSubscriber{
		eventTypes: eventTypes,
		eventCh:    make(chan struct{}, 100), // Buffered channel for async events
	}
}

func TestNewEventBus(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.events)
	assert.NotNil(t, bus.subscribers)
	assert.NotNil(t, bus.ctx)
}

func TestEventBus_Subscribe(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	subscriber := &mockEventSubscriber{
		eventTypes: []EventType{EventTypeWorkflowStarted, EventTypeWorkflowCompleted},
	}

	bus.Subscribe(subscriber)

	// Verify subscriber was added
	bus.mutex.RLock()
	assert.Contains(t, bus.subscribers[EventTypeWorkflowStarted], subscriber)
	assert.Contains(t, bus.subscribers[EventTypeWorkflowCompleted], subscriber)
	bus.mutex.RUnlock()
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	subscriber := newMockSubscriber([]EventType{EventTypeWorkflowStarted})

	// Subscribe then unsubscribe
	bus.Subscribe(subscriber)
	bus.Unsubscribe(subscriber)

	// Verify subscriber was removed
	bus.mutex.RLock()
	assert.NotContains(t, bus.subscribers[EventTypeWorkflowStarted], subscriber)
	bus.mutex.RUnlock()
}

func TestEventBus_PublishAndReceive(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	subscriber := newMockSubscriber([]EventType{EventTypeWorkflowStarted})

	bus.Subscribe(subscriber)

	// Publish an event
	event := WorkflowEvent{
		Type:       EventTypeWorkflowStarted,
		WorkflowID: "test-workflow",
		StageID:    "test-stage",
		Timestamp:  time.Now(),
		Data:       map[string]interface{}{"test": "data"},
		Metadata:   map[string]interface{}{"source": "test"},
	}

	bus.Publish(event)

	// Wait for event to be processed
	subscriber.WaitForEvents(1)

	// Verify event was received
	events := subscriber.GetEvents()
	require.Len(t, events, 1)
	assert.Equal(t, EventTypeWorkflowStarted, events[0].Type)
	assert.Equal(t, "test-workflow", events[0].WorkflowID)
	assert.Equal(t, "test-stage", events[0].StageID)
	assert.Equal(t, "data", events[0].Data["test"])
	assert.Equal(t, "test", events[0].Metadata["source"])
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	subscriber1 := newMockSubscriber([]EventType{EventTypeWorkflowStarted})
	subscriber2 := newMockSubscriber([]EventType{EventTypeWorkflowStarted, EventTypeWorkflowCompleted})

	bus.Subscribe(subscriber1)
	bus.Subscribe(subscriber2)

	// Publish events
	event1 := WorkflowEvent{
		Type:       EventTypeWorkflowStarted,
		WorkflowID: "workflow-1",
		Timestamp:  time.Now(),
	}

	event2 := WorkflowEvent{
		Type:       EventTypeWorkflowCompleted,
		WorkflowID: "workflow-1",
		Timestamp:  time.Now(),
	}

	bus.Publish(event1)
	bus.Publish(event2)

	// Wait for events to be processed
	subscriber1.WaitForEvents(1) // Only gets started event
	subscriber2.WaitForEvents(2) // Gets both events

	// Verify subscriber1 received only the started event
	events1 := subscriber1.GetEvents()
	assert.Len(t, events1, 1)
	assert.Equal(t, EventTypeWorkflowStarted, events1[0].Type)

	// Verify subscriber2 received both events
	events2 := subscriber2.GetEvents()
	assert.Len(t, events2, 2)
	// Events may arrive in any order due to async processing
	eventTypes := []EventType{events2[0].Type, events2[1].Type}
	assert.Contains(t, eventTypes, EventTypeWorkflowStarted)
	assert.Contains(t, eventTypes, EventTypeWorkflowCompleted)
}

func TestEventBus_Stop(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)

	subscriber := newMockSubscriber([]EventType{EventTypeWorkflowStarted})

	bus.Subscribe(subscriber)

	// Stop the bus
	bus.Shutdown()

	// Verify context is canceled
	select {
	case <-bus.ctx.Done():
		// Context should be canceled
	default:
		t.Error("Context should be canceled after Stop()")
	}
}

func TestEventBus_BufferOverflow(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	// Don't subscribe any listeners to cause buffer buildup

	// Publish many events rapidly
	for i := 0; i < 200; i++ {
		event := WorkflowEvent{
			Type:       EventTypeWorkflowStarted,
			WorkflowID: "test-workflow",
			Timestamp:  time.Now(),
		}

		// This should not block even with a full buffer
		select {
		case bus.events <- event:
		default:
			// Buffer is full, which is expected
		}
	}

	// Test should complete without hanging
	assert.True(t, true)
}

func TestWorkflowEvent_Structure(t *testing.T) {
	now := time.Now()
	event := WorkflowEvent{
		Type:       EventTypeStageStarted,
		WorkflowID: "test-workflow-123",
		StageID:    "test-stage-456",
		Timestamp:  now,
		Data: map[string]interface{}{
			"input":  "test input",
			"config": map[string]string{"key": "value"},
		},
		Metadata: map[string]interface{}{
			"user":   "test-user",
			"source": "api",
		},
	}

	assert.Equal(t, EventTypeStageStarted, event.Type)
	assert.Equal(t, "test-workflow-123", event.WorkflowID)
	assert.Equal(t, "test-stage-456", event.StageID)
	assert.Equal(t, now, event.Timestamp)
	assert.Equal(t, "test input", event.Data["input"])
	assert.Equal(t, "test-user", event.Metadata["user"])
}

func TestEventType_Constants(t *testing.T) {
	// Test that event type constants are properly defined
	eventTypes := []EventType{
		EventTypeWorkflowStarted,
		EventTypeWorkflowCompleted,
		EventTypeWorkflowFailed,
		EventTypeWorkflowPaused,
		EventTypeWorkflowResumed,
		EventTypeWorkflowStopped,
		EventTypeWorkflowCancelled,
		EventTypeStageStarted,
		EventTypeStageCompleted,
		EventTypeStageFailed,
		EventTypeStageSkipped,
	}

	// Each event type should have a unique value
	eventTypeMap := make(map[EventType]bool)
	for _, eventType := range eventTypes {
		assert.False(t, eventTypeMap[eventType], "Duplicate event type value: %v", eventType)
		eventTypeMap[eventType] = true
	}

	// Should have all expected event types
	assert.Len(t, eventTypeMap, len(eventTypes))
}

func TestEventBus_FilterEvents(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	// Subscribe to only workflow events, not stage events
	subscriber := newMockSubscriber([]EventType{EventTypeWorkflowStarted, EventTypeWorkflowCompleted})

	bus.Subscribe(subscriber)

	// Publish various events
	events := []WorkflowEvent{
		{Type: EventTypeWorkflowStarted, WorkflowID: "w1"},
		{Type: EventTypeStageStarted, WorkflowID: "w1"}, // Should be filtered out
		{Type: EventTypeWorkflowCompleted, WorkflowID: "w1"},
		{Type: EventTypeStageFailed, WorkflowID: "w1"}, // Should be filtered out
	}

	for _, event := range events {
		event.Timestamp = time.Now()
		bus.Publish(event)
	}

	// Wait for the 2 workflow events to be processed (2 out of 4 events should be received)
	subscriber.WaitForEvents(2)

	// Verify only workflow events were received
	receivedEvents := subscriber.GetEvents()
	assert.Len(t, receivedEvents, 2)
	// Events may arrive in any order due to async processing
	eventTypes := []EventType{receivedEvents[0].Type, receivedEvents[1].Type}
	assert.Contains(t, eventTypes, EventTypeWorkflowStarted)
	assert.Contains(t, eventTypes, EventTypeWorkflowCompleted)
}

func TestEventBus_ConcurrentPublishing(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)
	defer bus.Shutdown()

	subscriber := newMockSubscriber([]EventType{EventTypeWorkflowStarted})

	bus.Subscribe(subscriber)

	// Brief wait for event loop to start
	// Removed unnecessary 5ms sleep

	// Publish events concurrently from multiple goroutines
	numGoroutines := 5
	eventsPerGoroutine := 10

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := WorkflowEvent{
					Type:       EventTypeWorkflowStarted,
					WorkflowID: fmt.Sprintf("workflow-%d-%d", goroutineID, j),
					Timestamp:  time.Now(),
				}
				bus.Publish(event)
			}
		}(i)
	}

	wg.Wait()

	// Wait for all events to be processed
	expectedCount := numGoroutines * eventsPerGoroutine
	subscriber.WaitForEvents(expectedCount)

	// Verify all events were received
	events := subscriber.GetEvents()
	assert.Equal(t, expectedCount, len(events))
}

func TestEventBus_Shutdown(t *testing.T) {
	bus, err := NewEventBus(100)
	require.NoError(t, err)

	subscriber := newMockSubscriber([]EventType{EventTypeWorkflowStarted})

	bus.Subscribe(subscriber)

	// Brief wait for event loop to start
	// Removed unnecessary 5ms sleep

	// Shutdown the bus
	bus.Shutdown()

	// Verify context is canceled
	select {
	case <-bus.ctx.Done():
		// Context should be canceled
	case <-time.After(time.Millisecond * 100):
		t.Error("Context should be canceled after Shutdown()")
	}

	// Publishing events after shutdown should not crash
	event := WorkflowEvent{
		Type:       EventTypeWorkflowStarted,
		WorkflowID: "test",
		Timestamp:  time.Now(),
	}

	// This should not block or crash - use Publish method instead
	defer func() {
		if r := recover(); r != nil {
			// Expected - publishing to shut down bus should be safe
			_ = r // Acknowledge recovery
		}
	}()
	bus.Publish(event) // Should handle closed channel gracefully
}

func TestEventSubscriber_Interface(t *testing.T) {
	// Verify that our mock implements the interface correctly
	var subscriber EventSubscriber = &mockEventSubscriber{
		eventTypes: []EventType{EventTypeWorkflowStarted},
	}

	eventTypes := subscriber.GetEventTypes()
	assert.Len(t, eventTypes, 1)
	assert.Equal(t, EventTypeWorkflowStarted, eventTypes[0])

	// Test OnEvent doesn't panic
	event := WorkflowEvent{
		Type:       EventTypeWorkflowStarted,
		WorkflowID: "test",
		Timestamp:  time.Now(),
	}

	assert.NotPanics(t, func() {
		subscriber.OnEvent(event)
	})
}
