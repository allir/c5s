//go:build debug

package views

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/tui/theme"
)

// DiffDebugModel displays example diff lines for every theme.
type DiffDebugModel struct {
	scroll int
	width  int
	height int
}

// NewDiffDebugModel creates the debug view.
func NewDiffDebugModel() DiffDebugModel {
	return DiffDebugModel{}
}

// SetSize updates dimensions.
func (m *DiffDebugModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// ScrollUp scrolls up.
func (m *DiffDebugModel) ScrollUp() {
	if m.scroll > 0 {
		m.scroll--
	}
}

// ScrollDown scrolls down.
func (m *DiffDebugModel) ScrollDown() {
	m.scroll++
}

// View renders example diff blocks for every theme, grouped by appearance.
// Dark themes render on a dark terminal bg, light themes on a white terminal bg.
func (m *DiffDebugModel) View() string {
	var dark, light []theme.Theme
	for _, t := range theme.Themes {
		if t.Appearance == theme.Light {
			light = append(light, t)
		} else {
			dark = append(dark, t)
		}
	}

	var lines []string
	colW := min((m.width-10)/2, 44)

	// Dark section — no explicit terminal bg (uses terminal default)
	darkHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#CCCCCC")).Render
	darkMuted := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render

	lines = append(lines, "")
	lines = append(lines, "  "+darkHeader("Dark Themes")+"  "+darkMuted("left: terminal bg  right: theme bg"))
	lines = append(lines, "")

	for _, t := range dark {
		lines = append(lines, m.renderThemeBlock(t, nil, colW)...)
		lines = append(lines, "")
	}

	// Light section — white terminal bg painted across full width
	whiteBg := lipgloss.Color("#FFFFFF")
	row := lipgloss.NewStyle().Background(whiteBg).Width(m.width)
	lightHeader := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#333333")).Background(whiteBg).Render
	lightMuted := lipgloss.NewStyle().Foreground(lipgloss.Color("#999999")).Background(whiteBg).Render

	lines = append(lines, row.Render(""))
	lines = append(lines, row.Render("  "+lightHeader("Light Themes")+"  "+lightMuted("left: white terminal bg  right: theme bg")))
	lines = append(lines, row.Render(""))

	for _, t := range light {
		for _, l := range m.renderThemeBlock(t, &whiteBg, colW) {
			lines = append(lines, row.Render(l))
		}
		lines = append(lines, row.Render(""))
	}

	hint := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("esc:back  ↑/↓:scroll")
	lines = append(lines, "", "  "+hint)

	// Scroll window
	start := min(m.scroll, len(lines))
	end := min(start+m.height, len(lines))

	return strings.Join(lines[start:end], "\n")
}

// renderThemeBlock renders a theme's header + side-by-side diff columns.
// termBg is the simulated terminal background (nil = terminal default).
func (m *DiffDebugModel) renderThemeBlock(t theme.Theme, termBg *color.Color, colW int) []string {
	label := fmt.Sprintf("  %s", t.Name)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorText)
	if termBg != nil {
		headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#333333"))
	}
	header := headerStyle.Render(label)

	themeBg := color.Color(lipgloss.Color(t.Palette.Bg))
	left := buildDiffBlock(t.Palette, termBg, colW)    // terminal bg
	right := buildDiffBlock(t.Palette, &themeBg, colW) // theme bg
	gap := "  "

	var lines []string
	lines = append(lines, header)
	for i := range left {
		lines = append(lines, "    "+left[i]+gap+right[i])
	}
	return lines
}

// buildDiffBlock renders 4 example diff lines with the given theme's diff colors.
// bg sets the context line and padding background. nil = no explicit background.
func buildDiffBlock(p theme.Palette, bg *color.Color, w int) []string {
	d := p.Diff
	addFg := lipgloss.Color(d.AddFg)
	addBgC := lipgloss.Color(d.AddBg)
	addInline := lipgloss.Color(d.AddInlineBg)
	remFg := lipgloss.Color(d.RemoveFg)
	remBgC := lipgloss.Color(d.RemoveBg)
	remInline := lipgloss.Color(d.RemoveInlineBg)
	ctxFg := lipgloss.Color(p.Comment)

	ctx := lipgloss.NewStyle().Foreground(ctxFg)
	add := lipgloss.NewStyle().Foreground(addFg).Background(addBgC)
	addHi := lipgloss.NewStyle().Foreground(addFg).Background(addInline)
	rem := lipgloss.NewStyle().Foreground(remFg).Background(remBgC)
	remHi := lipgloss.NewStyle().Foreground(remFg).Background(remInline)
	pad := lipgloss.NewStyle().Width(w)

	if bg != nil {
		ctx = ctx.Background(*bg)
		pad = pad.Background(*bg)
	}

	return []string{
		pad.Render(ctx.Render("  10   func main() {")),
		pad.Render(rem.Render(" 11 - ") + rem.Render("fmt.Println(") + remHi.Render(`"hello"`) + rem.Render(")")),
		pad.Render(add.Render(" 11 + ") + add.Render("fmt.Println(") + addHi.Render(`"world"`) + add.Render(")")),
		pad.Render(ctx.Render("  12   }")),
	}
}
