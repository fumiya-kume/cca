package agents

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthMonitor(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	monitor := NewHealthMonitor(registry, messageBus)
	assert.NotNil(t, monitor)
	assert.Equal(t, registry, monitor.registry)
	assert.Equal(t, messageBus, monitor.messageBus)
	assert.Equal(t, 30*time.Second, monitor.checkInterval)
	assert.Equal(t, 10*time.Second, monitor.timeout)
	assert.Equal(t, 3, monitor.maxFailures)
	assert.NotNil(t, monitor.healthStatus)
	assert.NotNil(t, monitor.failureCounts)
}

func TestHealthMonitorStartStop(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	monitor := NewHealthMonitor(registry, messageBus)

	// Test start
	err := monitor.Start()
	assert.NoError(t, err)

	// Allow monitoring to run briefly using context
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Brief wait for startup
	select {
	case <-ctx.Done():
	default:
		// Continue with test
	}

	// Test stop
	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestHealthMonitorCheckAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create a mock agent that always passes health checks
	factory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(SecurityAgentID)
		agent.healthErr = nil // Health check passes
		return agent, nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	monitor := NewHealthMonitor(registry, messageBus)

	// Check the agent
	monitor.checkAgent(SecurityAgentID)

	// Verify health status is recorded
	status, err := monitor.GetHealthStatus(SecurityAgentID)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, SecurityAgentID, status.AgentID)
	assert.Equal(t, HealthHealthy, status.Status)
	assert.Equal(t, 0, status.ConsecutiveFails)
	assert.False(t, status.LastSuccess.IsZero())
}

func TestHealthMonitorCheckAgentFailure(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create a mock agent that fails health checks
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

	monitor := NewHealthMonitor(registry, messageBus)

	// Check the agent
	monitor.checkAgent(SecurityAgentID)

	// Verify health status shows failure
	status, err := monitor.GetHealthStatus(SecurityAgentID)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, SecurityAgentID, status.AgentID)
	assert.Equal(t, HealthDegraded, status.Status)
	assert.Equal(t, 1, status.ConsecutiveFails)
	assert.NotEmpty(t, status.ErrorMessage)
}

func TestHealthMonitorCheckAgentMultipleFailures(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create a mock agent that fails health checks
	factory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(SecurityAgentID)
		agent.healthErr = fmt.Errorf("persistent failure")
		return agent, nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	monitor := NewHealthMonitor(registry, messageBus)

	// Fail health check multiple times
	for i := 1; i <= 4; i++ {
		monitor.checkAgent(SecurityAgentID)

		status, err := monitor.GetHealthStatus(SecurityAgentID)
		assert.NoError(t, err)
		assert.Equal(t, i, status.ConsecutiveFails)

		if i >= monitor.maxFailures {
			assert.Equal(t, HealthUnhealthy, status.Status)
		} else {
			assert.Equal(t, HealthDegraded, status.Status)
		}
	}
}

func TestHealthMonitorCheckAgentRecovery(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create a mock agent with controllable health check
	mockAgent := newMockAgent(SecurityAgentID)
	factory := func(config AgentConfig) (Agent, error) {
		return mockAgent, nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	monitor := NewHealthMonitor(registry, messageBus)

	// First, fail the health check
	mockAgent.healthErr = fmt.Errorf("temporary failure")
	monitor.checkAgent(SecurityAgentID)

	status, err := monitor.GetHealthStatus(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, HealthDegraded, status.Status)
	assert.Equal(t, 1, status.ConsecutiveFails)

	// Now recover
	mockAgent.healthErr = nil
	monitor.checkAgent(SecurityAgentID)

	status, err = monitor.GetHealthStatus(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, HealthHealthy, status.Status)
	assert.Equal(t, 0, status.ConsecutiveFails)
	assert.False(t, status.LastSuccess.IsZero())
}

func TestHealthMonitorCheckNonExistentAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	monitor := NewHealthMonitor(registry, messageBus)

	// Check non-existent agent
	monitor.checkAgent(SecurityAgentID)

	// Should create unknown health status
	status, err := monitor.GetHealthStatus(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, HealthUnknown, status.Status)
}

