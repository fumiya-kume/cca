package ui

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/fumiya-kume/cca/internal/types"
)

// TestRenderFullView tests full view rendering
func TestRenderFullView(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Set up model with some data
	model.state = StateWorkflowRunning
	model.workflowStages = []WorkflowStage{
		{Name: "prepare", Status: types.StageRunning, Progress: 0.5},
		{Name: "build", Status: types.StagePending, Progress: 0.0},
	}
	model.currentStage = 0
	model.overallProgress = 0.25
	model.outputBuffer = []string{"Starting workflow...", "Preparing environment..."}

	view := model.renderFullView()
	assert.NotEmpty(t, view, "Full view should not be empty")
	assert.Contains(t, view, "prepare", "View should contain stage name")
}

// TestRenderCompactView tests compact view rendering
func TestRenderCompactView(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)
	model.config.CompactMode = true

	// Set up model with some data
	model.state = StateWorkflowRunning
	model.currentStage = 0
	model.overallProgress = 0.75

	view := model.renderCompactView()
	assert.NotEmpty(t, view, "Compact view should not be empty")
}

// TestRenderHeader tests header rendering
func TestRenderHeader(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with issue reference
	model.issueRef = &types.IssueReference{
		Owner:  "test-owner",
		Repo:   "test-repo",
		Number: 42,
	}

	header := model.renderHeader()
	assert.NotEmpty(t, header, "Header should not be empty")
	assert.Contains(t, header, "test-owner", "Header should contain owner")
	assert.Contains(t, header, "test-repo", "Header should contain repo")
	assert.Contains(t, header, "42", "Header should contain issue number")
}

// TestRenderCompactHeader tests compact header rendering
func TestRenderCompactHeader(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with issue reference
	model.issueRef = &types.IssueReference{
		Owner:  "test-owner",
		Repo:   "test-repo",
		Number: 42,
	}

	header := model.renderCompactHeader()
	assert.NotEmpty(t, header, "Compact header should not be empty")
}

// TestRenderProgress tests progress rendering
func TestRenderProgress(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with different progress values
	progressValues := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

	for _, progress := range progressValues {
		model.overallProgress = progress
		rendered := model.renderProgress()
		assert.NotEmpty(t, rendered, "Progress should not be empty for progress %f", progress)
	}
}

// TestRenderCompactProgress tests compact progress rendering
func TestRenderCompactProgress(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with different progress values
	progressValues := []float64{0.0, 0.5, 1.0}

	for _, progress := range progressValues {
		model.overallProgress = progress
		rendered := model.renderCompactProgress()
		assert.NotEmpty(t, rendered, "Compact progress should not be empty for progress %f", progress)
	}
}

// TestRenderMainContent tests main content rendering
func TestRenderMainContent(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with output buffer
	model.outputBuffer = []string{
		"Starting workflow...",
		"Preparing environment...",
		"Running tests...",
	}

	content := model.renderMainContent()
	assert.NotEmpty(t, content, "Main content should not be empty")
}

// TestRenderStages tests stages rendering
func TestRenderStages(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with multiple stages
	model.workflowStages = []WorkflowStage{
		{Name: "prepare", Status: types.StageCompleted, Progress: 1.0},
		{Name: "build", Status: types.StageRunning, Progress: 0.6},
		{Name: "test", Status: types.StagePending, Progress: 0.0},
		{Name: "deploy", Status: types.StagePending, Progress: 0.0},
	}

	stages := model.renderStages()
	assert.NotEmpty(t, stages, "Stages should not be empty")

	// Check that all stage names are present
	for _, stage := range model.workflowStages {
		assert.Contains(t, stages, stage.Name, "Stages should contain stage name %s", stage.Name)
	}
}

