// Package ui provides terminal user interface components and models for ccAgents
package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fumiya-kume/cca/internal/types"
)

// Workflow Messages

// WorkflowStartMsg is sent when a workflow begins
type WorkflowStartMsg struct {
	IssueRef *types.IssueReference
	Stages   []WorkflowStage
}

// WorkflowCompleteMsg is sent when a workflow completes successfully
type WorkflowCompleteMsg struct {
	Duration time.Duration
	Results  map[string]interface{}
}

// WorkflowErrorMsg is sent when a workflow encounters an error
type WorkflowErrorMsg struct {
	Error error
	Stage int
}

// WorkflowPauseMsg is sent when a workflow is paused
type WorkflowPauseMsg struct {
	Reason string
}

// WorkflowResumeMsg is sent when a workflow is resumed
type WorkflowResumeMsg struct{}

// Stage Messages

// StageStartMsg is sent when a workflow stage begins
type StageStartMsg struct {
	StageIndex int
	StageName  string
	Timestamp  time.Time
}

// StageCompleteMsg is sent when a workflow stage completes
type StageCompleteMsg struct {
	StageIndex int
	StageName  string
	Duration   time.Duration
	Results    map[string]interface{}
}

// StageErrorMsg is sent when a workflow stage fails
type StageErrorMsg struct {
	StageIndex  int
	StageName   string
	Error       error
	Recoverable bool
}

// StageProgressMsg is sent to update stage progress
type StageProgressMsg struct {
	StageIndex int
	Progress   float64 // 0.0 to 1.0
	Message    string
}

// Process Messages

// ProcessOutputMsg represents output from an external process
type ProcessOutputMsg struct {
	ProcessID string
	Line      string
	Timestamp time.Time
	IsStderr  bool
}

// ProcessStartMsg is sent when a process starts
type ProcessStartMsg struct {
	ProcessID string
	Command   string
	Args      []string
}

// ProcessExitedMsg is sent when a process exits
type ProcessExitedMsg struct {
	ProcessID string
	ExitCode  int
	Duration  time.Duration
}

// User Input Messages

// UserInputRequestMsg requests input from the user
type UserInputRequestMsg struct {
	Prompt      string
	Placeholder string
	Required    bool
	Validation  func(string) error
}

// UserInputResponseMsg contains the user's response
type UserInputResponseMsg struct {
	Response string
	Valid    bool
}

// Status Messages

// StatusUpdateMsg updates the status message
type StatusUpdateMsg struct {
	Message string
	Type    StatusType
}

// StatusType defines the type of status message
type StatusType int

const (
	StatusInfo StatusType = iota
	StatusSuccess
	StatusWarning
	StatusError
)

// ErrorMsg represents an error message
type ErrorMsg struct {
	Error   error
	Context string
	Fatal   bool
}

// SuccessMsg represents a success message
type SuccessMsg struct {
	Message string
	Details map[string]interface{}
}

// Command Functions - these return tea.Cmd functions

// WorkflowStart returns a command to start a workflow
func WorkflowStart(issueRef *types.IssueReference, stages []WorkflowStage) tea.Cmd {
	return func() tea.Msg {
		return WorkflowStartMsg{
			IssueRef: issueRef,
			Stages:   stages,
		}
	}
}

// WorkflowComplete returns a command to signal workflow completion
func WorkflowComplete(duration time.Duration, results map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		return WorkflowCompleteMsg{
			Duration: duration,
			Results:  results,
		}
	}
}

// WorkflowError returns a command to signal a workflow error
func WorkflowError(err error, stage int) tea.Cmd {
	return func() tea.Msg {
		return WorkflowErrorMsg{
			Error: err,
			Stage: stage,
		}
	}
}

// WorkflowCancel returns a command to cancel the current workflow
func WorkflowCancel() tea.Cmd {
	return func() tea.Msg {
		return tea.Quit()
	}
}

