package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	id           AgentID
	status       AgentStatus
	capabilities []string
	config       AgentConfig
	logger       *logger.Logger

	// Message handling
	messageBus  *MessageBus
	messageChan <-chan *AgentMessage

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// Metrics
	metrics *AgentMetrics

	// Message handlers
	handlers map[MessageType]MessageHandler
}

// MessageHandler processes a specific type of message
type MessageHandler func(ctx context.Context, msg *AgentMessage) error

// AgentMetrics tracks agent performance metrics
type AgentMetrics struct {
	MessagesReceived   int64
	MessagesProcessed  int64
	MessagesFailed     int64
	TotalProcessTime   time.Duration
	AverageProcessTime time.Duration
	LastProcessTime    time.Time
	mu                 sync.RWMutex
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(id AgentID, config AgentConfig, messageBus *MessageBus) (*BaseAgent, error) {
	if messageBus == nil {
		return nil, fmt.Errorf("message bus cannot be nil")
	}

	// Subscribe to message bus
	messageChan, err := messageBus.Subscribe(id, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to message bus: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	agent := &BaseAgent{
		id:          id,
		status:      StatusOffline,
		config:      config,
		logger:      logger.GetLogger(),
		messageBus:  messageBus,
		messageChan: messageChan,
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &AgentMetrics{},
		handlers:    make(map[MessageType]MessageHandler),
	}

	// Register default handlers
	agent.registerDefaultHandlers()

	return agent, nil
}

// GetID returns the agent's ID
func (ba *BaseAgent) GetID() AgentID {
	return ba.id
}

// GetStatus returns the agent's current status
func (ba *BaseAgent) GetStatus() AgentStatus {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.status
}

// SetStatus updates the agent's status
func (ba *BaseAgent) SetStatus(status AgentStatus) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	oldStatus := ba.status
	ba.status = status

	if oldStatus != status {
		ba.logger.Info("Agent status changed (old_status: %s, new_status: %s)", oldStatus, status)
	}
}

// GetCapabilities returns the agent's capabilities
func (ba *BaseAgent) GetCapabilities() []string {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	capsCopy := make([]string, len(ba.capabilities))
	copy(capsCopy, ba.capabilities)
	return capsCopy
}

// SetCapabilities sets the agent's capabilities
func (ba *BaseAgent) SetCapabilities(capabilities []string) {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.capabilities = capabilities
}

// Start starts the agent
func (ba *BaseAgent) Start(ctx context.Context) error {
	ba.SetStatus(StatusStarting)

	// Start message processing
	ba.wg.Add(1)
	go ba.processMessages()

	ba.SetStatus(StatusIdle)
	ba.logger.Info("Agent started")

	return nil
}

// Stop stops the agent
func (ba *BaseAgent) Stop(ctx context.Context) error {
	ba.SetStatus(StatusStopping)

	// Cancel context to stop goroutines
	ba.cancel()

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		ba.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown
	case <-ctx.Done():
		// Timeout
		ba.logger.Warn("Agent stop timeout")
	}

	// Unsubscribe from message bus
	if err := ba.messageBus.Unsubscribe(ba.id); err != nil {
		ba.logger.Error("Failed to unsubscribe from message bus (error: %v)", err)
	}

	ba.SetStatus(StatusOffline)
	ba.logger.Info("Agent stopped")

	return nil
}

// ProcessMessage processes an incoming message
func (ba *BaseAgent) ProcessMessage(ctx context.Context, msg *AgentMessage) error {
	if msg == nil {
		return fmt.Errorf("message cannot be nil")
	}

	// Update metrics
	ba.metrics.mu.Lock()
	ba.metrics.MessagesReceived++
	ba.metrics.mu.Unlock()

	startTime := time.Now()

	// Find handler for message type
	ba.mu.RLock()
	handler, exists := ba.handlers[msg.Type]
	ba.mu.RUnlock()

	if !exists {
		ba.logger.Warn("No handler for message type (message_type: %s, message_id: %s)", msg.Type, msg.ID)
		// Update metrics for failed message
		processTime := time.Since(startTime)
		ba.updateMetrics(false, processTime)
		return fmt.Errorf("no handler for message type %s", msg.Type)
	}

	// Execute handler
	err := handler(ctx, msg)

	// Update metrics
	processTime := time.Since(startTime)
	ba.updateMetrics(err == nil, processTime)

	return err
}

// HealthCheck performs a health check
func (ba *BaseAgent) HealthCheck(ctx context.Context) error {
	// Check if agent is running
	status := ba.GetStatus()
	if status == StatusOffline || status == StatusError {
		return fmt.Errorf("agent is not healthy: status=%s", status)
	}

	// Check message processing
	ba.metrics.mu.RLock()
	lastProcess := ba.metrics.LastProcessTime
	ba.metrics.mu.RUnlock()

	if !lastProcess.IsZero() && time.Since(lastProcess) > 5*time.Minute {
		return fmt.Errorf("no messages processed in last 5 minutes")
	}

	return nil
}