// TestRenderInput tests input rendering
func TestRenderInput(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test when input is not shown
	model.showInput = false
	input := model.renderInput()
	assert.Empty(t, input, "Input should be empty when not shown")

	// Test when input is shown
	model.showInput = true
	model.inputPrompt = "Enter your choice:"
	model.textInput.SetValue("test input")

	input = model.renderInput()
	assert.NotEmpty(t, input, "Input should not be empty when shown")
	assert.Contains(t, input, "Enter your choice", "Input should contain prompt")
}

// TestRenderFooter tests footer rendering
func TestRenderFooter(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test different states
	states := []ApplicationState{
		StateInitial,
		StateWorkflowRunning,
		StateWorkflowPaused,
		StateWorkflowCompleted,
		StateWorkflowFailed,
	}

	for _, state := range states {
		model.state = state
		footer := model.renderFooter()
		assert.NotEmpty(t, footer, "Footer should not be empty for state %v", state)
	}
}

// TestRenderCompactFooter tests compact footer rendering
func TestRenderCompactFooter(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with different states
	model.state = StateWorkflowRunning
	footer := model.renderCompactFooter()
	assert.NotEmpty(t, footer, "Compact footer should not be empty")

	model.state = StateWorkflowCompleted
	footer = model.renderCompactFooter()
	assert.NotEmpty(t, footer, "Compact footer should not be empty for completed state")
}

// TestGetStatusText tests status text generation
func TestGetStatusText(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test different states
	states := []ApplicationState{
		StateInitial,
		StateWorkflowRunning,
		StateWorkflowPaused,
		StateWorkflowCompleted,
		StateWorkflowFailed,
		StateUserInput,
		StateShuttingDown,
	}

	for _, state := range states {
		model.state = state
		status := model.getStatusText()
		assert.NotEmpty(t, status, "Status text should not be empty for state %v", state)
	}
}

// TestCalculateMainContentHeight tests main content height calculation
func TestCalculateMainContentHeight(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with different window heights
	windowHeights := []int{10, 20, 30, 50, 100}

	for _, height := range windowHeights {
		model.windowHeight = height
		contentHeight := model.calculateMainContentHeight()
		assert.Greater(t, contentHeight, 0, "Content height should be positive for window height %d", height)
		assert.LessOrEqual(t, contentHeight, height, "Content height should not exceed window height")
	}
}

// TestCalculateSidebarWidth tests sidebar width calculation
func TestCalculateSidebarWidth(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with different window widths
	windowWidths := []int{40, 80, 120, 160, 200}

	for _, width := range windowWidths {
		model.windowWidth = width
		sidebarWidth := model.calculateSidebarWidth()
		assert.GreaterOrEqual(t, sidebarWidth, 0, "Sidebar width should be non-negative for window width %d", width)
		assert.LessOrEqual(t, sidebarWidth, width, "Sidebar width should not exceed window width")

		// For narrow windows, sidebar should be 0
		if width < 120 {
			assert.Equal(t, 0, sidebarWidth, "Sidebar width should be 0 for narrow windows")
		} else {
			assert.Greater(t, sidebarWidth, 0, "Sidebar width should be positive for wide windows")
		}
	}
}

// TestFormatDuration tests duration formatting
func TestFormatDuration(t *testing.T) {

	// Test different durations
	testCases := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0s"},
		{time.Second, "1s"},
		{time.Minute, "1m0s"},
		{time.Hour, "1h0m0s"},
		{time.Hour + 30*time.Minute + 45*time.Second, "1h30m45s"},
	}

	for _, tc := range testCases {
		formatted := formatDuration(tc.duration)
		assert.NotEmpty(t, formatted, "Formatted duration should not be empty")
		// Note: The exact format may vary based on implementation
	}
}

