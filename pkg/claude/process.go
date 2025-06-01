package claude

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
)

var _ clock.Clock = (*clock.RealClock)(nil) // Ensure clock import is used

// startClaudeProcess starts a new Claude Code process for the session
func (c *Client) startClaudeProcess(session *Session) error {
	workingDir := session.WorkingDir
	if workingDir == "" {
		workingDir = c.workingDir
	}
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Create command with shell support for aliases
	// #nosec G204 - claudeCommand is validated at client creation and comes from trusted configuration
	var cmd *exec.Cmd
	if strings.Contains(c.claudeCommand, " ") || c.claudeCommand == "claude" {
		// If command contains spaces or is the default "claude", use shell to support aliases
		// #nosec G204 - claudeCommand is from config, not user input
		cmd = exec.CommandContext(session.Context, "sh", "-c", c.claudeCommand)
		fmt.Printf("🔍 Debug: Starting interactive Claude process with shell: sh -c \"%s\"\n", c.claudeCommand)
	} else {
		// Direct command execution for explicit paths
		// #nosec G204 - claudeCommand is from validated config, not user input
		cmd = exec.CommandContext(session.Context, c.claudeCommand)
		fmt.Printf("🔍 Debug: Starting interactive Claude process: %s\n", c.claudeCommand)
	}
	cmd.Dir = workingDir
	cmd.Env = append(os.Environ(),
		"CLAUDE_SESSION_ID="+session.ID,
		"CLAUDE_WORKING_DIR="+workingDir,
	)
	fmt.Printf("🔍 Debug: Working directory: %s\n", workingDir)
	fmt.Printf("🔍 Debug: Session ID: %s\n", session.ID)

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close() //nolint:errcheck // Cleanup on error
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close() //nolint:errcheck // Cleanup on error
		_ = stdout.Close() //nolint:errcheck // Cleanup on error
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		_ = stdin.Close() //nolint:errcheck // Cleanup on error
		_ = stdout.Close() //nolint:errcheck // Cleanup on error
		_ = stderr.Close() //nolint:errcheck // Cleanup on error
		fmt.Printf("❌ Claude Process Start Error: %v\n", err)
		fmt.Printf("❌ Claude Command: %s\n", c.claudeCommand)
		fmt.Printf("❌ Working Directory: %s\n", workingDir)
		return fmt.Errorf("failed to start claude process: %w", err)
	}

	// Update session
	session.Command = cmd
	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr
	session.Status = SessionIdle

	// Start output monitoring
	go c.monitorOutput(session, stdout, false)
	go c.monitorOutput(session, stderr, true)

	return nil
}

// monitorSession monitors a session for health and lifecycle events
func (c *Client) monitorSession(session *Session) {
	defer func() {
		if r := recover(); r != nil {
			session.errorChan <- fmt.Errorf("session monitor panic: %v", r)
		}
	}()

	healthTicker := c.clock.NewTicker(5 * time.Second)
	defer healthTicker.Stop()

	processTicker := c.clock.NewTicker(100 * time.Millisecond)
	defer processTicker.Stop()

	for {
		select {
		case <-session.Context.Done():
			// Session context canceled
			session.Status = SessionClosed
			// Send status without blocking
			select {
			case session.statusChan <- SessionClosed:
			default:
				// Channel is full, status will be detected elsewhere
			}
			return

		case <-healthTicker.C():
			// Periodic health check
			if err := c.checkSessionHealth(session); err != nil {
				// Send error without blocking
				select {
				case session.errorChan <- err:
				default:
					// Error channel is full, continue monitoring
				}
			}

		case <-processTicker.C():
			// Check if process has exited
			if session.Command != nil && session.Command.ProcessState != nil {
				if session.Command.ProcessState.Exited() {
					exitCode := session.Command.ProcessState.ExitCode()
					if exitCode != 0 {
						session.Status = SessionError
						errorMsg := fmt.Errorf("claude process exited with code %d", exitCode)
						fmt.Printf("❌ Claude Process Error: %v (Session: %s)\n", errorMsg, session.ID)
						select {
						case session.errorChan <- errorMsg:
						default:
						}
					} else {
						session.Status = SessionClosed
					}
					// Send status without blocking
					select {
					case session.statusChan <- session.Status:
					default:
					}
					return
				}
			}
		}
	}
}