// RegisterHandler registers a message handler
func (ba *BaseAgent) RegisterHandler(msgType MessageType, handler MessageHandler) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.handlers[msgType] = handler
	ba.logger.Debug("Handler registered (message_type: %s)", msgType)
}

// registerDefaultHandlers registers default message handlers
func (ba *BaseAgent) registerDefaultHandlers() {
	// Health check handler
	ba.RegisterHandler(HealthCheck, func(ctx context.Context, msg *AgentMessage) error {
		// Perform health check and send response
		err := ba.HealthCheck(ctx)

		response := &AgentMessage{
			Type:          ResultReporting,
			Sender:        ba.id,
			Receiver:      msg.Sender,
			CorrelationID: msg.ID,
			Payload: map[string]interface{}{
				"healthy": err == nil,
				"error":   err,
			},
			Priority: msg.Priority,
		}

		return ba.messageBus.Send(response)
	})

	// Progress update handler (for monitoring)
	ba.RegisterHandler(ProgressUpdate, func(ctx context.Context, msg *AgentMessage) error {
		ba.logger.Debug("Progress update received (payload: %v)", msg.Payload)
		return nil
	})
}

// processMessages processes incoming messages
func (ba *BaseAgent) processMessages() {
	defer ba.wg.Done()

	for {
		select {
		case msg := <-ba.messageChan:
			if msg == nil {
				continue
			}

			// Set status to busy while processing
			ba.SetStatus(StatusBusy)

			// Process message
			if err := ba.ProcessMessage(ba.ctx, msg); err != nil {
				ba.logger.Error("Failed to process message (message_id: %s, message_type: %s, error: %v)", msg.ID, msg.Type, err)
			}

			// Set status back to idle
			ba.SetStatus(StatusIdle)

		case <-ba.ctx.Done():
			ba.logger.Info("Message processing stopped")
			return
		}
	}
}

// updateMetrics updates agent metrics
func (ba *BaseAgent) updateMetrics(success bool, processTime time.Duration) {
	ba.metrics.mu.Lock()
	defer ba.metrics.mu.Unlock()

	if success {
		ba.metrics.MessagesProcessed++
	} else {
		ba.metrics.MessagesFailed++
	}

	ba.metrics.TotalProcessTime += processTime
	ba.metrics.LastProcessTime = time.Now()

	// Calculate average
	totalMessages := ba.metrics.MessagesProcessed + ba.metrics.MessagesFailed
	if totalMessages > 0 {
		ba.metrics.AverageProcessTime = ba.metrics.TotalProcessTime / time.Duration(totalMessages)
	}
}

// GetMetrics returns current agent metrics
func (ba *BaseAgent) GetMetrics() AgentMetrics {
	ba.metrics.mu.RLock()
	defer ba.metrics.mu.RUnlock()

	// Return a copy to avoid lock copying
	return AgentMetrics{
		MessagesReceived:   ba.metrics.MessagesReceived,
		MessagesProcessed:  ba.metrics.MessagesProcessed,
		MessagesFailed:     ba.metrics.MessagesFailed,
		TotalProcessTime:   ba.metrics.TotalProcessTime,
		AverageProcessTime: ba.metrics.AverageProcessTime,
		LastProcessTime:    ba.metrics.LastProcessTime,
	}
}

// SendMessage sends a message through the message bus
func (ba *BaseAgent) SendMessage(msg *AgentMessage) error {
	if msg.Sender == "" {
		msg.Sender = ba.id
	}

	return ba.messageBus.Send(msg)
}

// SendProgressUpdate sends a progress update
func (ba *BaseAgent) SendProgressUpdate(receiver AgentID, progress interface{}) error {
	msg := &AgentMessage{
		Type:     ProgressUpdate,
		Sender:   ba.id,
		Receiver: receiver,
		Payload:  progress,
		Priority: PriorityMedium,
	}

	return ba.SendMessage(msg)
}

// SendResult sends a result message
func (ba *BaseAgent) SendResult(receiver AgentID, result *AgentResult) error {
	msg := &AgentMessage{
		Type:     ResultReporting,
		Sender:   ba.id,
		Receiver: receiver,
		Payload:  result,
		Priority: PriorityHigh,
	}

	return ba.SendMessage(msg)
}

// SendError sends an error notification
func (ba *BaseAgent) SendError(receiver AgentID, err error, context map[string]interface{}) error {
	msg := &AgentMessage{
		Type:     ErrorNotification,
		Sender:   ba.id,
		Receiver: receiver,
		Payload: map[string]interface{}{
			"error":   err.Error(),
			"context": context,
		},
		Priority: PriorityHigh,
	}

	return ba.SendMessage(msg)
}
