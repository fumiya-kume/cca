package agents

import (
	"context"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/fumiya-kume/cca/pkg/clock"
	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthMonitorWithFakeClockIntegration(t *testing.T) {
	// Create fake clock starting at a known time
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	// Create test dependencies
	messageBus := NewMessageBus(100)
	registry := NewAgentRegistry(messageBus)

	// Create health monitor with fake clock
	monitor := NewHealthMonitorWithClock(registry, messageBus, fakeClock)
	monitor.checkInterval = 30 * time.Second // 30 second intervals

	// Register a mock agent
	mockAgent := &MockAgent{id: "test-agent"}
	config := AgentConfig{Enabled: true}
	factory := func(config AgentConfig) (Agent, error) {
		return mockAgent, nil
	}
	err := registry.RegisterFactory("test-agent", factory)
	require.NoError(t, err)
	err = registry.CreateAgent("test-agent", config)
	require.NoError(t, err)

	// Start monitoring in background

	go func() { _ = monitor.Start() }()
	defer func() { _ = monitor.Stop() }()

	// Give a moment for monitor to start
	time.Sleep(2 * time.Millisecond)

	// Advance clock to trigger first health check
	fakeClock.Advance(30 * time.Second)
	time.Sleep(5 * time.Millisecond) // Let goroutines process

	// Verify health check was performed
	status, err := monitor.GetHealthStatus("test-agent")
	require.NoError(t, err)
	assert.Equal(t, HealthHealthy, status.Status)
	assert.True(t, status.LastCheck.After(baseTime))

	// Test that advancing clock triggers more health checks instantly
	start := time.Now()

	// Advance by 5 intervals (normally 2.5 minutes)
	for i := 0; i < 5; i++ {
		fakeClock.Advance(30 * time.Second)
		time.Sleep(2 * time.Millisecond) // Let processing happen
	}

	elapsed := time.Since(start)

	// Should complete in well under a second with fake clock
	assert.Less(t, elapsed, 100*time.Millisecond, "Health checks should be instant with fake clock")

	// Verify final health status
	status, err = monitor.GetHealthStatus("test-agent")
	require.NoError(t, err)
	assert.Equal(t, HealthHealthy, status.Status)
}

func TestHealthMonitorPerformanceComparison(t *testing.T) {
	t.Run("FakeClock", func(t *testing.T) {
		start := time.Now()

		baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		fakeClock := clock.NewFakeClock(baseTime)

		// Create components
		messageBus := NewMessageBus(100)
		registry := NewAgentRegistry(messageBus)
		monitor := NewHealthMonitorWithClock(registry, messageBus, fakeClock)
		monitor.checkInterval = 30 * time.Second

		// Register agent
		mockAgent := &MockAgent{id: "perf-test-agent"}
		config := AgentConfig{Enabled: true}
		factory := func(config AgentConfig) (Agent, error) {
			return mockAgent, nil
		}
		_ = registry.RegisterFactory("perf-test-agent", factory)
		_ = registry.CreateAgent("perf-test-agent", config)

		go func() { _ = monitor.Start() }()
		defer func() { _ = monitor.Stop() }()

		time.Sleep(2 * time.Millisecond)

		// Simulate 1 hour of health monitoring (120 checks)
		for i := 0; i < 120; i++ {
			fakeClock.Advance(30 * time.Second)
			// Small delay to let goroutine process
			if i%10 == 0 {
				time.Sleep(time.Millisecond)
			}
		}

		elapsed := time.Since(start)
		t.Logf("FakeClock: Simulated 1 hour of health monitoring in %v", elapsed)

		// Should complete in well under 1 second
		assert.Less(t, elapsed, time.Second, "Should complete quickly with fake clock")

		// Verify health checks were performed
		status, err := monitor.GetHealthStatus("perf-test-agent")
		require.NoError(t, err)
		assert.Equal(t, HealthHealthy, status.Status)
	})

	t.Run("RealClock_Demonstration", func(t *testing.T) {
		t.Skip("Would take 1+ hour with real clock")

		// This demonstrates what the same test would look like with real time:
		// 120 health checks × 30 second intervals = 1 hour of real time
		// vs <1 second with fake clock = 3600x speedup!
	})
}

func TestRateLimiterWithFakeClockIntegration(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	// Create rate limiter: 5 requests per minute
	limiter := github.NewRateLimiterWithClock(5, time.Minute, fakeClock)

	// Should have 5 tokens initially
	assert.Equal(t, 5, limiter.GetAvailableTokens())

	// Consume all tokens
	for i := 0; i < 5; i++ {
		assert.True(t, limiter.TryTakeToken(), "Should have token %d", i)
	}

	// Should be empty now
	assert.False(t, limiter.TryTakeToken(), "Should be out of tokens")
	assert.Equal(t, 0, limiter.GetAvailableTokens())

	// Advance clock by 30 seconds - should refill 2.5 tokens (2 actual)
	fakeClock.Advance(30 * time.Second)
	assert.Equal(t, 2, limiter.GetAvailableTokens())

	// Advance clock by full minute - should refill to max
	fakeClock.Advance(time.Minute)
	assert.Equal(t, 5, limiter.GetAvailableTokens())
}

func TestAnalysisCacheWithFakeClockIntegration(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	// Create cache with 1 hour expiration
	cache := analysis.NewAnalysisCacheWithClock(time.Hour, fakeClock)

	// Create dummy analysis result
	result := &analysis.AnalysisResult{
		Languages:    []analysis.LanguageInfo{{Name: "go", FileCount: 10}},
		Frameworks:   []analysis.FrameworkInfo{{Name: "none", Version: ""}},
		AnalysisTime: time.Millisecond,
	}

	// Set cache entry
	cache.Set("/test/project", result)

	// Should be available immediately
	cached := cache.Get("/test/project")
	assert.NotNil(t, cached)
	assert.Equal(t, "go", cached.Languages[0].Name)

	// Advance by 30 minutes - should still be valid
	fakeClock.Advance(30 * time.Minute)
	cached = cache.Get("/test/project")
	assert.NotNil(t, cached, "Should still be cached after 30 minutes")

	// Advance by another 31 minutes - should be expired
	fakeClock.Advance(31 * time.Minute)
	cached = cache.Get("/test/project")
	assert.Nil(t, cached, "Should be expired after 61 minutes")
}

func TestRetryWithFakeClockIntegration(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	config := errors.RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 0, // No delay for testing
		Multiplier:      2.0,
		MaxInterval:     10 * time.Second,
	}

	attempts := 0
	operation := func() error {
		attempts++
		if attempts < 3 {
			// Return a retryable error (network error)
			return errors.NewError(errors.ErrorTypeNetwork).
				WithMessage("simulated network error for testing").
				Build()
		}
		return nil
	}

	ctx := context.Background()

	// With zero interval, this should complete immediately
	err := errors.RetryWithClock(ctx, fakeClock, config, operation, errors.NetworkShouldRetry)
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

// MockAgent for testing
type MockAgent struct {
	id AgentID
}

func (m *MockAgent) GetID() AgentID                                              { return m.id }
func (m *MockAgent) GetStatus() AgentStatus                                      { return StatusIdle }
func (m *MockAgent) Start(ctx context.Context) error                             { return nil }
func (m *MockAgent) Stop(ctx context.Context) error                              { return nil }
func (m *MockAgent) ProcessMessage(ctx context.Context, msg *AgentMessage) error { return nil }
func (m *MockAgent) GetCapabilities() []string                                   { return []string{"test"} }
func (m *MockAgent) HealthCheck(ctx context.Context) error                       { return nil }
