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

func TestNewPerformanceAgent(t *testing.T) {
	config := AgentConfig{
		Enabled:      true,
		MaxInstances: 1,
		Timeout:      time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, PerformanceAgentID, agent.GetID())
	assert.Equal(t, StatusOffline, agent.GetStatus())
}

func TestPerformanceAgent_Lifecycle(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
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

func TestPerformanceAgent_GetCapabilities(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	capabilities := agent.GetCapabilities()
	assert.NotEmpty(t, capabilities)

	// Should have core performance capabilities
	assert.Contains(t, capabilities, "performance_profiling")
	assert.Contains(t, capabilities, "benchmark_execution")
	assert.Contains(t, capabilities, "optimization_recommendations")
	assert.Contains(t, capabilities, "performance_monitoring")
	assert.Contains(t, capabilities, "regression_detection")
	assert.Contains(t, capabilities, "bottleneck_analysis")
}

func TestPerformanceAgent_ProcessMessage_TaskAssignment(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test performance analysis task
	message := &AgentMessage{
		ID:       "perf-msg-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: PerformanceAgentID,
		Payload: map[string]interface{}{
			"action": "analyze",
			"path":   "./testdata",
			"parameters": map[string]interface{}{
				"deep_analysis": true,
				"profile_cpu":   true,
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityHigh,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestPerformanceAgent_ProcessMessage_Profile(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test profiling task
	message := &AgentMessage{
		ID:       "perf-profile-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: PerformanceAgentID,
		Payload: map[string]interface{}{
			"action":       "profile",
			"profile_type": "cpu",
			"target":       "./main.go",
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestPerformanceAgent_ProcessMessage_Benchmark(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test benchmarking task - use load test type to avoid running actual go benchmark
	message := &AgentMessage{
		ID:       "perf-bench-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: PerformanceAgentID,
		Payload: map[string]interface{}{
			"action":         "benchmark",
			"benchmark_type": "load", // Use load test which has simpler implementation
			"target":         "./test",
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestPerformanceAgent_ProcessMessage_Optimize(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test optimization task
	message := &AgentMessage{
		ID:       "perf-opt-1",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: PerformanceAgentID,
		Payload: map[string]interface{}{
			"action":            "optimize",
			"optimization_type": "algorithm",
			"data":              map[string]interface{}{"functions": []string{"sort", "search"}},
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestPerformanceAgent_ProcessMessage_CollaborationRequest(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
		Timeout: time.Millisecond * 10, // Fast timeout for tests
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test collaboration request
	message := &AgentMessage{
		ID:       "perf-collab-1",
		Type:     CollaborationRequest,
		Sender:   ArchitectureAgentID,
		Receiver: PerformanceAgentID,
		Payload: &AgentCollaboration{
			CollaborationType: ExpertiseConsultation,
			Context: CollaborationContext{
				Purpose: "Need performance guidance for optimization",
				SharedData: map[string]interface{}{
					"bottlenecks": []string{"database queries", "memory allocations"},
					"target_sla":  "< 100ms response time",
				},
			},
		},
		Timestamp: time.Now(),
		Priority:  PriorityMedium,
	}

	err = agent.ProcessMessage(context.Background(), message)
	require.NoError(t, err)
}

func TestPerformanceAgent_CalculateSeverity(t *testing.T) {
	agent := &PerformanceAgent{}

	tests := []struct {
		impact   float64
		expected string
	}{
		{85.0, "critical"},
		{75.0, "critical"},
		{65.0, "high"},
		{50.0, "high"},
		{35.0, "medium"},
		{25.0, "medium"},
		{15.0, "low"},
		{5.0, "low"},
	}

	for _, tt := range tests {
		result := agent.calculateSeverity(tt.impact)
		assert.Equal(t, tt.expected, result, "Impact %.1f should map to %s", tt.impact, tt.expected)
	}
}

func TestPerformanceAgent_MapSeverityToPriority(t *testing.T) {
	agent := &PerformanceAgent{}

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

func TestPerformanceAgent_MapComplexityToPriority(t *testing.T) {
	agent := &PerformanceAgent{}

	tests := []struct {
		complexity string
		expected   Priority
	}{
		{"low", PriorityHigh}, // Low complexity = high priority (easy wins)
		{"LOW", PriorityHigh},
		{"medium", PriorityMedium},
		{"MEDIUM", PriorityMedium},
		{"high", PriorityLow}, // High complexity = low priority
		{"HIGH", PriorityLow},
		{"unknown", PriorityLow},
		{"", PriorityLow},
	}

	for _, tt := range tests {
		result := agent.mapComplexityToPriority(tt.complexity)
		assert.Equal(t, tt.expected, result, "Complexity %s should map to %s", tt.complexity, tt.expected)
	}
}

func TestPerformanceAgent_CountAutomatedFixes(t *testing.T) {
	agent := &PerformanceAgent{}

	recommendations := []PerformanceRecommendation{
		{Automated: true},
		{Automated: false},
		{Automated: true},
		{Automated: true},
		{Automated: false},
	}

	count := agent.countAutomatedFixes(recommendations)
	assert.Equal(t, 3, count)
}

func TestPerformanceAgent_GenerateWarnings(t *testing.T) {
	agent := &PerformanceAgent{}

	tests := []struct {
		name                string
		bottlenecks         []PerformanceBottleneck
		regressions         []PerformanceRegression
		expectedCount       int
		expectedBottlenecks int
		expectedRegressions int
	}{
		{
			name: "No critical issues",
			bottlenecks: []PerformanceBottleneck{
				{Severity: "high"},
				{Severity: "medium"},
			},
			regressions:   []PerformanceRegression{},
			expectedCount: 0,
		},
		{
			name: "Critical bottlenecks only",
			bottlenecks: []PerformanceBottleneck{
				{Severity: "critical"},
				{Severity: "critical"},
				{Severity: "high"},
			},
			regressions:         []PerformanceRegression{},
			expectedCount:       1,
			expectedBottlenecks: 2,
		},
		{
			name:        "Critical regressions only",
			bottlenecks: []PerformanceBottleneck{},
			regressions: []PerformanceRegression{
				{Severity: "critical"},
				{Severity: "high"},
			},
			expectedCount:       1,
			expectedRegressions: 1,
		},
		{
			name: "Both critical bottlenecks and regressions",
			bottlenecks: []PerformanceBottleneck{
				{Severity: "critical"},
			},
			regressions: []PerformanceRegression{
				{Severity: "critical"},
				{Severity: "critical"},
			},
			expectedCount:       2,
			expectedBottlenecks: 1,
			expectedRegressions: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := agent.generateWarnings(tt.bottlenecks, tt.regressions)
			assert.Len(t, warnings, tt.expectedCount)

			if tt.expectedBottlenecks > 0 {
				found := false
				for _, warning := range warnings {
					if strings.Contains(warning, "bottlenecks") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should contain bottleneck warning")
			}

			if tt.expectedRegressions > 0 {
				found := false
				for _, warning := range warnings {
					if strings.Contains(warning, "regressions") {
						found = true
						break
					}
				}
				assert.True(t, found, "Should contain regression warning")
			}
		})
	}
}

func TestPerformanceAgent_GeneratePriorityItems(t *testing.T) {
	agent := &PerformanceAgent{
		BaseAgent: &BaseAgent{id: PerformanceAgentID},
	}

	bottlenecks := []PerformanceBottleneck{
		{
			ID:          "bottleneck-1",
			Type:        "memory",
			Description: "Memory leak in data processing",
			Impact:      85.0,
			Severity:    "critical",
			Suggestion:  "Fix memory leak by proper cleanup",
		},
	}

	recommendations := []PerformanceRecommendation{
		{
			ID:          "rec-1",
			Type:        AlgorithmOptimization,
			Title:       "Optimize sorting algorithm",
			Description: "Replace bubble sort with quicksort",
			Complexity:  "low",
			Automated:   true,
			CodeExample: "sort.Slice(...)",
		},
	}

	items := agent.generatePriorityItems(bottlenecks, recommendations)

	assert.Len(t, items, 2)

	// Check bottleneck item
	bottleneckItem := items[0]
	assert.Equal(t, "perf-bottleneck-bottleneck-1", bottleneckItem.ID)
	assert.Equal(t, PerformanceAgentID, bottleneckItem.AgentID)
	assert.Equal(t, "Performance Bottleneck", bottleneckItem.Type)
	assert.Equal(t, "Memory leak in data processing", bottleneckItem.Description)
	assert.Equal(t, PriorityCritical, bottleneckItem.Severity)
	assert.False(t, bottleneckItem.AutoFixable)
	assert.Equal(t, "Fix memory leak by proper cleanup", bottleneckItem.FixDetails)

	// Check recommendation item
	recItem := items[1]
	assert.Equal(t, "perf-rec-rec-1", recItem.ID)
	assert.Equal(t, PerformanceAgentID, recItem.AgentID)
	assert.Equal(t, "Performance Optimization", recItem.Type)
	assert.Equal(t, "Replace bubble sort with quicksort", recItem.Description)
	assert.Equal(t, PriorityHigh, recItem.Severity) // Low complexity maps to high priority
	assert.True(t, recItem.AutoFixable)
	assert.Equal(t, "sort.Slice(...)", recItem.FixDetails)
}

// Test Profiler Interfaces

func TestCPUProfiler_GetProfileType(t *testing.T) {
	profiler := &CPUProfiler{}
	assert.Equal(t, CPUProfile, profiler.GetProfileType())
}

func TestMemoryProfiler_GetProfileType(t *testing.T) {
	profiler := &MemoryProfiler{}
	assert.Equal(t, MemoryProfile, profiler.GetProfileType())
}

func TestGoroutineProfiler_GetProfileType(t *testing.T) {
	profiler := &GoroutineProfiler{}
	assert.Equal(t, GoroutineProfile, profiler.GetProfileType())
}

func TestCPUProfiler_Profile(t *testing.T) {
	profiler := &CPUProfiler{
		logger: logger.NewDefault(),
	}

	ctx := context.Background()
	options := ProfileOptions{
		Duration:     10 * time.Second,
		SampleRate:   100,
		OutputFormat: "pprof",
	}

	result, err := profiler.Profile(ctx, "./test", options)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, CPUProfile, result.Type)
	assert.Equal(t, options.Duration, result.Duration)
	assert.Greater(t, result.Samples, 0)
	assert.NotEmpty(t, result.TopFunctions)
	assert.NotEmpty(t, result.Hotspots)
}

// Test Benchmarker Interfaces

func TestGoBenchmarker_GetBenchmarkType(t *testing.T) {
	benchmarker := &GoBenchmarker{}
	assert.Equal(t, Microbenchmark, benchmarker.GetBenchmarkType())
}

func TestLoadTestBenchmarker_GetBenchmarkType(t *testing.T) {
	benchmarker := &LoadTestBenchmarker{}
	assert.Equal(t, LoadTest, benchmarker.GetBenchmarkType())
}

// Test Optimizer Interfaces

func TestAlgorithmOptimizer_GetOptimizationType(t *testing.T) {
	optimizer := &AlgorithmOptimizer{}
	assert.Equal(t, AlgorithmOptimization, optimizer.GetOptimizationType())
}

func TestMemoryOptimizer_GetOptimizationType(t *testing.T) {
	optimizer := &MemoryOptimizer{}
	assert.Equal(t, MemoryOptimization, optimizer.GetOptimizationType())
}

func TestCachingOptimizer_GetOptimizationType(t *testing.T) {
	optimizer := &CachingOptimizer{}
	assert.Equal(t, CachingOptimization, optimizer.GetOptimizationType())
}

func TestAlgorithmOptimizer_Analyze(t *testing.T) {
	optimizer := &AlgorithmOptimizer{}

	ctx := context.Background()
	data := []ProfileResult{} // Simplified test data

	result, err := optimizer.Analyze(ctx, data)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, AlgorithmOptimization, result.Type)
	assert.Equal(t, "algorithm", result.Target)
	assert.NotEmpty(t, result.Recommendations)

	rec := result.Recommendations[0]
	assert.Equal(t, "algo-1", rec.ID)
	assert.Equal(t, AlgorithmOptimization, rec.Type)
	assert.Contains(t, rec.Title, "efficient sorting")
	assert.False(t, rec.Automated)
	assert.NotEmpty(t, rec.Benefits)
}

// Test Monitor Interfaces

func TestSystemMonitor_GetMetrics(t *testing.T) {
	monitor := &SystemMonitor{}
	metrics := monitor.GetMetrics()

	assert.Len(t, metrics, 4)
	assert.Contains(t, metrics, CPUUsage)
	assert.Contains(t, metrics, MemoryUsage)
	assert.Contains(t, metrics, DiskIO)
	assert.Contains(t, metrics, NetworkIO)
}

func TestApplicationMonitor_GetMetrics(t *testing.T) {
	monitor := &ApplicationMonitor{}
	metrics := monitor.GetMetrics()

	assert.Len(t, metrics, 4)
	assert.Contains(t, metrics, GoroutineCount)
	assert.Contains(t, metrics, GCStats)
	assert.Contains(t, metrics, Latency)
	assert.Contains(t, metrics, Throughput)
}

// Test Error Handling

func TestPerformanceAgent_ErrorHandling(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test invalid message payload
	message := &AgentMessage{
		ID:       "invalid-msg",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: PerformanceAgentID,
		Payload:  "invalid-payload", // Should be map[string]interface{}
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task data format")
}

func TestPerformanceAgent_InvalidAction(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	err = agent.Start(context.Background())
	require.NoError(t, err)
	defer func() { _ = agent.Stop(context.Background()) }()

	// Test unknown action
	message := &AgentMessage{
		ID:       "unknown-action",
		Type:     TaskAssignment,
		Sender:   "test-agent",
		Receiver: PerformanceAgentID,
		Payload: map[string]interface{}{
			"action": "unknown_action",
		},
	}

	err = agent.ProcessMessage(context.Background(), message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestPerformanceAgent_Metrics(t *testing.T) {
	config := AgentConfig{
		Enabled: true,
	}

	messageBus := NewMessageBus(100)
	defer messageBus.Stop()

	agent, err := NewPerformanceAgent(config, messageBus)
	require.NoError(t, err)

	// Get initial metrics
	metrics := agent.GetMetrics()
	assert.NotNil(t, &metrics) // Use pointer to avoid copying lock
	assert.Equal(t, int64(0), metrics.MessagesReceived)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
}

// Test Data Structures

func TestProfileOptions_Structure(t *testing.T) {
	options := ProfileOptions{
		Duration:      30 * time.Second,
		SampleRate:    100,
		OutputFormat:  "pprof",
		IncludePaths:  []string{"./pkg"},
		ExcludePaths:  []string{"./vendor"},
		CustomOptions: map[string]interface{}{"detail": true},
	}

	assert.Equal(t, 30*time.Second, options.Duration)
	assert.Equal(t, 100, options.SampleRate)
	assert.Equal(t, "pprof", options.OutputFormat)
	assert.Contains(t, options.IncludePaths, "./pkg")
	assert.Contains(t, options.ExcludePaths, "./vendor")
	assert.Equal(t, true, options.CustomOptions["detail"])
}

func TestProfileResult_Structure(t *testing.T) {
	result := ProfileResult{
		Type:     CPUProfile,
		Duration: 30 * time.Second,
		Samples:  1000,
		TopFunctions: []FunctionProfile{
			{
				Name:         "main.process",
				File:         "main.go",
				Line:         42,
				SelfTime:     500 * time.Millisecond,
				TotalTime:    2 * time.Second,
				SelfPercent:  25.0,
				TotalPercent: 40.0,
				Calls:        1000,
			},
		},
		Hotspots: []PerformanceHotspot{
			{
				Location:    "main.go:42",
				Type:        "cpu_intensive",
				Impact:      85.0,
				Description: "CPU hotspot",
				Suggestion:  "Optimize algorithm",
			},
		},
		Metrics: ProfileMetrics{
			TotalSamples:    1000,
			SampleRate:      100.0,
			TotalDuration:   30 * time.Second,
			MemoryAllocated: 1024 * 1024,
			GCCycles:        5,
		},
		RawData: "profile data",
	}

	assert.Equal(t, CPUProfile, result.Type)
	assert.Equal(t, 30*time.Second, result.Duration)
	assert.Equal(t, 1000, result.Samples)
	assert.Len(t, result.TopFunctions, 1)
	assert.Len(t, result.Hotspots, 1)

	fn := result.TopFunctions[0]
	assert.Equal(t, "main.process", fn.Name)
	assert.Equal(t, "main.go", fn.File)
	assert.Equal(t, 42, fn.Line)

	hotspot := result.Hotspots[0]
	assert.Equal(t, "main.go:42", hotspot.Location)
	assert.Equal(t, 85.0, hotspot.Impact)

	metrics := result.Metrics
	assert.Equal(t, int64(1000), metrics.TotalSamples)
	assert.Equal(t, 100.0, metrics.SampleRate)
}

func TestBenchmarkReport_Structure(t *testing.T) {
	report := BenchmarkReport{
		Type:            Microbenchmark,
		Name:            "BenchmarkSort",
		Iterations:      1000,
		TotalDuration:   10 * time.Second,
		AverageDuration: 100 * time.Microsecond,
		MinDuration:     50 * time.Microsecond,
		MaxDuration:     200 * time.Microsecond,
		Throughput:      10000.0,
		LatencyP50:      90 * time.Microsecond,
		LatencyP95:      150 * time.Microsecond,
		LatencyP99:      180 * time.Microsecond,
		MemoryAllocs:    100,
		BytesAllocated:  1024,
		ErrorRate:       0.1,
		Results: []BenchmarkResult{
			{
				Iteration: 1,
				Duration:  95 * time.Microsecond,
				Success:   true,
				Metrics:   map[string]float64{"cpu": 0.5},
			},
		},
	}

	assert.Equal(t, Microbenchmark, report.Type)
	assert.Equal(t, "BenchmarkSort", report.Name)
	assert.Equal(t, 1000, report.Iterations)
	assert.Equal(t, 10*time.Second, report.TotalDuration)
	assert.Equal(t, 100*time.Microsecond, report.AverageDuration)
	assert.Equal(t, 10000.0, report.Throughput)
	assert.Len(t, report.Results, 1)

	result := report.Results[0]
	assert.Equal(t, 1, result.Iteration)
	assert.True(t, result.Success)
	assert.Equal(t, 0.5, result.Metrics["cpu"])
}

func TestPerformanceRecommendation_Structure(t *testing.T) {
	rec := PerformanceRecommendation{
		ID:          "rec-1",
		Type:        AlgorithmOptimization,
		Title:       "Optimize sorting",
		Description: "Use efficient sorting algorithm",
		Location:    "main.go:42",
		Current: PerformanceMetric{
			CPUTime:    2 * time.Second,
			Throughput: 1000.0,
		},
		Predicted: PerformanceMetric{
			CPUTime:    500 * time.Millisecond,
			Throughput: 4000.0,
		},
		Complexity:  "medium",
		Confidence:  0.9,
		Automated:   false,
		CodeExample: "sort.Slice(...)",
		Benefits:    []string{"Faster sorting", "Lower CPU usage"},
		Risks:       []string{"Code complexity"},
	}

	assert.Equal(t, "rec-1", rec.ID)
	assert.Equal(t, AlgorithmOptimization, rec.Type)
	assert.Equal(t, "Optimize sorting", rec.Title)
	assert.Equal(t, "main.go:42", rec.Location)
	assert.Equal(t, "medium", rec.Complexity)
	assert.Equal(t, 0.9, rec.Confidence)
	assert.False(t, rec.Automated)
	assert.Len(t, rec.Benefits, 2)
	assert.Len(t, rec.Risks, 1)

	assert.Equal(t, 2*time.Second, rec.Current.CPUTime)
	assert.Equal(t, 1000.0, rec.Current.Throughput)
	assert.Equal(t, 500*time.Millisecond, rec.Predicted.CPUTime)
	assert.Equal(t, 4000.0, rec.Predicted.Throughput)
}

// Test Constants

func TestProfileType_Constants(t *testing.T) {
	assert.Equal(t, ProfileType("cpu"), CPUProfile)
	assert.Equal(t, ProfileType("memory"), MemoryProfile)
	assert.Equal(t, ProfileType("goroutine"), GoroutineProfile)
	assert.Equal(t, ProfileType("block"), BlockProfile)
	assert.Equal(t, ProfileType("mutex"), MutexProfile)
	assert.Equal(t, ProfileType("heap"), HeapProfile)
	assert.Equal(t, ProfileType("alloc"), AllocProfile)
}

func TestBenchmarkType_Constants(t *testing.T) {
	assert.Equal(t, BenchmarkType("micro"), Microbenchmark)
	assert.Equal(t, BenchmarkType("macro"), Macrobenchmark)
	assert.Equal(t, BenchmarkType("load"), LoadTest)
	assert.Equal(t, BenchmarkType("stress"), StressTest)
	assert.Equal(t, BenchmarkType("endurance"), EnduranceTest)
	assert.Equal(t, BenchmarkType("spike"), SpikeTest)
}

func TestOptimizationType_Constants(t *testing.T) {
	assert.Equal(t, OptimizationType("algorithm"), AlgorithmOptimization)
	assert.Equal(t, OptimizationType("memory"), MemoryOptimization)
	assert.Equal(t, OptimizationType("caching"), CachingOptimization)
	assert.Equal(t, OptimizationType("io"), IOOptimization)
	assert.Equal(t, OptimizationType("database"), DatabaseOptimization)
	assert.Equal(t, OptimizationType("network"), NetworkOptimization)
	assert.Equal(t, OptimizationType("concurrency"), ConcurrencyOptimization)
}

func TestMetricType_Constants(t *testing.T) {
	assert.Equal(t, MetricType("cpu_usage"), CPUUsage)
	assert.Equal(t, MetricType("memory_usage"), MemoryUsage)
	assert.Equal(t, MetricType("disk_io"), DiskIO)
	assert.Equal(t, MetricType("network_io"), NetworkIO)
	assert.Equal(t, MetricType("goroutine_count"), GoroutineCount)
	assert.Equal(t, MetricType("gc_stats"), GCStats)
	assert.Equal(t, MetricType("latency"), Latency)
	assert.Equal(t, MetricType("throughput"), Throughput)
}
