package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)

// AgentFactory creates instances of agents
type AgentFactory func(config AgentConfig) (Agent, error)

// AgentRegistry manages all agents in the system
type AgentRegistry struct {
	agents    map[AgentID]Agent
	factories map[AgentID]AgentFactory
	configs   map[AgentID]AgentConfig
	mu        sync.RWMutex

	messageBus *MessageBus
	logger     *logger.Logger

	healthCheckInterval time.Duration
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry(messageBus *MessageBus) *AgentRegistry {
	ctx, cancel := context.WithCancel(context.Background())

	registry := &AgentRegistry{
		agents:              make(map[AgentID]Agent),
		factories:           make(map[AgentID]AgentFactory),
		configs:             make(map[AgentID]AgentConfig),
		messageBus:          messageBus,
		logger:              logger.GetLogger(),
		healthCheckInterval: 30 * time.Second,
		ctx:                 ctx,
		cancel:              cancel,
	}

	// Start health monitoring
	registry.wg.Add(1)
	go registry.monitorHealth()

	return registry
}

// RegisterFactory registers an agent factory
func (r *AgentRegistry) RegisterFactory(agentID AgentID, factory AgentFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[agentID]; exists {
		return fmt.Errorf("factory for agent %s already registered", agentID)
	}

	r.factories[agentID] = factory
	r.logger.Info("Agent factory registered (agent_id: %s)", agentID)

	return nil
}

// CreateAgent creates and starts an agent
func (r *AgentRegistry) CreateAgent(agentID AgentID, config AgentConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if agent already exists
	if _, exists := r.agents[agentID]; exists {
		return fmt.Errorf("agent %s already exists", agentID)
	}

	// Get factory for agent type
	factory, exists := r.factories[agentID]
	if !exists {
		return fmt.Errorf("no factory registered for agent %s", agentID)
	}

	// Create agent instance
	agent, err := factory(config)
	if err != nil {
		return fmt.Errorf("failed to create agent %s: %w", agentID, err)
	}

	// Start the agent
	if err := agent.Start(r.ctx); err != nil {
		return fmt.Errorf("failed to start agent %s: %w", agentID, err)
	}

	// Register agent
	r.agents[agentID] = agent
	r.configs[agentID] = config

	r.logger.Info("Agent created and started (agent_id: %s, enabled: %t)", agentID, config.Enabled)

	return nil
}

// GetAgent retrieves an agent by ID
func (r *AgentRegistry) GetAgent(agentID AgentID) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// ListAgents returns all registered agents
func (r *AgentRegistry) ListAgents() []AgentID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]AgentID, 0, len(r.agents))
	for id := range r.agents {
		agents = append(agents, id)
	}

	return agents
}

// GetAgentStatus returns the status of an agent
func (r *AgentRegistry) GetAgentStatus(agentID AgentID) (AgentStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return StatusOffline, fmt.Errorf("agent %s not found", agentID)
	}

	return agent.GetStatus(), nil
}

// StopAgent stops a specific agent
func (r *AgentRegistry) StopAgent(agentID AgentID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Stop the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := agent.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop agent %s: %w", agentID, err)
	}

	// Remove from registry
	delete(r.agents, agentID)
	delete(r.configs, agentID)

	r.logger.Info("Agent stopped and removed (agent_id: %s)", agentID)

	return nil
}

// StopAgentOnly stops a specific agent but preserves its configuration
func (r *AgentRegistry) StopAgentOnly(agentID AgentID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Stop the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := agent.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop agent %s: %w", agentID, err)
	}

	// Remove only from agents registry, keep config
	delete(r.agents, agentID)

	r.logger.Info("Agent stopped (agent_id: %s)", agentID)

	return nil
}

// StopAll stops all agents
func (r *AgentRegistry) StopAll() error {
	r.logger.Info("Stopping all agents")

	// Cancel context to signal shutdown
	r.cancel()

	// Stop all agents
	r.mu.Lock()

	var errors []error
	agents := make(map[AgentID]Agent)
	for id, agent := range r.agents {
		agents[id] = agent
	}

	// Clear registries while holding lock
	r.agents = make(map[AgentID]Agent)
	r.configs = make(map[AgentID]AgentConfig)

	r.mu.Unlock()

	// Stop agents without holding lock
	for agentID, agent := range agents {
		// Use immediate function with defer to ensure context cleanup
		func(agentID AgentID, agent Agent) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := agent.Stop(ctx); err != nil {
				errors = append(errors, fmt.Errorf("failed to stop agent %s: %w", agentID, err))
			}

			r.logger.Info("Agent stopped (agent_id: %s)", agentID)
		}(agentID, agent)
	}

	// Wait for health monitor to stop
	r.wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some agents: %v", errors)
	}

	return nil
}

// RegisterAgent registers an agent configuration (simplified version for compatibility)
func (r *AgentRegistry) RegisterAgent(agentID AgentID, config AgentConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store configuration
	r.configs[agentID] = config

	r.logger.Info("Agent configuration registered (agent_id: %s)", agentID)
	return nil
}

