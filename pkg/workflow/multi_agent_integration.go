package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/fumiya-kume/cca/pkg/agents"
	"github.com/fumiya-kume/cca/pkg/config"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// MultiAgentWorkflowIntegration integrates the multi-agent system into the main workflow
type MultiAgentWorkflowIntegration struct {
	configManager   *config.AgentConfigManager
	agentRegistry   *agents.AgentRegistry
	messageBus      *agents.MessageBus
	multiAgentStage *MultiAgentStage
	healthMonitor   *agents.HealthMonitor
	logger          *logger.Logger
}

// WorkflowContext provides context for workflow execution
type WorkflowContext struct {
	IssueNumber   int                        `json:"issue_number"`
	Repository    string                     `json:"repository"`
	Branch        string                     `json:"branch"`
	WorkspacePath string                     `json:"workspace_path"`
	Configuration *agents.AgentConfiguration `json:"configuration"`
	Metadata      map[string]interface{}     `json:"metadata"`
}

// MultiAgentExecutionResult represents the result of multi-agent execution
type MultiAgentExecutionResult struct {
	Success            bool                                   `json:"success"`
	ExecutedAgents     []agents.AgentID                       `json:"executed_agents"`
	AgentResults       map[agents.AgentID]*agents.AgentResult `json:"agent_results"`
	ConsolidatedResult *ConsolidatedResult                    `json:"consolidated_result"`
	AutoFixesApplied   []FixExecution                         `json:"auto_fixes_applied"`
	Report             *MultiAgentReport                      `json:"report"`
	Duration           time.Duration                          `json:"duration"`
	Timestamp          time.Time                              `json:"timestamp"`
	Errors             []error                                `json:"errors,omitempty"`
	Warnings           []string                               `json:"warnings,omitempty"`
}

// NewMultiAgentWorkflowIntegration creates a new multi-agent workflow integration
func NewMultiAgentWorkflowIntegration(logger *logger.Logger) (*MultiAgentWorkflowIntegration, error) {
	// Initialize configuration manager
	configManager := config.NewAgentConfigManager(logger)
	if err := configManager.LoadConfiguration(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize message bus
	messageBus := agents.NewMessageBus(1000)

	// Initialize agent registry
	agentRegistry := agents.NewAgentRegistry(messageBus)

	// Initialize health monitor
	healthMonitor := agents.NewHealthMonitor(agentRegistry, messageBus)

	// Initialize multi-agent stage
	multiAgentStage := NewMultiAgentStage(messageBus, agentRegistry, logger)

	integration := &MultiAgentWorkflowIntegration{
		configManager:   configManager,
		agentRegistry:   agentRegistry,
		messageBus:      messageBus,
		multiAgentStage: multiAgentStage,
		healthMonitor:   healthMonitor,
		logger:          logger,
	}

	// Register configuration change callback
	configManager.RegisterCallback(integration.handleConfigurationChange)

	// Start health monitoring
	if err := healthMonitor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start health monitor: %w", err)
	}

	// Start hot configuration reloading
	if err := configManager.StartHotReload(); err != nil {
		logger.Warn("Failed to start configuration hot reload (error: %v)", err)
	}

	logger.Info("Multi-agent workflow integration initialized successfully")
	return integration, nil
}

// Initialize sets up all agents according to configuration
func (mawi *MultiAgentWorkflowIntegration) Initialize(ctx context.Context) error {
	mawi.logger.Info("Initializing multi-agent system")

	config := mawi.configManager.GetConfiguration()

	// Initialize all enabled agents
	enabledAgents := config.GetEnabledAgents()
	for _, agentID := range enabledAgents {
		agentConfig, _ := config.GetAgentConfig(agentID)

		if err := mawi.agentRegistry.RegisterAgent(agentID, agentConfig); err != nil {
			mawi.logger.Error("Failed to register agent (agent: %s, error: %v)", agentID, err)
			continue
		}

		if err := mawi.agentRegistry.StartAgent(ctx, agentID); err != nil {
			mawi.logger.Error("Failed to start agent (agent: %s, error: %v)", agentID, err)
			continue
		}

		mawi.logger.Info("Agent initialized successfully (agent: %s)", agentID)
	}

	mawi.logger.Info("Multi-agent system initialization completed (enabled_agents: %d)", len(enabledAgents))
	return nil
}

