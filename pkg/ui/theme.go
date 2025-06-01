package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/fumiya-kume/cca/internal/types"
)

// Theme defines colors and styles for the UI
type Theme struct {
	// Primary colors
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Background lipgloss.Color
	Surface    lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// Text colors
	Text        lipgloss.Color
	TextMuted   lipgloss.Color
	TextInverse lipgloss.Color

	// Border colors
	Border      lipgloss.Color
	BorderFocus lipgloss.Color

	// Progress colors
	ProgressBG lipgloss.Color
	ProgressFG lipgloss.Color

	// Styles
	Styles ThemeStyles
}

// ThemeStyles contains pre-configured lipgloss styles
type ThemeStyles struct {
	// Layout styles
	Title      lipgloss.Style
	Header     lipgloss.Style
	Footer     lipgloss.Style
	Panel      lipgloss.Style
	PanelFocus lipgloss.Style

	// Content styles
	Output     lipgloss.Style
	Input      lipgloss.Style
	Progress   lipgloss.Style
	Stage      lipgloss.Style
	StageFocus lipgloss.Style

	// Status styles
	StatusInfo    lipgloss.Style
	StatusSuccess lipgloss.Style
	StatusWarning lipgloss.Style
	StatusError   lipgloss.Style

	// Interactive styles
	Button       lipgloss.Style
	ButtonFocus  lipgloss.Style
	ButtonActive lipgloss.Style

	// Typography
	Bold   lipgloss.Style
	Italic lipgloss.Style
	Code   lipgloss.Style
	Link   lipgloss.Style
}

// NewDarkTheme creates a dark theme
func NewDarkTheme() Theme {
	// Dark theme color palette
	theme := Theme{
		Primary:    lipgloss.Color("#7c3aed"), // Purple
		Secondary:  lipgloss.Color("#10b981"), // Green
		Accent:     lipgloss.Color("#f59e0b"), // Amber
		Background: lipgloss.Color("#1f2937"), // Dark gray
		Surface:    lipgloss.Color("#374151"), // Medium gray

		Success: lipgloss.Color("#10b981"), // Green
		Warning: lipgloss.Color("#f59e0b"), // Amber
		Error:   lipgloss.Color("#ef4444"), // Red
		Info:    lipgloss.Color("#3b82f6"), // Blue

		Text:        lipgloss.Color("#f9fafb"), // Light gray
		TextMuted:   lipgloss.Color("#9ca3af"), // Muted gray
		TextInverse: lipgloss.Color("#111827"), // Dark

		Border:      lipgloss.Color("#4b5563"), // Gray
		BorderFocus: lipgloss.Color("#7c3aed"), // Purple

		ProgressBG: lipgloss.Color("#374151"), // Medium gray
		ProgressFG: lipgloss.Color("#7c3aed"), // Purple
	}

	theme.Styles = createThemeStyles(theme)
	return theme
}

// NewLightTheme creates a light theme
func NewLightTheme() Theme {
	// Light theme color palette
	theme := Theme{
		Primary:    lipgloss.Color("#5b21b6"), // Purple
		Secondary:  lipgloss.Color("#059669"), // Green
		Accent:     lipgloss.Color("#d97706"), // Amber
		Background: lipgloss.Color("#ffffff"), // White
		Surface:    lipgloss.Color("#f9fafb"), // Light gray

		Success: lipgloss.Color("#059669"), // Green
		Warning: lipgloss.Color("#d97706"), // Amber
		Error:   lipgloss.Color("#dc2626"), // Red
		Info:    lipgloss.Color("#2563eb"), // Blue

		Text:        lipgloss.Color("#111827"), // Dark
		TextMuted:   lipgloss.Color("#6b7280"), // Muted gray
		TextInverse: lipgloss.Color("#f9fafb"), // Light

		Border:      lipgloss.Color("#d1d5db"), // Light gray
		BorderFocus: lipgloss.Color("#5b21b6"), // Purple

		ProgressBG: lipgloss.Color("#e5e7eb"), // Light gray
		ProgressFG: lipgloss.Color("#5b21b6"), // Purple
	}

	theme.Styles = createThemeStyles(theme)
	return theme
}

