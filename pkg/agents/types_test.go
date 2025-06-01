package agents

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentID(t *testing.T) {
	tests := []struct {
		name     string
		agentID  AgentID
		expected string
	}{
		{"Security Agent", SecurityAgentID, "security"},
		{"Architecture Agent", ArchitectureAgentID, "architecture"},
		{"Documentation Agent", DocumentationAgentID, "documentation"},
		{"Testing Agent", TestingAgentID, "testing"},
		{"Performance Agent", PerformanceAgentID, "performance"},
		{"Claude Code Agent", ClaudeCodeAgentID, "claude-code"},
		{"Git Agent", GitAgentID, "git"},
		{"GitHub Agent", GitHubAgentID, "github"},
		{"Analysis Agent", AnalysisAgentID, "analysis"},
		{"Workflow Orchestrator", WorkflowOrchestratorID, "workflow_orchestrator"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.agentID))
		})
	}
}

func TestMessageType(t *testing.T) {
	tests := []struct {
		name        string
		messageType MessageType
		expected    string
	}{
		{"Task Assignment", TaskAssignment, "task_assignment"},
		{"Progress Update", ProgressUpdate, "progress_update"},
		{"Result Reporting", ResultReporting, "result_reporting"},
		{"Error Notification", ErrorNotification, "error_notification"},
		{"Health Check", HealthCheck, "health_check"},
		{"Resource Request", ResourceRequest, "resource_request"},
		{"Collaboration Request", CollaborationRequest, "collaboration_request"},
		{"Feedback Provision", FeedbackProvision, "feedback_provision"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.messageType))
		})
	}
}

func TestPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		expected int
	}{
		{"Low Priority", PriorityLow, 0},
		{"Medium Priority", PriorityMedium, 1},
		{"High Priority", PriorityHigh, 2},
		{"Critical Priority", PriorityCritical, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.priority))
		})
	}
}

func TestPriorityOrdering(t *testing.T) {
	assert.True(t, PriorityLow < PriorityMedium)
	assert.True(t, PriorityMedium < PriorityHigh)
	assert.True(t, PriorityHigh < PriorityCritical)
}

func TestAgentStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   AgentStatus
		expected string
	}{
		{"Idle", StatusIdle, "idle"},
		{"Busy", StatusBusy, "busy"},
		{"Error", StatusError, "error"},
		{"Offline", StatusOffline, "offline"},
		{"Starting", StatusStarting, "starting"},
		{"Stopping", StatusStopping, "stopping"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestMessageContext(t *testing.T) {
	context := MessageContext{
		WorkflowID:  "workflow-123",
		StageID:     "stage-456",
		IssueNumber: 789,
		Repository:  "owner/repo",
		Branch:      "feature/test",
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	assert.Equal(t, "workflow-123", context.WorkflowID)
	assert.Equal(t, "stage-456", context.StageID)
	assert.Equal(t, 789, context.IssueNumber)
	assert.Equal(t, "owner/repo", context.Repository)
	assert.Equal(t, "feature/test", context.Branch)
	assert.Equal(t, "value1", context.Metadata["key1"])
	assert.Equal(t, 42, context.Metadata["key2"])
}

func TestAgentMessage(t *testing.T) {
	timestamp := time.Now()
	context := MessageContext{
		WorkflowID: "workflow-123",
		Repository: "owner/repo",
	}

	msg := AgentMessage{
		ID:            "msg-123",
		Type:          TaskAssignment,
		Sender:        SecurityAgentID,
		Receiver:      ArchitectureAgentID,
		Payload:       map[string]interface{}{"task": "analyze"},
		Timestamp:     timestamp,
		Priority:      PriorityHigh,
		CorrelationID: "corr-456",
		Context:       context,
	}

	assert.Equal(t, "msg-123", msg.ID)
	assert.Equal(t, TaskAssignment, msg.Type)
	assert.Equal(t, SecurityAgentID, msg.Sender)
	assert.Equal(t, ArchitectureAgentID, msg.Receiver)
	assert.Equal(t, PriorityHigh, msg.Priority)
	assert.Equal(t, "corr-456", msg.CorrelationID)
	assert.Equal(t, timestamp, msg.Timestamp)
	assert.Equal(t, context, msg.Context)

	payload, ok := msg.Payload.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "analyze", payload["task"])
}

func TestResourceLimits(t *testing.T) {
	limits := ResourceLimits{
		MaxMemoryMB:    512,
		MaxCPUPercent:  75.5,
		MaxDiskSpaceGB: 10,
	}

	assert.Equal(t, 512, limits.MaxMemoryMB)
	assert.Equal(t, 75.5, limits.MaxCPUPercent)
	assert.Equal(t, 10, limits.MaxDiskSpaceGB)
}

func TestCollaborationType(t *testing.T) {
	tests := []struct {
		name     string
		colType  CollaborationType
		expected string
	}{
		{"Data Sharing", DataSharing, "data_sharing"},
		{"Result Validation", ResultValidation, "result_validation"},
		{"Conflict Resolution", ConflictResolution, "conflict_resolution"},
		{"Expertise Consultation", ExpertiseConsultation, "expertise_consultation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.colType))
		})
	}
}

func TestCollaborationContext(t *testing.T) {
	deadline := time.Now().Add(time.Hour)
	context := CollaborationContext{
		Purpose:    "Code review collaboration",
		SharedData: map[string]interface{}{"code": "package main"},
		Constraints: map[string]interface{}{
			"max_time": "30m",
			"scope":    "security",
		},
		Deadline: &deadline,
	}

	assert.Equal(t, "Code review collaboration", context.Purpose)
	assert.NotNil(t, context.SharedData)
	assert.NotNil(t, context.Constraints)
	assert.Equal(t, deadline, *context.Deadline)
}

func TestAgentCollaboration(t *testing.T) {
	requestTime := time.Now()
	deadline := requestTime.Add(time.Hour)

	collaboration := AgentCollaboration{
		RequestingAgent:   SecurityAgentID,
		TargetAgent:       ArchitectureAgentID,
		CollaborationType: ExpertiseConsultation,
		Context: CollaborationContext{
			Purpose:  "Security architecture review",
			Deadline: &deadline,
		},
		Priority:    PriorityHigh,
		RequestTime: requestTime,
	}

	assert.Equal(t, SecurityAgentID, collaboration.RequestingAgent)
	assert.Equal(t, ArchitectureAgentID, collaboration.TargetAgent)
	assert.Equal(t, ExpertiseConsultation, collaboration.CollaborationType)
	assert.Equal(t, PriorityHigh, collaboration.Priority)
	assert.Equal(t, requestTime, collaboration.RequestTime)
	assert.Equal(t, "Security architecture review", collaboration.Context.Purpose)
}

func TestAgentResult(t *testing.T) {
	timestamp := time.Now()
	duration := 5 * time.Minute

	result := AgentResult{
		AgentID:   SecurityAgentID,
		Success:   true,
		Results:   map[string]interface{}{"vulnerabilities": 0},
		Errors:    nil,
		Warnings:  []string{"Minor issue found"},
		Metrics:   map[string]interface{}{"scan_time": "5m"},
		Duration:  duration,
		Timestamp: timestamp,
	}

	assert.Equal(t, SecurityAgentID, result.AgentID)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.Results["vulnerabilities"])
	assert.Nil(t, result.Errors)
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "Minor issue found", result.Warnings[0])
	assert.Equal(t, "5m", result.Metrics["scan_time"])
	assert.Equal(t, duration, result.Duration)
	assert.Equal(t, timestamp, result.Timestamp)
}

func TestAgentResultWithErrors(t *testing.T) {
	result := AgentResult{
		AgentID: TestingAgentID,
		Success: false,
		Errors:  []error{assert.AnError},
	}

	assert.Equal(t, TestingAgentID, result.AgentID)
	assert.False(t, result.Success)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, assert.AnError, result.Errors[0])
}

