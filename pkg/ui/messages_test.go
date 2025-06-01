package ui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fumiya-kume/cca/internal/types"
)

// TestWorkflowMessages tests workflow-related message creation
func TestWorkflowMessages(t *testing.T) {
	t.Run("WorkflowStart", func(t *testing.T) {
		issueRef := &types.IssueReference{
			Owner:  "test-owner",
			Repo:   "test-repo",
			Number: 123,
			Source: "url",
		}
		stages := []WorkflowStage{
			{Name: "stage1", Status: types.StagePending},
			{Name: "stage2", Status: types.StagePending},
		}

		cmd := WorkflowStart(issueRef, stages)
		require.NotNil(t, cmd)

		msg := cmd()
		workflowMsg, ok := msg.(WorkflowStartMsg)
		require.True(t, ok)

		assert.Equal(t, issueRef, workflowMsg.IssueRef)
		assert.Equal(t, stages, workflowMsg.Stages)
	})

	t.Run("WorkflowComplete", func(t *testing.T) {
		duration := 5 * time.Minute
		results := map[string]interface{}{
			"files_changed": 3,
			"tests_passed":  true,
		}

		cmd := WorkflowComplete(duration, results)
		require.NotNil(t, cmd)

		msg := cmd()
		completeMsg, ok := msg.(WorkflowCompleteMsg)
		require.True(t, ok)

		assert.Equal(t, duration, completeMsg.Duration)
		assert.Equal(t, results, completeMsg.Results)
	})

	t.Run("WorkflowError", func(t *testing.T) {
		err := errors.New("test error")
		stage := 2

		cmd := WorkflowError(err, stage)
		require.NotNil(t, cmd)

		msg := cmd()
		errorMsg, ok := msg.(WorkflowErrorMsg)
		require.True(t, ok)

		assert.Equal(t, err, errorMsg.Error)
		assert.Equal(t, stage, errorMsg.Stage)
	})

	t.Run("WorkflowCancel", func(t *testing.T) {
		cmd := WorkflowCancel()
		require.NotNil(t, cmd)

		msg := cmd()
		_, ok := msg.(tea.QuitMsg)
		require.True(t, ok)
	})

	t.Run("WorkflowRetry", func(t *testing.T) {
		cmd := WorkflowRetry()
		require.NotNil(t, cmd)

		msg := cmd()
		_, ok := msg.(WorkflowResumeMsg)
		require.True(t, ok)
	})
}

// TestStageMessages tests stage-related message creation
func TestStageMessages(t *testing.T) {
	t.Run("StageStart", func(t *testing.T) {
		index := 1
		name := "test-stage"

		cmd := StageStart(index, name)
		require.NotNil(t, cmd)

		msg := cmd()
		stageMsg, ok := msg.(StageStartMsg)
		require.True(t, ok)

		assert.Equal(t, index, stageMsg.StageIndex)
		assert.Equal(t, name, stageMsg.StageName)
		assert.WithinDuration(t, time.Now(), stageMsg.Timestamp, time.Second)
	})

	t.Run("StageComplete", func(t *testing.T) {
		index := 1
		name := "test-stage"
		duration := 30 * time.Second
		results := map[string]interface{}{
			"success": true,
			"output":  "test output",
		}

		cmd := StageComplete(index, name, duration, results)
		require.NotNil(t, cmd)

		msg := cmd()
		completeMsg, ok := msg.(StageCompleteMsg)
		require.True(t, ok)

		assert.Equal(t, index, completeMsg.StageIndex)
		assert.Equal(t, name, completeMsg.StageName)
		assert.Equal(t, duration, completeMsg.Duration)
		assert.Equal(t, results, completeMsg.Results)
	})

	t.Run("StageError", func(t *testing.T) {
		index := 2
		name := "failing-stage"
		err := errors.New("stage failed")
		recoverable := true

		cmd := StageError(index, name, err, recoverable)
		require.NotNil(t, cmd)

		msg := cmd()
		errorMsg, ok := msg.(StageErrorMsg)
		require.True(t, ok)

		assert.Equal(t, index, errorMsg.StageIndex)
		assert.Equal(t, name, errorMsg.StageName)
		assert.Equal(t, err, errorMsg.Error)
		assert.Equal(t, recoverable, errorMsg.Recoverable)
	})

	t.Run("StageProgress", func(t *testing.T) {
		index := 0
		progress := 0.75
		message := "75% complete"

		cmd := StageProgress(index, progress, message)
		require.NotNil(t, cmd)

		msg := cmd()
		progressMsg, ok := msg.(StageProgressMsg)
		require.True(t, ok)

		assert.Equal(t, index, progressMsg.StageIndex)
		assert.Equal(t, progress, progressMsg.Progress)
		assert.Equal(t, message, progressMsg.Message)
	})
}