// createThemeStyles creates all the lipgloss styles for a theme
func createThemeStyles(theme Theme) ThemeStyles {
	return ThemeStyles{
		// Layout styles
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Primary).
			Background(theme.Background).
			Padding(0, 1).
			Margin(0, 0, 1, 0),

		Header: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Surface).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),

		Footer: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Background(theme.Background).
			Padding(0, 1).
			Align(lipgloss.Center),

		Panel: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Surface).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),

		PanelFocus: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Surface).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderFocus).
			Bold(true),

		// Content styles
		Output: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Background).
			Padding(1),

		Input: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Surface).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),

		Progress: lipgloss.NewStyle().
			Foreground(theme.ProgressFG).
			Background(theme.ProgressBG),

		Stage: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Surface).
			Padding(0, 1).
			Margin(0, 0, 0, 2),

		StageFocus: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Background(theme.Surface).
			Padding(0, 1).
			Margin(0, 0, 0, 2).
			Bold(true),

		// Status styles
		StatusInfo: lipgloss.NewStyle().
			Foreground(theme.Info).
			Bold(true),

		StatusSuccess: lipgloss.NewStyle().
			Foreground(theme.Success).
			Bold(true),

		StatusWarning: lipgloss.NewStyle().
			Foreground(theme.Warning).
			Bold(true),

		StatusError: lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true),

		// Interactive styles
		Button: lipgloss.NewStyle().
			Foreground(theme.TextInverse).
			Background(theme.Primary).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary),

		ButtonFocus: lipgloss.NewStyle().
			Foreground(theme.TextInverse).
			Background(theme.Primary).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Bold(true).
			Underline(true),

		ButtonActive: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Background(theme.Background).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Bold(true),

		// Typography
		Bold: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Text),

		Italic: lipgloss.NewStyle().
			Italic(true).
			Foreground(theme.TextMuted),

		Code: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Background(theme.Surface).
			Padding(0, 1),

		Link: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Underline(true),
	}
}

// GetStatusStyle returns the appropriate style for a status type
func (t Theme) GetStatusStyle(statusType StatusType) lipgloss.Style {
	switch statusType {
	case StatusSuccess:
		return t.Styles.StatusSuccess
	case StatusWarning:
		return t.Styles.StatusWarning
	case StatusError:
		return t.Styles.StatusError
	default:
		return t.Styles.StatusInfo
	}
}

// GetStageStatusIcon returns an icon for the stage status
func GetStageStatusIcon(status types.StageStatus) string {
	switch status {
	case types.StagePending:
		return "○"
	case types.StageRunning:
		return "⏳"
	case types.StageCompleted:
		return "✓"
	case types.StageFailed:
		return "✗"
	case types.StageSkipped:
		return "⊘"
	default:
		return "○"
	}
}

// GetStageStatusColor returns a color for the stage status
func (t Theme) GetStageStatusColor(status types.StageStatus) lipgloss.Color {
	switch status {
	case types.StagePending:
		return t.TextMuted
	case types.StageRunning:
		return t.Info
	case types.StageCompleted:
		return t.Success
	case types.StageFailed:
		return t.Error
	case types.StageSkipped:
		return t.Warning
	default:
		return t.TextMuted
	}
}

// ApplyTheme applies the theme to a style
func (t Theme) ApplyTheme(base lipgloss.Style) lipgloss.Style {
	return base.
		Foreground(t.Text).
		Background(t.Background)
}

// WithPadding adds padding to a style
func WithPadding(style lipgloss.Style, padding ...int) lipgloss.Style {
	switch len(padding) {
	case 1:
		return style.Padding(padding[0])
	case 2:
		return style.Padding(padding[0], padding[1])
	case 4:
		return style.Padding(padding[0], padding[1], padding[2], padding[3])
	default:
		return style
	}
}

// WithMargin adds margin to a style
func WithMargin(style lipgloss.Style, margin ...int) lipgloss.Style {
	switch len(margin) {
	case 1:
		return style.Margin(margin[0])
	case 2:
		return style.Margin(margin[0], margin[1])
	case 4:
		return style.Margin(margin[0], margin[1], margin[2], margin[3])
	default:
		return style
	}
}

// WithBorder adds a border to a style
func WithBorder(style lipgloss.Style, borderStyle lipgloss.Border, color lipgloss.Color) lipgloss.Style {
	return style.
		Border(borderStyle).
		BorderForeground(color)
}
