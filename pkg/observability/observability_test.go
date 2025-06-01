package observability

import (
	"context"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
	"github.com/fumiya-kume/cca/pkg/performance"
)

func TestObservabilityManager(t *testing.T) {
	// Create test logger
	testLogger := logger.NewDefault()

	// Create performance monitor
	perfMonitor := performance.NewPerformanceMonitor(1*time.Second, testLogger)

	// Create observability manager
	config := DefaultObservabilityConfig()
	config.Dashboard.Enabled = false // Disable dashboard for tests
	config.Telemetry.Enabled = false // Disable telemetry for tests

	manager := NewObservabilityManager(config, testLogger, perfMonitor)

	// Test initialization
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize observability manager: %v", err)
	}

	// Test start
	err = manager.Start()
	if err != nil {
		t.Fatalf("Failed to start observability manager: %v", err)
	}

	// Test component access
	if manager.GetStructuredLogger() == nil {
		t.Error("Structured logger should be available")
	}

	if manager.GetMetricsCollector() == nil {
		t.Error("Metrics collector should be available")
	}

	if manager.GetSessionTracker() == nil {
		t.Error("Session tracker should be available")
	}

	// Test recording activity
	metadata := map[string]interface{}{
		"test_key": "test_value",
		"count":    42,
	}

	manager.RecordActivity("test_activity", true, 100*time.Millisecond, nil, metadata)

	// Test status
	status := manager.GetOverallStatus()
	if !status["enabled"].(bool) {
		t.Error("Manager should be enabled")
	}

	if !status["running"].(bool) {
		t.Error("Manager should be running")
	}

	// Test stop
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop observability manager: %v", err)
	}
}

func TestStructuredLogger(t *testing.T) {
	testLogger := logger.NewDefault()
	structuredLogger := NewStructuredLogger(testLogger)

	// Test basic logging
	structuredLogger.Info("Test message", "key", "value")
	structuredLogger.Debug("Debug message", "debug_key", "debug_value")
	structuredLogger.Warn("Warning message", "warn_key", "warn_value")
	structuredLogger.Error("Error message", "error_key", "error_value")

	// Test context
	contextLogger := structuredLogger.WithContext("component", "test").WithTraceID("trace123")
	contextLogger.Info("Context message", "context_key", "context_value")

	// Test error logging
	testError := &testError{message: "test error"}
	structuredLogger.LogError(testError, "Error occurred", "error_context", "test")

	// Note: With real logger, we can't easily verify logs were written
	// but the test above verifies the methods can be called without error
}

func TestMetricsCollector(t *testing.T) {
	testLogger := logger.NewDefault()
	perfMonitor := performance.NewPerformanceMonitor(1*time.Second, testLogger)
	aggregator := NewLogAggregator(100)

	collector := NewApplicationMetricsCollector(perfMonitor, aggregator, testLogger)

	// Test metric registration
	definition := MetricDefinition{
		Name:        "test.metric",
		Type:        "gauge",
		Description: "Test metric",
		Unit:        "count",
		Labels:      map[string]string{"env": "test"},
	}

	err := collector.RegisterMetric(definition)
	if err != nil {
		t.Fatalf("Failed to register metric: %v", err)
	}

	// Test duplicate registration
	err = collector.RegisterMetric(definition)
	if err == nil {
		t.Error("Expected error for duplicate metric registration")
	}

	// Test collection
	metrics := collector.CollectApplicationMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be collected")
	}

	// Test summary
	summary := collector.GetMetricsSummary()
	if summary["registered_metrics"].(int) != 1 {
		t.Errorf("Expected 1 registered metric, got %d", summary["registered_metrics"].(int))
	}
}

