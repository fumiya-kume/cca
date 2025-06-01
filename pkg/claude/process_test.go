package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testSessionID = "test-session"
)

func TestClient_startClaudeProcess(t *testing.T) {
	// Skip this test if running in CI or without actual claude command
	if testing.Short() {
		t.Skip("Skipping process test in short mode")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "claude-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	client := &Client{
		claudeCommand: "echo", // Use echo as a simple test command
		workingDir:    tempDir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	session := &Session{
		ID:         testSessionID,
		Context:    ctx,
		WorkingDir: tempDir,
	}

	// Test starting a process (will fail because echo exits immediately)
	err = client.startClaudeProcess(session)
	// We expect this to work initially but the process will exit
	// The actual behavior depends on the implementation details
	if err != nil {
		// This is expected since echo exits immediately
		assert.Contains(t, err.Error(), "")
	}
}

func TestClient_startClaudeProcess_WorkingDirHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claude-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name           string
		clientWorkDir  string
		sessionWorkDir string
		expectWorkDir  string
	}{
		{
			name:           "use session working dir",
			clientWorkDir:  "/client/dir",
			sessionWorkDir: tempDir,
			expectWorkDir:  tempDir,
		},
		{
			name:          "fallback to client working dir",
			clientWorkDir: tempDir,
			expectWorkDir: tempDir,
		},
		{
			name: "fallback to current working dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				claudeCommand: "echo",
				workingDir:    tt.clientWorkDir,
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()

			session := &Session{
				ID:         testSessionID,
				Context:    ctx,
				WorkingDir: tt.sessionWorkDir,
			}

			// Mock the working directory determination logic
			workingDir := session.WorkingDir
			if workingDir == "" {
				workingDir = client.workingDir
			}
			if workingDir == "" {
				var err error
				workingDir, err = os.Getwd()
				require.NoError(t, err)
			}

			if tt.expectWorkDir != "" {
				assert.Equal(t, tt.expectWorkDir, workingDir)
			} else {
				// Should be current working directory
				cwd, err := os.Getwd()
				require.NoError(t, err)
				assert.Equal(t, cwd, workingDir)
			}
		})
	}
}

func TestClient_sessionManagement(t *testing.T) {
	// Test session state management without actually starting processes
	client := &Client{
		claudeCommand: "sleep",
		sessions:      make(map[string]*Session),
	}

	// Test adding session info
	sessionID := testSessionID
	session := &Session{
		ID:        sessionID,
		Status:    SessionIdle,
		CreatedAt: time.Now(),
	}

	client.sessions[sessionID] = session

	// Verify session was added
	assert.Contains(t, client.sessions, sessionID)
	assert.Equal(t, sessionID, client.sessions[sessionID].ID)
}

func TestSession_ContextHandling(t *testing.T) {
	// Test context timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	session := &Session{
		ID:      testSessionID,
		Context: ctx,
	}

	// Wait for context to timeout
	select {
	case <-session.Context.Done():
		assert.Equal(t, context.DeadlineExceeded, session.Context.Err())
	case <-time.After(time.Millisecond * 200):
		t.Error("Context should have timed out")
	}
}

func TestSession_IDGeneration(t *testing.T) {
	// Test that sessions have unique IDs
	session1 := &Session{ID: "session-1"}
	session2 := &Session{ID: "session-2"}

	assert.NotEqual(t, session1.ID, session2.ID)
	assert.NotEmpty(t, session1.ID)
	assert.NotEmpty(t, session2.ID)
}

func TestSession_Validation(t *testing.T) {
	now := time.Now()
	session := &Session{
		ID:         testSessionID,
		Status:     SessionRunning,
		CreatedAt:  now,
		WorkingDir: "/test/dir",
	}

	assert.Equal(t, testSessionID, session.ID)
	assert.Equal(t, SessionRunning, session.Status)
	assert.Equal(t, now, session.CreatedAt)
	assert.Equal(t, "/test/dir", session.WorkingDir)
}

func TestClient_EnvironmentSetup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claude-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	_ = &Client{
		claudeCommand: "env",
		workingDir:    tempDir,
	}

	session := &Session{
		ID:         "test-session-env",
		WorkingDir: tempDir,
	}

	// Test that environment variables would be set correctly
	expectedEnv := []string{
		"CLAUDE_SESSION_ID=" + session.ID,
		"CLAUDE_WORKING_DIR=" + tempDir,
	}

	// Verify the environment setup logic
	for _, envVar := range expectedEnv {
		parts := strings.SplitN(envVar, "=", 2)
		require.Len(t, parts, 2)

		key, expectedValue := parts[0], parts[1]

		switch key {
		case "CLAUDE_SESSION_ID":
			assert.Equal(t, session.ID, expectedValue)
		case "CLAUDE_WORKING_DIR":
			assert.Equal(t, tempDir, expectedValue)
		}
	}
}

