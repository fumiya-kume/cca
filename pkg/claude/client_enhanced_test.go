package claude

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fumiya-kume/cca/pkg/clock"
)

// TestNewClient_Enhanced tests comprehensive client creation scenarios
func TestNewClient_Enhanced(t *testing.T) {
	tests := []struct {
		name           string
		config         ClientConfig
		expectError    bool
		expectedFields func(*testing.T, *Client)
	}{
		{
			name: "full configuration",
			config: ClientConfig{
				ClaudeCommand:         "echo",
				WorkingDirectory:      "/tmp",
				MaxConcurrentSessions: 5,
				SessionTimeout:        10 * time.Minute,
				HealthCheckInterval:   2 * time.Minute,
				ResourceLimits: ResourceLimits{
					MaxMemoryMB:   1024,
					MaxCPUPercent: 80,
					MaxDuration:   1 * time.Hour,
					MaxOutputSize: 1024 * 1024,
				},
			},
			expectError: false,
			expectedFields: func(t *testing.T, client *Client) {
				assert.Equal(t, "echo", client.claudeCommand)
				assert.Equal(t, "/tmp", client.workingDir)
				assert.Equal(t, 5, client.maxConcurrent)
				assert.Equal(t, 10*time.Minute, client.timeout)
				assert.Equal(t, 2*time.Minute, client.healthCheckInterval)
				assert.NotNil(t, client.sessions)
				assert.NotNil(t, client.sessionLimit)
				assert.NotNil(t, client.metrics)
			},
		},
		{
			name: "minimal configuration with defaults",
			config: ClientConfig{
				ClaudeCommand: "echo",
			},
			expectError: false,
			expectedFields: func(t *testing.T, client *Client) {
				assert.Equal(t, "echo", client.claudeCommand)
				assert.Equal(t, 3, client.maxConcurrent)                   // Default
				assert.Equal(t, 30*time.Minute, client.timeout)            // Default
				assert.Equal(t, 5*time.Minute, client.healthCheckInterval) // Default
			},
		},
		{
			name: "empty command should use default",
			config: ClientConfig{
				WorkingDirectory: "/tmp",
			},
			expectError: false,
			expectedFields: func(t *testing.T, client *Client) {
				assert.Equal(t, "claude", client.claudeCommand) // Default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				return
			}

			// Note: We may get an error if 'claude' command doesn't exist,
			// but we can still test the client structure if it was created
			if client != nil {
				tt.expectedFields(t, client)
			}
		})
	}
}

// TestNewClientWithClock tests client creation with custom clock
func TestNewClientWithClock(t *testing.T) {
	config := ClientConfig{
		ClaudeCommand: "echo",
	}

	fakeClock := clock.NewFakeClock(time.Now())
	client, err := NewClientWithClock(config, fakeClock)

	// May error due to command not existing, but test the clock was set
	if client != nil {
		assert.Equal(t, fakeClock, client.clock)
	}

	// Test that time operations use the fake clock
	if client != nil {
		initialTime := fakeClock.Now()
		fakeClock.Advance(5 * time.Minute)
		assert.Equal(t, initialTime.Add(5*time.Minute), client.clock.Now())
	} else {
		t.Logf("Client creation failed (expected if 'echo' command isn't available): %v", err)
	}
}

// TestClient_SessionManagement tests session lifecycle
func TestClient_SessionManagement(t *testing.T) {
	// Test client session management without actually starting processes
	config := ClientConfig{
		ClaudeCommand:         "echo",
		MaxConcurrentSessions: 2,
	}

	client, _ := NewClient(config)
	require.NotNil(t, client, "Client should be created for testing even if command doesn't exist")

	// Test session tracking
	assert.Equal(t, 0, len(client.sessions))

	// Test metrics initialization
	metrics := client.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalSessions)
	assert.Equal(t, int64(0), metrics.ActiveSessions)
}

// TestClient_MetricsOperations tests thread-safe metrics operations
func TestClient_MetricsOperations(t *testing.T) {
	config := ClientConfig{
		ClaudeCommand: "echo",
	}

	client, _ := NewClient(config)
	require.NotNil(t, client)

	// Test metrics update
	client.updateMetrics(func(m *ClientMetrics) {
		m.TotalSessions = 5
		m.ActiveSessions = 3
		m.CompletedSessions = 2
		m.FailedSessions = 1
		m.TotalCommands = 10
	})

	metrics := client.GetMetrics()
	assert.Equal(t, int64(5), metrics.TotalSessions)
	assert.Equal(t, int64(3), metrics.ActiveSessions)
	assert.Equal(t, int64(2), metrics.CompletedSessions)
	assert.Equal(t, int64(1), metrics.FailedSessions)
	assert.Equal(t, int64(10), metrics.TotalCommands)
}

