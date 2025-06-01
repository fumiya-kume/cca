package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"

	"github.com/fumiya-kume/cca/internal/types"
)

// TestNewDarkTheme tests dark theme creation
func TestNewDarkTheme(t *testing.T) {
	theme := NewDarkTheme()

	// Test primary colors are set
	assert.NotEmpty(t, theme.Primary)
	assert.NotEmpty(t, theme.Secondary)
	assert.NotEmpty(t, theme.Accent)
	assert.NotEmpty(t, theme.Background)
	assert.NotEmpty(t, theme.Surface)

	// Test status colors are set
	assert.NotEmpty(t, theme.Success)
	assert.NotEmpty(t, theme.Warning)
	assert.NotEmpty(t, theme.Error)
	assert.NotEmpty(t, theme.Info)

	// Test text colors are set
	assert.NotEmpty(t, theme.Text)
	assert.NotEmpty(t, theme.TextMuted)
	assert.NotEmpty(t, theme.TextInverse)

	// Test border colors are set
	assert.NotEmpty(t, theme.Border)
	assert.NotEmpty(t, theme.BorderFocus)

	// Test progress colors are set
	assert.NotEmpty(t, theme.ProgressBG)
	assert.NotEmpty(t, theme.ProgressFG)

	// Test styles are initialized
	assert.NotNil(t, theme.Styles)

	// Test specific dark theme colors
	assert.Equal(t, lipgloss.Color("#7c3aed"), theme.Primary)
	assert.Equal(t, lipgloss.Color("#10b981"), theme.Secondary)
	assert.Equal(t, lipgloss.Color("#f59e0b"), theme.Accent)
	assert.Equal(t, lipgloss.Color("#1f2937"), theme.Background)
}

// TestNewLightTheme tests light theme creation
func TestNewLightTheme(t *testing.T) {
	theme := NewLightTheme()

	// Test primary colors are set
	assert.NotEmpty(t, theme.Primary)
	assert.NotEmpty(t, theme.Secondary)
	assert.NotEmpty(t, theme.Accent)
	assert.NotEmpty(t, theme.Background)
	assert.NotEmpty(t, theme.Surface)

	// Test status colors are set
	assert.NotEmpty(t, theme.Success)
	assert.NotEmpty(t, theme.Warning)
	assert.NotEmpty(t, theme.Error)
	assert.NotEmpty(t, theme.Info)

	// Test text colors are set
	assert.NotEmpty(t, theme.Text)
	assert.NotEmpty(t, theme.TextMuted)
	assert.NotEmpty(t, theme.TextInverse)

	// Test border colors are set
	assert.NotEmpty(t, theme.Border)
	assert.NotEmpty(t, theme.BorderFocus)

	// Test progress colors are set
	assert.NotEmpty(t, theme.ProgressBG)
	assert.NotEmpty(t, theme.ProgressFG)

	// Test styles are initialized
	assert.NotNil(t, theme.Styles)

	// Test specific light theme colors (different from dark)
	assert.NotEqual(t, lipgloss.Color("#1f2937"), theme.Background) // Should not be dark background
	assert.NotEqual(t, lipgloss.Color("#f9fafb"), theme.Text)       // Should not be light text
}

// TestThemeStylesCreation tests that theme styles are properly created
func TestThemeStylesCreation(t *testing.T) {
	theme := NewDarkTheme()
	styles := theme.Styles

	// Test layout styles exist
	assert.NotNil(t, styles.Title)
	assert.NotNil(t, styles.Header)
	assert.NotNil(t, styles.Footer)
	assert.NotNil(t, styles.Panel)
	assert.NotNil(t, styles.PanelFocus)

	// Test content styles exist
	assert.NotNil(t, styles.Output)
	assert.NotNil(t, styles.Input)
	assert.NotNil(t, styles.Progress)
	assert.NotNil(t, styles.Stage)
	assert.NotNil(t, styles.StageFocus)

	// Test status styles exist
	assert.NotNil(t, styles.StatusInfo)
	assert.NotNil(t, styles.StatusSuccess)
	assert.NotNil(t, styles.StatusWarning)
	assert.NotNil(t, styles.StatusError)

	// Test interactive styles exist
	assert.NotNil(t, styles.Button)
	assert.NotNil(t, styles.ButtonFocus)
	assert.NotNil(t, styles.ButtonActive)

	// Test typography styles exist
	assert.NotNil(t, styles.Bold)
	assert.NotNil(t, styles.Italic)
	assert.NotNil(t, styles.Code)
	assert.NotNil(t, styles.Link)
}

