package github

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_NewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(100, time.Hour)

	if rl.maxTokens != 100 {
		t.Errorf("expected maxTokens to be 100, got %d", rl.maxTokens)
	}

	if rl.tokens != 100 {
		t.Errorf("expected initial tokens to be 100, got %d", rl.tokens)
	}

	expectedRefillRate := time.Hour / 100
	if rl.refillRate != expectedRefillRate {
		t.Errorf("expected refillRate to be %v, got %v", expectedRefillRate, rl.refillRate)
	}
}

func TestRateLimiter_TryTakeToken(t *testing.T) {
	rl := NewRateLimiter(2, time.Hour)

	// Should be able to take tokens initially
	if !rl.TryTakeToken() {
		t.Error("expected to be able to take first token")
	}

	if !rl.TryTakeToken() {
		t.Error("expected to be able to take second token")
	}

	// Should not be able to take more tokens
	if rl.TryTakeToken() {
		t.Error("expected to not be able to take third token")
	}

	// Available tokens should be 0
	if tokens := rl.GetAvailableTokens(); tokens != 0 {
		t.Errorf("expected 0 available tokens, got %d", tokens)
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(1, 100*time.Millisecond)

	// Take the only token
	if !rl.TryTakeToken() {
		t.Error("expected to be able to take initial token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected Wait to succeed, got error: %v", err)
	}

	// Should have waited approximately 100ms for token refill
	if elapsed < 90*time.Millisecond || elapsed > 150*time.Millisecond {
		t.Errorf("expected to wait ~100ms, waited %v", elapsed)
	}
}

func TestRateLimiter_WaitWithCancellation(t *testing.T) {
	rl := NewRateLimiter(1, time.Hour) // Very slow refill

	// Take the only token
	if !rl.TryTakeToken() {
		t.Error("expected to be able to take initial token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected Wait to fail due to context cancellation")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}

	// Should have waited approximately 50ms before cancellation
	if elapsed < 40*time.Millisecond || elapsed > 70*time.Millisecond {
		t.Errorf("expected to wait ~50ms before cancellation, waited %v", elapsed)
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping time-dependent test in short mode")
	}

	rl := NewRateLimiter(10, 50*time.Millisecond) // Reduced from 100ms

	// Take all tokens
	for i := 0; i < 10; i++ {
		if !rl.TryTakeToken() {
			t.Errorf("expected to be able to take token %d", i+1)
		}
	}

	// Should have no tokens left
	if tokens := rl.GetAvailableTokens(); tokens != 0 {
		t.Errorf("expected 0 tokens after taking all, got %d", tokens)
	}

	// Wait for refill period with ticker polling
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tokens := rl.GetAvailableTokens()
			if tokens >= 8 {
				return // Success - tokens have been refilled
			}
		case <-ctx.Done():
			goto checkTokens
		}
	}

checkTokens:

	// Final check after timeout
	tokens := rl.GetAvailableTokens()
	if tokens < 8 {
		t.Errorf("expected at least 8 tokens after refill period, got %d", tokens)
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(5, time.Hour)

	// Take some tokens
	rl.TryTakeToken()
	rl.TryTakeToken()
	rl.TryTakeToken()

	if tokens := rl.GetAvailableTokens(); tokens != 2 {
		t.Errorf("expected 2 tokens after taking 3, got %d", tokens)
	}

	// Reset the rate limiter
	rl.Reset()

	// Should have all tokens back
	if tokens := rl.GetAvailableTokens(); tokens != 5 {
		t.Errorf("expected 5 tokens after reset, got %d", tokens)
	}
}

func TestRateLimiter_GetTimeUntilNextToken(t *testing.T) {
	rl := NewRateLimiter(1, 100*time.Millisecond)

	// Initially should have no wait time
	if wait := rl.GetTimeUntilNextToken(); wait != 0 {
		t.Errorf("expected 0 wait time initially, got %v", wait)
	}

	// Take the token
	rl.TryTakeToken()

	// Should now have wait time
	wait := rl.GetTimeUntilNextToken()
	if wait <= 0 || wait > 100*time.Millisecond {
		t.Errorf("expected wait time between 0 and 100ms, got %v", wait)
	}
}

func TestRateLimiter_String(t *testing.T) {
	rl := NewRateLimiter(10, time.Hour)

	str := rl.String()
	if str == "" {
		t.Error("expected non-empty string representation")
	}

	// Should contain token information
	if !contains(str, "10") {
		t.Errorf("expected string to contain token count, got: %s", str)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) != -1
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