// monitorOutput monitors stdout/stderr from a Claude process
func (c *Client) monitorOutput(session *Session, reader io.ReadCloser, isStderr bool) {
	defer c.handleMonitorPanic(session, reader)

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	scanDone := c.startScanning(session, scanner, isStderr)
	c.waitForScanCompletion(session, scanner, scanDone)
}

// handleMonitorPanic handles panic recovery and cleanup for output monitoring
func (c *Client) handleMonitorPanic(session *Session, reader io.ReadCloser) {
	if r := recover(); r != nil {
		// Send error without blocking
		select {
		case session.errorChan <- fmt.Errorf("output monitor panic: %v", r):
		default:
		}
	}
	// Ensure reader is closed
	defer func() { _ = reader.Close() }() //nolint:errcheck // Cleanup in defer is best effort
}

// startScanning starts the scanning goroutine and returns a completion channel
func (c *Client) startScanning(session *Session, scanner *bufio.Scanner, isStderr bool) <-chan struct{} {
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		c.scanLoop(session, scanner, isStderr)
	}()
	return scanDone
}

// scanLoop performs the main scanning loop
func (c *Client) scanLoop(session *Session, scanner *bufio.Scanner, isStderr bool) {
	for scanner.Scan() {
		if c.shouldStopScanning(session) {
			return
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		c.processOutputLine(session, line, isStderr)
	}
}

// shouldStopScanning checks if scanning should stop due to context cancellation
func (c *Client) shouldStopScanning(session *Session) bool {
	select {
	case <-session.Context.Done():
		return true
	default:
		return false
	}
}

// processOutputLine processes a single output line
func (c *Client) processOutputLine(session *Session, line string, isStderr bool) {
	c.updateSessionActivity(session)
	outputType := c.categorizeOutput(line)
	c.updateSessionStatus(session, outputType)
	c.sendOutputMessage(session, line, outputType, isStderr)
}

// updateSessionActivity updates the session's last activity timestamp
func (c *Client) updateSessionActivity(session *Session) {
	session.mutex.Lock()
	session.LastActivity = c.clock.Now()
	session.mutex.Unlock()
}

// updateSessionStatus updates session status based on output type
func (c *Client) updateSessionStatus(session *Session, outputType OutputType) {
	if outputType == OutputPrompt {
		c.handlePromptOutput(session)
	} else if outputType == OutputResult && session.Status == SessionRunning {
		c.handleResultOutput(session)
	}
}

// handlePromptOutput handles prompt output by updating session status
func (c *Client) handlePromptOutput(session *Session) {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.Status == SessionRunning {
		session.Status = SessionWaiting
		select {
		case session.statusChan <- SessionWaiting:
		default:
		}
	}
}

// handleResultOutput handles result output by updating session status
func (c *Client) handleResultOutput(session *Session) {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	session.Status = SessionIdle
	select {
	case session.statusChan <- SessionIdle:
	default:
	}
}

// sendOutputMessage sends output message to the output channel
func (c *Client) sendOutputMessage(session *Session, line string, outputType OutputType, isStderr bool) {
	msg := OutputMessage{
		SessionID: session.ID,
		Content:   line,
		Timestamp: c.clock.Now(),
		IsStderr:  isStderr,
		Type:      outputType,
	}

	select {
	case session.outputChan <- msg:
	case <-session.Context.Done():
		return
	default:
		// Channel full, drop message to prevent blocking
	}
}

// waitForScanCompletion waits for scanning to complete or context cancellation
func (c *Client) waitForScanCompletion(session *Session, scanner *bufio.Scanner, scanDone <-chan struct{}) {
	select {
	case <-scanDone:
		// Scanner completed normally
		c.handleScannerError(session, scanner)
	case <-session.Context.Done():
		// Context canceled, scanner goroutine will exit
		return
	}
}

// handleScannerError handles any scanner errors
func (c *Client) handleScannerError(session *Session, scanner *bufio.Scanner) {
	if err := scanner.Err(); err != nil {
		scannerErr := fmt.Errorf("output scanner error: %w", err)
		fmt.Printf("❌ Claude Scanner Error: %v (Session: %s)\n", scannerErr, session.ID)
		session.errorChan <- scannerErr
	}
}

// categorizeOutput determines the type of output from Claude
func (c *Client) categorizeOutput(line string) OutputType {
	normalizedLine := strings.TrimSpace(strings.ToLower(line))

	if c.isPromptOutput(normalizedLine) {
		return OutputPrompt
	}

	if c.isProgressOutput(normalizedLine) {
		return OutputProgress
	}

	if c.isErrorOutput(normalizedLine) {
		return OutputError
	}

	if c.isResultOutput(normalizedLine) {
		return OutputResult
	}

	if c.isDebugOutput(normalizedLine) {
		return OutputDebug
	}

	return OutputText
}

// isPromptOutput checks if the line indicates a prompt for user input
func (c *Client) isPromptOutput(line string) bool {
	promptKeywords := []string{"enter", "input", "confirm", "y/n"}
	return c.containsAnyKeyword(line, promptKeywords)
}

// isProgressOutput checks if the line indicates progress information
func (c *Client) isProgressOutput(line string) bool {
	progressKeywords := []string{"progress", "%", "loading", "processing"}
	return c.containsAnyKeyword(line, progressKeywords)
}

// isErrorOutput checks if the line indicates an error
func (c *Client) isErrorOutput(line string) bool {
	errorKeywords := []string{"error", "failed", "exception", "panic"}
	return c.containsAnyKeyword(line, errorKeywords)
}

// isResultOutput checks if the line indicates completion or success
func (c *Client) isResultOutput(line string) bool {
	resultKeywords := []string{"complete", "finished", "done", "success"}
	return c.containsAnyKeyword(line, resultKeywords)
}

// isDebugOutput checks if the line contains debug information
func (c *Client) isDebugOutput(line string) bool {
	debugKeywords := []string{"debug", "trace", "verbose"}
	return c.containsAnyKeyword(line, debugKeywords)
}

// containsAnyKeyword checks if the line contains any of the given keywords
func (c *Client) containsAnyKeyword(line string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	return false
}

// checkSessionHealth performs health checks on a session
func (c *Client) checkSessionHealth(session *Session) error {
	session.mutex.RLock()
	defer session.mutex.RUnlock()

	// Check if session has been inactive too long
	if c.clock.Since(session.LastActivity) > c.timeout {
		return fmt.Errorf("session %s inactive for %v", session.ID, c.clock.Since(session.LastActivity))
	}

	// Check if process is still running
	if session.Command != nil && session.Command.Process != nil {
		// Send signal 0 to check if process exists
		if err := session.Command.Process.Signal(syscall.Signal(0)); err != nil {
			return fmt.Errorf("session %s process no longer exists: %w", session.ID, err)
		}
	}

	return nil
}

// sendInput sends input to a Claude session
func (c *Client) sendInput(session *Session, input string) error {
	session.mutex.Lock()
	defer session.mutex.Unlock()

	if session.Status != SessionWaiting {
		return fmt.Errorf("session %s not waiting for input (status: %v)", session.ID, session.Status)
	}

	_, err := session.Stdin.Write([]byte(input + "\n"))
	if err != nil {
		session.Status = SessionError
		return fmt.Errorf("failed to send input to session %s: %w", session.ID, err)
	}

	session.LastActivity = c.clock.Now()
	session.Status = SessionRunning
	session.statusChan <- SessionRunning

	return nil
}

// getProcessMemoryUsage returns the memory usage of a session's process
func (c *Client) getProcessMemoryUsage(session *Session) (int64, error) {
	if session.Command == nil || session.Command.Process == nil {
		return 0, fmt.Errorf("no process found for session %s", session.ID)
	}

	// Read from /proc/[pid]/status on Linux
	pid := session.Command.Process.Pid
	statusFile := fmt.Sprintf("/proc/%d/status", pid)

	// #nosec G304 - statusFile path is constructed from system proc directory
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read process status: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				var memory int64
				if _, err := fmt.Sscanf(parts[1], "%d", &memory); err == nil {
					return memory * 1024, nil // Convert from kB to bytes
				}
			}
		}
	}

	return 0, fmt.Errorf("could not parse memory usage for session %s", session.ID)
}