func TestTelemetryCollector(t *testing.T) {
	config := DefaultTelemetryConfig()
	config.Enabled = true
	config.OutputPath = "./test_telemetry"

	testLogger := logger.NewDefault()
	collector := NewTelemetryCollector(config, testLogger)

	// Test enabling
	collector.Enable()
	if !collector.IsEnabled() {
		t.Error("Telemetry should be enabled")
	}

	// Test event recording
	data := map[string]interface{}{
		"test_key": "test_value",
		"count":    123,
	}

	collector.RecordEvent("test_event", data)
	collector.RecordTiming("test_timing", 50*time.Millisecond, data)
	collector.RecordError("test_error", &testError{message: "test error"}, data)

	// Test getting events
	events := collector.GetEvents()
	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	// Test summary
	summary := collector.GetSummary()
	if summary["event_count"].(int) != 3 {
		t.Errorf("Expected 3 events in summary, got %d", summary["event_count"].(int))
	}

	// Test disabling
	collector.Disable()
	if collector.IsEnabled() {
		t.Error("Telemetry should be disabled")
	}
}

func TestSessionTracker(t *testing.T) {
	testLogger := logger.NewDefault()
	tracker := NewSessionTracker(testLogger)

	sessionID := tracker.GetSessionID()
	if sessionID == "" {
		t.Error("Session ID should not be empty")
	}

	// Test metadata
	tracker.SetMetadata("test_key", "test_value")
	tracker.SetMetadata("user_id", 12345)

	// Test activity recording
	tracker.RecordActivity("test_activity", true, nil, map[string]interface{}{
		"activity_data": "test",
	})

	tracker.RecordTimedActivity("timed_activity", 100*time.Millisecond, true, nil, map[string]interface{}{
		"duration_data": "test",
	})

	// Test error activity
	tracker.RecordActivity("error_activity", false, &testError{message: "test error"}, nil)

	// Test summary
	summary := tracker.GetSessionSummary()
	if summary["session_id"].(string) != sessionID {
		t.Error("Session ID mismatch in summary")
	}

	if summary["total_activities"].(int) != 3 {
		t.Errorf("Expected 3 activities, got %d", summary["total_activities"].(int))
	}

	if summary["successful_activities"].(int) != 2 {
		t.Errorf("Expected 2 successful activities, got %d", summary["successful_activities"].(int))
	}

	// Test ending session
	tracker.EndSession()
	summary = tracker.GetSessionSummary()
	if summary["is_active"].(bool) {
		t.Error("Session should not be active after ending")
	}
}

func TestDebugManager(t *testing.T) {
	config := DefaultDebugConfig()
	config.Enabled = true
	config.EnableProfiling = true
	config.EnableTracing = true
	config.OutputPath = "./test_debug"

	testLogger := logger.NewDefault()
	manager := NewDebugManager(config, testLogger)

	// Test enabling
	manager.Enable()

	// Test trace recording
	event := TraceEvent{
		Type:      "test_trace",
		Component: "test_component",
		Operation: "test_operation",
		Data:      map[string]interface{}{"test": "data"},
	}

	manager.RecordTrace(event)

	// Test trace span
	span := manager.StartTrace("test_component", "test_operation", map[string]interface{}{
		"span_data": "test",
	})

	span.SetData("additional_key", "additional_value")
	span.AddEvent("span_event", map[string]interface{}{"event_data": "test"})
	span.RecordError(&testError{message: "span error"})
	span.Finish()

	// Test diagnostic info
	diagnostics := manager.GetDiagnosticInfo()
	if diagnostics.Runtime.NumCPU <= 0 {
		t.Error("Expected positive CPU count in diagnostics")
	}

	if diagnostics.Memory.Alloc <= 0 {
		t.Error("Expected positive memory allocation in diagnostics")
	}

	// Test disabling
	manager.Disable()
}

func TestLogAggregator(t *testing.T) {
	aggregator := NewLogAggregator(100)

	// Test filters
	levelFilter := &LevelFilter{MinLevel: "WARN"}
	aggregator.AddFilter(levelFilter)

	componentFilter := &ComponentFilter{
		AllowedComponents: map[string]bool{
			"test_component": true,
		},
	}
	aggregator.AddFilter(componentFilter)

	// Test processors
	sanitizer := &SanitizationProcessor{
		SensitiveFields: []string{"password", "secret"},
	}
	aggregator.AddProcessor(sanitizer)

	metricsCollector := NewMetricsCollector()
	aggregator.AddProcessor(metricsCollector)

	// Test file output
	output, err := NewFileLogOutput("./test_logs/test.log")
	if err != nil {
		t.Fatalf("Failed to create file output: %v", err)
	}
	aggregator.AddOutput(output)

	// Test log entry processing
	entry := LogEntry{
		Level:     "ERROR",
		Message:   "Test error message",
		Component: "test_component",
		Context: map[string]interface{}{
			"password": "secret123",
			"user_id":  12345,
		},
	}

	aggregator.AddEntry(entry)

	// Test retrieval
	entries := aggregator.GetEntries(time.Time{}, "", "")
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// Test stats
	stats := aggregator.GetStats()
	if stats["total_entries"].(int) != 1 {
		t.Errorf("Expected 1 total entry, got %d", stats["total_entries"].(int))
	}

	// Cleanup
	_ = output.Close()
}

