package agents

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResultAggregator(t *testing.T) {
	aggregator := NewResultAggregator()

	assert.NotNil(t, aggregator)
	assert.NotNil(t, aggregator.results)
	assert.NotNil(t, aggregator.conflictResolver)
	assert.NotNil(t, aggregator.priorityEngine)
	assert.NotNil(t, aggregator.logger)
}

func TestResultAggregatorAddResult(t *testing.T) {
	aggregator := NewResultAggregator()

	result := &AgentResult{
		AgentID:   SecurityAgentID,
		Success:   true,
		Duration:  5 * time.Minute,
		Timestamp: time.Now(),
		Results:   map[string]interface{}{"vulnerabilities": 0},
	}

	err := aggregator.AddResult(result)
	assert.NoError(t, err)

	// Verify result was stored
	aggregator.mu.RLock()
	storedResult, exists := aggregator.results[SecurityAgentID]
	aggregator.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, result, storedResult)
}

func TestResultAggregatorAddResultWithNil(t *testing.T) {
	aggregator := NewResultAggregator()

	err := aggregator.AddResult(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "result cannot be nil")
}

func TestResultAggregatorAggregate(t *testing.T) {
	aggregator := NewResultAggregator()

	// Add multiple results
	results := []*AgentResult{
		{
			AgentID:   SecurityAgentID,
			Success:   true,
			Duration:  3 * time.Minute,
			Timestamp: time.Now(),
			Results:   map[string]interface{}{"vulnerabilities": 0},
		},
		{
			AgentID:   ArchitectureAgentID,
			Success:   true,
			Duration:  4 * time.Minute,
			Timestamp: time.Now(),
			Results:   map[string]interface{}{"complexity": "medium"},
		},
		{
			AgentID:   TestingAgentID,
			Success:   false,
			Duration:  2 * time.Minute,
			Timestamp: time.Now(),
			Errors:    []error{assert.AnError},
		},
	}

	for _, result := range results {
		err := aggregator.AddResult(result)
		require.NoError(t, err)
	}

	workflowID := "test-workflow-123"
	aggregation, err := aggregator.Aggregate(workflowID)

	assert.NoError(t, err)
	assert.NotNil(t, aggregation)
	assert.Equal(t, workflowID, aggregation.WorkflowID)
	assert.Len(t, aggregation.AgentResults, 3)
	assert.NotEmpty(t, aggregation.Summary)
	assert.False(t, aggregation.Timestamp.IsZero())

	// Verify all agent results are included
	assert.Contains(t, aggregation.AgentResults, SecurityAgentID)
	assert.Contains(t, aggregation.AgentResults, ArchitectureAgentID)
	assert.Contains(t, aggregation.AgentResults, TestingAgentID)

	// Verify priorities include failure
	assert.NotEmpty(t, aggregation.Priorities)

	// Check for testing agent failure in priorities
	found := false
	for _, item := range aggregation.Priorities {
		if item.AgentID == TestingAgentID && item.Type == "Agent Failure" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should include testing agent failure in priorities")
}

func TestResultAggregatorAggregateWithNoResults(t *testing.T) {
	aggregator := NewResultAggregator()

	aggregation, err := aggregator.Aggregate("test-workflow")
	assert.Error(t, err)
	assert.Nil(t, aggregation)
	assert.Contains(t, err.Error(), "no results to aggregate")
}

func TestResultAggregatorClear(t *testing.T) {
	aggregator := NewResultAggregator()

	// Add a result
	result := &AgentResult{
		AgentID: SecurityAgentID,
		Success: true,
	}

	err := aggregator.AddResult(result)
	require.NoError(t, err)

	// Verify result exists
	aggregator.mu.RLock()
	assert.Len(t, aggregator.results, 1)
	aggregator.mu.RUnlock()

	// Clear results
	aggregator.Clear()

	// Verify results are cleared
	aggregator.mu.RLock()
	assert.Len(t, aggregator.results, 0)
	aggregator.mu.RUnlock()
}

func TestResultAggregatorGenerateSummary(t *testing.T) {
	aggregator := NewResultAggregator()

	results := map[AgentID]*AgentResult{
		SecurityAgentID: {
			AgentID:   SecurityAgentID,
			Success:   true,
			Duration:  3 * time.Minute,
			Timestamp: time.Now(),
		},
		TestingAgentID: {
			AgentID:   TestingAgentID,
			Success:   false,
			Duration:  2 * time.Minute,
			Timestamp: time.Now(),
		},
	}

	conflicts := []ConflictReport{
		{
			IssueType:   "Test Conflict",
			Description: "Test description",
		},
	}

	priorities := []PriorityItem{
		{
			AgentID:     SecurityAgentID,
			Type:        "Security Issue",
			Description: "Critical vulnerability",
			Severity:    PriorityCritical,
		},
		{
			AgentID:     TestingAgentID,
			Type:        "Test Issue",
			Description: "Low coverage",
			Severity:    PriorityLow,
		},
	}

	summary := aggregator.generateSummary(results, conflicts, priorities)

	assert.Contains(t, summary, "Workflow Analysis Complete")
	assert.Contains(t, summary, "Agent Results:")
	assert.Contains(t, summary, "security: Success")
	assert.Contains(t, summary, "testing: Failed")
	assert.Contains(t, summary, "High Priority Issues:")
	assert.Contains(t, summary, "Critical vulnerability")
	assert.Contains(t, summary, "Conflicts Detected: 1")
	assert.Contains(t, summary, "Test description")
}

func TestNewConflictResolver(t *testing.T) {
	resolver := NewConflictResolver()

	assert.NotNil(t, resolver)
	assert.NotNil(t, resolver.logger)
	assert.NotEmpty(t, resolver.rules)

	// Verify default rules are loaded
	ruleNames := make([]string, len(resolver.rules))
	for i, rule := range resolver.rules {
		ruleNames[i] = rule.Name
	}

	assert.Contains(t, ruleNames, "Security vs Performance")
	assert.Contains(t, ruleNames, "Architecture vs Quick Fix")
	assert.Contains(t, ruleNames, "Test Coverage vs Deadline")
}

func TestConflictResolverDetectConflicts(t *testing.T) {
	resolver := NewConflictResolver()

	results := map[AgentID]*AgentResult{
		SecurityAgentID: {
			AgentID: SecurityAgentID,
			Success: true,
		},
		PerformanceAgentID: {
			AgentID: PerformanceAgentID,
			Success: true,
		},
		ArchitectureAgentID: {
			AgentID: ArchitectureAgentID,
			Success: true,
		},
		ClaudeCodeAgentID: {
			AgentID: ClaudeCodeAgentID,
			Success: true,
		},
		TestingAgentID: {
			AgentID: TestingAgentID,
			Success: true,
		},
	}

	conflicts := resolver.DetectConflicts(results)

	// Should detect conflicts based on default rules
	assert.NotEmpty(t, conflicts)

	// Check for specific conflict types
	conflictTypes := make([]string, len(conflicts))
	for i, conflict := range conflicts {
		conflictTypes[i] = conflict.IssueType
	}

	// Should include security vs performance conflict
	assert.Contains(t, conflictTypes, "Encryption Overhead")
	// Should include architecture vs quick fix conflict
	assert.Contains(t, conflictTypes, "Technical Debt")
	// Should include test coverage conflict
	assert.Contains(t, conflictTypes, "Insufficient Test Coverage")
}

func TestDetectSecurityPerformanceConflicts(t *testing.T) {
	// Test with both agents present and successful
	results := map[AgentID]*AgentResult{
		SecurityAgentID: {
			AgentID: SecurityAgentID,
			Success: true,
		},
		PerformanceAgentID: {
			AgentID: PerformanceAgentID,
			Success: true,
		},
	}

	conflicts := detectSecurityPerformanceConflicts(results)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, "Encryption Overhead", conflict.IssueType)
	assert.Contains(t, conflict.ConflictingAgents, SecurityAgentID)
	assert.Contains(t, conflict.ConflictingAgents, PerformanceAgentID)
	assert.False(t, conflict.RequiresHuman)

	// Test with missing agents
	incompleteResults := map[AgentID]*AgentResult{
		SecurityAgentID: {
			AgentID: SecurityAgentID,
			Success: true,
		},
	}

	conflicts = detectSecurityPerformanceConflicts(incompleteResults)
	assert.Empty(t, conflicts)

	// Test with failed agents
	failedResults := map[AgentID]*AgentResult{
		SecurityAgentID: {
			AgentID: SecurityAgentID,
			Success: false,
		},
		PerformanceAgentID: {
			AgentID: PerformanceAgentID,
			Success: true,
		},
	}

	conflicts = detectSecurityPerformanceConflicts(failedResults)
	assert.Empty(t, conflicts)
}

