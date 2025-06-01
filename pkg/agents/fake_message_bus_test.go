package agents

import (
	"context"
	"sync"
)

// FakeMessageBus is a fast synchronous message bus for testing
type FakeMessageBus struct {
	messages []AgentMessage
	mu       sync.Mutex
	handlers map[AgentID]func(*AgentMessage)
}

// NewFakeMessageBus creates a fast fake message bus
func NewFakeMessageBus() *FakeMessageBus {
	return &FakeMessageBus{
		messages: make([]AgentMessage, 0),
		handlers: make(map[AgentID]func(*AgentMessage)),
	}
}

// Send immediately delivers the message synchronously
func (f *FakeMessageBus) Send(msg *AgentMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.messages = append(f.messages, *msg)

	// Immediately call handler if registered
	if handler, exists := f.handlers[msg.Receiver]; exists {
		go handler(msg) // Still async to avoid deadlocks
	}

	return nil
}

// Subscribe registers a handler for an agent
func (f *FakeMessageBus) Subscribe(agentID AgentID, handler func(*AgentMessage)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[agentID] = handler
}

// GetMessages returns all sent messages
func (f *FakeMessageBus) GetMessages() []AgentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]AgentMessage, len(f.messages))
	copy(result, f.messages)
	return result
}

// Stop does nothing for fake bus
func (f *FakeMessageBus) Stop() {}

// Clear removes all messages
func (f *FakeMessageBus) Clear() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = f.messages[:0]
}

// FakeAgent is a minimal fast agent for testing
type FakeAgent struct {
	id           AgentID
	status       AgentStatus
	capabilities []string
	mu           sync.Mutex
}

// NewFakeAgent creates a fast fake agent
func NewFakeAgent(id AgentID) *FakeAgent {
	return &FakeAgent{
		id:           id,
		status:       StatusOffline,
		capabilities: []string{"test_capability"},
	}
}

func (f *FakeAgent) GetID() AgentID {
	return f.id
}

func (f *FakeAgent) GetStatus() AgentStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *FakeAgent) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = StatusIdle
	return nil
}

func (f *FakeAgent) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = StatusOffline
	return nil
}

func (f *FakeAgent) ProcessMessage(ctx context.Context, msg *AgentMessage) error {
	// Fast no-op processing
	return nil
}

func (f *FakeAgent) GetCapabilities() []string {
	return f.capabilities
}

func (f *FakeAgent) HealthCheck(ctx context.Context) error {
	return nil
}