// ExecuteEnhancedStage6 executes the enhanced Stage 6 with multi-agent coordination
func (mawi *MultiAgentWorkflowIntegration) ExecuteEnhancedStage6(ctx context.Context, workflowCtx *WorkflowContext) (*MultiAgentExecutionResult, error) {
	startTime := time.Now()
	mawi.logger.Info("Starting enhanced Stage 6 execution (issue: %d, repository: %s)", workflowCtx.IssueNumber, workflowCtx.Repository)

	result := &MultiAgentExecutionResult{
		ExecutedAgents: []agents.AgentID{},
		AgentResults:   make(map[agents.AgentID]*agents.AgentResult),
		Timestamp:      startTime,
		Errors:         []error{},
		Warnings:       []string{},
	}

	// Prepare workspace configuration
	workspaceConfig := &agents.WorkspaceConfig{
		WorkspacePath: workflowCtx.WorkspacePath,
		ProjectType:   workflowCtx.Configuration.Project.Type,
		Configuration: workflowCtx.Metadata,
	}

	// Execute multi-agent stage
	stageResult, err := mawi.multiAgentStage.Execute(ctx, workspaceConfig)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Errorf("multi-agent stage execution failed: %w", err))
		result.Duration = time.Since(startTime)
		return result, err
	}

	// Extract results from stage execution
	if stageResult.Metadata != nil {
		if agentResults, ok := stageResult.Metadata["agent_results"].(map[agents.AgentID]*agents.AgentResult); ok {
			result.AgentResults = agentResults
			for agentID := range agentResults {
				result.ExecutedAgents = append(result.ExecutedAgents, agentID)
			}
		}

		if consolidated, ok := stageResult.Metadata["consolidated"].(*ConsolidatedResult); ok {
			result.ConsolidatedResult = consolidated
		}

		if autoFixes, ok := stageResult.Metadata["auto_fixes_applied"].([]FixExecution); ok {
			result.AutoFixesApplied = autoFixes
		}

		if report, ok := stageResult.Metadata["report"].(*MultiAgentReport); ok {
			result.Report = report
		}
	}

	result.Success = stageResult.Status == "completed"
	result.Duration = time.Since(startTime)

	// Generate summary warnings
	result.Warnings = mawi.generateSummaryWarnings(result)

	mawi.logger.Info("Enhanced Stage 6 execution completed (success: %t, duration: %v, executed_agents: %d, auto_fixes: %d)", result.Success, result.Duration, len(result.ExecutedAgents), len(result.AutoFixesApplied))

	return result, nil
}

// Shutdown gracefully shuts down the multi-agent system
func (mawi *MultiAgentWorkflowIntegration) Shutdown(ctx context.Context) error {
	mawi.logger.Info("Shutting down multi-agent system")

	// Stop configuration hot reload
	if err := mawi.configManager.StopHotReload(); err != nil {
		mawi.logger.Error("Failed to stop configuration hot reload (error: %v)", err)
	}

	// Stop health monitor
	if err := mawi.healthMonitor.Stop(); err != nil {
		mawi.logger.Error("Failed to stop health monitor (error: %v)", err)
	}

	// Stop all agents
	enabledAgents := mawi.configManager.GetConfiguration().GetEnabledAgents()
	for _, agentID := range enabledAgents {
		if err := mawi.agentRegistry.StopAgent(agentID); err != nil {
			mawi.logger.Error("Failed to stop agent (agent: %s, error: %v)", agentID, err)
		}
	}

	// Shutdown message bus
	if err := mawi.messageBus.Shutdown(ctx); err != nil {
		mawi.logger.Error("Failed to shutdown message bus (error: %v)", err)
	}

	mawi.logger.Info("Multi-agent system shutdown completed")
	return nil
}

// GetAgentStatus returns the status of all agents
func (mawi *MultiAgentWorkflowIntegration) GetAgentStatus() map[agents.AgentID]agents.AgentStatus {
	status := make(map[agents.AgentID]agents.AgentStatus)

	enabledAgents := mawi.configManager.GetConfiguration().GetEnabledAgents()
	for _, agentID := range enabledAgents {
		if agent, err := mawi.agentRegistry.GetAgent(agentID); err == nil {
			status[agentID] = agent.GetStatus()
		} else {
			status[agentID] = agents.StatusOffline
		}
	}

	return status
}