func TestDetectArchitectureQuickFixConflicts(t *testing.T) {
	results := map[AgentID]*AgentResult{
		ArchitectureAgentID: {
			AgentID: ArchitectureAgentID,
			Success: true,
		},
		ClaudeCodeAgentID: {
			AgentID: ClaudeCodeAgentID,
			Success: true,
		},
	}

	conflicts := detectArchitectureQuickFixConflicts(results)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, "Technical Debt", conflict.IssueType)
	assert.Contains(t, conflict.ConflictingAgents, ArchitectureAgentID)
	assert.Contains(t, conflict.ConflictingAgents, ClaudeCodeAgentID)
	assert.True(t, conflict.RequiresHuman)
}

func TestDetectTestCoverageConflicts(t *testing.T) {
	results := map[AgentID]*AgentResult{
		TestingAgentID: {
			AgentID: TestingAgentID,
			Success: true,
		},
	}

	conflicts := detectTestCoverageConflicts(results)
	assert.Len(t, conflicts, 1)

	conflict := conflicts[0]
	assert.Equal(t, "Insufficient Test Coverage", conflict.IssueType)
	assert.Contains(t, conflict.ConflictingAgents, TestingAgentID)
	assert.True(t, conflict.RequiresHuman)

	// Test with failed testing agent
	failedResults := map[AgentID]*AgentResult{
		TestingAgentID: {
			AgentID: TestingAgentID,
			Success: false,
		},
	}

	conflicts = detectTestCoverageConflicts(failedResults)
	assert.Empty(t, conflicts)
}

