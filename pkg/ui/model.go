package ui

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fumiya-kume/cca/internal/types"
)

// FocusState represents which component currently has focus
type FocusState int

const (
	FocusViewport FocusState = iota
	FocusInput
	FocusProgress
	FocusStages
)

// ApplicationState represents the current state of the application
type ApplicationState int

const (
	StateInitial ApplicationState = iota
	StateWorkflowRunning
	StateWorkflowPaused
	StateWorkflowCompleted
	StateWorkflowFailed
	StateUserInput
	StateShuttingDown
)

// WorkflowStage represents a single workflow stage
type WorkflowStage struct {
	Name      string
	Status    types.StageStatus
	Output    []string
	StartTime time.Time
	EndTime   time.Time
	Error     error
	Progress  float64 // 0.0 to 1.0
}

// Model represents the main application model following the Elm Architecture
type Model struct {
	// Core application state
	ctx             context.Context
	state           ApplicationState
	issueRef        *types.IssueReference
	workflowStages  []WorkflowStage
	currentStage    int
	overallProgress float64

	// UI components
	progress  progress.Model
	viewport  viewport.Model
	textInput textinput.Model

	// Layout and focus management
	focused      FocusState
	windowWidth  int
	windowHeight int

	// Theme and styling
	theme Theme

	// UI state
	showInput     bool
	inputPrompt   string
	outputBuffer  []string
	errorMessage  string
	statusMessage string

	// Configuration
	config UIConfig

	// Sound notifications
	soundNotifier *SoundNotifier

	// Synchronization for concurrent access
	mutex sync.RWMutex
}

// UIConfig holds UI configuration options
type UIConfig struct {
	ShowTimestamps    bool
	VerboseOutput     bool
	ViewportBuffer    int
	AutoScroll        bool
	CompactMode       bool
	AnimationsEnabled bool
	SoundEnabled      bool
}

// NewModel creates a new application model with default configuration
func NewModel(ctx context.Context) Model {
	// Initialize progress bar
	progressModel := progress.New(progress.WithDefaultGradient())
	progressModel.ShowPercentage = true
	progressModel.Width = 40

	// Initialize viewport for output
	viewportModel := viewport.New(80, 20)
	viewportModel.MouseWheelEnabled = true

	// Initialize text input
	textInputModel := textinput.New()
	textInputModel.Placeholder = "Enter command or response..."
	textInputModel.CharLimit = 500
	textInputModel.Width = 80

	// Default theme
	theme := NewDarkTheme()

	// Default configuration
	config := UIConfig{
		ShowTimestamps:    true,
		VerboseOutput:     false,
		ViewportBuffer:    10000,
		AutoScroll:        true,
		CompactMode:       false,
		AnimationsEnabled: true,
		SoundEnabled:      true,
	}

	// Initialize sound notifier
	soundConfig := DefaultSoundConfig()
	soundNotifier := NewSoundNotifier(soundConfig)

	return Model{
		ctx:             ctx,
		state:           StateInitial,
		workflowStages:  []WorkflowStage{},
		currentStage:    -1,
		overallProgress: 0.0,
		progress:        progressModel,
		viewport:        viewportModel,
		textInput:       textInputModel,
		focused:         FocusViewport,
		windowWidth:     80,
		windowHeight:    24,
		theme:           theme,
		showInput:       false,
		outputBuffer:    []string{},
		config:          config,
		soundNotifier:   soundNotifier,
	}
}

// Init implements tea.Model interface
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.progress.Init(),
	)
}

// Update implements tea.Model interface - handles all messages and state updates
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		cmd := m.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case WorkflowStartMsg:
		m.handleWorkflowStartMsg(msg)

	case WorkflowCompleteMsg:
		m.handleWorkflowCompleteMsg(msg)

	case WorkflowErrorMsg:
		m.handleWorkflowErrorMsg(msg)

	case StageStartMsg:
		m.handleStageStartMsg(msg)

	case StageCompleteMsg:
		m.handleStageCompleteMsg(msg)

	case StageErrorMsg:
		m.handleStageErrorMsg(msg)

	case ProcessOutputMsg:
		m.addOutputLine(msg.Line)

	case UserInputRequestMsg:
		m.handleUserInputRequestMsg(msg)

	case UserInputResponseMsg:
		return m.handleUserInputResponseMsg(msg)
	}

	// Update child components
	cmds = append(cmds, m.updateChildComponents(msg)...)

	return m, tea.Batch(cmds...)
}

