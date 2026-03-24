package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/tui/theme"
	"github.com/allir/c5s/internal/version"
)

// headerView renders the sessions list header, styled after Claude Code's banner.
func headerView(sessionCount, width int) string {
	// Line 1: title + version
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorText).Render("c5s")
	ver := lipgloss.NewStyle().Foreground(theme.ColorDimText).Render(" " + version.Version)
	line1 := title + ver

	// Line 2: subtitle + count
	subtitle := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(
		fmt.Sprintf("Claude Code Sessions · %d active", sessionCount),
	)
	line2 := subtitle

	block := lipgloss.NewStyle().PaddingLeft(1).Render(line1 + "\n" + line2)
	return lipgloss.NewStyle().Width(width).Render(block)
}
