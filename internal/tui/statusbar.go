package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/tui/theme"
)

type keyHint struct {
	Key  string
	Desc string
}

// sessionsStatusBar renders the status bar for the sessions list view.
func sessionsStatusBar(width int) string {
	return renderStatusBar(width, []keyHint{
		{"q", "quit"},
		{"a", "approve"},
		{"x", "deny"},
		{"enter", "details"},
		{"s", "settings"},
		{"?", "help"},
	})
}

// detailStatusBar renders the status bar for the detail view.
func detailStatusBar(width int) string {
	return renderStatusBar(width, []keyHint{
		{"esc", "back"},
		{"↑/↓", "scroll"},
		{"a", "approve"},
		{"x", "deny"},
		{"?", "help"},
	})
}

// settingsStatusBar renders the status bar for the settings view.
func settingsStatusBar(width int) string {
	return renderStatusBar(width, []keyHint{
		{"esc", "back"},
		{"↑/↓", "navigate"},
		{"enter", "select"},
	})
}

func renderStatusBar(width int, hints []keyHint) string {
	parts := make([]string, len(hints))
	for i, h := range hints {
		key := theme.StyleStatusBarKey.Render(h.Key)
		desc := lipgloss.NewStyle().Foreground(theme.ColorDimText).Render(":" + h.Desc)
		parts[i] = key + desc
	}
	return theme.StyleStatusBar.Width(width).Render(strings.Join(parts, "  "))
}