// TestGetStatusStyle tests status style retrieval
func TestGetStatusStyle(t *testing.T) {
	theme := NewDarkTheme()

	tests := []struct {
		name       string
		statusType StatusType
		expected   lipgloss.Style
	}{
		{"Info status", StatusInfo, theme.Styles.StatusInfo},
		{"Success status", StatusSuccess, theme.Styles.StatusSuccess},
		{"Warning status", StatusWarning, theme.Styles.StatusWarning},
		{"Error status", StatusError, theme.Styles.StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := theme.GetStatusStyle(tt.statusType)
			assert.Equal(t, tt.expected, style)
		})
	}
}

// TestGetStageStatusIcon tests stage status icon retrieval
func TestGetStageStatusIcon(t *testing.T) {

	tests := []struct {
		name   string
		status types.StageStatus
	}{
		{"Pending stage", types.StagePending},
		{"Running stage", types.StageRunning},
		{"Completed stage", types.StageCompleted},
		{"Failed stage", types.StageFailed},
		{"Skipped stage", types.StageSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := GetStageStatusIcon(tt.status)
			assert.NotEmpty(t, icon, "Icon should not be empty for status %v", tt.status)
		})
	}
}

// TestGetStageStatusColor tests stage status color retrieval
func TestGetStageStatusColor(t *testing.T) {
	theme := NewDarkTheme()

	tests := []struct {
		name   string
		status types.StageStatus
	}{
		{"Pending stage", types.StagePending},
		{"Running stage", types.StageRunning},
		{"Completed stage", types.StageCompleted},
		{"Failed stage", types.StageFailed},
		{"Skipped stage", types.StageSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := theme.GetStageStatusColor(tt.status)
			assert.NotEmpty(t, color, "Color should not be empty for status %v", tt.status)
		})
	}
}

// TestApplyTheme tests theme application functionality
func TestApplyTheme(t *testing.T) {
	theme := NewDarkTheme()
	style := lipgloss.NewStyle()

	// Test applying theme to a style
	styledComponent := theme.ApplyTheme(style)
	assert.NotNil(t, styledComponent)

	// Verify theme colors are applied
	assert.NotEqual(t, style, styledComponent, "Styled component should differ from base style")
}

// TestWithPadding tests padding helper function
func TestWithPadding(t *testing.T) {
	style := lipgloss.NewStyle()

	// Test different padding configurations
	paddedStyle := WithPadding(style, 1)
	assert.NotNil(t, paddedStyle)

	// Test with multiple padding values
	multiPaddedStyle := WithPadding(style, 1, 2)
	assert.NotNil(t, multiPaddedStyle)

	// Test with full padding specification
	fullPaddedStyle := WithPadding(style, 1, 2, 3, 4)
	assert.NotNil(t, fullPaddedStyle)
}

// TestWithMargin tests margin helper function
func TestWithMargin(t *testing.T) {
	style := lipgloss.NewStyle()

	// Test different margin configurations
	marginStyle := WithMargin(style, 1)
	assert.NotNil(t, marginStyle)

	// Test with multiple margin values
	multiMarginStyle := WithMargin(style, 1, 2)
	assert.NotNil(t, multiMarginStyle)

	// Test with full margin specification
	fullMarginStyle := WithMargin(style, 1, 2, 3, 4)
	assert.NotNil(t, fullMarginStyle)
}

// TestWithBorder tests border helper function
func TestWithBorder(t *testing.T) {
	theme := NewDarkTheme()
	style := lipgloss.NewStyle()

	// Test different border types
	normalBorder := WithBorder(style, lipgloss.NormalBorder(), theme.Border)
	assert.NotNil(t, normalBorder)

	// Test with focus state
	focusBorder := WithBorder(style, lipgloss.RoundedBorder(), theme.BorderFocus)
	assert.NotNil(t, focusBorder)

	// Test with different border styles
	doubleBorder := WithBorder(style, lipgloss.DoubleBorder(), theme.Border)
	assert.NotNil(t, doubleBorder)
}