func TestHealthMonitorGetMetrics(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	monitor := NewHealthMonitor(registry, messageBus)

	// Initial metrics
	metrics := monitor.GetMetrics()
	assert.Equal(t, 0, metrics.TotalAgents)
	assert.Equal(t, 0, metrics.HealthyAgents)

	// Add some agents with different health states
	healthyFactory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(SecurityAgentID)
		agent.healthErr = nil
		return agent, nil
	}

	unhealthyFactory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(ArchitectureAgentID)
		agent.healthErr = fmt.Errorf("failed")
		return agent, nil
	}

	config := AgentConfig{Enabled: true}

	err := registry.RegisterFactory(SecurityAgentID, healthyFactory)
	require.NoError(t, err)
	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	err = registry.RegisterFactory(ArchitectureAgentID, unhealthyFactory)
	require.NoError(t, err)
	err = registry.CreateAgent(ArchitectureAgentID, config)
	require.NoError(t, err)

	// Check agents
	monitor.checkAgent(SecurityAgentID)
	monitor.checkAgent(ArchitectureAgentID)

	// Get updated metrics
	metrics = monitor.GetMetrics()
	assert.Equal(t, 2, metrics.TotalAgents)
	assert.Equal(t, 1, metrics.HealthyAgents)
	assert.False(t, metrics.LastCheckTime.IsZero())
}

func TestHealthMonitorGetAllHealthStatus(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	monitor := NewHealthMonitor(registry, messageBus)

	// Initially empty
	statuses := monitor.GetAllHealthStatus()
	assert.Empty(t, statuses)

	// Add some health statuses manually (simulating health checks)
	monitor.mu.Lock()
	monitor.healthStatus[SecurityAgentID] = &HealthStatus{
		AgentID: SecurityAgentID,
		Status:  HealthHealthy,
	}
	monitor.healthStatus[ArchitectureAgentID] = &HealthStatus{
		AgentID: ArchitectureAgentID,
		Status:  HealthDegraded,
	}
	monitor.mu.Unlock()

	// Get all statuses
	statuses = monitor.GetAllHealthStatus()
	assert.Len(t, statuses, 2)
	assert.Contains(t, statuses, SecurityAgentID)
	assert.Contains(t, statuses, ArchitectureAgentID)
	assert.Equal(t, HealthHealthy, statuses[SecurityAgentID].Status)
	assert.Equal(t, HealthDegraded, statuses[ArchitectureAgentID].Status)

	// Verify they are copies (modifying shouldn't affect original)
	statuses[SecurityAgentID].Status = HealthUnhealthy
	originalStatus, _ := monitor.GetHealthStatus(SecurityAgentID)
	assert.Equal(t, HealthHealthy, originalStatus.Status)
}

func TestHealthMonitorGetHealthStatusNotFound(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	monitor := NewHealthMonitor(registry, messageBus)

	status, err := monitor.GetHealthStatus(SecurityAgentID)
	assert.Error(t, err)
	assert.Nil(t, status)
	assert.Contains(t, err.Error(), "no health status for agent security")
}

func TestHealthMonitorIntegrationWithRegistry(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Set shorter interval for testing
	monitor := NewHealthMonitor(registry, messageBus)
	monitor.checkInterval = 50 * time.Millisecond

	// Create agents
	factory := func(config AgentConfig) (Agent, error) {
		return newMockAgent(SecurityAgentID), nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Start monitoring
	err = monitor.Start()
	require.NoError(t, err)

	// Wait for health checks to run using ticker
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := monitor.GetHealthStatus(SecurityAgentID); err == nil {
				goto healthStatusFound
			}
		case <-ctx.Done():
			goto healthStatusFound
		}
	}
healthStatusFound:

	// Verify health status was recorded
	status, err := monitor.GetHealthStatus(SecurityAgentID)
	assert.NoError(t, err)
	assert.NotNil(t, status)

	// Stop monitoring
	err = monitor.Stop()
	assert.NoError(t, err)
}

func TestNewRecoveryManager(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	manager := NewRecoveryManager(registry, messageBus)
	assert.NotNil(t, manager)
	assert.Equal(t, registry, manager.registry)
	assert.Equal(t, messageBus, manager.messageBus)
	assert.NotNil(t, manager.recoveryStrategies)
}

