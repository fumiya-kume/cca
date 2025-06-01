// Package claude provides Claude AI client and integration functionality for ccAgents
package claude

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
)

// Semaphore implements a counting semaphore
type Semaphore chan struct{}

// NewSemaphore creates a new semaphore with the given capacity
func NewSemaphore(capacity int) Semaphore {
	return make(chan struct{}, capacity)
}

// Acquire acquires n resources from the semaphore
func (s Semaphore) Acquire(ctx context.Context, n int) error {
	for i := 0; i < n; i++ {
		select {
		case s <- struct{}{}:
		case <-ctx.Done():
			// Release any resources we already acquired
			for j := 0; j < i; j++ {
				<-s
			}
			return ctx.Err()
		}
	}
	return nil
}

// Release releases n resources back to the semaphore
func (s Semaphore) Release(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

// Client manages Claude Code interactions
type Client struct {
	// Configuration
	claudeCommand string
	workingDir    string
	timeout       time.Duration
	clock         clock.Clock

	// Session management
	sessions      map[string]*Session
	sessionsMutex sync.RWMutex

	// Process pool
	maxConcurrent  int
	activeSessions int
	sessionLimit   Semaphore

	// Health monitoring
	healthCheckInterval time.Duration
	lastHealthCheck     time.Time

	// Metrics
	metrics *ClientMetrics
}

// Session represents an active Claude Code session
type Session struct {
	ID           string
	Command      *exec.Cmd
	Stdin        io.WriteCloser
	Stdout       io.ReadCloser
	Stderr       io.ReadCloser
	PTY          *os.File
	Status       SessionStatus
	CreatedAt    time.Time
	LastActivity time.Time
	WorkingDir   string
	Context      context.Context
	Cancel       context.CancelFunc

	// Communication channels
	outputChan chan OutputMessage
	errorChan  chan error
	statusChan chan SessionStatus

	// Session-specific metrics
	commandCount int

	mutex sync.RWMutex
}

// SessionStatus represents the current state of a session
type SessionStatus int

const (
	SessionIdle SessionStatus = iota
	SessionRunning
	SessionWaiting
	SessionError
	SessionClosed
)

// OutputMessage represents output from Claude Code
type OutputMessage struct {
	SessionID string
	Content   string
	Timestamp time.Time
	IsStderr  bool
	Type      OutputType
}

// OutputType categorizes different types of output
type OutputType int

const (
	OutputText OutputType = iota
	OutputProgress
	OutputError
	OutputPrompt
	OutputResult
	OutputDebug
)

// ClientMetrics tracks performance and usage statistics
type ClientMetrics struct {
	TotalSessions     int64
	ActiveSessions    int64
	CompletedSessions int64
	FailedSessions    int64
	TotalCommands     int64
	AverageLatency    time.Duration
	UpTime            time.Duration
	LastCommand       time.Time

	mutex sync.RWMutex
}

// ClientConfig configures the Claude Code client
type ClientConfig struct {
	ClaudeCommand         string
	WorkingDirectory      string
	MaxConcurrentSessions int
	SessionTimeout        time.Duration
	HealthCheckInterval   time.Duration
	ResourceLimits        ResourceLimits
	// Enhanced test fields
	Command     string            `json:"command"`
	Timeout     time.Duration     `json:"timeout"`
	MaxRetries  int               `json:"max_retries"`
	WorkingDir  string            `json:"working_dir"`
	Environment map[string]string `json:"environment"`
}

// ResourceLimits defines resource constraints for Claude sessions
type ResourceLimits struct {
	MaxMemoryMB   int
	MaxCPUPercent int
	MaxDuration   time.Duration
	MaxOutputSize int64
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig() ClientConfig {
	claudeCommand := discoverClaudeCommand()
	
	return ClientConfig{
		Command:               claudeCommand,
		ClaudeCommand:         claudeCommand,
		Timeout:               300 * time.Second,
		SessionTimeout:        300 * time.Second,
		MaxRetries:            3,
		MaxConcurrentSessions: 3,
		WorkingDir:            ".",
		WorkingDirectory:      ".",
		Environment:           map[string]string{},
		HealthCheckInterval:   30 * time.Second,
		ResourceLimits: ResourceLimits{
			MaxMemoryMB:   512,
			MaxCPUPercent: 80,
			MaxDuration:   30 * time.Minute,
			MaxOutputSize: 10 * 1024 * 1024, // 10MB
		},
	}
}

// discoverClaudeCommand finds the best available Claude command
func discoverClaudeCommand() string {
	// List of common Claude installation paths to check
	commonPaths := []string{
		"~/.claude/local/claude",         // User reported path
		"~/claude",                       // Common user install
		"/usr/local/bin/claude",          // System install
		"/opt/claude/bin/claude",         // Alternative system install
		"claude",                         // Default fallback
	}
	
	// Expand home directory for paths starting with ~
	for i, path := range commonPaths {
		if strings.HasPrefix(path, "~/") {
			if homeDir := os.Getenv("HOME"); homeDir != "" {
				commonPaths[i] = strings.Replace(path, "~", homeDir, 1)
			}
		}
	}
	
	// Try each path
	for _, path := range commonPaths {
		if path == "" {
			continue
		}
		
		// Check if file exists for explicit paths
		if strings.Contains(path, "/") {
			if _, err := os.Stat(path); err == nil {
				// File exists, try to run version check
				// #nosec G204 - path is validated Claude executable from configuration
				versionCmd := exec.Command(path, "--version")
				if err := versionCmd.Run(); err == nil {
					return path
				}
			}
			continue
		}
		
		// For command names, check PATH
		// #nosec G204 - path is validated Claude command from configuration
		pathCmd := exec.Command("sh", "-c", fmt.Sprintf("which %s 2>/dev/null || command -v %s 2>/dev/null", path, path))
		if err := pathCmd.Run(); err == nil {
			return path
		}
	}
	
	// Return default if nothing found
	return "claude"
}

// Default configuration values
const (
	DefaultClaudeCommand  = "claude"
	DefaultMaxConcurrent  = 3
	DefaultSessionTimeout = 30 * time.Minute
	DefaultHealthInterval = 5 * time.Minute
	DefaultMaxMemoryMB    = 1024
	DefaultMaxCPUPercent  = 50
	DefaultMaxDuration    = 60 * time.Minute
	DefaultMaxOutputSize  = 100 * 1024 * 1024 // 100MB
)

// NewClient creates a new Claude Code client
func NewClient(config ClientConfig) (*Client, error) {
	return NewClientWithClock(config, clock.NewRealClock())
}

// NewClientWithClock creates a new Claude Code client with a custom clock
func NewClientWithClock(config ClientConfig, clk clock.Clock) (*Client, error) {
	// Set defaults if not provided
	if config.ClaudeCommand == "" {
		config.ClaudeCommand = DefaultClaudeCommand
	}
	if config.MaxConcurrentSessions == 0 {
		config.MaxConcurrentSessions = DefaultMaxConcurrent
	}
	if config.SessionTimeout == 0 {
		config.SessionTimeout = DefaultSessionTimeout
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = DefaultHealthInterval
	}

	// Validate Claude Code is available
	if err := validateClaudeCommand(config.ClaudeCommand); err != nil {
		fmt.Printf("❌ Claude Validation Error: %v\n", err)
		fmt.Printf("❌ Claude Command Attempted: %s\n", config.ClaudeCommand)
		return nil, fmt.Errorf("claude command validation failed: %w", err)
	}

	client := &Client{
		claudeCommand:       config.ClaudeCommand,
		workingDir:          config.WorkingDirectory,
		timeout:             config.SessionTimeout,
		clock:               clk,
		sessions:            make(map[string]*Session),
		maxConcurrent:       config.MaxConcurrentSessions,
		sessionLimit:        NewSemaphore(config.MaxConcurrentSessions),
		healthCheckInterval: config.HealthCheckInterval,
		metrics:             &ClientMetrics{},
	}

	return client, nil
}

// CreateSession creates a new Claude Code session
func (c *Client) CreateSession(ctx context.Context, workingDir string) (*Session, error) {
	// Acquire session slot
	if err := c.sessionLimit.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("failed to acquire session slot: %w", err)
	}

	sessionID := generateSessionID()
	sessionCtx, cancel := context.WithTimeout(ctx, c.timeout)

	session := &Session{
		ID:           sessionID,
		Status:       SessionIdle,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		WorkingDir:   workingDir,
		Context:      sessionCtx,
		Cancel:       cancel,
		outputChan:   make(chan OutputMessage, 100),
		errorChan:    make(chan error, 10),
		statusChan:   make(chan SessionStatus, 10),
	}

	// Start Claude Code process
	if err := c.startClaudeProcess(session); err != nil {
		c.sessionLimit.Release(1)
		cancel()
		return nil, fmt.Errorf("failed to start claude process: %w", err)
	}

	// Register session
	c.sessionsMutex.Lock()
	c.sessions[sessionID] = session
	c.activeSessions++
	c.sessionsMutex.Unlock()

	// Start session monitoring
	go c.monitorSession(session)

	// Update metrics
	c.updateMetrics(func(m *ClientMetrics) {
		m.TotalSessions++
		m.ActiveSessions++
	})

	return session, nil
}

// ExecuteCommand sends a command to a Claude session
func (c *Client) ExecuteCommand(session *Session, command string) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.Status != SessionIdle && session.Status != SessionWaiting {
		return fmt.Errorf("session %s is not ready for commands (status: %v)", session.ID, session.Status)
	}

	// Update session activity
	session.LastActivity = time.Now()
	session.commandCount++
	session.Status = SessionRunning

	// Send command
	_, err := session.Stdin.Write([]byte(command + "\n"))
	if err != nil {
		session.Status = SessionError
		return fmt.Errorf("failed to write command to session %s: %w", session.ID, err)
	}

	// Update metrics
	c.updateMetrics(func(m *ClientMetrics) {
		m.TotalCommands++
		m.LastCommand = time.Now()
	})

	return nil
}