// GetAgentConfig retrieves an agent's configuration
func (r *AgentRegistry) GetAgentConfig(agentID AgentID) (AgentConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, exists := r.configs[agentID]
	if !exists {
		return AgentConfig{}, fmt.Errorf("no configuration found for agent %s", agentID)
	}

	return config, nil
}

// StartAgent starts a specific agent
func (r *AgentRegistry) StartAgent(ctx context.Context, agentID AgentID) error {
	// Get configuration
	config, err := r.GetAgentConfig(agentID)
	if err != nil {
		return err
	}

	// Create and start agent using CreateAgent
	return r.CreateAgent(agentID, config)
}

// GetMetrics returns agent registry metrics
func (r *AgentRegistry) GetMetrics() *AgentRegistryMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	totalAgents := len(r.agents)
	activeAgents := 0
	failedAgents := 0

	for _, agent := range r.agents {
		status := agent.GetStatus()
		switch status {
		case StatusIdle, StatusBusy:
			activeAgents++
		case StatusError, StatusOffline:
			failedAgents++
		}
	}

	return &AgentRegistryMetrics{
		TotalAgents:   totalAgents,
		ActiveAgents:  activeAgents,
		FailedAgents:  failedAgents,
		Registrations: int64(totalAgents),
		LastUpdate:    time.Now(),
	}
}

// SetHealthCheckInterval sets the health check interval safely
func (r *AgentRegistry) SetHealthCheckInterval(interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.healthCheckInterval = interval
}

// monitorHealth periodically checks agent health
func (r *AgentRegistry) monitorHealth() {
	defer r.wg.Done()

	r.mu.RLock()
	interval := r.healthCheckInterval
	r.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.checkAllAgentsHealth()

		case <-r.ctx.Done():
			r.logger.Info("Health monitoring stopped")
			return
		}
	}
}

// checkAllAgentsHealth checks health of all agents
func (r *AgentRegistry) checkAllAgentsHealth() {
	r.mu.RLock()
	agents := make(map[AgentID]Agent)
	for id, agent := range r.agents {
		agents[id] = agent
	}
	r.mu.RUnlock()

	for agentID, agent := range agents {
		// Create context with timeout for each agent check
		ctx, cancel := context.WithTimeout(r.ctx, 10*time.Second)

		// Use immediate defer to ensure context is canceled
		func(agentID AgentID, agent Agent) {
			defer cancel()

			if err := agent.HealthCheck(ctx); err != nil {
				r.logger.Error("Agent health check failed (agent_id: %s, error: %v)", agentID, err)

				// Send health check failure notification
				msg := &AgentMessage{
					Type:     ErrorNotification,
					Sender:   AgentID("registry"),
					Receiver: AgentID("orchestrator"),
					Payload: map[string]interface{}{
						"agent_id": agentID,
						"error":    err.Error(),
						"type":     "health_check_failed",
					},
					Priority: PriorityHigh,
				}

				if err := r.messageBus.Send(msg); err != nil {
					r.logger.Error("Failed to send health check notification (agent_id: %s, error: %v)", agentID, err)
				}
			}
		}(agentID, agent)
	}
}

// AgentManager provides high-level agent management
type AgentManager struct {
	registry   *AgentRegistry
	messageBus *MessageBus
	router     *MessageRouter
	logger     *logger.Logger
}

// NewAgentManager creates a new agent manager
func NewAgentManager() *AgentManager {
	// Create message bus
	messageBus := NewMessageBus(1000)

	// Create registry
	registry := NewAgentRegistry(messageBus)

	// Create router
	router := NewMessageRouter(messageBus)

	return &AgentManager{
		registry:   registry,
		messageBus: messageBus,
		router:     router,
		logger:     logger.GetLogger(),
	}
}

// RegisterAgent registers an agent factory and configuration
func (m *AgentManager) RegisterAgent(agentID AgentID, factory AgentFactory, config AgentConfig) error {
	// Register factory
	if err := m.registry.RegisterFactory(agentID, factory); err != nil {
		return err
	}

	// Create agent if enabled
	if config.Enabled {
		if err := m.registry.CreateAgent(agentID, config); err != nil {
			return err
		}
	}

	return nil
}

// StartAgent starts a specific agent
func (m *AgentManager) StartAgent(agentID AgentID) error {
	// Get agent config
	config, err := m.registry.GetAgentConfig(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent config: %w", err)
	}

	// Create and start agent
	return m.registry.CreateAgent(agentID, config)
}

// StopAgent stops a specific agent
func (m *AgentManager) StopAgent(agentID AgentID) error {
	return m.registry.StopAgent(agentID)
}

// SendMessage sends a message through the message bus
func (m *AgentManager) SendMessage(msg *AgentMessage) error {
	return m.router.RouteMessage(msg)
}

// Shutdown gracefully shuts down the agent manager
func (m *AgentManager) Shutdown() error {
	m.logger.Info("Shutting down agent manager")

	// Stop all agents
	if err := m.registry.StopAll(); err != nil {
		m.logger.Error("Error stopping agents (error: %v)", err)
	}

	// Stop message bus
	m.messageBus.Stop()

	m.logger.Info("Agent manager shutdown complete")

	return nil
}
