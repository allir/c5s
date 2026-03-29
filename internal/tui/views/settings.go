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
	active int // index of the currently applied theme
	width  int
	height int
}

// NewSettingsModel creates a settings view with the cursor on the active theme.
func NewSettingsModel(activeTheme string) SettingsModel {
	cursor := 0
	if i, _, ok := theme.FindTheme(activeTheme); ok {
		cursor = i
	}
	return SettingsModel{cursor: cursor, active: cursor}
}

// SetActive updates the active theme index after a selection.
func (m *SettingsModel) SetActive(idx int) {
	m.active = idx
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
	if m.cursor < len(theme.Themes)-1 {
		m.cursor++
	}
}

// SelectedTheme returns the name and palette of the currently selected theme.
func (m *SettingsModel) SelectedTheme() (string, theme.Palette) {
	entry := theme.Themes[m.cursor]
	return entry.Name, entry.Palette
}

// Cursor returns the current cursor index.
func (m *SettingsModel) Cursor() int {
	return m.cursor
}

// View renders the settings screen.
func (m *SettingsModel) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorText).
		Render("Theme")

	subheader := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Bold(true)

	var rows []string
	rows = append(rows, title)

	// Split themes into dark and light groups
	var dark, light []indexedTheme
	for i, t := range theme.Themes {
		if t.Appearance == theme.Light {
			light = append(light, indexedTheme{i, t})
		} else {
			dark = append(dark, indexedTheme{i, t})
		}
	}

	if len(dark) > 0 {
		rows = append(rows, "", subheader.Render("  Dark"))
		rows = append(rows, m.renderThemeList(dark)...)
	}
	if len(light) > 0 {
		rows = append(rows, "", subheader.Render("  Light"))
		rows = append(rows, m.renderThemeList(light)...)
	}

	rows = append(rows, "")
	hint := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("enter:select  esc:back")
	rows = append(rows, hint)

	content := strings.Join(rows, "\n")
	return lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1).Render(content)
}

// indexedTheme pairs a theme with its index in the global Themes list.
type indexedTheme struct {
	index int
	theme theme.Theme
}

func (m *SettingsModel) renderThemeList(themes []indexedTheme) []string {
	var rows []string
	for _, it := range themes {
		swatch := colorSwatch(it.theme.Palette)
		check := "  "
		if it.index == m.active {
			check = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("✓ ")
		}

		var line string
		if it.index == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯")
			label := lipgloss.NewStyle().Foreground(theme.ColorText).Bold(true).Render(
				fmt.Sprintf(" %-18s", it.theme.Name),
			)
			line = cursor + label + check + swatch
		} else {
			label := lipgloss.NewStyle().Foreground(theme.ColorFgAlt).Render(
				fmt.Sprintf("  %-18s", it.theme.Name),
			)
			line = label + check + swatch
		}
		rows = append(rows, line)
	}
	return rows
}

// colorSwatch renders a row of colored blocks showing the palette's core colors
// on the palette's background, so light and dark themes are visually distinct.
func colorSwatch(p theme.Palette) string {
	colors := []string{p.Red, p.Orange, p.Yellow, p.Green, p.Cyan, p.Blue, p.Magenta, p.Comment, p.Fg}
	bg := lipgloss.Color(p.Bg)
	parts := make([]string, len(colors))
	for i, c := range colors {
		parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Background(bg).Render("██")
	}
	return strings.Join(parts, "")
}
