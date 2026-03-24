// Package theme defines the color palette and styles for the c5s TUI.
package theme

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/claude"
)

// Colors used throughout the TUI.
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // purple
	ColorSecondary = lipgloss.Color("#06B6D4") // cyan
	ColorMuted     = lipgloss.Color("#6B7280") // gray
	ColorSuccess   = lipgloss.Color("#10B981") // green
	ColorWarning   = lipgloss.Color("#F59E0B") // amber
	ColorDanger    = lipgloss.Color("#EF4444") // red
	ColorText      = lipgloss.Color("#E5E7EB") // light gray
	ColorDimText   = lipgloss.Color("#9CA3AF") // dim gray
	ColorBg        = lipgloss.Color("#111827") // dark bg
	ColorBgAlt     = lipgloss.Color("#1F2937") // slightly lighter bg
)

// Layout styles.
var (
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			PaddingLeft(1)

	StyleHeaderCount = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(ColorDimText).
			PaddingLeft(1)

	StyleStatusBarKey = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	StyleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorText)

	StyleTableRow = lipgloss.NewStyle().
			Foreground(ColorDimText)

	StyleTableRowSelected = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorBgAlt).
				Bold(true)

	StyleTableCell = lipgloss.NewStyle().
			PaddingRight(2)
)

// SeparatorLine renders a horizontal separator line at the given width.
func SeparatorLine(width int) string {
	return lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(strings.Repeat("─", width))
}

// StatusStyle returns a style for the given status string.
func StatusStyle(s claude.Status) lipgloss.Style {
	switch s {
	case claude.StatusWorking:
		return lipgloss.NewStyle().Foreground(ColorSuccess)
	case claude.StatusIdle:
		return lipgloss.NewStyle().Foreground(ColorMuted)
	case claude.StatusInput:
		return lipgloss.NewStyle().Foreground(ColorWarning)
	case claude.StatusFinished:
		return lipgloss.NewStyle().Foreground(ColorDimText)
	default:
		return lipgloss.NewStyle().Foreground(ColorDimText)
	}
}

// StatusIndicator returns a colored dot + label for the given status.
func StatusIndicator(s claude.Status) string {
	return StatusStyle(s).Render("●") + " " + s.String()
}
