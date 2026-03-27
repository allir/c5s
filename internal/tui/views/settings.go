package views

import (
	"fmt"
	"strconv"
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
	if i, _, ok := theme.FindTheme(activeTheme); ok {
		cursor = i
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
	if m.cursor < len(theme.Themes)-1 {
		m.cursor++
	}
}

// SelectedTheme returns the name and palette of the currently selected theme.
func (m *SettingsModel) SelectedTheme() (string, theme.Palette) {
	entry := theme.Themes[m.cursor]
	return entry.Name, entry.Palette
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
		if isLightBg(t.Palette.Bg) {
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

		var line string
		if it.index == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯")
			label := lipgloss.NewStyle().Foreground(theme.ColorText).Bold(true).Render(
				fmt.Sprintf(" %-18s", it.theme.Name),
			)
			line = cursor + label + "  " + swatch
		} else {
			label := lipgloss.NewStyle().Foreground(theme.ColorDimText).Render(
				fmt.Sprintf("  %-18s", it.theme.Name),
			)
			line = label + "  " + swatch
		}
		rows = append(rows, line)
	}
	return rows
}

// isLightBg returns true if a hex color string has high luminance (light background).
func isLightBg(hex string) bool {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return false
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	// Perceived luminance (ITU-R BT.601)
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	return lum > 128
}

// colorSwatch renders a row of colored blocks showing the palette's core colors
// on the palette's background, so light and dark themes are visually distinct.
func colorSwatch(p theme.Palette) string {
	colors := []string{p.Pink, p.Green, p.Yellow, p.Cyan, p.Purple, p.Orange, p.Comment, p.Fg}
	bg := lipgloss.Color(p.Bg)
	parts := make([]string, len(colors))
	for i, c := range colors {
		parts[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Background(bg).Render("██")
	}
	return strings.Join(parts, "")
}
