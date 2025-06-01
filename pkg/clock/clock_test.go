package clock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealClock(t *testing.T) {
	clock := NewRealClock()

	start := clock.Now()
	clock.Sleep(10 * time.Millisecond)
	elapsed := clock.Since(start)

	assert.True(t, elapsed >= 10*time.Millisecond)
}

func TestFakeClockNow(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	assert.Equal(t, baseTime, clock.Now())

	clock.Advance(time.Hour)
	expected := baseTime.Add(time.Hour)
	assert.Equal(t, expected, clock.Now())
}

func TestFakeClockAfter(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	// Test immediate firing for zero duration
	ch := clock.After(0)
	select {
	case received := <-ch:
		assert.Equal(t, baseTime, received)
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Expected immediate firing")
	}

	// Test delayed firing
	ch = clock.After(time.Hour)

	// Should not fire immediately
	select {
	case <-ch:
		t.Fatal("Should not fire immediately")
	default:
		// Expected - channel should be empty
	}

	// Advance clock and verify firing
	clock.Advance(time.Hour)
	select {
	case received := <-ch:
		assert.Equal(t, baseTime.Add(time.Hour), received)
	default:
		t.Fatal("Expected channel to fire after advancing clock")
	}
}

func TestFakeClockTicker(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	ticker := clock.NewTicker(time.Hour)
	defer ticker.Stop()

	// Should not fire immediately
	select {
	case <-ticker.C():
		t.Fatal("Should not fire immediately")
	default:
		// Expected - ticker should not fire yet
	}

	// Advance and check first tick
	clock.Advance(time.Hour)
	select {
	case received := <-ticker.C():
		assert.Equal(t, baseTime.Add(time.Hour), received)
	default:
		t.Fatal("Expected first tick after advancing clock")
	}

	// Advance and check second tick
	clock.Advance(time.Hour)
	select {
	case received := <-ticker.C():
		assert.Equal(t, baseTime.Add(2*time.Hour), received)
	default:
		t.Fatal("Expected second tick after advancing clock")
	}
}

func TestFakeClockTickerStop(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	ticker := clock.NewTicker(time.Hour)
	ticker.Stop()

	// Advance time - should not fire since stopped
	clock.Advance(time.Hour)
	select {
	case <-ticker.C():
		t.Fatal("Should not fire after stop")
	default:
		// Expected - stopped ticker should not fire
	}
}

func TestFakeClockSleep(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	done := make(chan bool, 1)
	started := make(chan bool, 1)

	go func() {
		started <- true
		clock.Sleep(time.Hour)
		done <- true
	}()

	// Wait for goroutine to start
	<-started

	// Give it a moment to enter Sleep call
	time.Sleep(time.Millisecond)

	// Should not complete immediately
	select {
	case <-done:
		t.Fatal("Sleep should not complete immediately")
	default:
		// Expected - sleep should be blocked
	}

	// Advance clock and verify completion
	clock.Advance(time.Hour)

	// Wait for completion with timeout
	select {
	case <-done:
		// Expected - sleep should complete
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Sleep should complete after advancing clock")
	}
}

func TestFakeClockSinceUntil(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	pastTime := baseTime.Add(-time.Hour)
	futureTime := baseTime.Add(time.Hour)

	assert.Equal(t, time.Hour, clock.Since(pastTime))
	assert.Equal(t, time.Hour, clock.Until(futureTime))

	clock.Advance(30 * time.Minute)
	assert.Equal(t, 90*time.Minute, clock.Since(pastTime))
	assert.Equal(t, 30*time.Minute, clock.Until(futureTime))
}

func TestWithTimeout(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	ctx := context.Background()
	ctx, cancel := WithTimeout(ctx, clock, time.Hour)
	defer cancel()

	// Give the timeout goroutine a moment to start
	time.Sleep(time.Millisecond)

	// Should not timeout immediately
	select {
	case <-ctx.Done():
		t.Fatal("Should not timeout immediately")
	default:
		// Expected - context should not be canceled yet
	}

	// Advance clock to trigger timeout
	clock.Advance(time.Hour)

	// Wait for context to be canceled
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected timeout after advancing clock")
	}
}

// Example: Health check that would normally take 30 seconds, now instant
func TestHealthCheckExample(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	// Simplified health check test that doesn't depend on ticker behavior
	healthyForDuration := func(duration time.Duration) bool {
		deadline := clock.Now().Add(duration)
		checkCount := 0

		// Instead of using ticker, use simple clock advances for testing
		for checkCount < 5 { // Do a few checks
			checkCount++
			clock.Advance(duration / 5) // Advance time in steps
			if clock.Now().After(deadline) || clock.Now().Equal(deadline) {
				return true // Healthy for full duration
			}
		}
		return false
	}

	// Test with fake clock
	result := healthyForDuration(30 * time.Second)
	assert.True(t, result)
}

// Example: Rate limiter that would normally take minutes, now instant
func TestRateLimiterExample(t *testing.T) {
	baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(baseTime)

	// Simple rate limiter: 1 token per minute, max 5 tokens
	type rateLimiter struct {
		tokens     int
		maxTokens  int
		lastRefill time.Time
		refillRate time.Duration
	}

	limiter := &rateLimiter{
		tokens:     5,
		maxTokens:  5,
		lastRefill: clock.Now(),
		refillRate: time.Minute,
	}

	refillTokens := func() {
		now := clock.Now()
		elapsed := now.Sub(limiter.lastRefill)
		tokensToAdd := int(elapsed / limiter.refillRate)

		if tokensToAdd > 0 {
			limiter.tokens = min(limiter.maxTokens, limiter.tokens+tokensToAdd)
			limiter.lastRefill = now
		}
	}

	consume := func() bool {
		refillTokens()
		if limiter.tokens > 0 {
			limiter.tokens--
			return true
		}
		return false
	}

	// Consume all tokens
	for i := 0; i < 5; i++ {
		require.True(t, consume(), "Should have tokens initially")
	}

	// Should be empty now
	assert.False(t, consume(), "Should be out of tokens")

	// Advance 3 minutes, should get 3 tokens back
	clock.Advance(3 * time.Minute)

	for i := 0; i < 3; i++ {
		assert.True(t, consume(), "Should have refilled tokens")
	}

	assert.False(t, consume(), "Should be limited again")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