func TestNewPriorityEngine(t *testing.T) {
	engine := NewPriorityEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.logger)
	assert.NotEmpty(t, engine.weights)

	// Verify weights are set for expected agents
	assert.Equal(t, 1.5, engine.weights[SecurityAgentID])
	assert.Equal(t, 1.2, engine.weights[ArchitectureAgentID])
	assert.Equal(t, 1.0, engine.weights[TestingAgentID])
	assert.Equal(t, 0.8, engine.weights[DocumentationAgentID])
	assert.Equal(t, 1.1, engine.weights[PerformanceAgentID])
}

func TestPriorityEnginePrioritize(t *testing.T) {
	engine := NewPriorityEngine()

	results := map[AgentID]*AgentResult{
		SecurityAgentID: {
			AgentID: SecurityAgentID,
			Success: true,
		},
		ArchitectureAgentID: {
			AgentID: ArchitectureAgentID,
			Success: true,
		},
		TestingAgentID: {
			AgentID: TestingAgentID,
			Success: false, // Failed agent should create high priority item
		},
		DocumentationAgentID: {
			AgentID: DocumentationAgentID,
			Success: true,
		},
	}

	items := engine.Prioritize(results)

	assert.NotEmpty(t, items)

	// Check for failure item
	found := false
	for _, item := range items {
		if item.AgentID == TestingAgentID && item.Type == "Agent Failure" {
			found = true
			assert.Equal(t, PriorityHigh, item.Severity)
			assert.False(t, item.AutoFixable)
			break
		}
	}
	assert.True(t, found, "Should include testing agent failure")

	// Verify items are sorted by priority (security should have high weight)
	if len(items) > 1 {
		// Items should be sorted by weighted priority
		// Security items should generally appear before documentation items
		securityIndex := -1
		docIndex := -1

		for i, item := range items {
			if item.AgentID == SecurityAgentID {
				securityIndex = i
			}
			if item.AgentID == DocumentationAgentID {
				docIndex = i
			}
		}

		if securityIndex >= 0 && docIndex >= 0 {
			// Security should typically come before documentation due to higher weight
			// (unless documentation has much higher severity)
			assert.True(t, securityIndex <= docIndex || items[docIndex].Severity > items[securityIndex].Severity)
		}
	}
}