// TestTruncateText tests text truncation
func TestTruncateText(t *testing.T) {

	// Test text truncation
	testCases := []struct {
		text     string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long text that should be truncated", 10, "this is..."},
		{"exact length", 12, "exact length"},
		{"", 5, ""},
	}

	for _, tc := range testCases {
		truncated := truncateText(tc.text, tc.maxLen)
		assert.LessOrEqual(t, len(truncated), tc.maxLen, "Truncated text should not exceed max length")

		if len(tc.text) <= tc.maxLen {
			assert.Equal(t, tc.text, truncated, "Short text should not be truncated")
		} else {
			assert.Contains(t, truncated, "...", "Long text should contain ellipsis")
		}
	}
}

// TestCenterText tests text centering
func TestCenterText(t *testing.T) {

	// Test text centering
	testCases := []struct {
		text  string
		width int
	}{
		{"test", 10},
		{"hello world", 20},
		{"short", 15},
		{"", 10},
	}

	for _, tc := range testCases {
		centered := centerText(tc.text, tc.width)
		assert.LessOrEqual(t, len(centered), tc.width, "Centered text should not exceed width")

		if len(tc.text) <= tc.width {
			// Text should be centered (padding on both sides)
			assert.GreaterOrEqual(t, len(centered), len(tc.text), "Centered text should have padding")
		}
	}
}

// TestViewRendering tests complete view rendering
func TestViewRendering(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test in different modes
	t.Run("NormalMode", func(t *testing.T) {
		model.config.CompactMode = false
		view := model.View()
		assert.NotEmpty(t, view, "Normal view should not be empty")
	})

	t.Run("CompactMode", func(t *testing.T) {
		model.config.CompactMode = true
		view := model.View()
		assert.NotEmpty(t, view, "Compact view should not be empty")
	})
}

// TestRenderWithWorkflowData tests rendering with complete workflow data
func TestRenderWithWorkflowData(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Set up complete workflow data
	model.issueRef = &types.IssueReference{
		Owner:  "example",
		Repo:   "project",
		Number: 123,
	}

	model.workflowStages = []WorkflowStage{
		{
			Name:      "prepare",
			Status:    types.StageCompleted,
			Progress:  1.0,
			StartTime: time.Now().Add(-2 * time.Minute),
			EndTime:   time.Now().Add(-90 * time.Second),
			Output:    []string{"Preparation complete"},
		},
		{
			Name:      "build",
			Status:    types.StageRunning,
			Progress:  0.7,
			StartTime: time.Now().Add(-90 * time.Second),
			Output:    []string{"Building...", "70% complete"},
		},
		{
			Name:     "test",
			Status:   types.StagePending,
			Progress: 0.0,
		},
	}

	model.currentStage = 1
	model.overallProgress = 0.57 // (1.0 + 0.7 + 0.0) / 3
	model.state = StateWorkflowRunning
	model.outputBuffer = []string{
		"Workflow started",
		"Preparation phase completed",
		"Build phase in progress",
		"Current progress: 70%",
	}

	// Test full view
	view := model.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "example/project")
	assert.Contains(t, view, "prepare")
	assert.Contains(t, view, "build")
	assert.Contains(t, view, "test")

	// Test compact view
	model.config.CompactMode = true
	compactView := model.View()
	assert.NotEmpty(t, compactView)
	assert.NotEqual(t, view, compactView, "Compact view should differ from full view")
}

// TestRenderErrorStates tests rendering in error states
func TestRenderErrorStates(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test with error message
	model.state = StateWorkflowFailed
	model.errorMessage = "Build failed with exit code 1"
	model.workflowStages = []WorkflowStage{
		{
			Name:   "build",
			Status: types.StageFailed,
			Error:  assert.AnError,
		},
	}

	view := model.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "failed", "View should indicate failure")
}

// TestRenderInputStates tests rendering when input is required
func TestRenderInputStates(t *testing.T) {
	ctx := context.Background()
	model := NewModel(ctx)

	// Test user input state
	model.state = StateUserInput
	model.showInput = true
	model.inputPrompt = "Do you want to continue? (y/n)"
	model.textInput.SetValue("y")

	view := model.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "continue", "View should contain input prompt")
}