// handleWindowSizeMsg handles window size changes
func (m *Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	m.windowWidth = msg.Width
	m.windowHeight = msg.Height
	*m = *m.updateLayout()
}

// handleWorkflowStartMsg handles workflow start messages
func (m *Model) handleWorkflowStartMsg(msg WorkflowStartMsg) {
	m.mutex.Lock()
	m.state = StateWorkflowRunning
	m.issueRef = msg.IssueRef
	m.workflowStages = msg.Stages
	m.currentStage = 0
	m.overallProgress = 0.0
	m.statusMessage = "Workflow started"
	m.mutex.Unlock()
	m.addOutputLine("🚀 Starting workflow for issue #" + string(rune(msg.IssueRef.Number)))
}

// handleWorkflowCompleteMsg handles workflow completion messages
func (m *Model) handleWorkflowCompleteMsg(msg WorkflowCompleteMsg) {
	m.mutex.Lock()
	m.state = StateWorkflowCompleted
	m.overallProgress = 1.0
	m.statusMessage = "Workflow completed successfully"
	soundEnabled := m.config.SoundEnabled
	m.mutex.Unlock()
	m.addOutputLine("✅ Workflow completed successfully!")
	
	// Play workflow complete sound
	if soundEnabled && m.soundNotifier != nil {
		m.soundNotifier.PlayWorkflowCompleteSound()
	}
}

// handleWorkflowErrorMsg handles workflow error messages
func (m *Model) handleWorkflowErrorMsg(msg WorkflowErrorMsg) {
	m.state = StateWorkflowFailed
	m.errorMessage = msg.Error.Error()
	m.statusMessage = "Workflow failed"
	m.addOutputLine("❌ Workflow failed: " + msg.Error.Error())
	
	// Play error sound
	if m.config.SoundEnabled && m.soundNotifier != nil {
		m.soundNotifier.PlayErrorSound()
	}
}

// handleStageStartMsg handles stage start messages
func (m *Model) handleStageStartMsg(msg StageStartMsg) {
	if msg.StageIndex < len(m.workflowStages) {
		m.workflowStages[msg.StageIndex].Status = types.StageRunning
		m.workflowStages[msg.StageIndex].StartTime = time.Now()
		m.currentStage = msg.StageIndex
		m.updateProgress()
		m.addOutputLine("▶️  " + m.workflowStages[msg.StageIndex].Name)
	}
}

// handleStageCompleteMsg handles stage completion messages
func (m *Model) handleStageCompleteMsg(msg StageCompleteMsg) {
	if msg.StageIndex < len(m.workflowStages) {
		m.workflowStages[msg.StageIndex].Status = types.StageCompleted
		m.workflowStages[msg.StageIndex].EndTime = time.Now()
		m.updateProgress()
		m.addOutputLine("✅ " + m.workflowStages[msg.StageIndex].Name + " completed")
		
		// Play success sound for stage completion
		if m.config.SoundEnabled && m.soundNotifier != nil {
			m.soundNotifier.PlaySuccessSound()
		}
	}
}

// handleStageErrorMsg handles stage error messages
func (m *Model) handleStageErrorMsg(msg StageErrorMsg) {
	if msg.StageIndex < len(m.workflowStages) {
		m.workflowStages[msg.StageIndex].Status = types.StageFailed
		m.workflowStages[msg.StageIndex].Error = msg.Error
		m.workflowStages[msg.StageIndex].EndTime = time.Now()
		m.addOutputLine("❌ " + m.workflowStages[msg.StageIndex].Name + " failed: " + msg.Error.Error())
	}
}

// handleUserInputRequestMsg handles user input request messages
func (m *Model) handleUserInputRequestMsg(msg UserInputRequestMsg) {
	m.state = StateUserInput
	m.showInput = true
	m.inputPrompt = msg.Prompt
	m.textInput.SetValue("")
	m.textInput.Focus()
	m.focused = FocusInput
	
	// Play confirmation sound when user input is requested
	if m.config.SoundEnabled && m.soundNotifier != nil {
		m.soundNotifier.PlayConfirmationSound()
	}
}

