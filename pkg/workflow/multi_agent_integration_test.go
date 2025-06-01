package workflow

import (
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/agents"
	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// Helper function to create test logger
func createTestLogger() *logger.Logger {
	testLogger, _ := logger.New(logger.Config{
		Level: logger.LevelInfo,
	})
	return testLogger
}

// Helper function to create a basic integration for testing simple functions
func createBasicIntegration() *MultiAgentWorkflowIntegration {
	testLogger := createTestLogger()
	return &MultiAgentWorkflowIntegration{
		logger: testLogger,
	}
}

func TestNewMultiAgentWorkflowIntegration(t *testing.T) {
	t.Run("constructor requires logger", func(t *testing.T) {
		// Test that the constructor would need a logger
		// This is a basic structural test since full mocking is complex
		testLogger := createTestLogger()
		assert.NotNil(t, testLogger)
	})
}

func TestMultiAgentWorkflowIntegration_GenerateSummaryWarnings(t *testing.T) {
	integration := createBasicIntegration()

	t.Run("no warnings for good results", func(t *testing.T) {
		result := &MultiAgentExecutionResult{
			Report: &MultiAgentReport{
				ExecutedAgents:    5,
				SuccessfulAgents:  5,
				ConflictsResolved: 0,
				OverallScore:      95.0,
			},
			AutoFixesApplied: []FixExecution{{}},
		}

		warnings := integration.generateSummaryWarnings(result)
		assert.Equal(t, 0, len(warnings))
	})

	t.Run("warnings for problematic results", func(t *testing.T) {
		result := &MultiAgentExecutionResult{
			Report: &MultiAgentReport{
				ExecutedAgents:    5,
				SuccessfulAgents:  3,
				ConflictsResolved: 2,
				OverallScore:      70.0,
			},
			AutoFixesApplied: make([]FixExecution, 15), // High number of fixes
		}

		warnings := integration.generateSummaryWarnings(result)
		assert.Greater(t, len(warnings), 0)

		warningText := ""
		for _, warning := range warnings {
			warningText += warning
		}
		assert.Contains(t, warningText, "agents failed")
		assert.Contains(t, warningText, "conflicts were resolved")
		assert.Contains(t, warningText, "quality score is low")
		assert.Contains(t, warningText, "High number of auto-fixes")
	})
}

func TestMultiAgentWorkflowIntegration_GenerateHealthRecommendations(t *testing.T) {
	integration := createBasicIntegration()

	t.Run("recommendations for offline agents", func(t *testing.T) {
		agentStatus := map[agents.AgentID]agents.AgentStatus{
			agents.SecurityAgentID:      agents.StatusOffline,
			agents.DocumentationAgentID: agents.StatusError,
			agents.ArchitectureAgentID:  agents.StatusIdle,
		}

		config := &agents.AgentConfiguration{
			Global: agents.GlobalAgentConfig{
				MaxConcurrentAgents: 10, // Too high for number of agents
			},
		}

		recommendations := integration.generateHealthRecommendations(agentStatus, config)
		assert.Greater(t, len(recommendations), 0)

		recommendationText := ""
		for _, rec := range recommendations {
			recommendationText += rec
		}
		assert.Contains(t, recommendationText, "Restart 2 offline/error agents")
		assert.Contains(t, recommendationText, "reducing max_concurrent_agents")
	})

	t.Run("no recommendations for healthy system", func(t *testing.T) {
		agentStatus := map[agents.AgentID]agents.AgentStatus{
			agents.SecurityAgentID:      agents.StatusIdle,
			agents.DocumentationAgentID: agents.StatusBusy,
		}

		config := &agents.AgentConfiguration{
			Global: agents.GlobalAgentConfig{
				MaxConcurrentAgents: 3, // Reasonable setting
			},
		}

		recommendations := integration.generateHealthRecommendations(agentStatus, config)
		assert.Equal(t, 0, len(recommendations))
	})
}

func TestMultiAgentWorkflowIntegration_AgentConfigsEqual(t *testing.T) {
	integration := createBasicIntegration()

	config1 := agents.AgentConfig{
		Enabled:       true,
		MaxInstances:  2,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		Priority:      agents.PriorityHigh,
	}

	config2 := agents.AgentConfig{
		Enabled:       true,
		MaxInstances:  2,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		Priority:      agents.PriorityHigh,
	}

	config3 := agents.AgentConfig{
		Enabled:       false,
		MaxInstances:  2,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		Priority:      agents.PriorityHigh,
	}

	assert.True(t, integration.agentConfigsEqual(config1, config2))
	assert.False(t, integration.agentConfigsEqual(config1, config3))
}

func TestWorkflowContext(t *testing.T) {
	ctx := &WorkflowContext{
		IssueNumber:   123,
		Repository:    "test/repo",
		Branch:        "main",
		WorkspacePath: "/tmp/workspace",
		Configuration: &agents.AgentConfiguration{
			Project: agents.ProjectAgentConfig{
				Type: "go",
			},
		},
		Metadata: map[string]interface{}{
			"test": "metadata",
		},
	}

	assert.Equal(t, 123, ctx.IssueNumber)
	assert.Equal(t, "test/repo", ctx.Repository)
	assert.Equal(t, "main", ctx.Branch)
	assert.Equal(t, "/tmp/workspace", ctx.WorkspacePath)
	assert.NotNil(t, ctx.Configuration)
	assert.NotNil(t, ctx.Metadata)
}

func TestSystemHealthReport(t *testing.T) {
	now := time.Now()
	report := &SystemHealthReport{
		Timestamp:            now,
		OverallHealth:        85.5,
		TotalAgents:          5,
		HealthyAgents:        4,
		AgentStatus:          map[agents.AgentID]agents.AgentStatus{},
		ConfigurationVersion: 1,
		LastConfigUpdate:     now,
		MessageBusHealth:     &agents.MessageBusHealth{},
		Recommendations:      []string{"test recommendation"},
	}

	assert.Equal(t, now, report.Timestamp)
	assert.Equal(t, 85.5, report.OverallHealth)
	assert.Equal(t, 5, report.TotalAgents)
	assert.Equal(t, 4, report.HealthyAgents)
	assert.Equal(t, 1, report.ConfigurationVersion)
	assert.Equal(t, 1, len(report.Recommendations))
}

func TestPerformanceMetrics(t *testing.T) {
	now := time.Now()
	metrics := &PerformanceMetrics{
		Timestamp:            now,
		MessageBusMetrics:    &agents.MessageBusMetrics{},
		AgentRegistryMetrics: &agents.AgentRegistryMetrics{},
		HealthMonitorMetrics: &agents.HealthMonitorMetrics{},
	}

	assert.Equal(t, now, metrics.Timestamp)
	assert.NotNil(t, metrics.MessageBusMetrics)
	assert.NotNil(t, metrics.AgentRegistryMetrics)
	assert.NotNil(t, metrics.HealthMonitorMetrics)
}

func TestMultiAgentExecutionResult(t *testing.T) {
	now := time.Now()
	result := &MultiAgentExecutionResult{
		Success:            true,
		ExecutedAgents:     []agents.AgentID{agents.SecurityAgentID},
		AgentResults:       map[agents.AgentID]*agents.AgentResult{},
		ConsolidatedResult: &ConsolidatedResult{},
		AutoFixesApplied:   []FixExecution{},
		Report:             &MultiAgentReport{},
		Duration:           2 * time.Minute,
		Timestamp:          now,
		Errors:             []error{},
		Warnings:           []string{},
	}

	assert.True(t, result.Success)
	assert.Equal(t, 1, len(result.ExecutedAgents))
	assert.Equal(t, agents.SecurityAgentID, result.ExecutedAgents[0])
	assert.Equal(t, 2*time.Minute, result.Duration)
	assert.Equal(t, now, result.Timestamp)
	assert.NotNil(t, result.ConsolidatedResult)
	assert.NotNil(t, result.Report)
}