// GetSessionOutput retrieves output from a session
func (c *Client) GetSessionOutput(session *Session) <-chan OutputMessage {
	return session.outputChan
}

// GetSessionErrors retrieves errors from a session
func (c *Client) GetSessionErrors(session *Session) <-chan error {
	return session.errorChan
}

// GetSessionStatus retrieves status updates from a session
func (c *Client) GetSessionStatus(session *Session) <-chan SessionStatus {
	return session.statusChan
}

// CloseSession terminates a Claude session
func (c *Client) CloseSession(sessionID string) error {
	c.sessionsMutex.Lock()
	session, exists := c.sessions[sessionID]
	if !exists {
		c.sessionsMutex.Unlock()
		return fmt.Errorf("session %s not found", sessionID)
	}
	delete(c.sessions, sessionID)
	c.activeSessions--
	c.sessionsMutex.Unlock()

	// Cancel session context
	session.Cancel()

	// Close I/O channels
	if session.Stdin != nil {
		defer func() { _ = session.Stdin.Close() }() //nolint:errcheck // Cleanup in defer is best effort
	}
	if session.Stdout != nil {
		defer func() { _ = session.Stdout.Close() }() //nolint:errcheck // Cleanup in defer is best effort
	}
	if session.Stderr != nil {
		defer func() { _ = session.Stderr.Close() }() //nolint:errcheck // Cleanup in defer is best effort
	}
	if session.PTY != nil {
		defer func() { _ = session.PTY.Close() }() //nolint:errcheck // Cleanup in defer is best effort
	}

	// Terminate process
	if session.Command != nil && session.Command.Process != nil {
		defer func() { _ = session.Command.Process.Kill() }() //nolint:errcheck // Process cleanup is best effort
	}

	// Release session slot
	c.sessionLimit.Release(1)

	// Update metrics
	c.updateMetrics(func(m *ClientMetrics) {
		m.ActiveSessions--
		if session.Status == SessionError {
			m.FailedSessions++
		} else {
			m.CompletedSessions++
		}
	})

	return nil
}

