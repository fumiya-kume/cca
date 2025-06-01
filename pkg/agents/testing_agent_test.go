package agents

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestingAgent(t *testing.T) {
	config := AgentConfig{
		Enabled:      true,
		MaxInstances: 1,
		Timeout:      time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, TestingAgentID, agent.GetID())
	assert.Equal(t, StatusOffline, agent.GetStatus())
}

func TestTestingAgent_Lifecycle(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	// Test starting the agent
	ctx := context.Background()
	err = agent.Start(ctx)
	require.NoError(t, err)
	assert.Equal(t, StatusIdle, agent.GetStatus())

	// Test health check
	err = agent.HealthCheck(ctx)
	require.NoError(t, err)

	// Test stopping the agent
	err = agent.Stop(ctx)
	require.NoError(t, err)
}

func TestTestingAgent_GetCapabilities(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	capabilities := agent.GetCapabilities()
	assert.NotEmpty(t, capabilities)

	// Should have core testing capabilities
	assert.Contains(t, capabilities, "test_coverage_analysis")
	assert.Contains(t, capabilities, "test_generation")
	assert.Contains(t, capabilities, "test_execution")
	assert.Contains(t, capabilities, "test_quality_assessment")
	assert.Contains(t, capabilities, "performance_testing")
	assert.Contains(t, capabilities, "mutation_testing")
}