// handleUserInputResponseMsg handles user input response messages
func (m *Model) handleUserInputResponseMsg(msg UserInputResponseMsg) (tea.Model, tea.Cmd) {
	m.state = StateWorkflowRunning
	m.showInput = false
	m.textInput.Blur()
	m.focused = FocusViewport
	return m, UserInputSubmitted(msg.Response)
}

// updateChildComponents updates all child components and returns their commands
func (m *Model) updateChildComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	// Update progress component
	progressModel, progressCmd := m.progress.Update(msg)
	if p, ok := progressModel.(progress.Model); ok {
		m.progress = p
	}
	if progressCmd != nil {
		cmds = append(cmds, progressCmd)
	}

	// Update viewport component
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Update text input component if shown
	if m.showInput {
		m.textInput, cmd = m.textInput.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}

// handleKeyPress handles keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	// Check for key-specific handlers
	handlers := map[string]func() tea.Cmd{
		"ctrl+c": m.handleQuitKeys,
		"q":      m.handleQuitKeys,
		"tab":    m.handleTabKey,
		"shift+tab": m.handleShiftTabKey,
		"ctrl+r": m.handleRetryKey,
		"enter":  m.handleEnterKey,
		"esc":    m.handleEscapeKey,
		"ctrl+s": m.handleSoundToggleKey,
	}

	if handler, exists := handlers[key]; exists {
		return handler()
	}

	// Handle navigation keys
	if m.isNavigationKey(key) {
		return m.handleNavigationKey()
	}

	return nil
}

// handleQuitKeys handles quit key combinations
func (m *Model) handleQuitKeys() tea.Cmd {
	if m.state == StateWorkflowRunning {
		return WorkflowCancel()
	}
	return tea.Quit
}

// handleTabKey handles tab key for focus navigation
func (m *Model) handleTabKey() tea.Cmd {
	m.focused = m.nextFocus()
	return nil
}

// handleShiftTabKey handles shift+tab key for reverse focus navigation
func (m *Model) handleShiftTabKey() tea.Cmd {
	m.focused = m.prevFocus()
	return nil
}

// handleRetryKey handles retry key combination
func (m *Model) handleRetryKey() tea.Cmd {
	if m.state == StateWorkflowFailed {
		return WorkflowRetry()
	}
	return nil
}

// handleEnterKey handles enter key for input submission
func (m *Model) handleEnterKey() tea.Cmd {
	if m.showInput && m.focused == FocusInput {
		response := m.textInput.Value()
		return func() tea.Msg {
			return UserInputResponseMsg{Response: response}
		}
	}
	return nil
}

// handleEscapeKey handles escape key for canceling input
func (m *Model) handleEscapeKey() tea.Cmd {
	if m.showInput {
		m.showInput = false
		m.textInput.Blur()
		m.focused = FocusViewport
	}
	return nil
}

// handleSoundToggleKey handles sound toggle key combination
func (m *Model) handleSoundToggleKey() tea.Cmd {
	m.config.SoundEnabled = !m.config.SoundEnabled
	if m.soundNotifier != nil {
		m.soundNotifier.SetEnabled(m.config.SoundEnabled)
	}

	if m.config.SoundEnabled {
		m.addOutputLine("🔊 Sound notifications enabled")
		if m.soundNotifier != nil {
			m.soundNotifier.PlayConfirmationSound()
		}
	} else {
		m.addOutputLine("🔇 Sound notifications disabled")
	}
	return nil
}

// isNavigationKey checks if the key is a navigation key
func (m *Model) isNavigationKey(key string) bool {
	navKeys := []string{"up", "down", "pgup", "pgdown", "home", "end"}
	for _, navKey := range navKeys {
		if key == navKey {
			return true
		}
	}
	return false
}

// handleNavigationKey handles navigation keys for viewport
func (m *Model) handleNavigationKey() tea.Cmd {
	if m.focused == FocusViewport {
		// Let viewport handle these keys
		return nil
	}
	return nil
}

// nextFocus returns the next focus state
func (m *Model) nextFocus() FocusState {
	switch m.focused {
	case FocusViewport:
		if m.showInput {
			return FocusInput
		}
		return FocusProgress
	case FocusInput:
		return FocusProgress
	case FocusProgress:
		return FocusStages
	case FocusStages:
		return FocusViewport
	default:
		return FocusViewport
	}
}