// getProcessCPUUsage returns the CPU usage percentage of a session's process
func (c *Client) getProcessCPUUsage(session *Session) (float64, error) {
	if session.Command == nil || session.Command.Process == nil {
		return 0, fmt.Errorf("no process found for session %s", session.ID)
	}

	// Read from /proc/[pid]/stat on Linux
	pid := session.Command.Process.Pid
	statFile := fmt.Sprintf("/proc/%d/stat", pid)

	// #nosec G304 - statFile path is constructed from system proc directory
	data, err := os.ReadFile(statFile)
	if err != nil {
		return 0, fmt.Errorf("failed to read process stat: %w", err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 22 {
		return 0, fmt.Errorf("invalid stat format for session %s", session.ID)
	}

	// Fields 13 and 14 are utime and stime (user and system CPU time)
	var utime, stime int64
	if _, err := fmt.Sscanf(fields[13], "%d", &utime); err != nil {
		return 0, err
	}
	if _, err := fmt.Sscanf(fields[14], "%d", &stime); err != nil {
		return 0, err
	}

	// Get system uptime
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, fmt.Errorf("failed to read system uptime: %w", err)
	}

	var systemUptime float64
	if _, err := fmt.Sscanf(string(uptimeData), "%f", &systemUptime); err != nil {
		return 0, err
	}

	// Calculate CPU usage percentage
	totalTime := float64(utime + stime)
	clockTicks := float64(100) // Typical value, could be obtained from sysconf
	processUptime := c.clock.Since(session.CreatedAt).Seconds()

	if processUptime == 0 {
		return 0, nil
	}

	cpuUsage := (totalTime / clockTicks) / processUptime * 100

	return cpuUsage, nil
}

// enforceResourceLimits checks and enforces resource limits for a session
func (c *Client) enforceResourceLimits(session *Session, limits ResourceLimits) error {
	// Check memory limit
	if limits.MaxMemoryMB > 0 {
		memUsage, err := c.getProcessMemoryUsage(session)
		if err == nil && memUsage > int64(limits.MaxMemoryMB)*1024*1024 {
			return fmt.Errorf("session %s exceeded memory limit: %d MB > %d MB",
				session.ID, memUsage/(1024*1024), limits.MaxMemoryMB)
		}
	}

	// Check CPU limit
	if limits.MaxCPUPercent > 0 {
		cpuUsage, err := c.getProcessCPUUsage(session)
		if err == nil && cpuUsage > float64(limits.MaxCPUPercent) {
			return fmt.Errorf("session %s exceeded CPU limit: %.1f%% > %d%%",
				session.ID, cpuUsage, limits.MaxCPUPercent)
		}
	}

	// Check duration limit
	if limits.MaxDuration > 0 && c.clock.Since(session.CreatedAt) > limits.MaxDuration {
		return fmt.Errorf("session %s exceeded duration limit: %v > %v",
			session.ID, c.clock.Since(session.CreatedAt), limits.MaxDuration)
	}

	return nil
}
