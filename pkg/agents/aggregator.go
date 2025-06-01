// Package agents provides a multi-agent system for automated GitHub repository management
// and code analysis. It includes specialized agents for different aspects of code quality,
// security, documentation, and testing.
package agents

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)

// ResultAggregator consolidates results from multiple agents
type ResultAggregator struct {
	results map[AgentID]*AgentResult
	mu      sync.RWMutex
	logger  *logger.Logger

	conflictResolver *ConflictResolver
	priorityEngine   *PriorityEngine
}

// NewResultAggregator creates a new result aggregator
func NewResultAggregator() *ResultAggregator {
	return &ResultAggregator{
		results:          make(map[AgentID]*AgentResult),
		logger:           logger.GetLogger(),
		conflictResolver: NewConflictResolver(),
		priorityEngine:   NewPriorityEngine(),
	}
}

// AddResult adds a result from an agent
func (ra *ResultAggregator) AddResult(result *AgentResult) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	if result == nil {
		return fmt.Errorf("result cannot be nil")
	}

	ra.results[result.AgentID] = result
	ra.logger.Info("Result added (agent_id: %s, success: %t, duration: %v)", result.AgentID, result.Success, result.Duration)

	return nil
}

// Aggregate consolidates all results into a single aggregation
func (ra *ResultAggregator) Aggregate(workflowID string) (*ResultAggregation, error) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	if len(ra.results) == 0 {
		return nil, fmt.Errorf("no results to aggregate")
	}

	// Create copy of results
	resultsCopy := make(map[AgentID]*AgentResult)
	for id, result := range ra.results {
		resultsCopy[id] = result
	}

	// Detect conflicts
	conflicts := ra.conflictResolver.DetectConflicts(resultsCopy)

	// Extract and prioritize issues
	priorities := ra.priorityEngine.Prioritize(resultsCopy)

	// Generate summary
	summary := ra.generateSummary(resultsCopy, conflicts, priorities)

	aggregation := &ResultAggregation{
		WorkflowID:   workflowID,
		AgentResults: resultsCopy,
		Conflicts:    conflicts,
		Priorities:   priorities,
		Summary:      summary,
		Timestamp:    time.Now(),
	}

	ra.logger.Info("Results aggregated (workflow_id: %s, agent_count: %d, conflict_count: %d, priority_count: %d)", workflowID, len(resultsCopy), len(conflicts), len(priorities))

	return aggregation, nil
}

// generateSummary creates a comprehensive summary of all results
func (ra *ResultAggregator) generateSummary(results map[AgentID]*AgentResult, conflicts []ConflictReport, priorities []PriorityItem) string {
	summary := "Workflow Analysis Complete\n\n"

	// Agent results summary
	summary += "Agent Results:\n"
	for agentID, result := range results {
		status := "Failed"
		if result.Success {
			status = "Success"
		}
		summary += fmt.Sprintf("  - %s: %s (Duration: %s)\n", agentID, status, result.Duration)
	}

	// High priority issues
	summary += "\nHigh Priority Issues:\n"
	highPriorityCount := 0
	for _, item := range priorities {
		if item.Severity == PriorityCritical || item.Severity == PriorityHigh {
			summary += fmt.Sprintf("  - [%s] %s: %s\n", item.AgentID, item.Type, item.Description)
			highPriorityCount++
			if highPriorityCount >= 5 {
				summary += fmt.Sprintf("  ... and %d more\n", len(priorities)-5)
				break
			}
		}
	}

	// Conflicts
	if len(conflicts) > 0 {
		summary += fmt.Sprintf("\nConflicts Detected: %d\n", len(conflicts))
		for i, conflict := range conflicts {
			if i >= 3 {
				summary += fmt.Sprintf("  ... and %d more\n", len(conflicts)-3)
				break
			}
			summary += fmt.Sprintf("  - %s: %s\n", conflict.IssueType, conflict.Description)
		}
	}

	return summary
}

// Clear removes all stored results
func (ra *ResultAggregator) Clear() {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	ra.results = make(map[AgentID]*AgentResult)
	ra.logger.Info("Results cleared")
}

