package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Palette defines the unified dark color palette.
var (
	// Base & neutrals
	ColorBackground = lipgloss.Color("#16161E")
	ColorSurface    = lipgloss.Color("#1F2335")
	ColorBorder     = lipgloss.Color("#3B4261")
	ColorBorderDim  = lipgloss.Color("#292E42")
	ColorText       = lipgloss.Color("#C0CAF5")
	ColorTextMuted  = lipgloss.Color("#7AA2F7")
	ColorTextDim    = lipgloss.Color("#565F89")

	// Accents & highlights
	ColorPrimary   = lipgloss.Color("#7AA2F7") // Blue
	ColorSecondary = lipgloss.Color("#BB9AF7") // Purple
	ColorAccent    = lipgloss.Color("#00E8C6") // Bright Teal/Cyan
	ColorSelection = lipgloss.Color("#283457") // Selected row background

	// Status indicators
	ColorStatusRunning  = lipgloss.Color("#9ECE6A") // Green
	ColorStatusStopped  = lipgloss.Color("#565F89") // Gray / Muted
	ColorStatusPending  = lipgloss.Color("#E0AF68") // Yellow / Orange
	ColorStatusError    = lipgloss.Color("#F7768E") // Red
	ColorStatusInfo     = lipgloss.Color("#7DCFFF") // Cyan
	ColorStatusDegraded = lipgloss.Color("#FF9E64") // Amber
)

// Typography and Common Styles
var (
	// App Header
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Background(ColorSurface).
			Padding(0, 1)

	AppVersionStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Padding(0, 1)

	// Panels & Windows
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	PanelActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent).
				Padding(0, 1)

	PanelHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Padding(0, 1)

	// Status Badges
	BadgeRunning = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1A1B26")).
			Background(ColorStatusRunning).
			Padding(0, 1)

	BadgeStopped = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorStatusStopped).
			Padding(0, 1)

	BadgePending = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#1A1B26")).
			Background(ColorStatusPending).
			Padding(0, 1)

	BadgeError = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorStatusError).
			Padding(0, 1)

	// Table Styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(ColorBorder)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	TableRowSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorSelection)

	// Tabs
	TabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Background(ColorSurface).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorAccent).
			Padding(0, 2)

	TabInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorTextDim).
				Padding(0, 2)

	// Keybinding & Footer Bar
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Background(ColorSurface).
			Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	// Dialog & Overlay
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ColorAccent).
			Background(ColorSurface).
			Padding(1, 2)

	DialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				MarginBottom(1)

	// Toast / Status Line
	ToastSuccessStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorStatusRunning).
				Background(ColorSurface).
				Padding(0, 1)

	ToastErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorStatusError).
			Background(ColorSurface).
			Padding(0, 1)

	ToastInfoStyle = lipgloss.NewStyle().
			Foreground(ColorStatusInfo).
			Background(ColorSurface).
			Padding(0, 1)

	ToastWarningStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorStatusPending).
				Background(ColorSurface).
				Padding(0, 1)
)

// StatusBadge returns a formatted status badge for a given instance status string.
func StatusBadge(status string) string {
	switch status {
	case "Running":
		return BadgeRunning.Render("RUNNING")
	case "Stopped":
		return BadgeStopped.Render("STOPPED")
	case "Broken", "Error":
		return BadgeError.Render("BROKEN")
	case "Starting":
		return BadgePending.Render("STARTING")
	case "Stopping":
		return BadgePending.Render("STOPPING")
	default:
		if status == "" {
			return BadgeStopped.Render("UNKNOWN")
		}
		return BadgePending.Render(status)
	}
}
