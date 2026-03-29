package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/tui/theme"
	"github.com/allir/c5s/internal/version"
)

// headerView renders the sessions list header with optional key hints right-aligned.
func headerView(sessionCount, width int, hints []keyHint) string {
	// Line 1: title + version
	title := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorText).Render("c5s")
	ver := lipgloss.NewStyle().Foreground(theme.ColorFgAlt).Render(" " + version.Version)
	line1 := title + ver

	// Line 2: subtitle + count
	subtitle := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(
		fmt.Sprintf("Claude Code Sessions · %d active", sessionCount),
	)
	line2 := subtitle

	left := lipgloss.NewStyle().PaddingLeft(1).Render(line1 + "\n" + line2)

	if len(hints) == 0 {
		return lipgloss.NewStyle().Width(width).Render(left)
	}

	// Key hints: right-aligned on the subtitle line
	parts := make([]string, len(hints))
	for i, h := range hints {
		key := theme.StyleStatusBarKey.Render(h.Key)
		desc := lipgloss.NewStyle().Foreground(theme.ColorFgAlt).Render(":" + h.Desc)
		parts[i] = key + desc
	}
	legend := strings.Join(parts, "  ")

	// Place header left, legend right on the same block
	gap := width - lipgloss.Width(left) - lipgloss.Width(legend)
	if gap < 2 {
		// Not enough room — put legend on its own line
		return lipgloss.NewStyle().Width(width).Render(left + "\n" +
			lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(legend))
	}
	// Overlay: left block + right-aligned legend on the second line
	return lipgloss.NewStyle().Width(width).Render(left) + "\n" +
		lipgloss.NewStyle().Width(width).Align(lipgloss.Right).PaddingRight(1).Render(legend)
}