// TestProcessMessages tests process-related message creation
func TestProcessMessages(t *testing.T) {
	t.Run("ProcessOutput", func(t *testing.T) {
		processID := "proc-123"
		line := "output line"
		isStderr := false

		cmd := ProcessOutput(processID, line, isStderr)
		require.NotNil(t, cmd)

		msg := cmd()
		outputMsg, ok := msg.(ProcessOutputMsg)
		require.True(t, ok)

		assert.Equal(t, processID, outputMsg.ProcessID)
		assert.Equal(t, line, outputMsg.Line)
		assert.Equal(t, isStderr, outputMsg.IsStderr)
		assert.WithinDuration(t, time.Now(), outputMsg.Timestamp, time.Second)
	})

	t.Run("ProcessStart", func(t *testing.T) {
		processID := "proc-456"
		command := "echo"
		args := []string{"hello", "world"}

		cmd := ProcessStart(processID, command, args)
		require.NotNil(t, cmd)

		msg := cmd()
		startMsg, ok := msg.(ProcessStartMsg)
		require.True(t, ok)

		assert.Equal(t, processID, startMsg.ProcessID)
		assert.Equal(t, command, startMsg.Command)
		assert.Equal(t, args, startMsg.Args)
	})

	t.Run("ProcessExited", func(t *testing.T) {
		processID := "proc-789"
		exitCode := 0
		duration := 2 * time.Second

		cmd := ProcessExited(processID, exitCode, duration)
		require.NotNil(t, cmd)

		msg := cmd()
		exitMsg, ok := msg.(ProcessExitedMsg)
		require.True(t, ok)

		assert.Equal(t, processID, exitMsg.ProcessID)
		assert.Equal(t, exitCode, exitMsg.ExitCode)
		assert.Equal(t, duration, exitMsg.Duration)
	})
}

// TestUserInputMessages tests user input-related message creation
func TestUserInputMessages(t *testing.T) {
	t.Run("UserInputRequest", func(t *testing.T) {
		prompt := "Enter your name:"
		placeholder := "John Doe"
		required := true
		validation := func(s string) error {
			if len(s) < 2 {
				return errors.New("name too short")
			}
			return nil
		}

		cmd := UserInputRequest(prompt, placeholder, required, validation)
		require.NotNil(t, cmd)

		msg := cmd()
		inputMsg, ok := msg.(UserInputRequestMsg)
		require.True(t, ok)

		assert.Equal(t, prompt, inputMsg.Prompt)
		assert.Equal(t, placeholder, inputMsg.Placeholder)
		assert.Equal(t, required, inputMsg.Required)
		assert.NotNil(t, inputMsg.Validation)

		// Test validation function
		err := inputMsg.Validation("A")
		assert.Error(t, err)

		err = inputMsg.Validation("Alice")
		assert.NoError(t, err)
	})

	t.Run("UserInputSubmitted", func(t *testing.T) {
		response := "user response"

		cmd := UserInputSubmitted(response)
		require.NotNil(t, cmd)

		msg := cmd()
		responseMsg, ok := msg.(UserInputResponseMsg)
		require.True(t, ok)

		assert.Equal(t, response, responseMsg.Response)
		assert.True(t, responseMsg.Valid)
	})
}

