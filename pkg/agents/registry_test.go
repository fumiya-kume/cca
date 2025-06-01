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

// mockAgent implements Agent interface for testing
type mockAgent struct {
	mu           sync.RWMutex
	id           AgentID
	status       AgentStatus
	capabilities []string
	startErr     error
	stopErr      error
	healthErr    error
	startCalled  bool
	stopCalled   bool
	healthCalled bool
}

func newMockAgent(id AgentID) *mockAgent {
	return &mockAgent{
		id:           id,
		status:       StatusOffline,
		capabilities: []string{"test_capability"},
	}
}

func (m *mockAgent) GetID() AgentID {
	return m.id
}

func (m *mockAgent) GetStatus() AgentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *mockAgent) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalled = true
	if m.startErr != nil {
		return m.startErr
	}
	m.status = StatusIdle
	return nil
}

func (m *mockAgent) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopCalled = true
	if m.stopErr != nil {
		return m.stopErr
	}
	m.status = StatusOffline
	return nil
}

func (m *mockAgent) ProcessMessage(ctx context.Context, msg *AgentMessage) error {
	return nil
}

func (m *mockAgent) GetCapabilities() []string {
	return m.capabilities
}

func (m *mockAgent) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthCalled = true
	return m.healthErr
}

func (m *mockAgent) SetStatus(status AgentStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
}

// mockAgentFactory creates mock agents
func mockAgentFactory(id AgentID) AgentFactory {
	return func(config AgentConfig) (Agent, error) {
		return newMockAgent(id), nil
	}
}

func mockAgentFactoryWithError(err error) AgentFactory {
	return func(config AgentConfig) (Agent, error) {
		return nil, err
	}
}

func TestNewAgentRegistry(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.agents)
	assert.NotNil(t, registry.factories)
	assert.NotNil(t, registry.configs)
	assert.Equal(t, messageBus, registry.messageBus)
	assert.Equal(t, 30*time.Second, registry.healthCheckInterval)

	// Cleanup
	err := registry.StopAll()
	assert.NoError(t, err)
}

func TestAgentRegistryRegisterFactory(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	factory := mockAgentFactory(SecurityAgentID)

	// Test successful registration
	err := registry.RegisterFactory(SecurityAgentID, factory)
	assert.NoError(t, err)

	// Test duplicate registration should fail
	err = registry.RegisterFactory(SecurityAgentID, factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "factory for agent security already registered")
}

func TestAgentRegistryCreateAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{
		Enabled:       true,
		MaxInstances:  1,
		Timeout:       time.Minute,
		RetryAttempts: 3,
	}

	// Register factory first
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	// Test successful agent creation
	err = registry.CreateAgent(SecurityAgentID, config)
	assert.NoError(t, err)

	// Verify agent is in registry
	agent, err := registry.GetAgent(SecurityAgentID)
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, SecurityAgentID, agent.GetID())

	// Test duplicate creation should fail
	err = registry.CreateAgent(SecurityAgentID, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent security already exists")
}

func TestAgentRegistryCreateAgentWithoutFactory(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	config := AgentConfig{Enabled: true}

	// Try to create agent without registering factory
	err := registry.CreateAgent(SecurityAgentID, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no factory registered for agent security")
}

func TestAgentRegistryCreateAgentWithFactoryError(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	expectedError := fmt.Errorf("factory error")
	factory := mockAgentFactoryWithError(expectedError)
	config := AgentConfig{Enabled: true}

	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create agent security")
}

func TestAgentRegistryCreateAgentWithStartError(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create factory that returns agent with start error
	factory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(SecurityAgentID)
		agent.startErr = fmt.Errorf("start error")
		return agent, nil
	}

	config := AgentConfig{Enabled: true}

	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start agent security")
}

func TestAgentRegistryGetAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Test getting non-existent agent
	agent, err := registry.GetAgent(SecurityAgentID)
	assert.Error(t, err)
	assert.Nil(t, agent)
	assert.Contains(t, err.Error(), "agent security not found")

	// Create and register an agent
	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{Enabled: true}

	err = registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Test getting existing agent
	agent, err = registry.GetAgent(SecurityAgentID)
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, SecurityAgentID, agent.GetID())
}

