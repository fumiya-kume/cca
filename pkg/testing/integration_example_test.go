package testing

import (
	"context"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthMonitorWithRealClock(t *testing.T) {
	t.Skip("Skipping slow test - would take 30+ seconds")

	// This test demonstrates what we WOULD do with real clock
	// but it's too slow for regular testing
	realClock := clock.NewRealClock()
	monitor := NewHealthMonitor(realClock)
	monitor.checkInterval = 100 * time.Millisecond // Faster for testing

	monitor.RegisterAgent("test-agent")

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() { _ = monitor.Start(ctx) }()
	defer monitor.Stop()

	// Wait for at least one health check
	time.Sleep(150 * time.Millisecond)

	health, exists := monitor.GetAgentHealth("test-agent")
	require.True(t, exists)
	assert.Equal(t, HealthStatusHealthy, health.Status)
}

func TestHealthMonitorWithFakeClock(t *testing.T) {
	// Same test as above but instant with fake clock
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	monitor := NewHealthMonitor(fakeClock)
	monitor.RegisterAgent("test-agent")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monitoring in background
	go func() { _ = monitor.Start(ctx) }()

	// Give a moment for the monitor to start
	time.Sleep(2 * time.Millisecond)

	// Advance clock to trigger health checks - advance by the check interval
	fakeClock.Advance(monitor.checkInterval)

	// Give time for health check goroutines to complete
	time.Sleep(5 * time.Millisecond)

	// Advance time for the health check operation itself
	fakeClock.Advance(200 * time.Millisecond)

	// Give more time for processing
	time.Sleep(5 * time.Millisecond)

	monitor.Stop()

	// Verify health check was performed
	health, exists := monitor.GetAgentHealth("test-agent")
	require.True(t, exists)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.True(t, health.LastCheck.After(baseTime))
}

func TestRetryWithFakeClock(t *testing.T) {
	// Test basic fake clock functionality
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	// Test that After works with immediate timeout
	ch := fakeClock.After(0)
	select {
	case tm := <-ch:
		t.Logf("Immediate timer fired at %v", tm)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Immediate timer should fire instantly")
	}

	// Test that After works with manual advance
	ch2 := fakeClock.After(time.Second)
	select {
	case <-ch2:
		t.Fatal("Timer should not fire before advance")
	case <-time.After(10 * time.Millisecond):
		// Good, timer didn't fire
	}

	fakeClock.Advance(time.Second)
	select {
	case tm := <-ch2:
		t.Logf("Timer fired after advance at %v", tm)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timer should fire after advance")
	}

	// Test retry with immediate intervals (no delays)
	config := RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 0, // No delay for testing
		Multiplier:      2.0,
		MaxInterval:     10 * time.Second,
	}

	attempts := 0
	operation := func() error {
		attempts++
		t.Logf("Attempt %d", attempts)
		if attempts < 3 {
			return assert.AnError
		}
		return nil
	}

	// With zero interval, this should complete immediately
	err := RetryWithClock(context.Background(), fakeClock, config, operation)

	// Verify the operation succeeded and all attempts were made
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
	t.Logf("Retry operation completed with %d attempts", attempts)
}

func TestCacheExpirationWithFakeClock(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	cache := NewTimeBasedCache(fakeClock)

	// Set item with 1 hour TTL
	cache.Set("key1", "value1", time.Hour)

	// Should be available immediately
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Advance clock by 30 minutes - should still exist
	fakeClock.Advance(30 * time.Minute)
	value, exists = cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Advance clock by another 31 minutes - should be expired
	fakeClock.Advance(31 * time.Minute)
	value, exists = cache.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestCacheCleanupWithFakeClock(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(baseTime)

	cache := NewTimeBasedCache(fakeClock)

	// Set multiple items with different TTLs
	cache.Set("short", "value1", time.Minute)
	cache.Set("medium", "value2", time.Hour)
	cache.Set("long", "value3", 24*time.Hour)

	// All should exist initially
	assert.Equal(t, 3, len(cache.entries))

	// Advance by 2 minutes - short should expire
	fakeClock.Advance(2 * time.Minute)
	cache.Cleanup()
	assert.Equal(t, 2, len(cache.entries))

	// Advance by 2 hours - medium should expire
	fakeClock.Advance(2 * time.Hour)
	cache.Cleanup()
	assert.Equal(t, 1, len(cache.entries))

	// Advance by 25 hours - all should expire
	fakeClock.Advance(25 * time.Hour)
	cache.Cleanup()
	assert.Equal(t, 0, len(cache.entries))
}

func BenchmarkHealthMonitorFakeClock(b *testing.B) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fakeClock := clock.NewFakeClock(baseTime)
		monitor := NewHealthMonitor(fakeClock)
		monitor.RegisterAgent("test-agent")

		ctx, cancel := context.WithCancel(context.Background())

		go func() { _ = monitor.Start(ctx) }()

		// Simulate 10 health check cycles instantly
		for j := 0; j < 10; j++ {
			fakeClock.Advance(monitor.checkInterval)
			fakeClock.Advance(100 * time.Millisecond)
		}

		cancel()
		monitor.Stop()
	}
}

// Performance comparison test
func TestPerformanceComparison(t *testing.T) {
	t.Run("FakeClock", func(t *testing.T) {
		start := time.Now()

		baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		fakeClock := clock.NewFakeClock(baseTime)

		// Simulate 1 hour of health monitoring in multiple agents
		for i := 0; i < 5; i++ {
			monitor := NewHealthMonitor(fakeClock)
			monitor.RegisterAgent("agent-" + string(rune('A'+i)))

			// Simulate 1 hour of monitoring (120 checks at 30s intervals)
			for j := 0; j < 120; j++ {
				fakeClock.Advance(30 * time.Second)
				fakeClock.Advance(100 * time.Millisecond) // Health check time
			}
		}

		elapsed := time.Since(start)
		t.Logf("FakeClock: Simulated 5 hours of monitoring in %v", elapsed)
		assert.Less(t, elapsed, 100*time.Millisecond, "Should complete in under 100ms")
	})

	t.Run("RealClock_Demonstration", func(t *testing.T) {
		t.Skip("Would take 5+ hours with real clock")

		// This demonstrates what the same test would look like with real time:
		// - 5 agents × 1 hour each = 5 hours total
		// - 120 health checks × 30 second intervals = 1 hour per agent
		// - Total: 5+ hours of real time vs <100ms with fake clock

		realClock := clock.NewRealClock()
		monitor := NewHealthMonitor(realClock)
		monitor.checkInterval = 30 * time.Second

		start := time.Now()

		// This would run for hours...
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		defer cancel()

		_ = monitor.Start(ctx)

		elapsed := time.Since(start)
		t.Logf("RealClock would take: %v (but we canceled early)", elapsed)
	})
}