// TestStatusAndErrorMessages tests status and error message creation
func TestStatusAndErrorMessages(t *testing.T) {
	t.Run("StatusUpdate", func(t *testing.T) {
		message := "Processing..."
		statusType := StatusInfo

		cmd := StatusUpdate(message, statusType)
		require.NotNil(t, cmd)

		msg := cmd()
		statusMsg, ok := msg.(StatusUpdateMsg)
		require.True(t, ok)

		assert.Equal(t, message, statusMsg.Message)
		assert.Equal(t, statusType, statusMsg.Type)
	})

	t.Run("ShowError", func(t *testing.T) {
		err := errors.New("critical error")
		context := "initialization"
		fatal := true

		cmd := ShowError(err, context, fatal)
		require.NotNil(t, cmd)

		msg := cmd()
		errorMsg, ok := msg.(ErrorMsg)
		require.True(t, ok)

		assert.Equal(t, err, errorMsg.Error)
		assert.Equal(t, context, errorMsg.Context)
		assert.Equal(t, fatal, errorMsg.Fatal)
	})

	t.Run("ShowSuccess", func(t *testing.T) {
		message := "Operation completed successfully"
		details := map[string]interface{}{
			"duration": "5s",
			"files":    3,
		}

		cmd := ShowSuccess(message, details)
		require.NotNil(t, cmd)

		msg := cmd()
		successMsg, ok := msg.(SuccessMsg)
		require.True(t, ok)

		assert.Equal(t, message, successMsg.Message)
		assert.Equal(t, details, successMsg.Details)
	})
}

// TestUtilityCommands tests utility command functions
func TestUtilityCommands(t *testing.T) {
	t.Run("Tick", func(t *testing.T) {
		duration := 100 * time.Millisecond

		cmd := Tick(duration)
		require.NotNil(t, cmd)

		// Tick returns a tea.Cmd from tea.Tick, so we can't easily test
		// the actual timing without running the tea program
		// We just verify it returns a non-nil command
	})

	t.Run("Every", func(t *testing.T) {
		duration := 100 * time.Millisecond
		callCount := 0
		fn := func() tea.Msg {
			callCount++
			return struct{}{}
		}

		cmd := Every(duration, fn)
		require.NotNil(t, cmd)

		// Every returns a tea.Cmd from tea.Every, so we can't easily test
		// the actual timing without running the tea program
		// We just verify it returns a non-nil command
	})

	t.Run("Sequence", func(t *testing.T) {
		cmd1 := func() tea.Msg { return "msg1" }
		cmd2 := func() tea.Msg { return "msg2" }

		cmd := Sequence(cmd1, cmd2)
		require.NotNil(t, cmd)

		// Sequence returns a tea.Cmd from tea.Sequence
		// We just verify it returns a non-nil command
	})

	t.Run("Batch", func(t *testing.T) {
		cmd1 := func() tea.Msg { return "msg1" }
		cmd2 := func() tea.Msg { return "msg2" }

		cmd := Batch(cmd1, cmd2)
		require.NotNil(t, cmd)

		// Batch returns a tea.Cmd from tea.Batch
		// We just verify it returns a non-nil command
	})
}

