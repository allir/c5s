// Package theme defines the color palette and styles for the c5s TUI.
package theme

// Palette defines a complete color scheme. All hex values are strings so they
// can be consumed by both lipgloss (via lipgloss.Color) and glamour (via *string).
type Palette struct {
	// Core colors
	Fg      string // primary foreground
	FgDim   string // dimmed foreground
	Bg      string // primary background
	BgAlt   string // alternate/highlight background
	Comment string // muted/comment text
	Pink    string // keywords, danger, headings
	Cyan    string // secondary accent, links, types
	Green   string // success, names, insertions
	Yellow  string // warnings, strings
	Purple  string // constants, numbers, primary accent
	Orange  string // not in classic monokai, but useful for some tokens

	// Diff-specific (derived from core, but distinct enough to warrant naming)
	DiffAddFg    string
	DiffAddBg    string
	DiffRemoveFg string
	DiffRemoveBg string
}

// Named palettes.
var (
	Monokai = Palette{
		Fg:           "#F8F8F2",
		FgDim:        "#90908A",
		Bg:           "#272822",
		BgAlt:        "#3E3D32",
		Comment:      "#75715E",
		Pink:         "#F92672",
		Cyan:         "#66D9EF",
		Green:        "#A6E22E",
		Yellow:       "#E6DB74",
		Purple:       "#AE81FF",
		Orange:       "#FD971F",
		DiffAddFg:    "#A6E22E",
		DiffAddBg:    "#2B3A1A",
		DiffRemoveFg: "#F92672",
		DiffRemoveBg: "#3A1A22",
	}

	Molokai = Palette{
		Fg:           "#F8F8F2",
		FgDim:        "#90908A",
		Bg:           "#1B1D1E",
		BgAlt:        "#2D2E27",
		Comment:      "#7E8E91",
		Pink:         "#F92672",
		Cyan:         "#66D9EF",
		Green:        "#A6E22E",
		Yellow:       "#E6DB74",
		Purple:       "#AE81FF",
		Orange:       "#FD971F",
		DiffAddFg:    "#A6E22E",
		DiffAddBg:    "#1E2E12",
		DiffRemoveFg: "#F92672",
		DiffRemoveBg: "#2E1218",
	}

	Dracula = Palette{
		Fg:           "#F8F8F2",
		FgDim:        "#6272A4",
		Bg:           "#282A36",
		BgAlt:        "#44475A",
		Comment:      "#6272A4",
		Pink:         "#FF79C6",
		Cyan:         "#8BE9FD",
		Green:        "#50FA7B",
		Yellow:       "#F1FA8C",
		Purple:       "#BD93F9",
		Orange:       "#FFB86C",
		DiffAddFg:    "#50FA7B",
		DiffAddBg:    "#1A2E1A",
		DiffRemoveFg: "#FF5555",
		DiffRemoveBg: "#3A1A1A",
	}
)

// PaletteEntry pairs a display name with a palette for ordered iteration.
type PaletteEntry struct {
	Name    string
	Palette Palette
}

// Palettes is the ordered list of available themes.
var Palettes = []PaletteEntry{
	{"Molokai", Molokai},
	{"Monokai", Monokai},
	{"Dracula", Dracula},
}

// P is the active palette. Use ApplyPalette to change it at runtime.
var P = Molokai

// ptr returns a pointer to a copy of v (glamour's ansi.StyleConfig needs *string).
func ptr[T any](v T) *T { return &v }
