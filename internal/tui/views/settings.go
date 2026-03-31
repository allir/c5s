package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/config"
	"github.com/allir/c5s/internal/tui/theme"
)

// MenuItemKind identifies the type of a selectable settings item.
type MenuItemKind int

const (
	MenuRefresh  MenuItemKind = iota // cycles through refresh intervals
	MenuTheme                        // selects a theme
	MenuBgToggle                     // toggles theme background on/off
	MenuFillMode                     // cycles background fill mode
)

// menuItem is one selectable entry in the settings menu.
type menuItem struct {
	kind     MenuItemKind
	themeIdx int // index into theme.Themes (only for MenuTheme)
}

// indexedItem pairs a menuItem with its position in the flat items list.
type indexedItem struct {
	flatIdx int
	item    menuItem
}

// SettingsModel displays the settings screen with theme selection.
type SettingsModel struct {
	items              []menuItem
	cursor             int
	active             int // index into items for the currently applied theme
	UseThemeBackground bool
	BackgroundFillMode config.BackgroundFillMode
	RefreshInterval    time.Duration
	refreshOptions     []time.Duration
	width              int
	height             int
}

// NewSettingsModel creates a settings view with the cursor on the active theme.
func NewSettingsModel(activeTheme string, useThemeBg bool, fillMode config.BackgroundFillMode, refreshInterval time.Duration, refreshOptions []time.Duration) SettingsModel {
	m := SettingsModel{
		UseThemeBackground: useThemeBg,
		BackgroundFillMode: fillMode,
		RefreshInterval:    refreshInterval,
		refreshOptions:     refreshOptions,
	}
	m.rebuildItems()

	// Find which item is the active theme
	for i, item := range m.items {
		if item.kind == MenuTheme {
			if t := theme.Themes[item.themeIdx]; t.Name == activeTheme {
				m.active = i
				break
			}
		}
	}
	return m
}

// rebuildItems constructs the flat menu item list based on current state.
func (m *SettingsModel) rebuildItems() {
	m.items = m.items[:0]

	// General
	m.items = append(m.items, menuItem{kind: MenuRefresh})

	// Themes
	for i := range theme.Themes {
		m.items = append(m.items, menuItem{kind: MenuTheme, themeIdx: i})
	}

	// Background settings
	m.items = append(m.items, menuItem{kind: MenuBgToggle})
	if m.UseThemeBackground {
		m.items = append(m.items, menuItem{kind: MenuFillMode})
	}
}

// SetActive updates the active theme marker.
func (m *SettingsModel) SetActive(idx int) {
	m.active = idx
}