// TestMessageStructures tests message struct creation and field access
func TestMessageStructures(t *testing.T) {
	t.Run("WorkflowStartMsg", func(t *testing.T) {
		issueRef := &types.IssueReference{Owner: "test", Repo: "repo", Number: 1}
		stages := []WorkflowStage{{Name: "test"}}

		msg := WorkflowStartMsg{
			IssueRef: issueRef,
			Stages:   stages,
		}

		assert.Equal(t, issueRef, msg.IssueRef)
		assert.Equal(t, stages, msg.Stages)
	})

	t.Run("WorkflowCompleteMsg", func(t *testing.T) {
		duration := time.Minute
		results := map[string]interface{}{"key": "value"}

		msg := WorkflowCompleteMsg{
			Duration: duration,
			Results:  results,
		}

		assert.Equal(t, duration, msg.Duration)
		assert.Equal(t, results, msg.Results)
	})

	t.Run("ProcessOutputMsg", func(t *testing.T) {
		timestamp := time.Now()

		msg := ProcessOutputMsg{
			ProcessID: "proc-1",
			Line:      "output",
			Timestamp: timestamp,
			IsStderr:  true,
		}

		assert.Equal(t, "proc-1", msg.ProcessID)
		assert.Equal(t, "output", msg.Line)
		assert.Equal(t, timestamp, msg.Timestamp)
		assert.True(t, msg.IsStderr)
	})

	t.Run("UserInputRequestMsg", func(t *testing.T) {
		validation := func(s string) error { return nil }

		msg := UserInputRequestMsg{
			Prompt:      "Enter value:",
			Placeholder: "default",
			Required:    true,
			Validation:  validation,
		}

		assert.Equal(t, "Enter value:", msg.Prompt)
		assert.Equal(t, "default", msg.Placeholder)
		assert.True(t, msg.Required)
		assert.NotNil(t, msg.Validation)
	})
}

// TestComplexWorkflowScenarios tests more complex workflow scenarios
func TestComplexWorkflowScenarios(t *testing.T) {
	t.Run("WorkflowWithMultipleStages", func(t *testing.T) {
		// Create a workflow with multiple stages
		stages := []WorkflowStage{
			{Name: "prepare", Status: types.StagePending},
			{Name: "build", Status: types.StagePending},
			{Name: "test", Status: types.StagePending},
			{Name: "deploy", Status: types.StagePending},
		}

		issueRef := &types.IssueReference{
			Owner:  "example",
			Repo:   "project",
			Number: 42,
		}

		startCmd := WorkflowStart(issueRef, stages)
		msg := startCmd()
		workflowMsg := msg.(WorkflowStartMsg)

		assert.Len(t, workflowMsg.Stages, 4)
		assert.Equal(t, "prepare", workflowMsg.Stages[0].Name)
		assert.Equal(t, "deploy", workflowMsg.Stages[3].Name)
	})

	t.Run("StageProgressSequence", func(t *testing.T) {
		stageIndex := 1

		// Test progress updates from 0% to 100%
		progressValues := []float64{0.0, 0.25, 0.5, 0.75, 1.0}
		messages := []string{"Starting", "25% done", "Halfway", "Almost done", "Complete"}

		for i, progress := range progressValues {
			cmd := StageProgress(stageIndex, progress, messages[i])
			msg := cmd()
			progressMsg := msg.(StageProgressMsg)

			assert.Equal(t, stageIndex, progressMsg.StageIndex)
			assert.Equal(t, progress, progressMsg.Progress)
			assert.Equal(t, messages[i], progressMsg.Message)
		}
	})

	t.Run("ProcessLifecycle", func(t *testing.T) {
		processID := "build-process"
		command := "go"
		args := []string{"build", "-v", "./..."}

		// Start process
		startCmd := ProcessStart(processID, command, args)
		startMsg := startCmd().(ProcessStartMsg)
		assert.Equal(t, processID, startMsg.ProcessID)
		assert.Equal(t, command, startMsg.Command)
		assert.Equal(t, args, startMsg.Args)

		// Process output
		outputCmd := ProcessOutput(processID, "Building package...", false)
		outputMsg := outputCmd().(ProcessOutputMsg)
		assert.Equal(t, processID, outputMsg.ProcessID)
		assert.Equal(t, "Building package...", outputMsg.Line)
		assert.False(t, outputMsg.IsStderr)

		// Process exit
		duration := 10 * time.Second
		exitCmd := ProcessExited(processID, 0, duration)
		exitMsg := exitCmd().(ProcessExitedMsg)
		assert.Equal(t, processID, exitMsg.ProcessID)
		assert.Equal(t, 0, exitMsg.ExitCode)
		assert.Equal(t, duration, exitMsg.Duration)
	})
}
