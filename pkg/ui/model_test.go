package ui

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fumiya-kume/cca/internal/types"
)

// TestNewModel tests model creation with default configuration
func TestNewModel(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial state
	assert.Equal(t, StateInitial, model.state)
	assert.Equal(t, ctx, model.ctx)
	assert.Equal(t, -1, model.currentStage)
	assert.Equal(t, 0.0, model.overallProgress)

	// Test initial components are set up
	assert.NotNil(t, model.progress)
	assert.NotNil(t, model.viewport)
	assert.NotNil(t, model.textInput)
	assert.NotNil(t, model.theme)
	assert.NotNil(t, model.soundNotifier)

	// Test initial focus
	assert.Equal(t, FocusViewport, model.focused)

	// Test initial configuration
	assert.True(t, model.config.ShowTimestamps)
	assert.False(t, model.config.VerboseOutput)
	assert.Equal(t, 10000, model.config.ViewportBuffer)
	assert.True(t, model.config.AutoScroll)
	assert.False(t, model.config.CompactMode)
	assert.True(t, model.config.AnimationsEnabled)
	assert.True(t, model.config.SoundEnabled)

	// Test initial dimensions
	assert.Equal(t, 80, model.windowWidth)
	assert.Equal(t, 24, model.windowHeight)

	// Test initial UI state
	assert.False(t, model.showInput)
	assert.Empty(t, model.inputPrompt)
	assert.Empty(t, model.outputBuffer)
	assert.Empty(t, model.errorMessage)
	assert.Empty(t, model.statusMessage)
	assert.Empty(t, model.workflowStages)
}

// TestModelInit tests the Init method
func TestModelInit(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	cmd := model.Init()
	assert.NotNil(t, cmd, "Init should return a command")
}

// TestModelSetIssueRef tests setting issue reference
func TestModelSetIssueRef(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	issueRef := &types.IssueReference{
		Owner:  "test-owner",
		Repo:   "test-repo",
		Number: 42,
		Source: "url",
	}

	model.SetIssueRef(issueRef)
	assert.Equal(t, issueRef, model.issueRef)
}

// TestModelGetState tests state retrieval
func TestModelGetState(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial state
	assert.Equal(t, StateInitial, model.GetState())

	// Test state change
	model.state = StateWorkflowRunning
	assert.Equal(t, StateWorkflowRunning, model.GetState())
}

// TestModelIsWorkflowRunning tests workflow running status
func TestModelIsWorkflowRunning(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial state (not running)
	assert.False(t, model.IsWorkflowRunning())

	// Test running state
	model.state = StateWorkflowRunning
	assert.True(t, model.IsWorkflowRunning())

	// Test paused state (not considered running according to implementation)
	model.state = StateWorkflowPaused
	assert.False(t, model.IsWorkflowRunning())

	// Test completed state (not running)
	model.state = StateWorkflowCompleted
	assert.False(t, model.IsWorkflowRunning())

	// Test failed state (not running)
	model.state = StateWorkflowFailed
	assert.False(t, model.IsWorkflowRunning())
}

// TestModelGetCurrentStage tests current stage retrieval
func TestModelGetCurrentStage(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial current stage
	stageIndex := model.GetCurrentStage()
	assert.Equal(t, -1, stageIndex, "Should return -1 when no current stage")

	// Test with stages
	model.workflowStages = []WorkflowStage{
		{Name: "stage1", Status: types.StagePending},
		{Name: "stage2", Status: types.StageRunning},
	}
	model.currentStage = 1

	stageIndex = model.GetCurrentStage()
	assert.Equal(t, 1, stageIndex)

	// Get the actual stage from the slice
	stage := model.workflowStages[stageIndex]
	assert.Equal(t, "stage2", stage.Name)
	assert.Equal(t, types.StageRunning, stage.Status)
}

// TestModelGetProgress tests progress retrieval
func TestModelGetProgress(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial progress
	assert.Equal(t, 0.0, model.GetProgress())

	// Test with progress set
	model.overallProgress = 0.75
	assert.Equal(t, 0.75, model.GetProgress())
}