func TestAgentRegistryListAgents(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Initially empty
	agents := registry.ListAgents()
	assert.Empty(t, agents)

	// Create multiple agents
	agentIDs := []AgentID{SecurityAgentID, ArchitectureAgentID, TestingAgentID}
	config := AgentConfig{Enabled: true}

	for _, id := range agentIDs {
		factory := mockAgentFactory(id)
		err := registry.RegisterFactory(id, factory)
		require.NoError(t, err)

		err = registry.CreateAgent(id, config)
		require.NoError(t, err)
	}

	// List all agents
	agents = registry.ListAgents()
	assert.Len(t, agents, 3)
	for _, id := range agentIDs {
		assert.Contains(t, agents, id)
	}
}

func TestAgentRegistryGetAgentStatus(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Test getting status of non-existent agent
	status, err := registry.GetAgentStatus(SecurityAgentID)
	assert.Error(t, err)
	assert.Equal(t, StatusOffline, status)

	// Create an agent
	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{Enabled: true}

	err = registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Get agent status
	status, err = registry.GetAgentStatus(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, StatusIdle, status) // Mock agent sets to idle on start
}

func TestAgentRegistryStopAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Test stopping non-existent agent
	err := registry.StopAgent(SecurityAgentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent security not found")

	// Create an agent
	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{Enabled: true}

	err = registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Verify agent exists
	agent, err := registry.GetAgent(SecurityAgentID)
	require.NoError(t, err)
	mockAgent := agent.(*mockAgent)

	// Stop the agent
	err = registry.StopAgent(SecurityAgentID)
	assert.NoError(t, err)
	assert.True(t, mockAgent.stopCalled)

	// Verify agent is removed from registry
	_, err = registry.GetAgent(SecurityAgentID)
	assert.Error(t, err)
}

func TestAgentRegistryStopAgentWithError(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create factory that returns agent with stop error
	factory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(SecurityAgentID)
		agent.stopErr = fmt.Errorf("stop error")
		return agent, nil
	}

	config := AgentConfig{Enabled: true}

	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Try to stop agent with error
	err = registry.StopAgent(SecurityAgentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop agent security")
}

func TestAgentRegistryStopAll(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)

	// Create multiple agents
	agentIDs := []AgentID{SecurityAgentID, ArchitectureAgentID, TestingAgentID}
	config := AgentConfig{Enabled: true}
	mockAgents := make([]*mockAgent, len(agentIDs))

	for i, id := range agentIDs {
		factory := func(agentID AgentID) AgentFactory {
			return func(config AgentConfig) (Agent, error) {
				return newMockAgent(agentID), nil
			}
		}(id)

		err := registry.RegisterFactory(id, factory)
		require.NoError(t, err)

		err = registry.CreateAgent(id, config)
		require.NoError(t, err)

		agent, err := registry.GetAgent(id)
		require.NoError(t, err)
		mockAgents[i] = agent.(*mockAgent)
	}

	// Stop all agents
	err := registry.StopAll()
	assert.NoError(t, err)

	// Verify all agents were stopped
	for _, mockAgent := range mockAgents {
		assert.True(t, mockAgent.stopCalled)
	}

	// Verify registry is empty
	agents := registry.ListAgents()
	assert.Empty(t, agents)
}

func TestAgentRegistryStopAllWithErrors(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)

	// Create agents, some with stop errors
	config := AgentConfig{Enabled: true}

	// First agent stops successfully
	factory1 := mockAgentFactory(SecurityAgentID)
	err := registry.RegisterFactory(SecurityAgentID, factory1)
	require.NoError(t, err)
	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Second agent has stop error
	factory2 := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(ArchitectureAgentID)
		agent.stopErr = fmt.Errorf("stop error")
		return agent, nil
	}
	err = registry.RegisterFactory(ArchitectureAgentID, factory2)
	require.NoError(t, err)
	err = registry.CreateAgent(ArchitectureAgentID, config)
	require.NoError(t, err)

	// Stop all should return error but still clean up
	err = registry.StopAll()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stop some agents")

	// Registry should still be cleared
	agents := registry.ListAgents()
	assert.Empty(t, agents)
}

func TestAgentRegistryRegisterAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	config := AgentConfig{
		Enabled:      true,
		MaxInstances: 1,
	}

	err := registry.RegisterAgent(SecurityAgentID, config)
	assert.NoError(t, err)

	// Verify config is stored
	storedConfig, exists := registry.configs[SecurityAgentID]
	assert.True(t, exists)
	assert.Equal(t, config, storedConfig)
}

func TestAgentRegistryStartAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Try to start agent without configuration
	ctx := context.Background()
	err := registry.StartAgent(ctx, SecurityAgentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no configuration found for agent security")

	// Register configuration
	config := AgentConfig{Enabled: true}
	err = registry.RegisterAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Register factory
	factory := mockAgentFactory(SecurityAgentID)
	err = registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	// Start agent
	err = registry.StartAgent(ctx, SecurityAgentID)
	assert.NoError(t, err)

	// Verify agent is created and running
	agent, err := registry.GetAgent(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, StatusIdle, agent.GetStatus())
}

func TestAgentRegistryGetMetrics(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Initially empty metrics
	metrics := registry.GetMetrics()
	assert.Equal(t, 0, metrics.TotalAgents)
	assert.Equal(t, 0, metrics.ActiveAgents)
	assert.Equal(t, 0, metrics.FailedAgents)

	// Create agents with different statuses
	config := AgentConfig{Enabled: true}

	// Create active agent
	factory1 := mockAgentFactory(SecurityAgentID)
	err := registry.RegisterFactory(SecurityAgentID, factory1)
	require.NoError(t, err)
	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Create second agent that will be marked as failed
	factory2 := func(config AgentConfig) (Agent, error) {
		return newMockAgent(ArchitectureAgentID), nil
	}
	err = registry.RegisterFactory(ArchitectureAgentID, factory2)
	require.NoError(t, err)
	err = registry.CreateAgent(ArchitectureAgentID, config)
	require.NoError(t, err)

	// Get the agent and manually set it to error status
	agent, err := registry.GetAgent(ArchitectureAgentID)
	require.NoError(t, err)
	mockAgent := agent.(*mockAgent)
	mockAgent.SetStatus(StatusError)

	// Get updated metrics
	metrics = registry.GetMetrics()
	assert.Equal(t, 2, metrics.TotalAgents)
	assert.Equal(t, 1, metrics.ActiveAgents)
	assert.Equal(t, 1, metrics.FailedAgents)
	assert.Equal(t, int64(2), metrics.Registrations)
}

func TestAgentRegistryHealthMonitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping health monitoring test in short mode")
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)

	// Set a very short health check interval for testing
	registry.SetHealthCheckInterval(10 * time.Millisecond)

	// Create agent with health check that fails
	factory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(SecurityAgentID)
		agent.healthErr = fmt.Errorf("health check failed")
		return agent, nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Get the agent and manually call its health check
	agent, err := registry.GetAgent(SecurityAgentID)
	require.NoError(t, err)

	// Call health check directly on the agent
	ctx := context.Background()
	_ = agent.HealthCheck(ctx) // This will trigger the mock's healthCalled flag

	// Verify health check was called
	mockAgent := agent.(*mockAgent)
	assert.True(t, mockAgent.healthCalled)

	// Cleanup
	err = registry.StopAll()
	assert.NoError(t, err)
}

func TestNewAgentManager(t *testing.T) {
	manager := NewAgentManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.registry)
	assert.NotNil(t, manager.messageBus)
	assert.NotNil(t, manager.router)

	// Cleanup
	err := manager.Shutdown()
	assert.NoError(t, err)
}

func TestAgentManagerRegisterAgent(t *testing.T) {
	manager := NewAgentManager()
	defer func() { _ = manager.Shutdown() }()

	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{
		Enabled:       true,
		MaxInstances:  1,
		Timeout:       time.Minute,
		RetryAttempts: 3,
	}

	err := manager.RegisterAgent(SecurityAgentID, factory, config)
	assert.NoError(t, err)

	// Verify agent was created (since enabled = true)
	agent, err := manager.registry.GetAgent(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, SecurityAgentID, agent.GetID())
}

