package agents

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// HealthMonitor monitors the health of all agents
type HealthMonitor struct {
	registry   *AgentRegistry
	messageBus *MessageBus
	logger     *logger.Logger
	clock      clock.Clock

	// Health check configuration
	checkInterval time.Duration
	timeout       time.Duration
	maxFailures   int

	// Health status tracking
	healthStatus  map[AgentID]*HealthStatus
	failureCounts map[AgentID]int
	mu            sync.RWMutex

	// Monitoring control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// HealthStatus represents the health status of an agent
type HealthStatus struct {
	AgentID          AgentID       `json:"agent_id"`
	Status           HealthState   `json:"status"`
	LastCheck        time.Time     `json:"last_check"`
	LastSuccess      time.Time     `json:"last_success"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	ResponseTime     time.Duration `json:"response_time"`
	ErrorMessage     string        `json:"error_message,omitempty"`
	Metrics          HealthMetrics `json:"metrics"`
}

// HealthState represents the health state of an agent
type HealthState string

const (
	HealthHealthy   HealthState = "healthy"
	HealthDegraded  HealthState = "degraded"
	HealthUnhealthy HealthState = "unhealthy"
	HealthUnknown   HealthState = "unknown"
)

// HealthMetrics contains health-related metrics
type HealthMetrics struct {
	CPUUsage          float64       `json:"cpu_usage"`
	MemoryUsage       int64         `json:"memory_usage"`
	MessagesProcessed int64         `json:"messages_processed"`
	ErrorRate         float64       `json:"error_rate"`
	AverageLatency    time.Duration `json:"average_latency"`
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(registry *AgentRegistry, messageBus *MessageBus) *HealthMonitor {
	return NewHealthMonitorWithClock(registry, messageBus, clock.NewRealClock())
}

// NewHealthMonitorWithClock creates a new health monitor with a custom clock
func NewHealthMonitorWithClock(registry *AgentRegistry, messageBus *MessageBus, clk clock.Clock) *HealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	monitor := &HealthMonitor{
		registry:      registry,
		messageBus:    messageBus,
		logger:        logger.GetLogger(),
		clock:         clk,
		checkInterval: 30 * time.Second,
		timeout:       10 * time.Second,
		maxFailures:   3,
		healthStatus:  make(map[AgentID]*HealthStatus),
		failureCounts: make(map[AgentID]int),
		ctx:           ctx,
		cancel:        cancel,
	}

	return monitor
}

// Start begins health monitoring
func (hm *HealthMonitor) Start() error {
	hm.logger.Info("Starting health monitor (check_interval: %v, timeout: %v)", hm.checkInterval, hm.timeout)

	// Start monitoring goroutine
	hm.wg.Add(1)
	go hm.monitorLoop()

	return nil
}

// Stop halts health monitoring
func (hm *HealthMonitor) Stop() error {
	hm.logger.Info("Stopping health monitor")

	hm.cancel()
	hm.wg.Wait()

	return nil
}

// GetMetrics returns health monitoring metrics
func (hm *HealthMonitor) GetMetrics() *HealthMonitorMetrics {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	totalChecks := 0
	failedChecks := 0
	healthyAgents := 0

	for _, status := range hm.healthStatus {
		totalChecks += status.ConsecutiveFails
		if status.Status == HealthHealthy {
			healthyAgents++
		} else {
			failedChecks += status.ConsecutiveFails
		}
	}

	return &HealthMonitorMetrics{
		TotalAgents:        len(hm.healthStatus),
		HealthyAgents:      healthyAgents,
		TotalHealthChecks:  totalChecks,
		FailedHealthChecks: failedChecks,
		LastCheckTime:      hm.clock.Now(),
	}
}

// monitorLoop continuously monitors agent health
func (hm *HealthMonitor) monitorLoop() {
	defer hm.wg.Done()

	ticker := hm.clock.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	// Initial check
	hm.checkAllAgents()

	for {
		select {
		case <-ticker.C():
			hm.checkAllAgents()

		case <-hm.ctx.Done():
			hm.logger.Info("Health monitoring stopped")
			return
		}
	}
}

// checkAllAgents performs health checks on all registered agents
func (hm *HealthMonitor) checkAllAgents() {
	agents := hm.registry.ListAgents()

	var wg sync.WaitGroup
	for _, agentID := range agents {
		wg.Add(1)
		go func(id AgentID) {
			defer func() {
				if r := recover(); r != nil {
					hm.logger.Error("Health check goroutine panic for agent %s: %v", id, r)
				}
				wg.Done()
			}()
			hm.checkAgent(id)
		}(agentID)
	}

	wg.Wait()

	// Analyze overall health
	hm.analyzeOverallHealth()
}

// checkAgent performs a health check on a single agent
func (hm *HealthMonitor) checkAgent(agentID AgentID) {
	startTime := hm.clock.Now()

	// Get agent
	agent, err := hm.registry.GetAgent(agentID)
	if err != nil {
		hm.updateHealthStatus(agentID, HealthUnknown, 0, err.Error())
		return
	}

	// Create health check context with timeout
	ctx, cancel := clock.WithTimeout(hm.ctx, hm.clock, hm.timeout)
	defer cancel()

	// Perform health check
	err = agent.HealthCheck(ctx)
	responseTime := hm.clock.Since(startTime)

	if err != nil {
		hm.handleHealthCheckFailure(agentID, responseTime, err)
	} else {
		hm.handleHealthCheckSuccess(agentID, responseTime)
	}
}

// handleHealthCheckSuccess processes a successful health check
func (hm *HealthMonitor) handleHealthCheckSuccess(agentID AgentID, responseTime time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Reset failure count
	delete(hm.failureCounts, agentID)

	// Update health status
	status := &HealthStatus{
		AgentID:          agentID,
		Status:           HealthHealthy,
		LastCheck:        hm.clock.Now(),
		LastSuccess:      hm.clock.Now(),
		ConsecutiveFails: 0,
		ResponseTime:     responseTime,
		Metrics:          hm.getAgentMetrics(agentID),
	}

	hm.healthStatus[agentID] = status

	hm.logger.Debug("Health check passed (agent_id: %s, response_time: %v)", agentID, responseTime)
}

// handleHealthCheckFailure processes a failed health check
func (hm *HealthMonitor) handleHealthCheckFailure(agentID AgentID, responseTime time.Duration, err error) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Increment failure count
	hm.failureCounts[agentID]++
	failures := hm.failureCounts[agentID]

	// Determine health state based on failure count
	var state HealthState
	if failures >= hm.maxFailures {
		state = HealthUnhealthy
	} else if failures > 1 {
		state = HealthDegraded
	} else {
		state = HealthDegraded
	}

	// Get previous status for last success time
	lastSuccess := time.Time{}
	if prev, exists := hm.healthStatus[agentID]; exists {
		lastSuccess = prev.LastSuccess
	}

	// Update health status
	status := &HealthStatus{
		AgentID:          agentID,
		Status:           state,
		LastCheck:        hm.clock.Now(),
		LastSuccess:      lastSuccess,
		ConsecutiveFails: failures,
		ResponseTime:     responseTime,
		ErrorMessage:     err.Error(),
		Metrics:          hm.getAgentMetrics(agentID),
	}

	hm.healthStatus[agentID] = status

	hm.logger.Warn("Health check failed (agent_id: %s, failures: %d, state: %s, error: %v)", agentID, failures, state, err)

	// Send alert if unhealthy
	if state == HealthUnhealthy {
		hm.sendHealthAlert(agentID, status)
	}
}

// updateHealthStatus updates the health status of an agent
func (hm *HealthMonitor) updateHealthStatus(agentID AgentID, state HealthState, responseTime time.Duration, errorMsg string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	status := &HealthStatus{
		AgentID:      agentID,
		Status:       state,
		LastCheck:    hm.clock.Now(),
		ResponseTime: responseTime,
		ErrorMessage: errorMsg,
		Metrics:      hm.getAgentMetrics(agentID),
	}

	hm.healthStatus[agentID] = status
}

// getAgentMetrics retrieves metrics for an agent
func (hm *HealthMonitor) getAgentMetrics(agentID AgentID) HealthMetrics {
	// This is a placeholder - real implementation would collect actual metrics
	return HealthMetrics{
		CPUUsage:          25.5,
		MemoryUsage:       1024 * 1024 * 256, // 256 MB
		MessagesProcessed: 1000,
		ErrorRate:         0.01,
		AverageLatency:    50 * time.Millisecond,
	}
}

// sendHealthAlert sends an alert for unhealthy agents
func (hm *HealthMonitor) sendHealthAlert(agentID AgentID, status *HealthStatus) {
	msg := &AgentMessage{
		Type:     ErrorNotification,
		Sender:   AgentID("health_monitor"),
		Receiver: AgentID("orchestrator"),
		Payload: map[string]interface{}{
			"agent_id":   agentID,
			"status":     status.Status,
			"failures":   status.ConsecutiveFails,
			"error":      status.ErrorMessage,
			"alert_type": "agent_unhealthy",
		},
		Priority: PriorityCritical,
		Context: MessageContext{
			Metadata: map[string]interface{}{
				"health_check": true,
			},
		},
	}

	if err := hm.messageBus.Send(msg); err != nil {
		hm.logger.Error("Failed to send health alert (agent_id: %s, error: %v)", agentID, err)
	}
}

// analyzeOverallHealth analyzes the overall health of the system
func (hm *HealthMonitor) analyzeOverallHealth() {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	totalAgents := len(hm.healthStatus)
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	for _, status := range hm.healthStatus {
		switch status.Status {
		case HealthHealthy:
			healthyCount++
		case HealthDegraded:
			degradedCount++
		case HealthUnhealthy:
			unhealthyCount++
		}
	}

	hm.logger.Info("Overall health status (total_agents: %d, healthy: %d, degraded: %d, unhealthy: %d)", totalAgents, healthyCount, degradedCount, unhealthyCount)

	// Send overall health notification if issues exist
	if unhealthyCount > 0 || degradedCount > totalAgents/2 {
		hm.sendOverallHealthAlert(healthyCount, degradedCount, unhealthyCount)
	}
}

// sendOverallHealthAlert sends an alert about overall system health
func (hm *HealthMonitor) sendOverallHealthAlert(healthy, degraded, unhealthy int) {
	msg := &AgentMessage{
		Type:     ErrorNotification,
		Sender:   AgentID("health_monitor"),
		Receiver: AgentID("orchestrator"),
		Payload: map[string]interface{}{
			"healthy_count":   healthy,
			"degraded_count":  degraded,
			"unhealthy_count": unhealthy,
			"alert_type":      "system_health_degraded",
		},
		Priority: PriorityHigh,
	}

	if err := hm.messageBus.Send(msg); err != nil {
		hm.logger.Error("Failed to send overall health alert (error: %v)", err)
	}
}

// GetHealthStatus returns the current health status of an agent
func (hm *HealthMonitor) GetHealthStatus(agentID AgentID) (*HealthStatus, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	status, exists := hm.healthStatus[agentID]
	if !exists {
		return nil, fmt.Errorf("no health status for agent %s", agentID)
	}

	// Return a copy
	statusCopy := *status
	return &statusCopy, nil
}

// GetAllHealthStatus returns health status for all agents
func (hm *HealthMonitor) GetAllHealthStatus() map[AgentID]*HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Create a copy of the map
	statusCopy := make(map[AgentID]*HealthStatus)
	for id, status := range hm.healthStatus {
		s := *status
		statusCopy[id] = &s
	}

	return statusCopy
}

// RecoveryManager handles agent recovery operations
type RecoveryManager struct {
	registry   *AgentRegistry
	messageBus *MessageBus
	logger     *logger.Logger
	clock      clock.Clock

	recoveryStrategies map[AgentID]RecoveryStrategy
	mu                 sync.RWMutex
}

// RecoveryStrategy defines how to recover an unhealthy agent
type RecoveryStrategy struct {
	MaxRetries     int
	RetryDelay     time.Duration
	BackoffFactor  float64
	RecoveryAction RecoveryAction
}

// RecoveryAction represents an action to take for recovery
type RecoveryAction string

const (
	RecoveryRestart  RecoveryAction = "restart"
	RecoveryReload   RecoveryAction = "reload"
	RecoveryRecreate RecoveryAction = "recreate"
	RecoveryEscalate RecoveryAction = "escalate"
)

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(registry *AgentRegistry, messageBus *MessageBus) *RecoveryManager {
	return NewRecoveryManagerWithClock(registry, messageBus, clock.NewRealClock())
}

// NewRecoveryManagerWithClock creates a new recovery manager with a custom clock
func NewRecoveryManagerWithClock(registry *AgentRegistry, messageBus *MessageBus, clk clock.Clock) *RecoveryManager {
	return &RecoveryManager{
		registry:           registry,
		messageBus:         messageBus,
		logger:             logger.GetLogger(),
		clock:              clk,
		recoveryStrategies: make(map[AgentID]RecoveryStrategy),
	}
}

// SetRecoveryStrategy sets the recovery strategy for an agent
func (rm *RecoveryManager) SetRecoveryStrategy(agentID AgentID, strategy RecoveryStrategy) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.recoveryStrategies[agentID] = strategy
}

// RecoverAgent attempts to recover an unhealthy agent
func (rm *RecoveryManager) RecoverAgent(agentID AgentID, healthStatus *HealthStatus) error {
	rm.mu.RLock()
	strategy, hasStrategy := rm.recoveryStrategies[agentID]
	rm.mu.RUnlock()

	if !hasStrategy {
		// Use default strategy
		strategy = RecoveryStrategy{
			MaxRetries:     3,
			RetryDelay:     5 * time.Second,
			BackoffFactor:  2.0,
			RecoveryAction: RecoveryRestart,
		}
	}

	rm.logger.Info("Starting agent recovery (agent_id: %s, action: %s, max_retries: %d)", agentID, strategy.RecoveryAction, strategy.MaxRetries)

	retryDelay := strategy.RetryDelay
	for attempt := 1; attempt <= strategy.MaxRetries; attempt++ {
		err := rm.executeRecoveryAction(agentID, strategy.RecoveryAction)
		if err == nil {
			rm.logger.Info("Agent recovery successful (agent_id: %s, attempt: %d)", agentID, attempt)
			return nil
		}

		// If it's an unknown action error, return immediately without retrying
		if strings.Contains(err.Error(), "unknown recovery action") {
			return err
		}

		rm.logger.Warn("Recovery attempt failed (agent_id: %s, attempt: %d, error: %v)", agentID, attempt, err)

		if attempt < strategy.MaxRetries {
			select {
			case <-rm.clock.After(retryDelay):
			case <-context.Background().Done():
				return fmt.Errorf("recovery canceled during retry delay")
			}
			retryDelay = time.Duration(float64(retryDelay) * strategy.BackoffFactor)
		}
	}

	return fmt.Errorf("failed to recover agent %s after %d attempts", agentID, strategy.MaxRetries)
}

// executeRecoveryAction executes a specific recovery action
func (rm *RecoveryManager) executeRecoveryAction(agentID AgentID, action RecoveryAction) error {
	switch action {
	case RecoveryRestart:
		return rm.restartAgent(agentID)

	case RecoveryReload:
		return rm.reloadAgent(agentID)

	case RecoveryRecreate:
		return rm.recreateAgent(agentID)

	case RecoveryEscalate:
		return rm.escalateRecovery(agentID)

	default:
		return fmt.Errorf("unknown recovery action: %s", action)
	}
}

// restartAgent restarts an agent
func (rm *RecoveryManager) restartAgent(agentID AgentID) error {
	// Stop the agent but preserve configuration
	if err := rm.registry.StopAgentOnly(agentID); err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}

	// Wait a moment for graceful shutdown
	select {
	case <-rm.clock.After(5 * time.Millisecond):
	case <-context.Background().Done():
		return fmt.Errorf("shutdown interrupted")
	}

	// Get agent config
	config, err := rm.registry.GetAgentConfig(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent config: %w", err)
	}

	// Recreate the agent
	if err := rm.registry.CreateAgent(agentID, config); err != nil {
		return fmt.Errorf("failed to recreate agent: %w", err)
	}

	return nil
}

// reloadAgent reloads an agent's configuration
func (rm *RecoveryManager) reloadAgent(agentID AgentID) error {
	// Send reload message to agent
	msg := &AgentMessage{
		Type:     TaskAssignment,
		Sender:   AgentID("recovery_manager"),
		Receiver: agentID,
		Payload: map[string]interface{}{
			"action": "reload_config",
		},
		Priority: PriorityCritical,
	}

	return rm.messageBus.Send(msg)
}

// recreateAgent completely recreates an agent
func (rm *RecoveryManager) recreateAgent(agentID AgentID) error {
	// This is similar to restart but may involve additional cleanup
	return rm.restartAgent(agentID)
}

// escalateRecovery escalates the recovery to human intervention
func (rm *RecoveryManager) escalateRecovery(agentID AgentID) error {
	msg := &AgentMessage{
		Type:     ErrorNotification,
		Sender:   AgentID("recovery_manager"),
		Receiver: AgentID("orchestrator"),
		Payload: map[string]interface{}{
			"agent_id":   agentID,
			"alert_type": "recovery_failed",
			"action":     "human_intervention_required",
		},
		Priority: PriorityCritical,
	}

	return rm.messageBus.Send(msg)
}