// WorkflowRetry returns a command to retry a failed workflow
func WorkflowRetry() tea.Cmd {
	return func() tea.Msg {
		// This would trigger a workflow restart
		return WorkflowResumeMsg{}
	}
}

// StageStart returns a command to start a workflow stage
func StageStart(index int, name string) tea.Cmd {
	return func() tea.Msg {
		return StageStartMsg{
			StageIndex: index,
			StageName:  name,
			Timestamp:  time.Now(),
		}
	}
}

// StageComplete returns a command to signal stage completion
func StageComplete(index int, name string, duration time.Duration, results map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		return StageCompleteMsg{
			StageIndex: index,
			StageName:  name,
			Duration:   duration,
			Results:    results,
		}
	}
}

// StageError returns a command to signal a stage error
func StageError(index int, name string, err error, recoverable bool) tea.Cmd {
	return func() tea.Msg {
		return StageErrorMsg{
			StageIndex:  index,
			StageName:   name,
			Error:       err,
			Recoverable: recoverable,
		}
	}
}

// StageProgress returns a command to update stage progress
func StageProgress(index int, progress float64, message string) tea.Cmd {
	return func() tea.Msg {
		return StageProgressMsg{
			StageIndex: index,
			Progress:   progress,
			Message:    message,
		}
	}
}

// ProcessOutput returns a command to add process output
func ProcessOutput(processID, line string, isStderr bool) tea.Cmd {
	return func() tea.Msg {
		return ProcessOutputMsg{
			ProcessID: processID,
			Line:      line,
			Timestamp: time.Now(),
			IsStderr:  isStderr,
		}
	}
}

// ProcessStart returns a command to signal process start
func ProcessStart(processID, command string, args []string) tea.Cmd {
	return func() tea.Msg {
		return ProcessStartMsg{
			ProcessID: processID,
			Command:   command,
			Args:      args,
		}
	}
}

// ProcessExited returns a command to signal process exit
func ProcessExited(processID string, exitCode int, duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		return ProcessExitedMsg{
			ProcessID: processID,
			ExitCode:  exitCode,
			Duration:  duration,
		}
	}
}

// UserInputRequest returns a command to request user input
func UserInputRequest(prompt, placeholder string, required bool, validation func(string) error) tea.Cmd {
	return func() tea.Msg {
		return UserInputRequestMsg{
			Prompt:      prompt,
			Placeholder: placeholder,
			Required:    required,
			Validation:  validation,
		}
	}
}

// UserInputSubmitted returns a command when user submits input
func UserInputSubmitted(response string) tea.Cmd {
	return func() tea.Msg {
		return UserInputResponseMsg{
			Response: response,
			Valid:    true, // Validation would happen elsewhere
		}
	}
}

// StatusUpdate returns a command to update the status
func StatusUpdate(message string, statusType StatusType) tea.Cmd {
	return func() tea.Msg {
		return StatusUpdateMsg{
			Message: message,
			Type:    statusType,
		}
	}
}

// ShowError returns a command to display an error
func ShowError(err error, context string, fatal bool) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{
			Error:   err,
			Context: context,
			Fatal:   fatal,
		}
	}
}

// ShowSuccess returns a command to display a success message
func ShowSuccess(message string, details map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		return SuccessMsg{
			Message: message,
			Details: details,
		}
	}
}

// Utility Commands

// Tick returns a command that sends a message after a duration
func Tick(duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return struct{}{}
	})
}

// Every returns a command that sends a message repeatedly
func Every(duration time.Duration, fn func() tea.Msg) tea.Cmd {
	return tea.Every(duration, func(time.Time) tea.Msg {
		return fn()
	})
}

// Sequence returns a command that executes commands in sequence
func Sequence(cmds ...tea.Cmd) tea.Cmd {
	return tea.Sequence(cmds...)
}

// Batch returns a command that executes all commands concurrently
func Batch(cmds ...tea.Cmd) tea.Cmd {
	return tea.Batch(cmds...)
}