func TestTestingAgent_ProcessMessage_TaskAssignment(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test analysis task
	message := &AgentMessage{
		ID:       "test-msg-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: TestingAgentID,
		Payload: map[string]interface{}{
			"action": "analyze",
			"path":   "./testdata",
			"parameters": map[string]interface{}{
				"coverage_analysis": true,
				"quality_check":     true,
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestTestingAgent_ProcessMessage_Generate(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test generation task
	message := &AgentMessage{
		ID:       "test-gen-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: TestingAgentID,
		Payload: map[string]interface{}{
			"action":    "generate",
			"test_type": "unit",
			"target": map[string]interface{}{
				"type":     "function",
				"name":     "ProcessData",
				"location": "main.go:42",
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestTestingAgent_ProcessMessage_Run(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test execution task - use unknown framework to avoid actual execution
	message := &AgentMessage{
		ID:       "test-run-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: TestingAgentID,
		Payload: map[string]interface{}{
			"action":    "run",
			"path":      ".",
			"framework": "unknown", // Use unknown framework to test error handling
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no runner for framework")
}

func TestTestingAgent_ProcessMessage_CollaborationRequest(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test collaboration request
	message := &AgentMessage{
		ID:       "test-collab-1",
		Type:     CollaborationRequest,
		Sender:   ArchitectureAgentID,
		Receiver: TestingAgentID,
		Payload: &AgentCollaboration{
			CollaborationType: ExpertiseConsultation,
			Context: CollaborationContext{
				Purpose: "Need testing guidance for new feature",
				SharedData: map[string]interface{}{
					"feature_type": "authentication",
					"complexity":   "high",
				},
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestTestingAgent_CalculateOverallCoverage(t *testing.T) {
	agent := &TestingAgent{}

	coverage := TestCoverage{
		LineCoverage:     CoverageMetric{Percentage: 80.0},
		BranchCoverage:   CoverageMetric{Percentage: 70.0},
		FunctionCoverage: CoverageMetric{Percentage: 90.0},
	}

	overall := agent.calculateOverallCoverage(coverage)
	expected := 80.0*0.4 + 70.0*0.3 + 90.0*0.3 // 32 + 21 + 27 = 80.0
	assert.Equal(t, expected, overall)
}

func TestTestingAgent_MapPriorityToSeverity(t *testing.T) {
	agent := &TestingAgent{}

	tests := []struct {
		priority string
		expected Priority
	}{
		{"critical", PriorityCritical},
		{"CRITICAL", PriorityCritical},
		{"high", PriorityHigh},
		{"HIGH", PriorityHigh},
		{"medium", PriorityMedium},
		{"MEDIUM", PriorityMedium},
		{"low", PriorityLow},
		{"LOW", PriorityLow},
		{"unknown", PriorityLow},
		{"", PriorityLow},
	}

	for _, tt := range tests {
		result := agent.mapPriorityToSeverity(tt.priority)
		assert.Equal(t, tt.expected, result, "Priority %s should map to %s", tt.priority, tt.expected)
	}
}

func TestTestingAgent_MapSeverityToPriority(t *testing.T) {
	agent := &TestingAgent{}

	tests := []struct {
		severity string
		expected Priority
	}{
		{"critical", PriorityCritical},
		{"CRITICAL", PriorityCritical},
		{"high", PriorityHigh},
		{"HIGH", PriorityHigh},
		{"medium", PriorityMedium},
		{"MEDIUM", PriorityMedium},
		{"low", PriorityLow},
		{"LOW", PriorityLow},
		{"unknown", PriorityLow},
		{"", PriorityLow},
	}

	for _, tt := range tests {
		result := agent.mapSeverityToPriority(tt.severity)
		assert.Equal(t, tt.expected, result, "Severity %s should map to %s", tt.severity, tt.expected)
	}
}

func TestTestingAgent_CountAutoGenerable(t *testing.T) {
	agent := &TestingAgent{}

	gaps := []TestGap{
		{AutoGenerate: true},
		{AutoGenerate: false},
		{AutoGenerate: true},
		{AutoGenerate: true},
		{AutoGenerate: false},
	}

	count := agent.countAutoGenerable(gaps)
	assert.Equal(t, 3, count)
}

func TestTestingAgent_GenerateWarnings(t *testing.T) {
	agent := &TestingAgent{}

	tests := []struct {
		name           string
		gaps           []TestGap
		issues         []TestQualityIssue
		expectedCount  int
		expectedGaps   bool
		expectedIssues bool
	}{
		{
			name: "No critical issues",
			gaps: []TestGap{
				{Priority: "low"},
				{Priority: "medium"},
			},
			issues:        []TestQualityIssue{},
			expectedCount: 0,
		},
		{
			name: "High priority gaps only",
			gaps: []TestGap{
				{Priority: "high"},
				{Priority: "critical"},
				{Priority: "high"},
				{Priority: "high"},
				{Priority: "medium"},
			},
			issues:         []TestQualityIssue{},
			expectedCount:  1,
			expectedGaps:   true,
			expectedIssues: false,
		},
		{
			name: "Critical issues only",
			gaps: []TestGap{{Priority: "low"}},
			issues: []TestQualityIssue{
				{Severity: "critical"},
				{Severity: "high"},
			},
			expectedCount:  1,
			expectedGaps:   false,
			expectedIssues: true,
		},
		{
			name: "Both high gaps and critical issues",
			gaps: []TestGap{
				{Priority: "high"},
				{Priority: "critical"},
				{Priority: "high"},
				{Priority: "high"},
			},
			issues: []TestQualityIssue{
				{Severity: "critical"},
				{Severity: "critical"},
			},
			expectedCount:  2,
			expectedGaps:   true,
			expectedIssues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := agent.generateWarnings(tt.gaps, tt.issues)
			assert.Len(t, warnings, tt.expectedCount)

			if tt.expectedGaps {
				found := false
				for _, warning := range warnings {
					if strings.Contains(warning, "test coverage gaps") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should contain test gaps warning")
			}

			if tt.expectedIssues {
				found := false
				for _, warning := range warnings {
					if strings.Contains(warning, "quality issues") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should contain quality issues warning")
			}
		})
	}
}

func TestTestingAgent_GeneratePriorityItems(t *testing.T) {
	agent := &TestingAgent{
		BaseAgent: &BaseAgent{id: TestingAgentID},
	}

	gaps := []TestGap{
		{
			Type:         "missing_test_file",
			Target:       "user_service.go",
			Description:  "No test file found for user_service.go",
			Priority:     "high",
			TestType:     UnitTest,
			AutoGenerate: true,
		},
	}

	issues := []TestQualityIssue{
		{
			Type:        "poor_assertion",
			Severity:    "medium",
			Test:        "TestUserLogin",
			Description: "Test uses weak assertions",
			Suggestion:  "Use more specific assertions",
			AutoFixable: true,
		},
	}

	suggestions := []TestSuggestion{
		{
			Type:        "coverage_improvement",
			Target:      "auth_handler.go",
			Description: "Add edge case tests",
			TestType:    UnitTest,
			Priority:    "medium",
		},
	}

	items := agent.generatePriorityItems(gaps, issues, suggestions)

	assert.Len(t, items, 2) // gaps + issues (suggestions not included in current implementation)

	// Check gap item
	gapItem := items[0]
	assert.Equal(t, "test-gap-missing_test_file-user_service.go", gapItem.ID)
	assert.Equal(t, TestingAgentID, gapItem.AgentID)
	assert.Equal(t, "Test Coverage Gap", gapItem.Type)
	assert.Equal(t, "No test file found for user_service.go", gapItem.Description)
	assert.Equal(t, PriorityHigh, gapItem.Severity)
	assert.True(t, gapItem.AutoFixable)
	assert.Equal(t, string(UnitTest), gapItem.FixDetails)

	// Check issue item
	issueItem := items[1]
	assert.Equal(t, "test-issue-poor_assertion", issueItem.ID)
	assert.Equal(t, TestingAgentID, issueItem.AgentID)
	assert.Equal(t, "Test Quality Issue", issueItem.Type)
	assert.Equal(t, "Test uses weak assertions", issueItem.Description)
	assert.Equal(t, PriorityMedium, issueItem.Severity)
	assert.True(t, issueItem.AutoFixable)
	assert.Equal(t, "Use more specific assertions", issueItem.FixDetails)
}

// Test Analyzer Interfaces

func TestCoverageTestAnalyzer_GetName(t *testing.T) {
	analyzer := &CoverageTestAnalyzer{}
	assert.Equal(t, "coverage", analyzer.GetName())
}

func TestQualityTestAnalyzer_GetName(t *testing.T) {
	analyzer := &QualityTestAnalyzer{}
	assert.Equal(t, "quality", analyzer.GetName())
}

func TestPerformanceTestAnalyzer_GetName(t *testing.T) {
	analyzer := &PerformanceTestAnalyzer{}
	assert.Equal(t, "performance", analyzer.GetName())
}

// Test Generator Interfaces

func TestUnitTestGenerator_GetTestType(t *testing.T) {
	generator := &UnitTestGenerator{}
	assert.Equal(t, UnitTest, generator.GetTestType())
}

func TestIntegrationTestGenerator_GetTestType(t *testing.T) {
	generator := &IntegrationTestGenerator{}
	assert.Equal(t, IntegrationTest, generator.GetTestType())
}

func TestBenchmarkTestGenerator_GetTestType(t *testing.T) {
	generator := &BenchmarkTestGenerator{}
	assert.Equal(t, BenchmarkTest, generator.GetTestType())
}

func TestUnitTestGenerator_Generate(t *testing.T) {
	generator := &UnitTestGenerator{
		logger: logger.NewDefault(),
	}

	target := TestTarget{
		Type:     "function",
		Name:     "ProcessData",
		Location: "main.go:42",
	}

	ctx := context.Background()
	generated, err := generator.Generate(ctx, target)
	require.NoError(t, err)
	assert.NotNil(t, generated)

	assert.Equal(t, "TestProcessData", generated.Name)
	assert.Equal(t, UnitTest, generated.Type)
	assert.Contains(t, generated.Code, "TestProcessData")
	assert.Contains(t, generated.Code, "ProcessData")
	assert.Contains(t, generated.Description, "ProcessData")
	assert.Len(t, generated.TestCases, 1)

	testCase := generated.TestCases[0]
	assert.Equal(t, "should handle valid input", testCase.Name)
	assert.Equal(t, "test", testCase.Input)
	assert.Equal(t, "result", testCase.Expected)
}

// Test Runner Interfaces

func TestGoTestRunner_GetFramework(t *testing.T) {
	runner := &GoTestRunner{}
	assert.Equal(t, "go", runner.GetFramework())
}

func TestGoTestRunner_Run_ErrorHandling(t *testing.T) {
	runner := &GoTestRunner{
		logger: logger.NewDefault(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	options := TestRunOptions{
		Coverage: true,
		Timeout:  "1ms", // Very short timeout to cause failure
	}

	// This should fail due to timeout or non-existent path
	_, err := runner.Run(ctx, "/non/existent/path", options)
	assert.Error(t, err)
}

// Test Validator Interfaces

func TestCodeQualityValidator_Validate(t *testing.T) {
	validator := &CodeQualityValidator{}

	ctx := context.Background()
	testCode := `func TestExample(t *testing.T) { t.Log("test") }`

	validation, err := validator.Validate(ctx, testCode)
	require.NoError(t, err)
	assert.NotNil(t, validation)
	assert.True(t, validation.Valid)
	assert.Equal(t, 85.0, validation.Score)
	assert.Equal(t, 90.0, validation.Metrics.Readability)
	assert.Equal(t, 85.0, validation.Metrics.Maintainability)
	assert.Equal(t, 80.0, validation.Metrics.Reliability)
	assert.Equal(t, 85.0, validation.Metrics.Performance)
}

func TestConventionValidator_Validate(t *testing.T) {
	validator := &ConventionValidator{}

	ctx := context.Background()
	testCode := `func TestExample(t *testing.T) { t.Log("test") }`

	validation, err := validator.Validate(ctx, testCode)
	require.NoError(t, err)
	assert.NotNil(t, validation)
	assert.True(t, validation.Valid)
	assert.Equal(t, 90.0, validation.Score)
}

// Test Error Handling

func TestTestingAgent_ErrorHandling(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test invalid message payload
	message := &AgentMessage{
		ID:       "invalid-msg",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: TestingAgentID,
		Payload:  "invalid-payload", // Should be map[string]interface{}
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task data format")
}

func TestTestingAgent_InvalidAction(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test unknown action
	message := &AgentMessage{
		ID:       "unknown-action",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: TestingAgentID,
		Payload: map[string]interface{}{
			"action": "unknown_action",
		},
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestTestingAgent_Metrics(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewTestingAgent(config, messageBus)
	require.NoError(t, err)

	// Get initial metrics
	metrics := agent.GetMetrics()
	assert.NotNil(t, &metrics) // Use pointer to avoid copying lock
	assert.Equal(t, int64(0), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

// Test Data Structures

func TestTestCoverage_Structure(t *testing.T) {
	coverage := TestCoverage{
		LineCoverage: CoverageMetric{
			Total:      1000,
			Documented: 800,
			Percentage: 80.0,
		},
		BranchCoverage: CoverageMetric{
			Total:      200,
			Documented: 140,
			Percentage: 70.0,
		},
		FunctionCoverage: CoverageMetric{
			Total:      50,
			Documented: 45,
			Percentage: 90.0,
		},
		UnitTests: TestMetric{
			Total:       100,
			Passing:     95,
			Failing:     3,
			Skipped:     2,
			SuccessRate: 95.0,
		},
		IntegrationTests: TestMetric{
			Total:       20,
			Passing:     18,
			Failing:     1,
			Skipped:     1,
			SuccessRate: 90.0,
		},
		E2ETests: TestMetric{
			Total:       5,
			Passing:     5,
			Failing:     0,
			Skipped:     0,
			SuccessRate: 100.0,
		},
		OverallCoverage: 83.0,
	}

	assert.Equal(t, 1000, coverage.LineCoverage.Total)
	assert.Equal(t, 800, coverage.LineCoverage.Documented)
	assert.Equal(t, 80.0, coverage.LineCoverage.Percentage)

	assert.Equal(t, 200, coverage.BranchCoverage.Total)
	assert.Equal(t, 140, coverage.BranchCoverage.Documented)
	assert.Equal(t, 70.0, coverage.BranchCoverage.Percentage)

	assert.Equal(t, 50, coverage.FunctionCoverage.Total)
	assert.Equal(t, 45, coverage.FunctionCoverage.Documented)
	assert.Equal(t, 90.0, coverage.FunctionCoverage.Percentage)

	assert.Equal(t, 100, coverage.UnitTests.Total)
	assert.Equal(t, 95, coverage.UnitTests.Passing)
	assert.Equal(t, 3, coverage.UnitTests.Failing)
	assert.Equal(t, 2, coverage.UnitTests.Skipped)
	assert.Equal(t, 95.0, coverage.UnitTests.SuccessRate)

	assert.Equal(t, 83.0, coverage.OverallCoverage)
}

func TestTestGap_Structure(t *testing.T) {
	gap := TestGap{
		Type:         "missing_test_file",
		Target:       "user_service.go",
		Location:     "/src/user_service.go",
		Description:  "No test file found for user_service.go",
		Priority:     "high",
		TestType:     UnitTest,
		AutoGenerate: true,
		Complexity:   "medium",
	}

	assert.Equal(t, "missing_test_file", gap.Type)
	assert.Equal(t, "user_service.go", gap.Target)
	assert.Equal(t, "/src/user_service.go", gap.Location)
	assert.Equal(t, "No test file found for user_service.go", gap.Description)
	assert.Equal(t, "high", gap.Priority)
	assert.Equal(t, UnitTest, gap.TestType)
	assert.True(t, gap.AutoGenerate)
	assert.Equal(t, "medium", gap.Complexity)
}

func TestTestResult_Structure(t *testing.T) {
	result := TestResult{
		Success:      true,
		TotalTests:   100,
		PassedTests:  95,
		FailedTests:  3,
		SkippedTests: 2,
		Duration:     5 * time.Second,
		Coverage:     85.5,
		Failures: []TestFailure{
			{
				Test:     "TestUserLogin",
				Message:  "assertion failed",
				Location: "user_test.go:42",
				Output:   "Expected true, got false",
			},
		},
		Benchmarks: []TestBenchmarkResult{
			{
				Name:        "BenchmarkProcessData",
				Iterations:  1000,
				Duration:    time.Second,
				PerOp:       time.Microsecond,
				AllocsPerOp: 10,
				BytesPerOp:  1024,
			},
		},
	}

	assert.True(t, result.Success)
	assert.Equal(t, 100, result.TotalTests)
	assert.Equal(t, 95, result.PassedTests)
	assert.Equal(t, 3, result.FailedTests)
	assert.Equal(t, 2, result.SkippedTests)
	assert.Equal(t, 5*time.Second, result.Duration)
	assert.Equal(t, 85.5, result.Coverage)
	assert.Len(t, result.Failures, 1)
	assert.Len(t, result.Benchmarks, 1)

	failure := result.Failures[0]
	assert.Equal(t, "TestUserLogin", failure.Test)
	assert.Equal(t, "assertion failed", failure.Message)
	assert.Equal(t, "user_test.go:42", failure.Location)
	assert.Equal(t, "Expected true, got false", failure.Output)

	benchmark := result.Benchmarks[0]
	assert.Equal(t, "BenchmarkProcessData", benchmark.Name)
	assert.Equal(t, 1000, benchmark.Iterations)
	assert.Equal(t, time.Second, benchmark.Duration)
	assert.Equal(t, time.Microsecond, benchmark.PerOp)
	assert.Equal(t, 10, benchmark.AllocsPerOp)
	assert.Equal(t, 1024, benchmark.BytesPerOp)
}

func TestGeneratedTest_Structure(t *testing.T) {
	test := GeneratedTest{
		Name:        "TestProcessData",
		Type:        UnitTest,
		Code:        "func TestProcessData(t *testing.T) { ... }",
		Description: "Unit test for ProcessData function",
		TestCases: []TestCase{
			{
				Name:        "should handle valid input",
				Description: "Test with valid input data",
				Input:       "test data",
				Expected:    "processed data",
				Setup:       "setup code",
				Teardown:    "cleanup code",
				Tags:        []string{"unit", "fast"},
			},
		},
		Fixtures: []TestFixture{
			{
				Name:    "user_data",
				Type:    "json",
				Data:    map[string]string{"name": "test"},
				Setup:   "create test user",
				Cleanup: "delete test user",
			},
		},
		Metadata: map[string]string{
			"author":     "testing-agent",
			"generated":  "true",
			"complexity": "low",
		},
	}

	assert.Equal(t, "TestProcessData", test.Name)
	assert.Equal(t, UnitTest, test.Type)
	assert.Contains(t, test.Code, "TestProcessData")
	assert.Equal(t, "Unit test for ProcessData function", test.Description)
	assert.Len(t, test.TestCases, 1)
	assert.Len(t, test.Fixtures, 1)
	assert.Len(t, test.Metadata, 3)

	testCase := test.TestCases[0]
	assert.Equal(t, "should handle valid input", testCase.Name)
	assert.Equal(t, "test data", testCase.Input)
	assert.Equal(t, "processed data", testCase.Expected)
	assert.Contains(t, testCase.Tags, "unit")

	fixture := test.Fixtures[0]
	assert.Equal(t, "user_data", fixture.Name)
	assert.Equal(t, "json", fixture.Type)
	assert.Equal(t, "create test user", fixture.Setup)

	assert.Equal(t, "testing-agent", test.Metadata["author"])
	assert.Equal(t, "true", test.Metadata["generated"])
}

// Test Constants

func TestTestType_Constants(t *testing.T) {
	assert.Equal(t, TestType("unit"), UnitTest)
	assert.Equal(t, TestType("integration"), IntegrationTest)
	assert.Equal(t, TestType("e2e"), E2ETest)
	assert.Equal(t, TestType("benchmark"), BenchmarkTest)
	assert.Equal(t, TestType("property"), PropertyTest)
	assert.Equal(t, TestType("mutation"), MutationTest)
}