// TestModelSoundSettings tests sound notification settings
func TestModelSoundSettings(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial sound enabled state
	assert.True(t, model.IsSoundEnabled())

	// Test disabling sound
	model.SetSoundEnabled(false)
	assert.False(t, model.IsSoundEnabled())

	// Test enabling sound
	model.SetSoundEnabled(true)
	assert.True(t, model.IsSoundEnabled())
}

// TestModelView tests the View method
func TestModelView(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	view := model.View()
	assert.NotEmpty(t, view, "View should return non-empty string")
	assert.IsType(t, "", view, "View should return a string")
}

// TestApplicationStates tests all application states
func TestApplicationStates(t *testing.T) {
	states := []ApplicationState{
		StateInitial,
		StateWorkflowRunning,
		StateWorkflowPaused,
		StateWorkflowCompleted,
		StateWorkflowFailed,
		StateUserInput,
		StateShuttingDown,
	}

	for i, state := range states {
		assert.Equal(t, ApplicationState(i), state, "State constant should match index")
	}
}

// TestFocusStates tests all focus states
func TestFocusStates(t *testing.T) {
	focusStates := []FocusState{
		FocusViewport,
		FocusInput,
		FocusProgress,
		FocusStages,
	}

	for i, focusState := range focusStates {
		assert.Equal(t, FocusState(i), focusState, "Focus state constant should match index")
	}
}

// TestWorkflowStageStructure tests WorkflowStage structure
func TestWorkflowStageStructure(t *testing.T) {
	now := time.Now()
	stage := WorkflowStage{
		Name:      "test-stage",
		Status:    types.StageRunning,
		Output:    []string{"line1", "line2"},
		StartTime: now,
		EndTime:   now.Add(time.Minute),
		Error:     nil,
		Progress:  0.5,
	}

	assert.Equal(t, "test-stage", stage.Name)
	assert.Equal(t, types.StageRunning, stage.Status)
	assert.Equal(t, []string{"line1", "line2"}, stage.Output)
	assert.Equal(t, now, stage.StartTime)
	assert.Equal(t, now.Add(time.Minute), stage.EndTime)
	assert.Nil(t, stage.Error)
	assert.Equal(t, 0.5, stage.Progress)
}

// TestUIConfigStructure tests UIConfig structure
func TestUIConfigStructure(t *testing.T) {
	config := UIConfig{
		ShowTimestamps:    true,
		VerboseOutput:     false,
		ViewportBuffer:    5000,
		AutoScroll:        true,
		CompactMode:       false,
		AnimationsEnabled: true,
		SoundEnabled:      false,
	}

	assert.True(t, config.ShowTimestamps)
	assert.False(t, config.VerboseOutput)
	assert.Equal(t, 5000, config.ViewportBuffer)
	assert.True(t, config.AutoScroll)
	assert.False(t, config.CompactMode)
	assert.True(t, config.AnimationsEnabled)
	assert.False(t, config.SoundEnabled)
}

// TestModelWorkflowStageOperations tests workflow stage operations
func TestModelWorkflowStageOperations(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Initially no stages
	assert.Empty(t, model.workflowStages)
	assert.Equal(t, -1, model.currentStage)

	// Add stages
	stages := []WorkflowStage{
		{Name: "prepare", Status: types.StagePending},
		{Name: "build", Status: types.StagePending},
		{Name: "test", Status: types.StagePending},
	}
	model.workflowStages = stages
	model.currentStage = 0

	assert.Len(t, model.workflowStages, 3)
	assert.Equal(t, 0, model.currentStage)

	// Get current stage
	currentStageIndex := model.GetCurrentStage()
	assert.Equal(t, 0, currentStageIndex)
	currentStage := model.workflowStages[currentStageIndex]
	assert.Equal(t, "prepare", currentStage.Name)
	assert.Equal(t, types.StagePending, currentStage.Status)

	// Update current stage
	model.currentStage = 1
	currentStageIndex = model.GetCurrentStage()
	assert.Equal(t, 1, currentStageIndex)
	currentStage = model.workflowStages[currentStageIndex]
	assert.Equal(t, "build", currentStage.Name)
}