func TestDashboard(t *testing.T) {
	// Create test components
	testLogger := logger.NewDefault()
	perfMonitor := performance.NewPerformanceMonitor(1*time.Second, testLogger)
	aggregator := NewLogAggregator(100)
	metricsCollector := NewApplicationMetricsCollector(perfMonitor, aggregator, testLogger)
	sessionTracker := NewSessionTracker(testLogger)

	config := DefaultDashboardConfig()
	config.Enabled = false // Don't start server in tests
	config.Port = 9999     // Use different port for tests

	dashboard := NewDashboard(
		metricsCollector,
		perfMonitor,
		nil, // No telemetry
		sessionTracker,
		nil, // No debug manager
		testLogger,
	)

	dashboard.SetConfig(config)

	// Test data collection
	data := dashboard.collectDashboardData()
	if data.Title == "" {
		t.Error("Dashboard title should not be empty")
	}

	if data.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", data.Status)
	}

	// Note: We don't test the HTTP server to avoid port conflicts
}

// Helper types for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func TestDefaultConfigs(t *testing.T) {
	// Test default observability config
	obsConfig := DefaultObservabilityConfig()
	if !obsConfig.Enabled {
		t.Error("Default observability config should be enabled")
	}

	// Test default metrics config
	metricsConfig := DefaultMetricsConfig()
	if metricsConfig.CollectionInterval <= 0 {
		t.Error("Default metrics collection interval should be positive")
	}

	// Test default telemetry config
	telemetryConfig := DefaultTelemetryConfig()
	if telemetryConfig.Enabled {
		t.Error("Default telemetry should be disabled for privacy")
	}

	// Test default debug config
	debugConfig := DefaultDebugConfig()
	if debugConfig.Enabled {
		t.Error("Default debug config should be disabled")
	}

	// Test default dashboard config
	dashboardConfig := DefaultDashboardConfig()
	if dashboardConfig.Port <= 0 {
		t.Error("Default dashboard port should be positive")
	}
}

func TestIntegration(t *testing.T) {
	// Test full integration
	testLogger := logger.NewDefault()
	perfMonitor := performance.NewPerformanceMonitor(1*time.Second, testLogger)

	config := DefaultObservabilityConfig()
	config.Dashboard.Enabled = false
	config.Telemetry.Enabled = false
	config.Debug.Enabled = false
	config.FlushInterval = 100 * time.Millisecond

	manager := NewObservabilityManager(config, testLogger, perfMonitor)

	// Initialize and start
	err := manager.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	err = manager.Start()
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Record some activities
	for i := 0; i < 5; i++ {
		metadata := map[string]interface{}{
			"iteration": i,
			"test_data": "integration_test",
		}

		manager.RecordActivity(
			"integration_test",
			i%2 == 0, // Alternate success/failure
			time.Duration(i)*10*time.Millisecond,
			nil,
			metadata,
		)
	}

	// Wait for background processing with shorter timeout
	if !testing.Short() {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Wait for processing time
		select {
		case <-time.After(25 * time.Millisecond):
		case <-ctx.Done():
		}
	}

	// Verify status
	status := manager.GetOverallStatus()
	if !status["running"].(bool) {
		t.Error("Manager should be running")
	}

	// Clean shutdown
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop: %v", err)
	}

	status = manager.GetOverallStatus()
	if status["running"].(bool) {
		t.Error("Manager should not be running after stop")
	}
}