// GetSystemHealth returns overall system health information
func (mawi *MultiAgentWorkflowIntegration) GetSystemHealth() *SystemHealthReport {
	agentStatus := mawi.GetAgentStatus()
	config := mawi.configManager.GetConfiguration()

	healthy := 0
	total := 0
	for _, status := range agentStatus {
		total++
		if status == agents.StatusIdle || status == agents.StatusBusy {
			healthy++
		}
	}

	return &SystemHealthReport{
		Timestamp:            time.Now(),
		OverallHealth:        float64(healthy) / float64(total) * 100,
		TotalAgents:          total,
		HealthyAgents:        healthy,
		AgentStatus:          agentStatus,
		ConfigurationVersion: mawi.configManager.GetVersion(),
		LastConfigUpdate:     mawi.configManager.GetLastModified(),
		MessageBusHealth:     mawi.messageBus.GetHealth(),
		Recommendations:      mawi.generateHealthRecommendations(agentStatus, config),
	}
}

// SystemHealthReport represents system health information
type SystemHealthReport struct {
	Timestamp            time.Time                             `json:"timestamp"`
	OverallHealth        float64                               `json:"overall_health"` // 0-100
	TotalAgents          int                                   `json:"total_agents"`
	HealthyAgents        int                                   `json:"healthy_agents"`
	AgentStatus          map[agents.AgentID]agents.AgentStatus `json:"agent_status"`
	ConfigurationVersion int                                   `json:"configuration_version"`
	LastConfigUpdate     time.Time                             `json:"last_config_update"`
	MessageBusHealth     *agents.MessageBusHealth              `json:"message_bus_health"`
	Recommendations      []string                              `json:"recommendations"`
}

// ReconfigureAgent dynamically reconfigures a specific agent
func (mawi *MultiAgentWorkflowIntegration) ReconfigureAgent(ctx context.Context, agentID agents.AgentID, newConfig agents.AgentConfig) error {
	mawi.logger.Info("Reconfiguring agent (agent: %s)", agentID)

	// Validate new configuration
	if err := mawi.configManager.GetConfiguration().UpdateAgentConfig(agentID, newConfig); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Stop existing agent instance
	if err := mawi.agentRegistry.StopAgent(agentID); err != nil {
		mawi.logger.Warn("Failed to stop agent during reconfiguration (agent: %s, error: %v)", agentID, err)
	}

	// Update registration with new config
	if err := mawi.agentRegistry.RegisterAgent(agentID, newConfig); err != nil {
		return fmt.Errorf("failed to register agent with new config: %w", err)
	}

	// Start agent with new configuration
	if err := mawi.agentRegistry.StartAgent(ctx, agentID); err != nil {
		return fmt.Errorf("failed to start agent with new config: %w", err)
	}

	mawi.logger.Info("Agent reconfiguration completed (agent: %s)", agentID)
	return nil
}

// EnableAgent enables a previously disabled agent
func (mawi *MultiAgentWorkflowIntegration) EnableAgent(ctx context.Context, agentID agents.AgentID) error {
	config := mawi.configManager.GetConfiguration()
	config.EnableAgent(agentID)

	agentConfig, exists := config.GetAgentConfig(agentID)
	if !exists {
		return fmt.Errorf("agent %s not found in configuration", agentID)
	}

	return mawi.ReconfigureAgent(ctx, agentID, agentConfig)
}

// DisableAgent disables an active agent
func (mawi *MultiAgentWorkflowIntegration) DisableAgent(ctx context.Context, agentID agents.AgentID) error {
	config := mawi.configManager.GetConfiguration()
	config.DisableAgent(agentID)

	return mawi.agentRegistry.StopAgent(agentID)
}

// GetPerformanceMetrics returns performance metrics for the multi-agent system
func (mawi *MultiAgentWorkflowIntegration) GetPerformanceMetrics() *PerformanceMetrics {
	messageBusMetrics := mawi.messageBus.GetMetrics()
	agentRegistryMetrics := mawi.agentRegistry.GetMetrics()
	healthMonitorMetrics := mawi.healthMonitor.GetMetrics()

	return &PerformanceMetrics{
		Timestamp:            time.Now(),
		MessageBusMetrics:    &messageBusMetrics,
		AgentRegistryMetrics: agentRegistryMetrics,
		HealthMonitorMetrics: healthMonitorMetrics,
	}
}

