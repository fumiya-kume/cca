package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/google/uuid"
)

// MessageBus handles message routing between agents
type MessageBus struct {
	subscribers   map[AgentID]chan *AgentMessage
	subscribersMu sync.RWMutex

	messages      chan *AgentMessage
	messageBuffer int

	metrics *MessageBusMetrics
	clock   clock.Clock

	logger *logger.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// MessageBusMetrics tracks message bus performance
type MessageBusMetrics struct {
	TotalMessages      int64
	SuccessfulDelivery int64
	FailedDelivery     int64
	AverageLatency     time.Duration
	mu                 sync.RWMutex
}

// NewMessageBus creates a new message bus instance
func NewMessageBus(bufferSize int) *MessageBus {
	return NewMessageBusWithClock(bufferSize, clock.NewRealClock())
}

// NewMessageBusWithClock creates a new message bus instance with a custom clock
func NewMessageBusWithClock(bufferSize int, clk clock.Clock) *MessageBus {
	ctx, cancel := context.WithCancel(context.Background())

	mb := &MessageBus{
		subscribers:   make(map[AgentID]chan *AgentMessage),
		messages:      make(chan *AgentMessage, bufferSize),
		messageBuffer: bufferSize,
		metrics:       &MessageBusMetrics{},
		clock:         clk,
		logger:        logger.GetLogger(),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start message routing goroutine
	mb.wg.Add(1)
	go mb.routeMessages()

	return mb
}

// Subscribe registers an agent to receive messages
func (mb *MessageBus) Subscribe(agentID AgentID, bufferSize int) (<-chan *AgentMessage, error) {
	mb.subscribersMu.Lock()
	defer mb.subscribersMu.Unlock()

	if _, exists := mb.subscribers[agentID]; exists {
		return nil, fmt.Errorf("agent %s is already subscribed", agentID)
	}

	ch := make(chan *AgentMessage, bufferSize)
	mb.subscribers[agentID] = ch

	mb.logger.Info("Agent subscribed to message bus (agent_id: %s)", agentID)

	return ch, nil
}

// Unsubscribe removes an agent from message bus
func (mb *MessageBus) Unsubscribe(agentID AgentID) error {
	mb.subscribersMu.Lock()
	defer mb.subscribersMu.Unlock()

	ch, exists := mb.subscribers[agentID]
	if !exists {
		return fmt.Errorf("agent %s is not subscribed", agentID)
	}

	close(ch)
	delete(mb.subscribers, agentID)

	mb.logger.Info("Agent unsubscribed from message bus (agent_id: %s)", agentID)

	return nil
}

// Send publishes a message to the message bus
func (mb *MessageBus) Send(msg *AgentMessage) error {
	// Check if message bus is shut down first
	select {
	case <-mb.ctx.Done():
		return fmt.Errorf("message bus is shutting down")
	default:
	}

	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	if msg.Timestamp.IsZero() {
		msg.Timestamp = mb.clock.Now()
	}

	// Safely try to send to channel
	defer func() {
		if r := recover(); r != nil {
			// Channel was closed during send
			_ = r // Acknowledge recovery
		}
	}()

	select {
	case mb.messages <- msg:
		mb.updateMetrics(true, 0)
		return nil
	case <-mb.ctx.Done():
		return fmt.Errorf("message bus is shutting down")
	default:
		mb.updateMetrics(false, 0)
		return fmt.Errorf("message bus buffer is full")
	}
}

// Broadcast sends a message to all subscribed agents
func (mb *MessageBus) Broadcast(msg *AgentMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}

	if msg.Timestamp.IsZero() {
		msg.Timestamp = mb.clock.Now()
	}

	// Get snapshot of subscribers with safe channel references
	mb.subscribersMu.RLock()
	var deliveries []struct {
		agentID AgentID
		ch      chan *AgentMessage
	}
	for agentID, ch := range mb.subscribers {
		deliveries = append(deliveries, struct {
			agentID AgentID
			ch      chan *AgentMessage
		}{agentID, ch})
	}
	mb.subscribersMu.RUnlock()

	// Use direct delivery to avoid recursive Send() calls
	for _, delivery := range deliveries {
		// Check if we're shutting down before each delivery
		select {
		case <-mb.ctx.Done():
			return fmt.Errorf("message bus is shutting down")
		default:
		}

		broadcastMsg := *msg
		broadcastMsg.Receiver = delivery.agentID

		start := mb.clock.Now()

		// Safely deliver with panic recovery for closed channels
		func() {
			defer func() {
				if r := recover(); r != nil {
					mb.logger.Warn("Channel closed during broadcast delivery (receiver: %s)", delivery.agentID)
					mb.updateMetrics(false, time.Since(start))
				}
			}()

			select {
			case delivery.ch <- &broadcastMsg:
				mb.updateMetrics(true, time.Since(start))
				mb.logger.Debug("Broadcast message delivered (receiver: %s, message_id: %s)", delivery.agentID, broadcastMsg.ID)
			case <-time.After(5 * time.Second):
				mb.logger.Error("Broadcast message delivery timeout (receiver: %s, message_id: %s)", delivery.agentID, broadcastMsg.ID)
				mb.updateMetrics(false, time.Since(start))
			case <-mb.ctx.Done():
				// Context canceled during delivery
			}
		}()
	}

	return nil
}

// routeMessages handles message routing to subscribers
func (mb *MessageBus) routeMessages() {
	defer mb.wg.Done()

	for {
		select {
		case msg := <-mb.messages:
			mb.deliverMessage(msg)

		case <-mb.ctx.Done():
			mb.logger.Info("Message bus routing stopped")
			return
		}
	}
}

// deliverMessage delivers a message to the target agent
func (mb *MessageBus) deliverMessage(msg *AgentMessage) {
	start := mb.clock.Now()

	mb.subscribersMu.RLock()
	defer mb.subscribersMu.RUnlock()

	ch, exists := mb.subscribers[msg.Receiver]
	if !exists {
		mb.logger.Warn("No subscriber found for message (receiver: %s, message_id: %s, message_type: %s)", msg.Receiver, msg.ID, msg.Type)
		mb.updateMetrics(false, time.Since(start))
		return
	}

	select {
	case ch <- msg:
		mb.updateMetrics(true, time.Since(start))
		mb.logger.Debug("Message delivered (receiver: %s, message_id: %s, message_type: %s, latency: %v)", msg.Receiver, msg.ID, msg.Type, time.Since(start))

	case <-time.After(5 * time.Second):
		mb.logger.Error("Message delivery timeout (receiver: %s, message_id: %s, message_type: %s)", msg.Receiver, msg.ID, msg.Type)
		mb.updateMetrics(false, time.Since(start))

	case <-mb.ctx.Done():
		mb.logger.Info("Message delivery canceled due to shutdown")
	}
}

// updateMetrics updates message bus metrics
func (mb *MessageBus) updateMetrics(success bool, latency time.Duration) {
	mb.metrics.mu.Lock()
	defer mb.metrics.mu.Unlock()

	mb.metrics.TotalMessages++

	if success {
		mb.metrics.SuccessfulDelivery++

		// Update average latency
		if mb.metrics.AverageLatency == 0 {
			mb.metrics.AverageLatency = latency
		} else {
			// Simple moving average
			mb.metrics.AverageLatency = (mb.metrics.AverageLatency + latency) / 2
		}
	} else {
		mb.metrics.FailedDelivery++
	}
}

// GetMetrics returns current message bus metrics
func (mb *MessageBus) GetMetrics() MessageBusMetrics {
	mb.metrics.mu.RLock()
	defer mb.metrics.mu.RUnlock()

	// Return a copy to avoid lock copying
	return MessageBusMetrics{
		TotalMessages:      mb.metrics.TotalMessages,
		SuccessfulDelivery: mb.metrics.SuccessfulDelivery,
		FailedDelivery:     mb.metrics.FailedDelivery,
		AverageLatency:     mb.metrics.AverageLatency,
	}
}

// Stop gracefully shuts down the message bus
func (mb *MessageBus) Stop() {
	mb.logger.Info("Stopping message bus")

	mb.cancel()

	// Close all subscriber channels safely
	mb.subscribersMu.Lock()
	for agentID, ch := range mb.subscribers {
		// Safely close channel with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					mb.logger.Debug("Channel already closed during shutdown (agent_id: %s)", agentID)
				}
			}()
			close(ch)
			mb.logger.Debug("Closed subscriber channel (agent_id: %s)", agentID)
		}()
	}
	mb.subscribers = make(map[AgentID]chan *AgentMessage)
	mb.subscribersMu.Unlock()

	// Wait for routing goroutine to finish
	mb.wg.Wait()

	// Close messages channel
	close(mb.messages)

	mb.logger.Info("Message bus stopped")
}