func TestPriorityEngineExtractPriorityItems(t *testing.T) {
	engine := NewPriorityEngine()

	result := &AgentResult{
		AgentID: SecurityAgentID,
		Success: true,
	}

	items := engine.extractPriorityItems(SecurityAgentID, result)

	assert.Len(t, items, 1)
	item := items[0]
	assert.Equal(t, SecurityAgentID, item.AgentID)
	assert.Equal(t, "Security Vulnerability", item.Type)
	assert.Equal(t, PriorityCritical, item.Severity)
	assert.True(t, item.AutoFixable)
	assert.NotEmpty(t, item.FixDetails)

	// Test architecture agent
	items = engine.extractPriorityItems(ArchitectureAgentID, result)
	assert.Len(t, items, 1)
	assert.Equal(t, "Architecture Violation", items[0].Type)
	assert.False(t, items[0].AutoFixable)

	// Test testing agent
	items = engine.extractPriorityItems(TestingAgentID, result)
	assert.Len(t, items, 1)
	assert.Equal(t, "Test Coverage", items[0].Type)
	assert.True(t, items[0].AutoFixable)

	// Test unknown agent (should return empty)
	items = engine.extractPriorityItems(AgentID("unknown"), result)
	assert.Empty(t, items)
}

func TestPriorityEngineCountBySeverity(t *testing.T) {
	engine := NewPriorityEngine()

	items := []PriorityItem{
		{Severity: PriorityCritical},
		{Severity: PriorityHigh},
		{Severity: PriorityHigh},
		{Severity: PriorityMedium},
		{Severity: PriorityLow},
	}

	assert.Equal(t, 1, engine.countBySeverity(items, PriorityCritical))
	assert.Equal(t, 2, engine.countBySeverity(items, PriorityHigh))
	assert.Equal(t, 1, engine.countBySeverity(items, PriorityMedium))
	assert.Equal(t, 1, engine.countBySeverity(items, PriorityLow))
}

func TestNewAutoFixCoordinator(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	coordinator := NewAutoFixCoordinator(registry, messageBus)

	assert.NotNil(t, coordinator)
	assert.Equal(t, registry, coordinator.registry)
	assert.Equal(t, messageBus, coordinator.messageBus)
	assert.NotNil(t, coordinator.fixQueue)
	assert.NotNil(t, coordinator.fixHistory)
	assert.NotNil(t, coordinator.logger)
}

func TestAutoFixCoordinatorQueueFixes(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	coordinator := NewAutoFixCoordinator(registry, messageBus)

	items := []PriorityItem{
		{
			ID:          "fix-1",
			AutoFixable: true,
			FixDetails:  "Fix details 1",
		},
		{
			ID:          "fix-2",
			AutoFixable: false, // Should not be queued
			FixDetails:  "Fix details 2",
		},
		{
			ID:          "fix-3",
			AutoFixable: true,
			FixDetails:  "Fix details 3",
		},
	}

	coordinator.QueueFixes(items)

	coordinator.mu.Lock()
	queueLength := len(coordinator.fixQueue)
	coordinator.mu.Unlock()

	assert.Equal(t, 2, queueLength) // Only auto-fixable items should be queued
}