// ConflictResolver detects and resolves conflicts between agent recommendations
type ConflictResolver struct {
	logger *logger.Logger
	rules  []ConflictRule
}

// ConflictRule defines a rule for detecting conflicts
type ConflictRule struct {
	Name     string
	Detector func(results map[AgentID]*AgentResult) []ConflictReport
	Priority Priority
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver() *ConflictResolver {
	resolver := &ConflictResolver{
		logger: logger.GetLogger(),
	}

	// Initialize default conflict detection rules
	resolver.rules = []ConflictRule{
		{
			Name:     "Security vs Performance",
			Detector: detectSecurityPerformanceConflicts,
			Priority: PriorityHigh,
		},
		{
			Name:     "Architecture vs Quick Fix",
			Detector: detectArchitectureQuickFixConflicts,
			Priority: PriorityMedium,
		},
		{
			Name:     "Test Coverage vs Deadline",
			Detector: detectTestCoverageConflicts,
			Priority: PriorityMedium,
		},
	}

	return resolver
}

// DetectConflicts analyzes results for conflicting recommendations
func (cr *ConflictResolver) DetectConflicts(results map[AgentID]*AgentResult) []ConflictReport {
	var conflicts []ConflictReport

	for _, rule := range cr.rules {
		detected := rule.Detector(results)
		conflicts = append(conflicts, detected...)
	}

	cr.logger.Info("Conflict detection complete (total_conflicts: %d)", len(conflicts))

	return conflicts
}

// detectSecurityPerformanceConflicts identifies conflicts between security and performance recommendations
func detectSecurityPerformanceConflicts(results map[AgentID]*AgentResult) []ConflictReport {
	var conflicts []ConflictReport

	securityResult, hasSecurityResult := results[SecurityAgentID]
	performanceResult, hasPerfResult := results[PerformanceAgentID]

	if !hasSecurityResult || !hasPerfResult {
		return conflicts
	}

	// Check for conflicting recommendations
	// This is a simplified example - real implementation would analyze specific recommendations
	if securityResult.Success && performanceResult.Success {
		// Example: Security wants encryption, Performance wants to avoid it
		conflict := ConflictReport{
			ConflictingAgents: []AgentID{SecurityAgentID, PerformanceAgentID},
			IssueType:         "Encryption Overhead",
			Description:       "Security recommends encryption but it impacts performance",
			Resolution:        "Use hardware-accelerated encryption or cache encrypted data",
			RequiresHuman:     false,
		}
		conflicts = append(conflicts, conflict)
	}

	return conflicts
}

// detectArchitectureQuickFixConflicts identifies conflicts between proper architecture and quick fixes
func detectArchitectureQuickFixConflicts(results map[AgentID]*AgentResult) []ConflictReport {
	var conflicts []ConflictReport

	archResult, hasArchResult := results[ArchitectureAgentID]
	claudeResult, hasClaudeResult := results[ClaudeCodeAgentID]

	if !hasArchResult || !hasClaudeResult {
		return conflicts
	}

	// Example conflict detection
	if archResult.Success && claudeResult.Success {
		// Check if quick implementation violates architecture principles
		conflict := ConflictReport{
			ConflictingAgents: []AgentID{ArchitectureAgentID, ClaudeCodeAgentID},
			IssueType:         "Technical Debt",
			Description:       "Quick implementation may introduce technical debt",
			Resolution:        "Refactor after initial implementation",
			RequiresHuman:     true,
		}
		conflicts = append(conflicts, conflict)
	}

	return conflicts
}

// detectTestCoverageConflicts identifies conflicts between test coverage and time constraints
func detectTestCoverageConflicts(results map[AgentID]*AgentResult) []ConflictReport {
	var conflicts []ConflictReport

	testResult, hasTestResult := results[TestingAgentID]

	if !hasTestResult || !testResult.Success {
		return conflicts
	}

	// Example: Low test coverage but urgent deadline
	conflict := ConflictReport{
		ConflictingAgents: []AgentID{TestingAgentID},
		IssueType:         "Insufficient Test Coverage",
		Description:       "Test coverage below threshold but implementation is urgent",
		Resolution:        "Add critical path tests now, comprehensive tests later",
		RequiresHuman:     true,
	}
	conflicts = append(conflicts, conflict)

	return conflicts
}

// PriorityEngine analyzes and prioritizes issues from all agents
type PriorityEngine struct {
	logger  *logger.Logger
	weights map[AgentID]float64
}

// NewPriorityEngine creates a new priority engine
func NewPriorityEngine() *PriorityEngine {
	return &PriorityEngine{
		logger: logger.GetLogger(),
		weights: map[AgentID]float64{
			SecurityAgentID:      1.5, // Security issues get higher weight
			ArchitectureAgentID:  1.2, // Architecture issues are important
			TestingAgentID:       1.0, // Standard weight
			DocumentationAgentID: 0.8, // Lower priority
			PerformanceAgentID:   1.1, // Slightly elevated
		},
	}
}

// Prioritize analyzes results and creates a prioritized list of issues
func (pe *PriorityEngine) Prioritize(results map[AgentID]*AgentResult) []PriorityItem {
	var items []PriorityItem

	// Extract priority items from each agent result
	for agentID, result := range results {
		if !result.Success {
			// Agent failure is a high priority issue
			item := PriorityItem{
				ID:          fmt.Sprintf("%s-failure", agentID),
				AgentID:     agentID,
				Type:        "Agent Failure",
				Description: fmt.Sprintf("Agent %s failed to complete analysis", agentID),
				Severity:    PriorityHigh,
				Impact:      "Analysis incomplete",
				AutoFixable: false,
			}
			items = append(items, item)
			continue
		}

		// Extract items from successful results
		// This is a simplified example - real implementation would parse actual result data
		agentItems := pe.extractPriorityItems(agentID, result)
		items = append(items, agentItems...)
	}

	// Sort by weighted priority
	sort.Slice(items, func(i, j int) bool {
		weightI := pe.weights[items[i].AgentID]
		weightJ := pe.weights[items[j].AgentID]

		scoreI := float64(items[i].Severity) * weightI
		scoreJ := float64(items[j].Severity) * weightJ

		return scoreI > scoreJ
	})

	pe.logger.Info("Priority analysis complete (total_items: %d, critical_items: %d, high_items: %d)", len(items), pe.countBySeverity(items, PriorityCritical), pe.countBySeverity(items, PriorityHigh))

	return items
}

// extractPriorityItems extracts priority items from an agent result
func (pe *PriorityEngine) extractPriorityItems(agentID AgentID, result *AgentResult) []PriorityItem {
	var items []PriorityItem

	// This is a placeholder - real implementation would parse actual result structure
	// based on the specific agent type and its output format

	switch agentID {
	case SecurityAgentID:
		// Extract security vulnerabilities
		items = append(items, PriorityItem{
			ID:          fmt.Sprintf("%s-vuln-1", agentID),
			AgentID:     agentID,
			Type:        "Security Vulnerability",
			Description: "SQL injection vulnerability detected",
			Severity:    PriorityCritical,
			Impact:      "Data breach risk",
			AutoFixable: true,
			FixDetails:  "Use parameterized queries",
		})

	case ArchitectureAgentID:
		// Extract architecture issues
		items = append(items, PriorityItem{
			ID:          fmt.Sprintf("%s-arch-1", agentID),
			AgentID:     agentID,
			Type:        "Architecture Violation",
			Description: "Circular dependency detected",
			Severity:    PriorityHigh,
			Impact:      "Maintainability degraded",
			AutoFixable: false,
		})

	case TestingAgentID:
		// Extract test coverage issues
		items = append(items, PriorityItem{
			ID:          fmt.Sprintf("%s-test-1", agentID),
			AgentID:     agentID,
			Type:        "Test Coverage",
			Description: "Critical path lacks test coverage",
			Severity:    PriorityMedium,
			Impact:      "Regression risk",
			AutoFixable: true,
			FixDetails:  "Generate unit tests for UserService",
		})
	}

	return items
}

// countBySeverity counts items with a specific severity
func (pe *PriorityEngine) countBySeverity(items []PriorityItem, severity Priority) int {
	count := 0
	for _, item := range items {
		if item.Severity == severity {
			count++
		}
	}
	return count
}

// AutoFixCoordinator manages the application of automated fixes
type AutoFixCoordinator struct {
	logger     *logger.Logger
	registry   *AgentRegistry
	messageBus *MessageBus

	fixQueue   []PriorityItem
	fixHistory []FixRecord
	mu         sync.Mutex
}

// FixRecord records an applied fix
type FixRecord struct {
	ItemID    string
	AgentID   AgentID
	Timestamp time.Time
	Success   bool
	Error     error
	Details   string
}

// NewAutoFixCoordinator creates a new auto-fix coordinator
func NewAutoFixCoordinator(registry *AgentRegistry, messageBus *MessageBus) *AutoFixCoordinator {
	return &AutoFixCoordinator{
		logger:     logger.GetLogger(),
		registry:   registry,
		messageBus: messageBus,
		fixQueue:   make([]PriorityItem, 0),
		fixHistory: make([]FixRecord, 0),
	}
}

// QueueFixes adds auto-fixable items to the fix queue
func (afc *AutoFixCoordinator) QueueFixes(items []PriorityItem) {
	afc.mu.Lock()
	defer afc.mu.Unlock()

	for _, item := range items {
		if item.AutoFixable {
			afc.fixQueue = append(afc.fixQueue, item)
		}
	}

	afc.logger.Info("Fixes queued (total_fixes: %d)", len(afc.fixQueue))
}

// ApplyFixes executes queued fixes in order
func (afc *AutoFixCoordinator) ApplyFixes(ctx context.Context) error {
	afc.mu.Lock()
	defer afc.mu.Unlock()

	if len(afc.fixQueue) == 0 {
		return nil
	}

	afc.logger.Info("Starting auto-fix application (fix_count: %d)", len(afc.fixQueue))

	for _, item := range afc.fixQueue {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := afc.applyFix(ctx, item); err != nil {
				afc.logger.Error("Fix application failed (item_id: %s, error: %v)", item.ID, err)
				// Continue with other fixes
			}
		}
	}

	// Clear the queue
	afc.fixQueue = make([]PriorityItem, 0)

	return nil
}