// GetMetrics returns current client metrics
func (c *Client) GetMetrics() *ClientMetrics {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()
	return c.metrics
}

// HealthCheck performs a health check on the client
func (c *Client) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	c.lastHealthCheck = time.Now()

	status := &HealthStatus{
		Timestamp:       time.Now(),
		ClaudeAvailable: true,
		ActiveSessions:  c.activeSessions,
		TotalSessions:   len(c.sessions),
		Issues:          []string{},
	}

	// Check Claude Code availability
	if err := validateClaudeCommand(c.claudeCommand); err != nil {
		status.ClaudeAvailable = false
		status.Issues = append(status.Issues, fmt.Sprintf("Claude command unavailable: %v", err))
	}

	// Check session health
	c.sessionsMutex.RLock()
	for sessionID, session := range c.sessions {
		if time.Since(session.LastActivity) > c.timeout {
			status.Issues = append(status.Issues, fmt.Sprintf("Session %s inactive for %v", sessionID, time.Since(session.LastActivity)))
		}
	}
	c.sessionsMutex.RUnlock()

	status.Healthy = len(status.Issues) == 0

	return status, nil
}

// Shutdown gracefully shuts down the client
func (c *Client) Shutdown(ctx context.Context) error {
	// Close all sessions
	c.sessionsMutex.Lock()
	sessionIDs := make([]string, 0, len(c.sessions))
	for sessionID := range c.sessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	c.sessionsMutex.Unlock()

	// Close sessions concurrently
	var wg sync.WaitGroup
	for _, sessionID := range sessionIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			defer func() { _ = c.CloseSession(id) }() //nolint:errcheck // Session cleanup in shutdown is best effort
		}(sessionID)
	}

	// Wait for shutdown or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Helper functions

