package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#00CED1")  // Dark Turquoise
	secondaryColor = lipgloss.Color("#FFD700")  // Gold
	successColor   = lipgloss.Color("#32CD32")  // Lime Green
	warningColor   = lipgloss.Color("#FFA500")  // Orange
	errorColor     = lipgloss.Color("#DC143C")  // Crimson
	mutedColor     = lipgloss.Color("#696969")  // Dim Gray
	bgColor        = lipgloss.Color("#1E1E1E")  // Dark background
	fgColor        = lipgloss.Color("#E0E0E0")  // Light foreground

	// Base styles
	BaseStyle = lipgloss.NewStyle().
		Foreground(fgColor).
		Background(bgColor)

	// Header and Footer
	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(bgColor).
		Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(bgColor).
		Padding(0, 1)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(secondaryColor).
		Background(bgColor).
		Underline(true)

	HeaderHighlightStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(bgColor).
		Underline(true)

	SelectedRowStyle = lipgloss.NewStyle().
		Foreground(bgColor).
		Background(primaryColor)

	// Process state colors
	HighCPUStyle = lipgloss.NewStyle().
		Foreground(errorColor)

	HighMemoryStyle = lipgloss.NewStyle().
		Foreground(warningColor)

	HighIOStyle = lipgloss.NewStyle().
		Foreground(primaryColor)

	// Summary view styles
	SectionStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mutedColor).
		Padding(0, 1).
		Margin(0, 1)

	SectionTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Underline(true)

	// Progress bar styles
	ProgressBarFullStyle = lipgloss.NewStyle().
		Foreground(successColor)

	ProgressBarEmptyStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	// Status indicators
	NormalStyle = lipgloss.NewStyle().
		Foreground(successColor)

	WarningStyle = lipgloss.NewStyle().
		Foreground(warningColor)

	CriticalStyle = lipgloss.NewStyle().
		Foreground(errorColor)

	// Help screen styles
	HelpSectionStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(secondaryColor)

	HelpKeyStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)

	HelpDescriptionStyle = lipgloss.NewStyle().
		Foreground(fgColor)

	HelpTipStyle = lipgloss.NewStyle().
		Italic(true).
		Foreground(successColor)

	URLStyle = lipgloss.NewStyle().
		Underline(true).
		Foreground(primaryColor)

	// Dialog styles
	FilterDialogStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		Width(60).
		Align(lipgloss.Center)

	// Error styles
	ErrorStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(errorColor).
		Background(bgColor).
		Padding(1, 2)

	// Info styles
	InfoStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Padding(0, 1)

	SummaryStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Padding(0, 1)

	// Utility styles
	DimStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	CenterStyle = lipgloss.NewStyle().
		Align(lipgloss.Center)

	ScrollIndicatorStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		Align(lipgloss.Right)
)

// GetTheme returns the current theme
type Theme struct {
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Muted     lipgloss.Color
	BG        lipgloss.Color
	FG        lipgloss.Color
}

// DefaultTheme returns the default color theme
func DefaultTheme() Theme {
	return Theme{
		Primary:   primaryColor,
		Secondary: secondaryColor,
		Success:   successColor,
		Warning:   warningColor,
		Error:     errorColor,
		Muted:     mutedColor,
		BG:        bgColor,
		FG:        fgColor,
	}
}

// LightTheme returns a light color theme
func LightTheme() Theme {
	return Theme{
		Primary:   lipgloss.Color("#006400"),  // Dark Green
		Secondary: lipgloss.Color("#FF8C00"),  // Dark Orange
		Success:   lipgloss.Color("#228B22"),  // Forest Green
		Warning:   lipgloss.Color("#FF6347"),  // Tomato
		Error:     lipgloss.Color("#B22222"),  // Fire Brick
		Muted:     lipgloss.Color("#A9A9A9"),  // Dark Gray
		BG:        lipgloss.Color("#F5F5F5"),  // White Smoke
		FG:        lipgloss.Color("#2F4F4F"),  // Dark Slate Gray
	}
}

// ApplyTheme applies a theme to all styles
func ApplyTheme(theme Theme) {
	BaseStyle = BaseStyle.
		Foreground(theme.FG).
		Background(theme.BG)

	HeaderStyle = HeaderStyle.
		Foreground(theme.Primary).
		Background(theme.BG)

	TableHeaderStyle = TableHeaderStyle.
		Foreground(theme.Secondary).
		Background(theme.BG)

	SelectedRowStyle = SelectedRowStyle.
		Foreground(theme.BG).
		Background(theme.Primary)

	HighCPUStyle = HighCPUStyle.
		Foreground(theme.Error)

	HighMemoryStyle = HighMemoryStyle.
		Foreground(theme.Warning)

	HighIOStyle = HighIOStyle.
		Foreground(theme.Primary)

	// Update other styles as needed
}