// TestModelOutputBuffer tests output buffer operations
func TestModelOutputBuffer(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Initially empty
	assert.Empty(t, model.outputBuffer)

	// Add output lines
	model.outputBuffer = append(model.outputBuffer, "Line 1")
	model.outputBuffer = append(model.outputBuffer, "Line 2")
	model.outputBuffer = append(model.outputBuffer, "Line 3")

	assert.Len(t, model.outputBuffer, 3)
	assert.Equal(t, "Line 1", model.outputBuffer[0])
	assert.Equal(t, "Line 2", model.outputBuffer[1])
	assert.Equal(t, "Line 3", model.outputBuffer[2])
}

// TestModelErrorAndStatusMessages tests error and status message handling
func TestModelErrorAndStatusMessages(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Initially empty
	assert.Empty(t, model.errorMessage)
	assert.Empty(t, model.statusMessage)

	// Set error message
	model.errorMessage = "Something went wrong"
	assert.Equal(t, "Something went wrong", model.errorMessage)

	// Set status message
	model.statusMessage = "Processing..."
	assert.Equal(t, "Processing...", model.statusMessage)

	// Clear messages
	model.errorMessage = ""
	model.statusMessage = ""
	assert.Empty(t, model.errorMessage)
	assert.Empty(t, model.statusMessage)
}

// TestModelInputHandling tests input handling functionality
func TestModelInputHandling(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Initially not showing input
	assert.False(t, model.showInput)
	assert.Empty(t, model.inputPrompt)

	// Show input with prompt
	model.showInput = true
	model.inputPrompt = "Enter your choice:"

	assert.True(t, model.showInput)
	assert.Equal(t, "Enter your choice:", model.inputPrompt)

	// Hide input
	model.showInput = false
	model.inputPrompt = ""

	assert.False(t, model.showInput)
	assert.Empty(t, model.inputPrompt)
}

// TestModelProgressHandling tests progress handling
func TestModelProgressHandling(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial progress
	assert.Equal(t, 0.0, model.overallProgress)

	// Test progress updates
	progressValues := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

	for _, progress := range progressValues {
		model.overallProgress = progress
		assert.Equal(t, progress, model.GetProgress())
	}
}

// TestModelWindowDimensions tests window dimension handling
func TestModelWindowDimensions(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial dimensions
	assert.Equal(t, 80, model.windowWidth)
	assert.Equal(t, 24, model.windowHeight)

	// Test dimension updates
	model.windowWidth = 120
	model.windowHeight = 40

	assert.Equal(t, 120, model.windowWidth)
	assert.Equal(t, 40, model.windowHeight)
}

// TestModelFocusManagement tests focus management
func TestModelFocusManagement(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test initial focus
	assert.Equal(t, FocusViewport, model.focused)

	// Test focus changes
	focusStates := []FocusState{
		FocusViewport,
		FocusInput,
		FocusProgress,
		FocusStages,
	}

	for _, focusState := range focusStates {
		model.focused = focusState
		assert.Equal(t, focusState, model.focused)
	}
}

// TestModelContextHandling tests context handling
func TestModelContextHandling(t *testing.T) {
	// Test with regular context
	ctx := context.Background()
	model := NewModel(ctx)
	assert.Equal(t, ctx, model.ctx)

	// Test with canceled context
	ctxWithCancel, cancel := context.WithCancel(context.Background())
	modelWithCancel := NewModel(ctxWithCancel)
	assert.Equal(t, ctxWithCancel, modelWithCancel.ctx)

	cancel() // Cancel the context
	assert.Error(t, modelWithCancel.ctx.Err())

	// Test with timeout context
	ctxWithTimeout, timeoutCancel := context.WithTimeout(context.Background(), time.Second)
	defer timeoutCancel()
	modelWithTimeout := NewModel(ctxWithTimeout)
	assert.Equal(t, ctxWithTimeout, modelWithTimeout.ctx)
}