// prevFocus returns the previous focus state
func (m *Model) prevFocus() FocusState {
	switch m.focused {
	case FocusViewport:
		return FocusStages
	case FocusInput:
		return FocusViewport
	case FocusProgress:
		if m.showInput {
			return FocusInput
		}
		return FocusViewport
	case FocusStages:
		return FocusProgress
	default:
		return FocusViewport
	}
}

// updateLayout adjusts component sizes based on window dimensions
func (m *Model) updateLayout() *Model {
	// Responsive layout calculations
	if m.windowWidth < 80 {
		m.config.CompactMode = true
	} else {
		m.config.CompactMode = false
	}

	// Update viewport size
	headerHeight := 3 // Title and progress
	footerHeight := 1 // Status line
	stagesHeight := 5 // Stages panel
	inputHeight := 0

	if m.showInput {
		inputHeight = 3
	}

	availableHeight := m.windowHeight - headerHeight - footerHeight - inputHeight
	if !m.config.CompactMode {
		availableHeight -= stagesHeight
	}

	if availableHeight < 5 {
		availableHeight = 5
	}

	m.viewport.Width = m.windowWidth - 4 // Account for borders
	m.viewport.Height = availableHeight

	// Update progress bar width
	progressWidth := m.windowWidth - 20 // Leave space for percentage
	if progressWidth < 20 {
		progressWidth = 20
	}
	m.progress.Width = progressWidth

	// Update text input width
	m.textInput.Width = m.windowWidth - 4

	return m
}

// updateProgress calculates and updates the overall progress
func (m *Model) updateProgress() {
	if len(m.workflowStages) == 0 {
		m.overallProgress = 0.0
		return
	}

	completed := 0
	for _, stage := range m.workflowStages {
		if stage.Status == types.StageCompleted {
			completed++
		}
	}

	m.overallProgress = float64(completed) / float64(len(m.workflowStages))
}

// addOutputLine adds a new line to the output buffer and viewport
func (m *Model) addOutputLine(line string) {
	if m.config.ShowTimestamps {
		timestamp := time.Now().Format("15:04:05")
		line = "[" + timestamp + "] " + line
	}

	m.outputBuffer = append(m.outputBuffer, line)

	// Limit buffer size
	if len(m.outputBuffer) > m.config.ViewportBuffer {
		m.outputBuffer = m.outputBuffer[len(m.outputBuffer)-m.config.ViewportBuffer:]
	}

	// Update viewport content
	content := ""
	for _, outputLine := range m.outputBuffer {
		content += outputLine + "\n"
	}
	m.viewport.SetContent(content)

	// Auto-scroll to bottom if enabled
	if m.config.AutoScroll {
		m.viewport.GotoBottom()
	}
}

// View implements tea.Model interface - renders the entire UI
func (m *Model) View() string {
	if m.windowWidth == 0 || m.windowHeight == 0 {
		return "Initializing..."
	}

	if m.config.CompactMode {
		return m.renderCompactView()
	}
	return m.renderFullView()
}

// SetIssueRef sets the issue reference for the workflow
func (m *Model) SetIssueRef(issueRef *types.IssueReference) {
	m.issueRef = issueRef
}

// GetState returns the current application state (thread-safe)
func (m *Model) GetState() ApplicationState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state
}

// IsWorkflowRunning returns true if a workflow is currently running (thread-safe)
func (m *Model) IsWorkflowRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state == StateWorkflowRunning
}

// GetCurrentStage returns the index of the current workflow stage (thread-safe)
func (m *Model) GetCurrentStage() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.currentStage
}

// GetProgress returns the overall workflow progress (thread-safe)
func (m *Model) GetProgress() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.overallProgress
}

// SetSoundEnabled enables or disables sound notifications (thread-safe)
func (m *Model) SetSoundEnabled(enabled bool) {
	m.mutex.Lock()
	m.config.SoundEnabled = enabled
	m.mutex.Unlock()
	if m.soundNotifier != nil {
		m.soundNotifier.SetEnabled(enabled)
	}
}

// IsSoundEnabled returns whether sound notifications are enabled (thread-safe)
func (m *Model) IsSoundEnabled() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.SoundEnabled
}
