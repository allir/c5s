package theme

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/claude"
)

// Semantic color aliases — derived from the active palette.
var (
	ColorPrimary   color.Color
	ColorSecondary color.Color
	ColorMuted     color.Color
	ColorSuccess   color.Color
	ColorWarning   color.Color
	ColorDanger    color.Color
	ColorText      color.Color
	ColorFgAlt     color.Color
	ColorBg        color.Color
	ColorBgAlt     color.Color

	ColorDiffAddFg          color.Color
	ColorDiffAddBg          color.Color
	ColorDiffAddInlineBg    color.Color
	ColorDiffRemoveFg       color.Color
	ColorDiffRemoveBg       color.Color
	ColorDiffRemoveInlineBg color.Color
)

// Layout styles — rebuilt on palette change.
var (
	StyleHeader           lipgloss.Style
	StyleHeaderCount      lipgloss.Style
	StyleStatusBar        lipgloss.Style
	StyleStatusBarKey     lipgloss.Style
	StyleTableHeader      lipgloss.Style
	StyleTableRow         lipgloss.Style
	StyleTableRowSelected lipgloss.Style
	StyleTableCell        lipgloss.Style
)

func init() {
	ApplyPalette(P)
}

// ApplyPalette sets the active palette and rebuilds all derived colors and styles.
func ApplyPalette(p Palette) {
	P = p

	ColorPrimary = lipgloss.Color(p.Magenta)
	ColorSecondary = lipgloss.Color(p.Blue)
	ColorMuted = lipgloss.Color(p.Comment)
	ColorSuccess = lipgloss.Color(p.Green)
	ColorWarning = lipgloss.Color(p.Yellow)
	ColorDanger = lipgloss.Color(p.Red)
	ColorText = lipgloss.Color(p.Fg)
	ColorFgAlt = lipgloss.Color(p.FgAlt)
	ColorBg = lipgloss.Color(p.Bg)
	ColorBgAlt = lipgloss.Color(p.BgAlt)

	ColorDiffAddFg = lipgloss.Color(p.Diff.AddFg)
	ColorDiffAddBg = lipgloss.Color(p.Diff.AddBg)
	ColorDiffAddInlineBg = lipgloss.Color(p.Diff.AddInlineBg)
	ColorDiffRemoveFg = lipgloss.Color(p.Diff.RemoveFg)
	ColorDiffRemoveBg = lipgloss.Color(p.Diff.RemoveBg)
	ColorDiffRemoveInlineBg = lipgloss.Color(p.Diff.RemoveInlineBg)

	applyStyles()
}

func applyStyles() {
	StyleHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		PaddingLeft(1)

	StyleHeaderCount = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	StyleStatusBar = lipgloss.NewStyle().
		Foreground(ColorFgAlt).
		PaddingLeft(1)

	StyleStatusBarKey = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	StyleTableHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText)

	StyleTableRow = lipgloss.NewStyle().
		Foreground(ColorFgAlt)

	StyleTableRowSelected = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBgAlt).
		Bold(true)

	StyleTableCell = lipgloss.NewStyle().
		PaddingRight(2)
}

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
		return lipgloss.NewStyle().Foreground(ColorFgAlt)
	default:
		return lipgloss.NewStyle().Foreground(ColorFgAlt)
	}
}

// StatusIndicator returns a colored dot + label for the given status.
// If bg is non-nil, it's applied to the entire indicator (for selected rows).
func StatusIndicator(s claude.Status, bg ...color.Color) string {
	dotStyle := StatusStyle(s)
	labelStyle := lipgloss.NewStyle()
	if len(bg) > 0 && bg[0] != nil {
		dotStyle = dotStyle.Background(bg[0])
		labelStyle = labelStyle.Background(bg[0])
	}
	return dotStyle.Render("●") + labelStyle.Render(" "+s.String())
}