// Shutdown gracefully shuts down the message bus with context
func (mb *MessageBus) Shutdown(ctx context.Context) error {
	mb.Stop()
	return nil
}

// GetHealth returns message bus health information
func (mb *MessageBus) GetHealth() *MessageBusHealth {
	mb.metrics.mu.RLock()
	defer mb.metrics.mu.RUnlock()

	mb.subscribersMu.RLock()
	activeSubscribers := len(mb.subscribers)
	mb.subscribersMu.RUnlock()

	return &MessageBusHealth{
		TotalMessages:     mb.metrics.TotalMessages,
		FailedMessages:    mb.metrics.FailedDelivery,
		AverageLatency:    mb.metrics.AverageLatency,
		ActiveSubscribers: activeSubscribers,
		LastActivity:      time.Now(),
	}
}

// MessageRouter provides advanced routing capabilities
type MessageRouter struct {
	bus    *MessageBus
	routes map[MessageType][]AgentID
	mu     sync.RWMutex
	logger *logger.Logger
}

// NewMessageRouter creates a new message router
func NewMessageRouter(bus *MessageBus) *MessageRouter {
	return &MessageRouter{
		bus:    bus,
		routes: make(map[MessageType][]AgentID),
		logger: logger.GetLogger(),
	}
}

// AddRoute adds a routing rule for a message type
func (mr *MessageRouter) AddRoute(msgType MessageType, agents []AgentID) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.routes[msgType] = agents
	mr.logger.Info("Route added (message_type: %s, agents: %v)", msgType, agents)
}