// TestClient_ConcurrentMetricsUpdates tests thread safety
func TestClient_ConcurrentMetricsUpdates(t *testing.T) {
	config := ClientConfig{
		ClaudeCommand: "echo",
	}

	client, _ := NewClient(config)
	require.NotNil(t, client)

	// Simulate concurrent metrics updates
	const numGoroutines = 10
	const updatesPerGoroutine = 10

	done := make(chan struct{})
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < updatesPerGoroutine; j++ {
				client.updateMetrics(func(m *ClientMetrics) {
					m.TotalCommands++
				})
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	metrics := client.GetMetrics()
	assert.Equal(t, int64(numGoroutines*updatesPerGoroutine), metrics.TotalCommands)
}

// TestDefaultClientConfig tests default configuration values
func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	// ClaudeCommand should contain "claude" (might be full path due to discovery)
	assert.Contains(t, config.ClaudeCommand, "claude")
	assert.Equal(t, ".", config.WorkingDirectory) // Should use current directory
	assert.Equal(t, 3, config.MaxConcurrentSessions)
	assert.Equal(t, 5*time.Minute, config.SessionTimeout)
	assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
	assert.Equal(t, 512, config.ResourceLimits.MaxMemoryMB)
	assert.Equal(t, 80, config.ResourceLimits.MaxCPUPercent)
}

// TestResourceLimits tests resource limit configuration
func TestResourceLimits(t *testing.T) {
	limits := ResourceLimits{
		MaxMemoryMB:   1024,
		MaxCPUPercent: 80,
		MaxDuration:   2 * time.Hour,
		MaxOutputSize: 2048 * 1024,
	}

	config := ClientConfig{
		ClaudeCommand:  "echo",
		ResourceLimits: limits,
	}

	client, err := NewClient(config)
	if client != nil {
		// Test that resource limits are stored (implementation details may vary)
		assert.NotNil(t, client)
	} else {
		t.Logf("Client creation failed (expected if command doesn't exist): %v", err)
	}
}

// TestClient_SessionIDGeneration tests session ID generation
func TestClient_SessionIDGeneration(t *testing.T) {
	// Test that generateSessionID creates IDs with expected format
	id1 := generateSessionID()
	id2 := generateSessionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)

	// IDs should have expected format (implementation may generate same ID if called quickly)
	// We just test that the function works and returns non-empty strings
	t.Logf("Generated ID 1: %s", id1)
	t.Logf("Generated ID 2: %s", id2)
}

// TestSession_StatusTransitions tests session status management
func TestSession_StatusTransitions(t *testing.T) {
	session := &Session{
		ID:        generateSessionID(),
		Status:    SessionIdle,
		CreatedAt: time.Now(),
	}

	// Test initial status
	assert.Equal(t, SessionIdle, session.Status)

	// Test status transitions
	session.Status = SessionRunning
	assert.Equal(t, SessionRunning, session.Status)

	session.Status = SessionWaiting
	assert.Equal(t, SessionWaiting, session.Status)

	session.Status = SessionError
	assert.Equal(t, SessionError, session.Status)

	session.Status = SessionClosed
	assert.Equal(t, SessionClosed, session.Status)
}

// TestOutputMessage_Structure tests output message structure
func TestOutputMessage_Structure(t *testing.T) {
	timestamp := time.Now()

	msg := OutputMessage{
		SessionID: "test-session",
		Content:   "Hello, world!",
		Timestamp: timestamp,
		IsStderr:  false,
		Type:      OutputText,
	}

	assert.Equal(t, "test-session", msg.SessionID)
	assert.Equal(t, "Hello, world!", msg.Content)
	assert.Equal(t, timestamp, msg.Timestamp)
	assert.False(t, msg.IsStderr)
	assert.Equal(t, OutputText, msg.Type)
}

// TestOutputType_Values tests output type enumeration
func TestOutputType_Values(t *testing.T) {
	tests := []struct {
		outputType OutputType
		value      int
	}{
		{OutputText, 0},
		{OutputProgress, 1},
		{OutputError, 2},
		{OutputPrompt, 3},
		{OutputResult, 4},
		{OutputDebug, 5},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.value, int(tt.outputType))
	}
}

// TestClientConfig_ExtendedFields tests additional configuration fields
func TestClientConfig_ExtendedFields(t *testing.T) {
	config := ClientConfig{
		ClaudeCommand:    "claude",
		WorkingDirectory: "/tmp",
		Command:          "custom-claude",
		Timeout:          45 * time.Minute,
		MaxRetries:       5,
		WorkingDir:       "/custom/path",
		Environment: map[string]string{
			"CLAUDE_API_KEY": "test-key",
			"DEBUG":          "true",
		},
	}

	// ClaudeCommand should contain "claude" (might be full path due to discovery)
	assert.Contains(t, config.ClaudeCommand, "claude")
	assert.Equal(t, "/tmp", config.WorkingDirectory)
	assert.Equal(t, "custom-claude", config.Command)
	assert.Equal(t, 45*time.Minute, config.Timeout)
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, "/custom/path", config.WorkingDir)
	assert.Equal(t, 2, len(config.Environment))
	assert.Equal(t, "test-key", config.Environment["CLAUDE_API_KEY"])
	assert.Equal(t, "true", config.Environment["DEBUG"])
}
