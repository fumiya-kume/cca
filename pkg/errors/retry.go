// Package errors provides retry mechanisms with exponential backoff and jitter
// for resilient operation handling in ccAgents.
package errors

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts         int
	InitialInterval     time.Duration
	MaxInterval         time.Duration
	Multiplier          float64
	RandomizationFactor float64
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:         3,
		InitialInterval:     time.Second,
		MaxInterval:         30 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	}
}

// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// ShouldRetryFunc determines if an error should trigger a retry
type ShouldRetryFunc func(error) bool

// DefaultShouldRetry determines if an error should be retried based on its type and recoverability
func DefaultShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a ccAgentsError and if it's recoverable
	if ccErr, ok := err.(*ccAgentsError); ok {
		return ccErr.IsRecoverable()
	}

	// For non-ccAgentsError types, be conservative and don't retry
	return false
}

// NetworkShouldRetry determines if network errors should be retried
func NetworkShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Retry network errors and authentication errors
	if ccErr, ok := err.(*ccAgentsError); ok {
		return ccErr.Type() == ErrorTypeNetwork || ccErr.Type() == ErrorTypeAuthentication
	}

	return false
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, config RetryConfig, fn RetryableFunc, shouldRetry ShouldRetryFunc) error {
	return RetryWithClock(ctx, clock.NewRealClock(), config, fn, shouldRetry)
}

// RetryWithClock executes a function with retry logic using a custom clock
func RetryWithClock(ctx context.Context, clk clock.Clock, config RetryConfig, fn RetryableFunc, shouldRetry ShouldRetryFunc) error {
	if shouldRetry == nil {
		shouldRetry = DefaultShouldRetry
	}

	var lastErr error
	interval := config.InitialInterval

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry
		if !shouldRetry(err) {
			return err
		}

		// Don't wait after the last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Calculate next interval with jitter
		nextInterval := time.Duration(float64(interval) * config.Multiplier)
		if nextInterval > config.MaxInterval {
			nextInterval = config.MaxInterval
		}

		// Add randomization to prevent thundering herd using crypto/rand
		maxJitter := int64(float64(nextInterval) * config.RandomizationFactor)
		if maxJitter > 0 {
			jitterValue, err := rand.Int(rand.Reader, big.NewInt(maxJitter*2))
			if err == nil {
				jitter := time.Duration(jitterValue.Int64() - maxJitter)
				nextInterval += jitter
			}
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-clk.After(interval):
			// Continue to next attempt
		}

		interval = nextInterval
	}

	// All retries exhausted
	return NewError(ErrorTypeWorkflow).
		WithMessage("operation failed after maximum retry attempts").
		WithCause(lastErr).
		WithSeverity(SeverityHigh).
		WithContext("max_attempts", config.MaxAttempts).
		WithSuggestion("Check the underlying error cause").
		WithSuggestion("Consider increasing retry limits if appropriate").
		Build()
}

// RetryWithExponentialBackoff is a convenience function for exponential backoff retry
func RetryWithExponentialBackoff(ctx context.Context, maxAttempts int, fn RetryableFunc) error {
	config := RetryConfig{
		MaxAttempts:         maxAttempts,
		InitialInterval:     500 * time.Millisecond,
		MaxInterval:         30 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.1,
	}

	return Retry(ctx, config, fn, DefaultShouldRetry)
}

// RetryNetworkOperation is a convenience function for retrying network operations
func RetryNetworkOperation(ctx context.Context, fn RetryableFunc) error {
	config := RetryConfig{
		MaxAttempts:         5,
		InitialInterval:     time.Second,
		MaxInterval:         10 * time.Second,
		Multiplier:          1.5,
		RandomizationFactor: 0.2,
	}

	return Retry(ctx, config, fn, NetworkShouldRetry)
}
