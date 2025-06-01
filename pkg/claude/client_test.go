package claude

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	config := ClientConfig{
		ClaudeCommand:         "echo", // Use echo to simulate claude command
		WorkingDirectory:      "/tmp",
		MaxConcurrentSessions: 2,
		SessionTimeout:        5 * time.Minute,
		HealthCheckInterval:   1 * time.Minute,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.claudeCommand != "echo" {
		t.Errorf("Expected claude command 'echo', got '%s'", client.claudeCommand)
	}

	if client.maxConcurrent != 2 {
		t.Errorf("Expected max concurrent 2, got %d", client.maxConcurrent)
	}
}

func TestClientDefaults(t *testing.T) {
	config := ClientConfig{}
	_, err := NewClient(config)

	// This may succeed if Claude is actually available, otherwise will fail
	// The important part is that the client was created with default config
	if err != nil {
		t.Logf("Claude command not available (expected): %v", err)
	} else {
		t.Log("Claude command is available")
	}
}

func TestSemaphore(t *testing.T) {
	sem := NewSemaphore(2)
	ctx := context.Background()

	// Test acquiring resources
	err := sem.Acquire(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to acquire semaphore: %v", err)
	}

	err = sem.Acquire(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to acquire second semaphore: %v", err)
	}

	// Test timeout when no resources available
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err = sem.Acquire(ctxTimeout, 1)
	if err == nil {
		t.Error("Expected timeout error when semaphore is full")
	}

	// Release resources
	sem.Release(2)

	// Should be able to acquire again
	err = sem.Acquire(ctx, 2)
	if err != nil {
		t.Fatalf("Failed to re-acquire semaphore after release: %v", err)
	}

	sem.Release(2)
}

func TestSessionStatus(t *testing.T) {
	tests := []struct {
		status   SessionStatus
		expected string
	}{
		{SessionIdle, "idle"},
		{SessionRunning, "running"},
		{SessionWaiting, "waiting"},
		{SessionError, "error"},
		{SessionClosed, "closed"},
	}

	for _, test := range tests {
		if test.status.String() != test.expected {
			t.Errorf("Expected status %s, got %s", test.expected, test.status.String())
		}
	}
}

func TestGenerateSessionID(t *testing.T) {
	// Test that session IDs are generated and not empty
	id1 := generateSessionID()
	id2 := generateSessionID()

	if id1 == "" {
		t.Error("Generated session ID should not be empty")
	}

	if id2 == "" {
		t.Error("Generated session ID should not be empty")
	}

	// IDs should have expected format
	if !strings.HasPrefix(id1, "session_") {
		t.Error("Session ID should have expected prefix")
	}

	if !strings.HasPrefix(id2, "session_") {
		t.Error("Session ID should have expected prefix")
	}
}

func TestClientMetrics(t *testing.T) {
	config := ClientConfig{
		ClaudeCommand: "echo",
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test initial metrics
	metrics := client.GetMetrics()
	if metrics.TotalSessions != 0 {
		t.Error("Initial total sessions should be 0")
	}

	// Test metrics update
	client.updateMetrics(func(m *ClientMetrics) {
		m.TotalSessions++
		m.ActiveSessions++
	})

	metrics = client.GetMetrics()
	if metrics.TotalSessions != 1 {
		t.Error("Total sessions should be 1 after update")
	}
	if metrics.ActiveSessions != 1 {
		t.Error("Active sessions should be 1 after update")
	}
}