func TestRecoveryManagerSetRecoveryStrategy(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	manager := NewRecoveryManager(registry, messageBus)

	strategy := RecoveryStrategy{
		MaxRetries:     5,
		RetryDelay:     2 * time.Second,
		BackoffFactor:  1.5,
		RecoveryAction: RecoveryRestart,
	}

	manager.SetRecoveryStrategy(SecurityAgentID, strategy)

	manager.mu.RLock()
	storedStrategy, exists := manager.recoveryStrategies[SecurityAgentID]
	manager.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, strategy, storedStrategy)
}

func TestRecoveryManagerRecoverAgentRestart(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create a mock agent
	factory := func(config AgentConfig) (Agent, error) {
		return newMockAgent(SecurityAgentID), nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	// Register the configuration separately to persist it
	err = registry.RegisterAgent(SecurityAgentID, config)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	manager := NewRecoveryManager(registry, messageBus)

	// Set recovery strategy
	strategy := RecoveryStrategy{
		MaxRetries:     1,
		RetryDelay:     10 * time.Millisecond,
		BackoffFactor:  1.0,
		RecoveryAction: RecoveryRestart,
	}
	manager.SetRecoveryStrategy(SecurityAgentID, strategy)

	// Simulate health status
	healthStatus := &HealthStatus{
		AgentID:          SecurityAgentID,
		Status:           HealthUnhealthy,
		ConsecutiveFails: 3,
	}

	// Attempt recovery
	err = manager.RecoverAgent(SecurityAgentID, healthStatus)
	assert.NoError(t, err)

	// Verify agent was restarted (should exist in registry)
	agent, err := registry.GetAgent(SecurityAgentID)
	assert.NoError(t, err)
	assert.Equal(t, SecurityAgentID, agent.GetID())
}

func TestRecoveryManagerRecoverAgentWithDefaultStrategy(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// Create a mock agent
	factory := func(config AgentConfig) (Agent, error) {
		return newMockAgent(SecurityAgentID), nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, factory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	manager := NewRecoveryManager(registry, messageBus)

	// Don't set custom strategy - should use default

	healthStatus := &HealthStatus{
		AgentID: SecurityAgentID,
		Status:  HealthUnhealthy,
	}

	// Attempt recovery
	err = manager.RecoverAgent(SecurityAgentID, healthStatus)
	assert.NoError(t, err)
}

func TestRecoveryManagerRecoverAgentReload(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	manager := NewRecoveryManager(registry, messageBus)

	// Set reload strategy
	strategy := RecoveryStrategy{
		MaxRetries:     1,
		RetryDelay:     10 * time.Millisecond,
		BackoffFactor:  1.0,
		RecoveryAction: RecoveryReload,
	}
	manager.SetRecoveryStrategy(SecurityAgentID, strategy)

	healthStatus := &HealthStatus{
		AgentID: SecurityAgentID,
		Status:  HealthUnhealthy,
	}

	// Attempt recovery - should send reload message
	err := manager.RecoverAgent(SecurityAgentID, healthStatus)
	assert.NoError(t, err)

	// Wait for message to be sent using context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Wait for processing time
	select {
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
	}
}

func TestRecoveryManagerRecoverAgentEscalate(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	manager := NewRecoveryManager(registry, messageBus)

	strategy := RecoveryStrategy{
		MaxRetries:     1,
		RetryDelay:     10 * time.Millisecond,
		BackoffFactor:  1.0,
		RecoveryAction: RecoveryEscalate,
	}
	manager.SetRecoveryStrategy(SecurityAgentID, strategy)

	healthStatus := &HealthStatus{
		AgentID: SecurityAgentID,
		Status:  HealthUnhealthy,
	}

	// Attempt recovery - should escalate
	err := manager.RecoverAgent(SecurityAgentID, healthStatus)
	assert.NoError(t, err)

	// Wait for message to be sent using context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Wait for processing time
	select {
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
	}
}

func TestRecoveryManagerUnknownAction(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	manager := NewRecoveryManager(registry, messageBus)

	strategy := RecoveryStrategy{
		MaxRetries:     1,
		RetryDelay:     10 * time.Millisecond,
		BackoffFactor:  1.0,
		RecoveryAction: RecoveryAction("unknown"),
	}
	manager.SetRecoveryStrategy(SecurityAgentID, strategy)

	healthStatus := &HealthStatus{
		AgentID: SecurityAgentID,
		Status:  HealthUnhealthy,
	}

	// Attempt recovery with unknown action
	err := manager.RecoverAgent(SecurityAgentID, healthStatus)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown recovery action")
}

func TestRecoveryManagerMultipleRetries(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	// First create a working agent
	workingFactory := func(config AgentConfig) (Agent, error) {
		return newMockAgent(SecurityAgentID), nil
	}

	config := AgentConfig{Enabled: true}
	err := registry.RegisterFactory(SecurityAgentID, workingFactory)
	require.NoError(t, err)

	err = registry.CreateAgent(SecurityAgentID, config)
	require.NoError(t, err)

	// Now replace with a factory that fails during recovery by registering under a different ID
	// and then swapping the factory map (this is a test-only workaround)
	failFactory := func(config AgentConfig) (Agent, error) {
		agent := newMockAgent(SecurityAgentID)
		agent.startErr = fmt.Errorf("start failed")
		return agent, nil
	}

	// Stop all first to clear the registry
	_ = registry.StopAll()

	// Create new registry for this test part
	testRegistry := NewAgentRegistry(messageBus)
	defer func() { _ = testRegistry.StopAll() }()

	err = testRegistry.RegisterFactory(SecurityAgentID, failFactory)
	require.NoError(t, err)

	// Register config and create failing agent
	err = testRegistry.RegisterAgent(SecurityAgentID, config)
	require.NoError(t, err)

	manager := NewRecoveryManager(testRegistry, messageBus)

	strategy := RecoveryStrategy{
		MaxRetries:     2,
		RetryDelay:     10 * time.Millisecond,
		BackoffFactor:  2.0,
		RecoveryAction: RecoveryRestart,
	}
	manager.SetRecoveryStrategy(SecurityAgentID, strategy)

	healthStatus := &HealthStatus{
		AgentID: SecurityAgentID,
		Status:  HealthUnhealthy,
	}

	// Recovery should fail after retries
	err = manager.RecoverAgent(SecurityAgentID, healthStatus)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to recover agent")
}

func TestHealthStateConstants(t *testing.T) {
	assert.Equal(t, "healthy", string(HealthHealthy))
	assert.Equal(t, "degraded", string(HealthDegraded))
	assert.Equal(t, "unhealthy", string(HealthUnhealthy))
	assert.Equal(t, "unknown", string(HealthUnknown))
}

func TestRecoveryActionConstants(t *testing.T) {
	assert.Equal(t, "restart", string(RecoveryRestart))
	assert.Equal(t, "reload", string(RecoveryReload))
	assert.Equal(t, "recreate", string(RecoveryRecreate))
	assert.Equal(t, "escalate", string(RecoveryEscalate))
}

func TestHealthStatusStruct(t *testing.T) {
	status := HealthStatus{
		AgentID:          SecurityAgentID,
		Status:           HealthHealthy,
		LastCheck:        time.Now(),
		LastSuccess:      time.Now().Add(-time.Minute),
		ConsecutiveFails: 0,
		ResponseTime:     50 * time.Millisecond,
		ErrorMessage:     "",
		Metrics: HealthMetrics{
			CPUUsage:          25.0,
			MemoryUsage:       1024 * 1024 * 128, // 128MB
			MessagesProcessed: 100,
			ErrorRate:         0.01,
			AverageLatency:    10 * time.Millisecond,
		},
	}

	assert.Equal(t, SecurityAgentID, status.AgentID)
	assert.Equal(t, HealthHealthy, status.Status)
	assert.Equal(t, 0, status.ConsecutiveFails)
	assert.Equal(t, 50*time.Millisecond, status.ResponseTime)
	assert.Empty(t, status.ErrorMessage)
	assert.Equal(t, 25.0, status.Metrics.CPUUsage)
	assert.Equal(t, int64(100), status.Metrics.MessagesProcessed)
}

func TestRecoveryStrategyStruct(t *testing.T) {
	strategy := RecoveryStrategy{
		MaxRetries:     5,
		RetryDelay:     30 * time.Second,
		BackoffFactor:  2.0,
		RecoveryAction: RecoveryRestart,
	}

	assert.Equal(t, 5, strategy.MaxRetries)
	assert.Equal(t, 30*time.Second, strategy.RetryDelay)
	assert.Equal(t, 2.0, strategy.BackoffFactor)
	assert.Equal(t, RecoveryRestart, strategy.RecoveryAction)
}