func TestAutoFixCoordinatorApplyFixes(t *testing.T) {
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

	coordinator := NewAutoFixCoordinator(registry, messageBus)

	items := []PriorityItem{
		{
			ID:          "fix-1",
			AgentID:     SecurityAgentID,
			AutoFixable: true,
			FixDetails:  "Apply security fix",
		},
	}

	coordinator.QueueFixes(items)

	ctx := context.Background()
	err = coordinator.ApplyFixes(ctx)
	assert.NoError(t, err)

	// Verify fix history
	history := coordinator.GetFixHistory()
	assert.Len(t, history, 1)

	record := history[0]
	assert.Equal(t, "fix-1", record.ItemID)
	assert.Equal(t, SecurityAgentID, record.AgentID)
	assert.True(t, record.Success)
	assert.Contains(t, record.Details, "Apply security fix")

	// Verify queue is cleared
	coordinator.mu.Lock()
	queueLength := len(coordinator.fixQueue)
	coordinator.mu.Unlock()
	assert.Equal(t, 0, queueLength)
}

func TestAutoFixCoordinatorApplyFixesWithContext(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	coordinator := NewAutoFixCoordinator(registry, messageBus)

	items := []PriorityItem{
		{
			ID:          "fix-1",
			AgentID:     SecurityAgentID,
			AutoFixable: true,
		},
	}

	coordinator.QueueFixes(items)

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := coordinator.ApplyFixes(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestAutoFixCoordinatorApplyFixesWithNonExistentAgent(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	coordinator := NewAutoFixCoordinator(registry, messageBus)

	items := []PriorityItem{
		{
			ID:          "fix-1",
			AgentID:     SecurityAgentID, // Agent not created
			AutoFixable: true,
		},
	}

	coordinator.QueueFixes(items)

	ctx := context.Background()
	err := coordinator.ApplyFixes(ctx)
	assert.NoError(t, err) // Should continue despite individual failures

	// Verify no successful fixes in history
	history := coordinator.GetFixHistory()
	assert.Empty(t, history)
}

func TestAutoFixCoordinatorGetFixHistory(t *testing.T) {
	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	registry := NewAgentRegistry(messageBus)
	defer func() { _ = registry.StopAll() }()

	coordinator := NewAutoFixCoordinator(registry, messageBus)

	// Initially empty
	history := coordinator.GetFixHistory()
	assert.Empty(t, history)

	// Add some history manually
	coordinator.mu.Lock()
	coordinator.fixHistory = []FixRecord{
		{
			ItemID:    "fix-1",
			AgentID:   SecurityAgentID,
			Timestamp: time.Now(),
			Success:   true,
		},
		{
			ItemID:    "fix-2",
			AgentID:   TestingAgentID,
			Timestamp: time.Now(),
			Success:   false,
		},
	}
	coordinator.mu.Unlock()

	history = coordinator.GetFixHistory()
	assert.Len(t, history, 2)

	// Verify it's a copy (modifying shouldn't affect original)
	history[0].Success = false

	originalHistory := coordinator.GetFixHistory()
	assert.True(t, originalHistory[0].Success)
}

func TestConflictRuleStruct(t *testing.T) {
	rule := ConflictRule{
		Name:     "Test Rule",
		Priority: PriorityHigh,
		Detector: func(results map[AgentID]*AgentResult) []ConflictReport {
			return []ConflictReport{
				{
					IssueType:   "Test Conflict",
					Description: "Test description",
				},
			}
		},
	}

	assert.Equal(t, "Test Rule", rule.Name)
	assert.Equal(t, PriorityHigh, rule.Priority)
	assert.NotNil(t, rule.Detector)

	// Test detector function
	results := make(map[AgentID]*AgentResult)
	conflicts := rule.Detector(results)
	assert.Len(t, conflicts, 1)
	assert.Equal(t, "Test Conflict", conflicts[0].IssueType)
}

func TestFixRecordStruct(t *testing.T) {
	timestamp := time.Now()
	testError := assert.AnError

	record := FixRecord{
		ItemID:    "fix-123",
		AgentID:   SecurityAgentID,
		Timestamp: timestamp,
		Success:   false,
		Error:     testError,
		Details:   "Fix failed due to validation error",
	}

	assert.Equal(t, "fix-123", record.ItemID)
	assert.Equal(t, SecurityAgentID, record.AgentID)
	assert.Equal(t, timestamp, record.Timestamp)
	assert.False(t, record.Success)
	assert.Equal(t, testError, record.Error)
	assert.Equal(t, "Fix failed due to validation error", record.Details)
}