// RouteMessage routes a message based on configured rules
func (mr *MessageRouter) RouteMessage(msg *AgentMessage) error {
	mr.mu.RLock()
	agents, exists := mr.routes[msg.Type]
	mr.mu.RUnlock()

	if !exists {
		// If no specific route, send to the specified receiver
		return mr.bus.Send(msg)
	}

	// Get snapshot of required subscribers for safe access
	mr.bus.subscribersMu.RLock()
	var routeDeliveries []struct {
		agentID AgentID
		ch      chan *AgentMessage
	}
	for _, agentID := range agents {
		if ch, exists := mr.bus.subscribers[agentID]; exists {
			routeDeliveries = append(routeDeliveries, struct {
				agentID AgentID
				ch      chan *AgentMessage
			}{agentID, ch})
		} else {
			mr.logger.Warn("Route target agent not found (agent_id: %s, message_type: %s)", agentID, msg.Type)
		}
	}
	mr.bus.subscribersMu.RUnlock()

	// Route to specified agents with safe delivery
	for _, delivery := range routeDeliveries {
		routedMsg := *msg
		routedMsg.Receiver = delivery.agentID

		// Set message metadata if missing
		if routedMsg.ID == "" {
			routedMsg.ID = uuid.New().String()
		}
		if routedMsg.Timestamp.IsZero() {
			routedMsg.Timestamp = mr.bus.clock.Now()
		}

		// Safe delivery with panic recovery
		start := mr.bus.clock.Now()
		func() {
			defer func() {
				if r := recover(); r != nil {
					mr.logger.Warn("Channel closed during route delivery (agent_id: %s)", delivery.agentID)
					mr.bus.updateMetrics(false, mr.bus.clock.Since(start))
				}
			}()

			select {
			case delivery.ch <- &routedMsg:
				mr.bus.updateMetrics(true, mr.bus.clock.Since(start))
				mr.logger.Debug("Routed message delivered (agent_id: %s, message_type: %s)", delivery.agentID, msg.Type)
			case <-mr.bus.clock.After(5 * time.Second):
				mr.logger.Error("Routed message delivery timeout (agent_id: %s, message_type: %s)", delivery.agentID, msg.Type)
				mr.bus.updateMetrics(false, mr.bus.clock.Since(start))
			case <-mr.bus.ctx.Done():
				// Context canceled during delivery
			}
		}()
	}

	return nil
}