// SetSize updates the available dimensions.
func (m *SettingsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// MoveDown moves the cursor down.
func (m *SettingsModel) MoveDown() {
	if m.cursor < len(m.items)-1 {
		m.cursor++
	}
}

// MoveUp moves the cursor up.
func (m *SettingsModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// CurrentKind returns the kind of the menu item under the cursor.
func (m *SettingsModel) CurrentKind() MenuItemKind {
	return m.items[m.cursor].kind
}

// CycleFillMode toggles BackgroundFillMode between standard and fill.
func (m *SettingsModel) CycleFillMode() {
	if m.BackgroundFillMode == config.BackgroundFillFill {
		m.BackgroundFillMode = config.BackgroundFillStandard
	} else {
		m.BackgroundFillMode = config.BackgroundFillFill
	}
}

// CycleRefresh advances RefreshInterval to the next option.
func (m *SettingsModel) CycleRefresh() {
	for i, opt := range m.refreshOptions {
		if opt == m.RefreshInterval {
			m.RefreshInterval = m.refreshOptions[(i+1)%len(m.refreshOptions)]
			return
		}
	}
	if len(m.refreshOptions) > 0 {
		m.RefreshInterval = m.refreshOptions[0]
	}
}

// ClampCursor ensures the cursor stays within the menu after items change.
func (m *SettingsModel) ClampCursor() {
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
}

// RebuildAndClamp rebuilds the menu (e.g., after toggling bg) and clamps the cursor.
func (m *SettingsModel) RebuildAndClamp() {
	m.rebuildItems()
	m.ClampCursor()
}

// SelectedTheme returns the name and palette of the currently selected theme.
// Returns zero values if the cursor is not on a theme item.
func (m *SettingsModel) SelectedTheme() (string, theme.Palette) {
	item := m.items[m.cursor]
	if item.kind != MenuTheme {
		return "", theme.Palette{}
	}
	entry := theme.Themes[item.themeIdx]
	return entry.Name, entry.Palette
}

// Cursor returns the current cursor index.
func (m *SettingsModel) Cursor() int {
	return m.cursor
}

// View renders the settings screen.
func (m *SettingsModel) View() string {
	muted := lipgloss.NewStyle().Foreground(theme.ColorFgAlt)
	subheader := lipgloss.NewStyle().Foreground(theme.ColorMuted).Bold(true)
	sectionTitle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorText)

	var rows []string

	// General settings
	rows = append(rows, sectionTitle.Render("General"))
	rows = append(rows, "")
	rows = append(rows, m.renderOption("Refresh interval", muted.Render(formatDuration(m.RefreshInterval)), m.CurrentKind() == MenuRefresh))

	// Theme section
	rows = append(rows, "")
	rows = append(rows, sectionTitle.Render("Theme"))

	var dark, light []indexedItem
	for i, item := range m.items {
		if item.kind != MenuTheme {
			continue
		}
		ii := indexedItem{i, item}
		if theme.Themes[item.themeIdx].Appearance == theme.Light {
			light = append(light, ii)
		} else {
			dark = append(dark, ii)
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

	// Background settings
	rows = append(rows, "")
	bgVal := muted.Render("off")
	if m.UseThemeBackground {
		bgVal = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("on")
	}
	rows = append(rows, m.renderOption("Use theme background", bgVal, m.CurrentKind() == MenuBgToggle))
	if m.UseThemeBackground {
		rows = append(rows, m.renderOption("Background fill mode", muted.Render(string(m.BackgroundFillMode)), m.CurrentKind() == MenuFillMode))
	}

	rows = append(rows, "")
	hint := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("enter:select  esc:back")
	rows = append(rows, hint)

	content := strings.Join(rows, "\n")
	return lipgloss.NewStyle().PaddingLeft(2).PaddingTop(1).Render(content)
}

func (m *SettingsModel) renderThemeList(themes []indexedItem) []string {
	var rows []string
	for _, ii := range themes {
		t := theme.Themes[ii.item.themeIdx]
		swatch := colorSwatch(t.Palette)

		check := "  "
		if ii.flatIdx == m.active {
			check = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("✓ ")
		}

		var line string
		if ii.flatIdx == m.cursor {
			cursor := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯")
			label := lipgloss.NewStyle().Foreground(theme.ColorText).Bold(true).Render(
				fmt.Sprintf(" %-18s", t.Name),
			)
			line = cursor + label + check + swatch
		} else {
			label := lipgloss.NewStyle().Foreground(theme.ColorFgAlt).Render(
				fmt.Sprintf("  %-18s", t.Name),
			)
			line = label + check + swatch
		}
		rows = append(rows, line)
	}
	return rows
}

// renderOption renders a label: value pair with cursor highlight when selected.
// The value should be pre-styled by the caller.
func (m *SettingsModel) renderOption(label, styledValue string, selected bool) string {
	if selected {
		cursor := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯")
		lbl := lipgloss.NewStyle().Foreground(theme.ColorText).Bold(true).Render(" " + label + ": ")
		return cursor + lbl + styledValue
	}
	return lipgloss.NewStyle().Foreground(theme.ColorFgAlt).Render("  "+label+": ") + styledValue
}

func formatDuration(d time.Duration) string {
	if d >= time.Second && d%time.Second == 0 {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
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