func TestConflictReport(t *testing.T) {
	conflict := ConflictReport{
		ConflictingAgents: []AgentID{SecurityAgentID, PerformanceAgentID},
		IssueType:         "optimization_vs_security",
		Description:       "Security recommendations conflict with performance optimizations",
		Resolution:        "Apply security fixes with performance consideration",
		RequiresHuman:     true,
	}

	assert.Len(t, conflict.ConflictingAgents, 2)
	assert.Contains(t, conflict.ConflictingAgents, SecurityAgentID)
	assert.Contains(t, conflict.ConflictingAgents, PerformanceAgentID)
	assert.Equal(t, "optimization_vs_security", conflict.IssueType)
	assert.NotEmpty(t, conflict.Description)
	assert.NotEmpty(t, conflict.Resolution)
	assert.True(t, conflict.RequiresHuman)
}

func TestPriorityItem(t *testing.T) {
	item := PriorityItem{
		ID:          "issue-123",
		AgentID:     SecurityAgentID,
		Type:        "vulnerability",
		Description: "SQL injection vulnerability detected",
		Severity:    PriorityCritical,
		Impact:      "High - could lead to data breach",
		AutoFixable: true,
		FixDetails:  map[string]interface{}{"fix_type": "parameterized_query"},
	}

	assert.Equal(t, "issue-123", item.ID)
	assert.Equal(t, SecurityAgentID, item.AgentID)
	assert.Equal(t, "vulnerability", item.Type)
	assert.NotEmpty(t, item.Description)
	assert.Equal(t, PriorityCritical, item.Severity)
	assert.NotEmpty(t, item.Impact)
	assert.True(t, item.AutoFixable)
	assert.NotNil(t, item.FixDetails)
}

func TestResultAggregation(t *testing.T) {
	timestamp := time.Now()

	result1 := &AgentResult{
		AgentID: SecurityAgentID,
		Success: true,
	}

	result2 := &AgentResult{
		AgentID: ArchitectureAgentID,
		Success: true,
	}

	conflict := ConflictReport{
		ConflictingAgents: []AgentID{SecurityAgentID, PerformanceAgentID},
		IssueType:         "test_conflict",
	}

	priority := PriorityItem{
		ID:      "item-1",
		AgentID: SecurityAgentID,
		Type:    "issue",
	}

	aggregation := ResultAggregation{
		WorkflowID: "workflow-123",
		AgentResults: map[AgentID]*AgentResult{
			SecurityAgentID:     result1,
			ArchitectureAgentID: result2,
		},
		Conflicts:  []ConflictReport{conflict},
		Priorities: []PriorityItem{priority},
		Summary:    "All agents completed successfully with minor conflicts",
		Timestamp:  timestamp,
	}

	assert.Equal(t, "workflow-123", aggregation.WorkflowID)
	assert.Len(t, aggregation.AgentResults, 2)
	assert.Equal(t, result1, aggregation.AgentResults[SecurityAgentID])
	assert.Equal(t, result2, aggregation.AgentResults[ArchitectureAgentID])
	assert.Len(t, aggregation.Conflicts, 1)
	assert.Equal(t, conflict, aggregation.Conflicts[0])
	assert.Len(t, aggregation.Priorities, 1)
	assert.Equal(t, priority, aggregation.Priorities[0])
	assert.NotEmpty(t, aggregation.Summary)
	assert.Equal(t, timestamp, aggregation.Timestamp)
}

func TestWorkspaceConfig(t *testing.T) {
	config := WorkspaceConfig{
		WorkspacePath: "/path/to/workspace",
		ProjectType:   "golang_web_service",
		Configuration: map[string]interface{}{
			"go_version": "1.21",
			"modules":    []string{"web", "api"},
		},
	}

	assert.Equal(t, "/path/to/workspace", config.WorkspacePath)
	assert.Equal(t, "golang_web_service", config.ProjectType)
	assert.Equal(t, "1.21", config.Configuration["go_version"])
}

