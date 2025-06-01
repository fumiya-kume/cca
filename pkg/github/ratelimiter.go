package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
)

// RateLimiter implements a token bucket rate limiter for GitHub API calls
type RateLimiter struct {
	clock      clock.Clock
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// maxRequests: maximum number of requests allowed
// window: time window for the rate limit (e.g., time.Hour for hourly limit)
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	return NewRateLimiterWithClock(maxRequests, window, clock.NewRealClock())
}

// NewRateLimiterWithClock creates a new rate limiter with a custom clock
func NewRateLimiterWithClock(maxRequests int, window time.Duration, clk clock.Clock) *RateLimiter {
	return &RateLimiter{
		clock:      clk,
		tokens:     maxRequests,
		maxTokens:  maxRequests,
		refillRate: window / time.Duration(maxRequests),
		lastRefill: clk.Now(),
	}
}

// Wait blocks until a token is available or context is canceled
func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.tryTakeToken() {
			return nil
		}

		// Wait for next refill or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.clock.After(r.refillRate):
			// Continue loop to try again
		}
	}
}

// TryTakeToken attempts to take a token without blocking
func (r *RateLimiter) TryTakeToken() bool {
	return r.tryTakeToken()
}

// tryTakeToken attempts to take a token and refills if needed
func (r *RateLimiter) tryTakeToken() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Refill tokens based on elapsed time
	now := r.clock.Now()
	elapsed := r.clock.Since(r.lastRefill)
	tokensToAdd := int(elapsed / r.refillRate)

	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = now
	}

	// Try to take a token
	if r.tokens > 0 {
		r.tokens--
		return true
	}

	return false
}

// GetAvailableTokens returns the current number of available tokens
func (r *RateLimiter) GetAvailableTokens() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Refill tokens based on elapsed time
	now := r.clock.Now()
	elapsed := r.clock.Since(r.lastRefill)
	tokensToAdd := int(elapsed / r.refillRate)

	if tokensToAdd > 0 {
		r.tokens += tokensToAdd
		if r.tokens > r.maxTokens {
			r.tokens = r.maxTokens
		}
		r.lastRefill = now
	}

	return r.tokens
}

// GetTimeUntilNextToken returns the duration until the next token is available
func (r *RateLimiter) GetTimeUntilNextToken() time.Duration {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.tokens > 0 {
		return 0
	}

	return r.refillRate - r.clock.Since(r.lastRefill)
}

// Reset resets the rate limiter to its initial state
func (r *RateLimiter) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.tokens = r.maxTokens
	r.lastRefill = r.clock.Now()
}

// String returns a string representation of the rate limiter status
func (r *RateLimiter) String() string {
	return fmt.Sprintf("RateLimiter{tokens: %d/%d, nextRefill: %v}",
		r.GetAvailableTokens(), r.maxTokens, r.GetTimeUntilNextToken())
}