// TestThemeColorConsistency tests that colors are consistent across themes
func TestThemeColorConsistency(t *testing.T) {
	darkTheme := NewDarkTheme()
	lightTheme := NewLightTheme()

	// Both themes should have all required colors set
	requiredColors := []func(Theme) lipgloss.Color{
		func(t Theme) lipgloss.Color { return t.Primary },
		func(t Theme) lipgloss.Color { return t.Secondary },
		func(t Theme) lipgloss.Color { return t.Accent },
		func(t Theme) lipgloss.Color { return t.Background },
		func(t Theme) lipgloss.Color { return t.Surface },
		func(t Theme) lipgloss.Color { return t.Success },
		func(t Theme) lipgloss.Color { return t.Warning },
		func(t Theme) lipgloss.Color { return t.Error },
		func(t Theme) lipgloss.Color { return t.Info },
		func(t Theme) lipgloss.Color { return t.Text },
		func(t Theme) lipgloss.Color { return t.TextMuted },
		func(t Theme) lipgloss.Color { return t.TextInverse },
		func(t Theme) lipgloss.Color { return t.Border },
		func(t Theme) lipgloss.Color { return t.BorderFocus },
		func(t Theme) lipgloss.Color { return t.ProgressBG },
		func(t Theme) lipgloss.Color { return t.ProgressFG },
	}

	for i, colorGetter := range requiredColors {
		darkColor := colorGetter(darkTheme)
		lightColor := colorGetter(lightTheme)

		assert.NotEmpty(t, darkColor, "Dark theme color %d should not be empty", i)
		assert.NotEmpty(t, lightColor, "Light theme color %d should not be empty", i)
	}
}

// TestStatusStyleMapping tests that all status types have corresponding styles
func TestStatusStyleMapping(t *testing.T) {
	theme := NewDarkTheme()

	statusTypes := []StatusType{
		StatusInfo,
		StatusSuccess,
		StatusWarning,
		StatusError,
	}

	for _, statusType := range statusTypes {
		style := theme.GetStatusStyle(statusType)
		assert.NotNil(t, style, "Status type %v should have a corresponding style", statusType)
	}
}

// TestStageStatusMappings tests that all stage statuses have icons and colors
func TestStageStatusMappings(t *testing.T) {
	theme := NewDarkTheme()

	stageStatuses := []types.StageStatus{
		types.StagePending,
		types.StageRunning,
		types.StageCompleted,
		types.StageFailed,
		types.StageSkipped,
	}

	for _, status := range stageStatuses {
		icon := GetStageStatusIcon(status)
		color := theme.GetStageStatusColor(status)

		assert.NotEmpty(t, icon, "Stage status %v should have an icon", status)
		assert.NotEmpty(t, color, "Stage status %v should have a color", status)
	}
}

// TestThemeStructure tests the theme structure completeness
func TestThemeStructure(t *testing.T) {
	theme := NewDarkTheme()

	// Test that all theme fields are initialized
	assert.NotEmpty(t, theme.Primary)
	assert.NotEmpty(t, theme.Secondary)
	assert.NotEmpty(t, theme.Accent)
	assert.NotEmpty(t, theme.Background)
	assert.NotEmpty(t, theme.Surface)
	assert.NotEmpty(t, theme.Success)
	assert.NotEmpty(t, theme.Warning)
	assert.NotEmpty(t, theme.Error)
	assert.NotEmpty(t, theme.Info)
	assert.NotEmpty(t, theme.Text)
	assert.NotEmpty(t, theme.TextMuted)
	assert.NotEmpty(t, theme.TextInverse)
	assert.NotEmpty(t, theme.Border)
	assert.NotEmpty(t, theme.BorderFocus)
	assert.NotEmpty(t, theme.ProgressBG)
	assert.NotEmpty(t, theme.ProgressFG)

	// Test that styles are not nil
	styles := theme.Styles
	assert.NotNil(t, styles.Title)
	assert.NotNil(t, styles.Header)
	assert.NotNil(t, styles.Footer)
	assert.NotNil(t, styles.Panel)
	assert.NotNil(t, styles.PanelFocus)
	assert.NotNil(t, styles.Output)
	assert.NotNil(t, styles.Input)
	assert.NotNil(t, styles.Progress)
	assert.NotNil(t, styles.Stage)
	assert.NotNil(t, styles.StageFocus)
	assert.NotNil(t, styles.StatusInfo)
	assert.NotNil(t, styles.StatusSuccess)
	assert.NotNil(t, styles.StatusWarning)
	assert.NotNil(t, styles.StatusError)
	assert.NotNil(t, styles.Button)
	assert.NotNil(t, styles.ButtonFocus)
	assert.NotNil(t, styles.ButtonActive)
	assert.NotNil(t, styles.Bold)
	assert.NotNil(t, styles.Italic)
	assert.NotNil(t, styles.Code)
	assert.NotNil(t, styles.Link)
}
