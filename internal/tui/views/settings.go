package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/tui/theme"
)

// SettingsModel displays the settings screen with theme selection.
type SettingsModel struct {
	cursor int
	width  int
	height int
}

// NewSettingsModel creates a settings view with the cursor on the active theme.
func NewSettingsModel(activeTheme string) SettingsModel {
	cursor := 0
	for i, entry := range theme.Palettes {
		if entry.Name == activeTheme {
			cursor = i
			break
		}
	}
	return SettingsModel{cursor: cursor}
}

// SetSize updates the available dimensions.
func (m *SettingsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// MoveUp moves the cursor up.
func (m *SettingsModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// MoveDown moves the cursor down.
func (m *SettingsModel) MoveDown() {
	if m.cursor < len(theme.Palettes)-1 {
		m.cursor++
	}
}

// SelectedTheme returns the name and palette of the currently selected theme.
func (m *SettingsModel) SelectedTheme() (string, theme.Palette) {
	entry := theme.Palettes[m.cursor]
	return entry.Name, entry.Palette
}

// View renders the settings screen.
func (m *SettingsModel) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorText).
		Render("Theme")

	var rows []string
	rows = append(rows, title)
	rows = append(rows, "")

	for i, entry := range theme.Palettes {
		name := entry.Name
		p := entry.Palette

		// Color swatch: show the 8 core palette colors as filled blocks
		swatch := colorSwatch(p)

		var line string
		if i == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯")
			label := lipgloss.NewStyle().Foreground(theme.ColorText).Bold(true).Render(
				fmt.Sprintf(" %-10s", name),
			)
			line = cursor + label + "  " + swatch
		} else {
			label := lipgloss.NewStyle().Foreground(theme.ColorDimText).Render(
				fmt.Sprintf("  %-10s", name),
			)
			line = label + "  " + swatch
		}
		rows = append(rows, line)
	}

	rows = append(rows, "")
	hint := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("enter:select  esc:back")
	rows = append(rows, hint)

	content := strings.Join(rows, "\n")
	return lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1).Render(content)
}

// colorSwatch renders a row of colored blocks showing the palette's core colors.
func colorSwatch(p theme.Palette) string {
	colors := []string{p.Pink, p.Green, p.Yellow, p.Cyan, p.Purple, p.Orange, p.Comment, p.Fg}
	parts := make([]string, len(colors))
	for i, c := range colors {
		parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render("██")
	}
	return strings.Join(parts, "")
}