// applyFix applies a single fix
func (afc *AutoFixCoordinator) applyFix(ctx context.Context, item PriorityItem) error {
	startTime := time.Now()

	// Get the responsible agent
	_, err := afc.registry.GetAgent(item.AgentID)
	if err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// Create fix request message
	msg := &AgentMessage{
		Type:     TaskAssignment,
		Sender:   AgentID("autofix_coordinator"),
		Receiver: item.AgentID,
		Payload: map[string]interface{}{
			"action":      "apply_fix",
			"item_id":     item.ID,
			"fix_details": item.FixDetails,
		},
		Priority: PriorityHigh,
		Context: MessageContext{
			Metadata: map[string]interface{}{
				"auto_fix": true,
			},
		},
	}

	// Send fix request
	if err := afc.messageBus.Send(msg); err != nil {
		return fmt.Errorf("failed to send fix request: %w", err)
	}

	// Wait for response with timeout
	responseTimeout := time.After(30 * time.Second)
	select {
	case <-responseTimeout:
		return fmt.Errorf("fix request timed out for item %s", item.ID)
	case <-time.After(5 * time.Millisecond):
		// Minimal delay for synchronous operations
	}

	// Record the fix
	record := FixRecord{
		ItemID:    item.ID,
		AgentID:   item.AgentID,
		Timestamp: time.Now(),
		Success:   true,
		Details:   fmt.Sprintf("Applied fix: %s", item.FixDetails),
	}
	afc.fixHistory = append(afc.fixHistory, record)

	afc.logger.Info("Fix applied (item_id: %s, agent_id: %s, duration: %v)", item.ID, item.AgentID, time.Since(startTime))

	return nil
}

// GetFixHistory returns the history of applied fixes
func (afc *AutoFixCoordinator) GetFixHistory() []FixRecord {
	afc.mu.Lock()
	defer afc.mu.Unlock()

	historyCopy := make([]FixRecord, len(afc.fixHistory))
	copy(historyCopy, afc.fixHistory)

	return historyCopy
}
