package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fumiya-kume/cca/internal/types"
)

// renderFullView renders the complete UI layout
func (m *Model) renderFullView() string {
	header := m.renderHeader()
	progress := m.renderProgress()
	main := m.renderMainContent()
	stages := m.renderStages()
	input := m.renderInput()
	footer := m.renderFooter()

	// Layout sections
	sections := []string{header, progress}

	// Main content area - split between output and stages
	if m.windowWidth >= 120 {
		// Wide layout: side-by-side
		mainContent := lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.Styles.Panel.Width(m.windowWidth*2/3).Render(main),
			m.theme.Styles.Panel.Width(m.windowWidth/3-4).Render(stages),
		)
		sections = append(sections, mainContent)
	} else {
		// Narrow layout: stacked
		sections = append(sections, m.theme.Styles.Panel.Render(main))
		sections = append(sections, m.theme.Styles.Panel.Render(stages))
	}

	if m.showInput {
		sections = append(sections, input)
	}

	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderCompactView renders a compact UI for smaller terminals
func (m *Model) renderCompactView() string {
	title := m.renderCompactHeader()
	progress := m.renderCompactProgress()
	main := m.renderMainContent()
	input := m.renderInput()
	footer := m.renderCompactFooter()

	sections := []string{title, progress, main}

	if m.showInput {
		sections = append(sections, input)
	}

	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the application header
func (m *Model) renderHeader() string {
	title := "ccAgents"
	if m.issueRef != nil {
		title += fmt.Sprintf(" - Issue #%d (%s/%s)",
			m.issueRef.Number, m.issueRef.Owner, m.issueRef.Repo)
	}

	headerStyle := m.theme.Styles.Title
	if m.focused == FocusViewport {
		headerStyle = m.theme.Styles.PanelFocus
	}

	return headerStyle.Width(m.windowWidth).Render(title)
}

// renderCompactHeader renders a compact header
func (m *Model) renderCompactHeader() string {
	title := "ccAgents"
	if m.issueRef != nil {
		title += fmt.Sprintf(" #%d", m.issueRef.Number)
	}

	return m.theme.Styles.Title.Render(title)
}

// renderProgress renders the progress bar and status
func (m *Model) renderProgress() string {
	progressBar := m.progress.ViewAs(m.overallProgress)

	status := m.getStatusText()
	statusStyle := m.theme.Styles.StatusInfo

	switch m.state {
	case StateWorkflowCompleted:
		statusStyle = m.theme.Styles.StatusSuccess
	case StateWorkflowFailed:
		statusStyle = m.theme.Styles.StatusError
	case StateWorkflowRunning:
		statusStyle = m.theme.Styles.StatusInfo
	}

	progressSection := lipgloss.JoinVertical(
		lipgloss.Left,
		progressBar,
		statusStyle.Render(status),
	)

	panelStyle := m.theme.Styles.Panel
	if m.focused == FocusProgress {
		panelStyle = m.theme.Styles.PanelFocus
	}

	return panelStyle.Width(m.windowWidth).Render(progressSection)
}

// renderCompactProgress renders a compact progress bar
func (m *Model) renderCompactProgress() string {
	progress := fmt.Sprintf("%.0f%%", m.overallProgress*100)
	status := m.getStatusText()

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.theme.Styles.StatusInfo.Render(progress),
		" ",
		m.theme.Styles.StatusInfo.Render(status),
	)
}

// renderMainContent renders the main viewport content
func (m *Model) renderMainContent() string {
	viewportStyle := m.theme.Styles.Output
	if m.focused == FocusViewport {
		viewportStyle = m.theme.Styles.PanelFocus
	}

	content := m.viewport.View()
	if content == "" {
		content = m.theme.Styles.StatusInfo.Render("Ready to start workflow...")
	}

	return viewportStyle.Render(content)
}

// renderStages renders the workflow stages panel
func (m *Model) renderStages() string {
	if len(m.workflowStages) == 0 {
		return m.theme.Styles.Panel.Render("No stages defined")
	}

	var stageLines []string
	stageLines = append(stageLines, m.theme.Styles.Bold.Render("Workflow Stages:"))
	stageLines = append(stageLines, "")

	for i, stage := range m.workflowStages {
		icon := GetStageStatusIcon(stage.Status)
		color := m.theme.GetStageStatusColor(stage.Status)

		stageStyle := m.theme.Styles.Stage
		if m.focused == FocusStages && i == m.currentStage {
			stageStyle = m.theme.Styles.StageFocus
		}

		stageLine := fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(color).Render(icon),
			stage.Name)

		// Add timing information
		if stage.Status == types.StageCompleted && !stage.EndTime.IsZero() {
			duration := stage.EndTime.Sub(stage.StartTime)
			stageLine += m.theme.Styles.StatusSuccess.Render(fmt.Sprintf(" (%.1fs)", duration.Seconds()))
		} else if stage.Status == types.StageRunning {
			stageLine += m.theme.Styles.StatusInfo.Render(" (running...)")
		} else if stage.Status == types.StageFailed {
			stageLine += m.theme.Styles.StatusError.Render(" (failed)")
		}

		stageLines = append(stageLines, stageStyle.Render(stageLine))
	}

	panelStyle := m.theme.Styles.Panel
	if m.focused == FocusStages {
		panelStyle = m.theme.Styles.PanelFocus
	}

	return panelStyle.Render(strings.Join(stageLines, "\n"))
}