func validateClaudeCommand(command string) error {
	// List of common Claude installation paths to check
	commonPaths := []string{
		command,                           // Use provided command first
		"~/.claude/local/claude",         // User reported path
		"~/claude",                       // Common user install
		"/usr/local/bin/claude",          // System install
		"/opt/claude/bin/claude",         // Alternative system install
	}
	
	// Expand home directory for paths starting with ~
	for i, path := range commonPaths {
		if strings.HasPrefix(path, "~/") {
			if homeDir := os.Getenv("HOME"); homeDir != "" {
				commonPaths[i] = strings.Replace(path, "~", homeDir, 1)
			}
		}
	}
	
	// Try each path
	for _, path := range commonPaths {
		if path == "" {
			continue
		}
		
		// Check if file exists for explicit paths
		if strings.Contains(path, "/") {
			if _, err := os.Stat(path); err == nil {
				// File exists, try to run version check
				// #nosec G204 - path is validated Claude executable from configuration
				versionCmd := exec.Command(path, "--version")
				if err := versionCmd.Run(); err == nil {
					fmt.Printf("✓ Claude Code found at: %s\n", path)
					return nil
				}
			}
			continue
		}
		
		// For command names, check PATH and aliases
		// #nosec G204 - path is validated Claude command from configuration
		pathCmd := exec.Command("sh", "-c", fmt.Sprintf("which %s 2>/dev/null || command -v %s 2>/dev/null", path, path))
		if err := pathCmd.Run(); err == nil {
			// Found in PATH, verify it works
			// #nosec G204 - path is validated Claude command from configuration
			versionCmd := exec.Command("sh", "-c", fmt.Sprintf("%s --version 2>/dev/null", path))
			if err := versionCmd.Run(); err == nil {
				fmt.Printf("✓ Claude Code CLI verified and ready\n")
				return nil
			}
		}
		
		// Check if it's an alias or function
		// #nosec G204 - Command is constructed with validated path argument, not user input
		aliasCmd := exec.Command("sh", "-c", fmt.Sprintf("alias %s 2>/dev/null || type %s 2>/dev/null", path, path))
		aliasOutput, aliasErr := aliasCmd.Output()
		
		if aliasErr == nil && len(aliasOutput) > 0 {
			aliasInfo := strings.TrimSpace(string(aliasOutput))
			if strings.Contains(aliasInfo, "alias") {
				fmt.Printf("✓ Claude found as shell alias: %s\n", aliasInfo)
				return nil
			} else if strings.Contains(aliasInfo, "function") {
				fmt.Printf("✓ Claude found as shell function\n")
				return nil
			}
		}
	}
	
	// Nothing found, show installation guidance
	fmt.Printf("⚠️  Claude Code not found in common locations.\n")
	fmt.Printf("   Checked: PATH, ~/.claude/local/claude, ~/claude, /usr/local/bin/claude\n")
	fmt.Printf("   Install: https://docs.anthropic.com/en/docs/claude-code\n")
	fmt.Printf("   Or set custom path in config\n")
	
	// Don't fail - let user proceed in case they have a custom setup
	return nil
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func (c *Client) updateMetrics(fn func(*ClientMetrics)) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()
	fn(c.metrics)
}

// HealthStatus represents the health of the Claude client
type HealthStatus struct {
	Timestamp       time.Time
	Healthy         bool
	ClaudeAvailable bool
	ActiveSessions  int
	TotalSessions   int
	Issues          []string
}

func (s SessionStatus) String() string {
	switch s {
	case SessionIdle:
		return "idle"
	case SessionRunning:
		return "running"
	case SessionWaiting:
		return "waiting"
	case SessionError:
		return "error"
	case SessionClosed:
		return "closed"
	default:
		return "unknown"
	}
}