// PerformanceMetrics represents system performance metrics
type PerformanceMetrics struct {
	Timestamp            time.Time                    `json:"timestamp"`
	MessageBusMetrics    *agents.MessageBusMetrics    `json:"message_bus_metrics"`
	AgentRegistryMetrics *agents.AgentRegistryMetrics `json:"agent_registry_metrics"`
	HealthMonitorMetrics *agents.HealthMonitorMetrics `json:"health_monitor_metrics"`
}

// handleConfigurationChange handles configuration changes
func (mawi *MultiAgentWorkflowIntegration) handleConfigurationChange(oldConfig, newConfig *agents.AgentConfiguration) error {
	mawi.logger.Info("Configuration changed, applying updates")

	// Compare agent configurations and apply changes
	ctx := context.Background()

	// Check for newly enabled agents
	for agentID := range newConfig.Agents {
		newAgentConfig, _ := newConfig.GetAgentConfig(agentID)
		oldAgentConfig, hadOldConfig := oldConfig.GetAgentConfig(agentID)

		if newAgentConfig.Enabled && (!hadOldConfig || !oldAgentConfig.Enabled) {
			// Agent was enabled
			if err := mawi.EnableAgent(ctx, agentID); err != nil {
				mawi.logger.Error("Failed to enable agent (agent: %s, error: %v)", agentID, err)
			}
		} else if !newAgentConfig.Enabled && hadOldConfig && oldAgentConfig.Enabled {
			// Agent was disabled
			if err := mawi.DisableAgent(ctx, agentID); err != nil {
				mawi.logger.Error("Failed to disable agent (agent: %s, error: %v)", agentID, err)
			}
		} else if newAgentConfig.Enabled && hadOldConfig && oldAgentConfig.Enabled {
			// Agent configuration changed
			if !mawi.agentConfigsEqual(oldAgentConfig, newAgentConfig) {
				if err := mawi.ReconfigureAgent(ctx, agentID, newAgentConfig); err != nil {
					mawi.logger.Error("Failed to reconfigure agent (agent: %s, error: %v)", agentID, err)
				}
			}
		}
	}

	mawi.logger.Info("Configuration change processing completed")
	return nil
}

// generateSummaryWarnings generates summary warnings for execution results
func (mawi *MultiAgentWorkflowIntegration) generateSummaryWarnings(result *MultiAgentExecutionResult) []string {
	warnings := []string{}

	if result.Report != nil {
		if result.Report.SuccessfulAgents < result.Report.ExecutedAgents {
			warnings = append(warnings,
				fmt.Sprintf("%d agents failed during execution",
					result.Report.ExecutedAgents-result.Report.SuccessfulAgents))
		}

		if result.Report.ConflictsResolved > 0 {
			warnings = append(warnings,
				fmt.Sprintf("%d conflicts were resolved between agent recommendations",
					result.Report.ConflictsResolved))
		}

		if result.Report.OverallScore < 80.0 {
			warnings = append(warnings,
				fmt.Sprintf("Overall quality score is low: %.1f", result.Report.OverallScore))
		}
	}

	if len(result.AutoFixesApplied) > 10 {
		warnings = append(warnings,
			fmt.Sprintf("High number of auto-fixes applied: %d", len(result.AutoFixesApplied)))
	}

	return warnings
}

// generateHealthRecommendations generates health recommendations
func (mawi *MultiAgentWorkflowIntegration) generateHealthRecommendations(agentStatus map[agents.AgentID]agents.AgentStatus, config *agents.AgentConfiguration) []string {
	recommendations := []string{}

	offlineCount := 0
	for _, status := range agentStatus {
		if status == agents.StatusOffline || status == agents.StatusError {
			offlineCount++
		}
	}

	if offlineCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Restart %d offline/error agents", offlineCount))
	}

	if config.Global.MaxConcurrentAgents > len(agentStatus)*2 {
		recommendations = append(recommendations,
			"Consider reducing max_concurrent_agents setting")
	}

	return recommendations
}

// agentConfigsEqual compares two agent configurations for equality
func (mawi *MultiAgentWorkflowIntegration) agentConfigsEqual(a, b agents.AgentConfig) bool {
	return a.Enabled == b.Enabled &&
		a.MaxInstances == b.MaxInstances &&
		a.Timeout == b.Timeout &&
		a.RetryAttempts == b.RetryAttempts &&
		a.Priority == b.Priority
	// Note: This is a simplified comparison, full implementation would compare all fields
}