func TestStageResult(t *testing.T) {
	duration := 2 * time.Minute

	result := StageResult{
		Name:     "security_scan",
		Status:   "completed",
		Duration: duration,
		Output:   []string{"Scan completed", "No vulnerabilities found"},
		Metadata: map[string]interface{}{
			"files_scanned": 150,
			"scan_type":     "sast",
		},
	}

	assert.Equal(t, "security_scan", result.Name)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, duration, result.Duration)
	assert.Len(t, result.Output, 2)
	assert.Contains(t, result.Output, "Scan completed")
	assert.Contains(t, result.Output, "No vulnerabilities found")
	assert.Equal(t, 150, result.Metadata["files_scanned"])
	assert.Equal(t, "sast", result.Metadata["scan_type"])
}

func TestHealthMonitorMetrics(t *testing.T) {
	lastCheck := time.Now()

	metrics := HealthMonitorMetrics{
		TotalAgents:        5,
		HealthyAgents:      4,
		TotalHealthChecks:  100,
		FailedHealthChecks: 5,
		LastCheckTime:      lastCheck,
	}

	assert.Equal(t, 5, metrics.TotalAgents)
	assert.Equal(t, 4, metrics.HealthyAgents)
	assert.Equal(t, 100, metrics.TotalHealthChecks)
	assert.Equal(t, 5, metrics.FailedHealthChecks)
	assert.Equal(t, lastCheck, metrics.LastCheckTime)
}

func TestMessageBusHealth(t *testing.T) {
	lastActivity := time.Now()
	latency := 50 * time.Millisecond

	health := MessageBusHealth{
		TotalMessages:     1000,
		FailedMessages:    10,
		AverageLatency:    latency,
		ActiveSubscribers: 5,
		LastActivity:      lastActivity,
	}

	assert.Equal(t, int64(1000), health.TotalMessages)
	assert.Equal(t, int64(10), health.FailedMessages)
	assert.Equal(t, latency, health.AverageLatency)
	assert.Equal(t, 5, health.ActiveSubscribers)
	assert.Equal(t, lastActivity, health.LastActivity)
}

func TestAgentRegistryMetrics(t *testing.T) {
	lastUpdate := time.Now()

	metrics := AgentRegistryMetrics{
		TotalAgents:   10,
		ActiveAgents:  8,
		FailedAgents:  1,
		Registrations: 15,
		LastUpdate:    lastUpdate,
	}

	assert.Equal(t, 10, metrics.TotalAgents)
	assert.Equal(t, 8, metrics.ActiveAgents)
	assert.Equal(t, 1, metrics.FailedAgents)
	assert.Equal(t, int64(15), metrics.Registrations)
	assert.Equal(t, lastUpdate, metrics.LastUpdate)
}

// Test edge cases and boundary conditions
func TestEmptyStructs(t *testing.T) {
	// Test creating structs with minimal data
	msg := AgentMessage{}
	assert.Empty(t, msg.ID)
	assert.Empty(t, msg.Type)
	assert.True(t, msg.Timestamp.IsZero())

	context := MessageContext{}
	assert.Empty(t, context.WorkflowID)
	assert.Nil(t, context.Metadata)

	result := AgentResult{}
	assert.Empty(t, result.AgentID)
	assert.False(t, result.Success)
	assert.True(t, result.Timestamp.IsZero())
}

func TestNilPointers(t *testing.T) {
	// Test handling of nil pointers in structs
	context := CollaborationContext{
		Purpose:     "test",
		SharedData:  nil,
		Constraints: nil,
		Deadline:    nil,
	}

	assert.Equal(t, "test", context.Purpose)
	assert.Nil(t, context.SharedData)
	assert.Nil(t, context.Constraints)
	assert.Nil(t, context.Deadline)
}

func TestLargeData(t *testing.T) {
	// Test with large data structures
	largeMetadata := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeMetadata[string(rune('a'+i%26))+fmt.Sprintf("%d", i)] = i
	}

	context := MessageContext{
		WorkflowID: "large-workflow",
		Metadata:   largeMetadata,
	}

	assert.Equal(t, "large-workflow", context.WorkflowID)
	assert.Len(t, context.Metadata, 1000)
}