// renderInput renders the user input area when needed
func (m *Model) renderInput() string {
	if !m.showInput {
		return ""
	}

	prompt := m.inputPrompt
	if prompt == "" {
		prompt = "Input required:"
	}

	inputStyle := m.theme.Styles.Input
	if m.focused == FocusInput {
		inputStyle = m.theme.Styles.PanelFocus
	}

	inputSection := lipgloss.JoinVertical(
		lipgloss.Left,
		m.theme.Styles.Bold.Render(prompt),
		m.textInput.View(),
		m.theme.Styles.StatusInfo.Render("Press Enter to submit, Esc to cancel"),
	)

	return inputStyle.Width(m.windowWidth).Render(inputSection)
}

// renderFooter renders the application footer with shortcuts
func (m *Model) renderFooter() string {
	shortcuts := []string{}

	switch m.state {
	case StateWorkflowRunning:
		shortcuts = append(shortcuts, "Ctrl+C: Cancel")
		if m.showInput {
			shortcuts = append(shortcuts, "Enter: Submit", "Esc: Cancel Input")
		}
	case StateWorkflowFailed:
		shortcuts = append(shortcuts, "Ctrl+R: Retry", "Q: Quit")
	case StateWorkflowCompleted:
		shortcuts = append(shortcuts, "Q: Quit")
	default:
		shortcuts = append(shortcuts, "Q: Quit")
	}

	shortcuts = append(shortcuts, "Tab: Focus")

	footerText := strings.Join(shortcuts, " • ")
	return m.theme.Styles.Footer.Width(m.windowWidth).Render(footerText)
}

// renderCompactFooter renders a compact footer
func (m *Model) renderCompactFooter() string {
	var shortcut string
	switch m.state {
	case StateWorkflowRunning:
		shortcut = "Ctrl+C: Cancel"
	case StateWorkflowFailed:
		shortcut = "Ctrl+R: Retry"
	default:
		shortcut = "Q: Quit"
	}

	return m.theme.Styles.Footer.Render(shortcut)
}

// getStatusText returns the current status text
func (m *Model) getStatusText() string {
	if m.errorMessage != "" {
		return "Error: " + m.errorMessage
	}

	if m.statusMessage != "" {
		return m.statusMessage
	}

	switch m.state {
	case StateInitial:
		return "Ready"
	case StateWorkflowRunning:
		if m.currentStage >= 0 && m.currentStage < len(m.workflowStages) {
			return "Running: " + m.workflowStages[m.currentStage].Name
		}
		return "Running workflow..."
	case StateWorkflowPaused:
		return "Paused"
	case StateWorkflowCompleted:
		return "Completed successfully"
	case StateWorkflowFailed:
		return "Failed"
	case StateUserInput:
		return "Waiting for user input"
	case StateShuttingDown:
		return "Shutting down..."
	default:
		return "Unknown"
	}
}

// Helper functions for layout calculations

// calculateMainContentHeight calculates the available height for main content
func (m *Model) calculateMainContentHeight() int {
	headerHeight := 3   // Title and header
	progressHeight := 4 // Progress bar and status
	footerHeight := 1   // Footer
	inputHeight := 0

	if m.showInput {
		inputHeight = 4 // Input area
	}

	usedHeight := headerHeight + progressHeight + footerHeight + inputHeight
	availableHeight := m.windowHeight - usedHeight

	if availableHeight < 5 {
		availableHeight = 5
	}

	return availableHeight
}

// calculateSidebarWidth calculates the width for the sidebar
func (m *Model) calculateSidebarWidth() int {
	if m.windowWidth < 120 {
		return 0 // No sidebar in narrow mode
	}

	return m.windowWidth / 3
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
}

// truncateText truncates text to fit within a given width
func truncateText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	if width < 3 {
		return text[:width]
	}

	return text[:width-3] + "..."
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	if len(text) >= width {
		return truncateText(text, width)
	}

	padding := width - len(text)
	leftPadding := padding / 2
	rightPadding := padding - leftPadding

	return strings.Repeat(" ", leftPadding) + text + strings.Repeat(" ", rightPadding)
}