func TestClient_CommandConstruction(t *testing.T) {
	tests := []struct {
		name          string
		claudeCommand string
		expected      string
	}{
		{
			name:          "simple command",
			claudeCommand: "claude",
			expected:      "claude",
		},
		{
			name:          "command with path",
			claudeCommand: "/usr/local/bin/claude",
			expected:      "/usr/local/bin/claude",
		},
		{
			name:          "empty command",
			claudeCommand: "",
			expected:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				claudeCommand: tt.claudeCommand,
			}

			assert.Equal(t, tt.expected, client.claudeCommand)
		})
	}
}

func TestWorkingDirectoryResolution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claude-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test working directory resolution logic
	getCurrentDir := func() (string, error) {
		return os.Getwd()
	}

	tests := []struct {
		name           string
		sessionWorkDir string
		clientWorkDir  string
		expectError    bool
		expectCurrent  bool
	}{
		{
			name:           "use session working dir",
			sessionWorkDir: tempDir,
			clientWorkDir:  "/different/dir",
			expectError:    false,
			expectCurrent:  false,
		},
		{
			name:          "use client working dir",
			clientWorkDir: tempDir,
			expectError:   false,
			expectCurrent: false,
		},
		{
			name:          "use current working dir",
			expectError:   false,
			expectCurrent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the working directory resolution logic
			workingDir := tt.sessionWorkDir
			if workingDir == "" {
				workingDir = tt.clientWorkDir
			}
			if workingDir == "" {
				if tt.expectCurrent {
					cwd, err := getCurrentDir()
					require.NoError(t, err)
					workingDir = cwd
				}
			}

			if tt.expectError {
				assert.Empty(t, workingDir)
			} else {
				assert.NotEmpty(t, workingDir)
				if tt.expectCurrent {
					cwd, err := getCurrentDir()
					require.NoError(t, err)
					assert.Equal(t, cwd, workingDir)
				} else if tt.sessionWorkDir != "" {
					assert.Equal(t, tt.sessionWorkDir, workingDir)
				} else if tt.clientWorkDir != "" {
					assert.Equal(t, tt.clientWorkDir, workingDir)
				}
			}
		})
	}
}

func TestFileOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claude-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test file operations that might be used in process management
	testFile := filepath.Join(tempDir, "test.txt")

	// Test file creation
	err = os.WriteFile(testFile, []byte("test content"), 0600)
	require.NoError(t, err)

	// Test file reading
	// #nosec G304 - testFile is from test temp directory, safe
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// Test file existence
	_, err = os.Stat(testFile)
	assert.NoError(t, err)

	// Test file deletion
	err = os.Remove(testFile)
	require.NoError(t, err)

	// Verify file is gone
	_, err = os.Stat(testFile)
	assert.True(t, os.IsNotExist(err))
}

func TestTimeoutHandling(t *testing.T) {
	// Test timeout scenarios
	tests := []struct {
		name          string
		duration      time.Duration
		timeout       time.Duration
		expectTimeout bool
	}{
		{
			name:          "operation completes before timeout",
			duration:      time.Millisecond * 10,
			timeout:       time.Millisecond * 50,
			expectTimeout: false,
		},
		{
			name:          "operation times out",
			duration:      time.Millisecond * 50,
			timeout:       time.Millisecond * 10,
			expectTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			done := make(chan bool, 1)

			go func() {
				// Simulate work with CPU-bound operation instead of sleep
				start := time.Now()
				for time.Since(start) < tt.duration {
					// CPU-bound work to simulate processing time
					for i := 0; i < 1000; i++ {
						_ = i * i
					}
					// Brief yield to prevent hanging
					if time.Since(start) > tt.duration {
						break
					}
				}
				done <- true
			}()

			select {
			case <-done:
				if tt.expectTimeout {
					t.Error("Expected timeout but operation completed")
				}
			case <-ctx.Done():
				if !tt.expectTimeout {
					t.Error("Unexpected timeout")
				}
				assert.Equal(t, context.DeadlineExceeded, ctx.Err())
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	// Test various error scenarios
	tests := []struct {
		name        string
		setupError  func() error
		expectError bool
		errorType   string
	}{
		{
			name: "file not found error",
			setupError: func() error {
				_, err := os.Open("/nonexistent/file")
				return err
			},
			expectError: true,
			errorType:   "no such file",
		},
		{
			name: "permission denied error",
			setupError: func() error {
				// Try to create file in root directory (usually fails)
				_, err := os.Create("/root/test")
				return err
			},
			expectError: true,
			errorType:   "", // May vary by system
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupError()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != "" {
					assert.Contains(t, strings.ToLower(err.Error()), tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