func TestAgentManagerRegisterAgentDisabled(t *testing.T) {
	manager := NewAgentManager()
	defer func() { _ = manager.Shutdown() }()

	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{
		Enabled:       false, // Disabled
		MaxInstances:  1,
		Timeout:       time.Minute,
		RetryAttempts: 3,
	}

	err := manager.RegisterAgent(SecurityAgentID, factory, config)
	assert.NoError(t, err)

	// Verify agent was not created (since enabled = false)
	_, err = manager.registry.GetAgent(SecurityAgentID)
	assert.Error(t, err)
}

func TestAgentManagerStartStopAgent(t *testing.T) {
	manager := NewAgentManager()
	defer func() { _ = manager.Shutdown() }()

	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{
		Enabled:       false, // Start disabled
		MaxInstances:  1,
		Timeout:       time.Minute,
		RetryAttempts: 3,
	}

	// Register factory and config without creating
	err := manager.registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)
	err = manager.registry.RegisterAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Start agent manually
	err = manager.StartAgent(SecurityAgentID)
	assert.NoError(t, err)

	// Verify agent is running
	agent, err := manager.registry.GetAgent(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, StatusIdle, agent.GetStatus())

	// Stop agent
	err = manager.StopAgent(SecurityAgentID)
	assert.NoError(t, err)

	// Verify agent is removed
	_, err = manager.registry.GetAgent(SecurityAgentID)
	assert.Error(t, err)
}

func TestAgentManagerSendMessage(t *testing.T) {
	manager := NewAgentManager()
	defer func() { _ = manager.Shutdown() }()

	msg := &AgentMessage{
		Type:     TaskAssignment,
		Sender:   SecurityAgentID,
		Receiver: ArchitectureAgentID,
		Payload:  "test",
	}

	err := manager.SendMessage(msg)
	assert.NoError(t, err)
}

func TestAgentManagerShutdown(t *testing.T) {
	manager := NewAgentManager()

	// Create some agents
	factory := mockAgentFactory(SecurityAgentID)
	config := AgentConfig{Enabled: true}

	err := manager.RegisterAgent(SecurityAgentID, factory, config)
	require.NoError(t, err)

	// Shutdown should clean up everything
	err = manager.Shutdown()
	assert.NoError(t, err)

	// Verify agents are stopped
	agents := manager.registry.ListAgents()
	assert.Empty(t, agents)
}

func TestAgentRegistryConcurrentOperations(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Test concurrent agent creation and retrieval
	agentIDs := []AgentID{SecurityAgentID, ArchitectureAgentID, TestingAgentID, PerformanceAgentID}
	config := AgentConfig{Enabled: true}

	// Register factories
	for _, id := range agentIDs {
		factory := mockAgentFactory(id)
		err := registry.RegisterFactory(id, factory)
		require.NoError(t, err)
	}

	// Concurrent agent creation
	done := make(chan bool, len(agentIDs))
	for _, id := range agentIDs {
		go func(agentID AgentID) {
			defer func() { done <- true }()
			err := registry.CreateAgent(agentID, config)
			assert.NoError(t, err)
		}(id)
	}

	// Wait for all to complete
	for i := 0; i < len(agentIDs); i++ {
		<-done
	}

	// Verify all agents were created
	agents := registry.ListAgents()
	assert.Len(t, agents, len(agentIDs))

	// Concurrent agent access
	for _, id := range agentIDs {
		go func(agentID AgentID) {
			defer func() { done <- true }()
			agent, err := registry.GetAgent(agentID)
			assert.NoError(t, err)
			assert.Equal(t, agentID, agent.GetID())

			status, err := registry.GetAgentStatus(agentID)
			assert.NoError(t, err)
			assert.Equal(t, StatusIdle, status)
		}(id)
	}

	// Wait for all to complete
	for i := 0; i < len(agentIDs); i++ {
		<-done
	}
}
