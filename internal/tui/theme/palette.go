// Package theme defines the color palette and styles for the c5s TUI.
package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Palette defines a complete color scheme. All hex values are strings so they
// can be consumed by both lipgloss (via lipgloss.Color) and glamour (via *string).
type Palette struct {
	// Core colors
	Fg      string `json:"fg"`      // primary foreground
	FgDim   string `json:"fg_dim"`  // dimmed foreground
	Bg      string `json:"bg"`      // primary background
	BgAlt   string `json:"bg_alt"`  // alternate/highlight background
	Comment string `json:"comment"` // muted/comment text
	Pink    string `json:"pink"`    // keywords, danger, headings
	Cyan    string `json:"cyan"`    // secondary accent, links, types
	Green   string `json:"green"`   // success, names, insertions
	Yellow  string `json:"yellow"`  // warnings, strings
	Purple  string `json:"purple"`  // constants, numbers, primary accent
	Orange  string `json:"orange"`  // secondary accent

	// Diff-specific (derived from core, but distinct enough to warrant naming)
	DiffAddFg    string `json:"diff_add_fg"`
	DiffAddBg    string `json:"diff_add_bg"`
	DiffRemoveFg string `json:"diff_remove_fg"`
	DiffRemoveBg string `json:"diff_remove_bg"`
}

// Dark palettes.
var (
	PaletteCatppuccinMocha = Palette{
		Fg:           "#CDD6F4", // text
		FgDim:        "#6C7086", // overlay0
		Bg:           "#1E1E2E", // base
		BgAlt:        "#313244", // surface0
		Comment:      "#6C7086", // overlay0
		Pink:         "#F38BA8", // red
		Cyan:         "#89DCEB", // sky
		Green:        "#A6E3A1", // green
		Yellow:       "#F9E2AF", // yellow
		Purple:       "#CBA6F7", // mauve
		Orange:       "#FAB387", // peach
		DiffAddFg:    "#A6E3A1",
		DiffAddBg:    "#233022",
		DiffRemoveFg: "#F38BA8",
		DiffRemoveBg: "#42252D",
	}

	PaletteDracula = Palette{
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
		DiffAddBg:    "#0F3017",
		DiffRemoveFg: "#FF5555",
		DiffRemoveBg: "#421616",
	}

	PaletteGitHubDark = Palette{
		Fg:           "#E6EDF3",
		FgDim:        "#7D8590",
		Bg:           "#0D1117",
		BgAlt:        "#161B22",
		Comment:      "#7D8590",
		Pink:         "#FF7B72", // red
		Cyan:         "#79C0FF", // blue
		Green:        "#7EE787",
		Yellow:       "#E3B341",
		Purple:       "#D2A8FF",
		Orange:       "#FFA657",
		DiffAddFg:    "#7EE787",
		DiffAddBg:    "#1A301C",
		DiffRemoveFg: "#FF7B72",
		DiffRemoveBg: "#421F1D",
	}

	PaletteMolokai = Palette{
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
		DiffAddBg:    "#233009",
		DiffRemoveFg: "#F92672",
		DiffRemoveBg: "#420A1E",
	}

	PaletteNord = Palette{
		Fg:           "#D8DEE9", // nord4
		FgDim:        "#4C566A", // nord3
		Bg:           "#2E3440", // nord0
		BgAlt:        "#3B4252", // nord1
		Comment:      "#616E88", // brightened nord3
		Pink:         "#BF616A", // nord11
		Cyan:         "#88C0D0", // nord8
		Green:        "#A3BE8C", // nord14
		Yellow:       "#EBCB8B", // nord13
		Purple:       "#B48EAD", // nord15
		Orange:       "#D08770", // nord12
		DiffAddFg:    "#A3BE8C",
		DiffAddBg:    "#293023",
		DiffRemoveFg: "#BF616A",
		DiffRemoveBg: "#422124",
	}

	PaletteSolarizedDark = Palette{
		Fg:           "#839496", // base0
		FgDim:        "#586E75", // base01
		Bg:           "#002B36", // base03
		BgAlt:        "#073642", // base02
		Comment:      "#586E75", // base01
		Pink:         "#DC322F", // red
		Cyan:         "#2AA198", // cyan
		Green:        "#859900", // green
		Yellow:       "#B58900", // yellow
		Purple:       "#6C71C4", // violet
		Orange:       "#CB4B16", // orange
		DiffAddFg:    "#859900",
		DiffAddBg:    "#293000",
		DiffRemoveFg: "#DC322F",
		DiffRemoveBg: "#420F0E",
	}

	PaletteTokyoNight = Palette{
		Fg:           "#C0CAF5",
		FgDim:        "#565F89",
		Bg:           "#1A1B26",
		BgAlt:        "#24283B",
		Comment:      "#565F89",
		Pink:         "#F7768E",
		Cyan:         "#7DCFFF",
		Green:        "#9ECE6A",
		Yellow:       "#E0AF68",
		Purple:       "#BB9AF7",
		Orange:       "#FF9E64",
		DiffAddFg:    "#9ECE6A",
		DiffAddBg:    "#243018",
		DiffRemoveFg: "#F7768E",
		DiffRemoveBg: "#421F25",
	}
)

// Light palettes.
var (
	PaletteCatppuccinLatte = Palette{
		Fg:           "#4C4F69", // text
		FgDim:        "#9CA0B0", // overlay0
		Bg:           "#EFF1F5", // base
		BgAlt:        "#E6E9EF", // mantle
		Comment:      "#9CA0B0", // overlay0
		Pink:         "#D20F39", // red
		Cyan:         "#04A5E5", // sky
		Green:        "#40A02B", // green
		Yellow:       "#DF8E1D", // yellow
		Purple:       "#8839EF", // mauve
		Orange:       "#FE640B", // peach
		DiffAddFg:    "#40A02B",
		DiffAddBg:    "#C0E0BA",
		DiffRemoveFg: "#D20F39",
		DiffRemoveBg: "#F2BAC6",
	}

	PaletteGitHubLight = Palette{
		Fg:           "#1F2328",
		FgDim:        "#656D76",
		Bg:           "#FFFFFF",
		BgAlt:        "#F6F8FA",
		Comment:      "#656D76",
		Pink:         "#CF222E", // red
		Cyan:         "#0969DA", // blue
		Green:        "#116329",
		Yellow:       "#9A6700",
		Purple:       "#8250DF",
		Orange:       "#BC4C00",
		DiffAddFg:    "#116329",
		DiffAddBg:    "#DAFBE1", // official GitHub diff green
		DiffRemoveFg: "#CF222E",
		DiffRemoveBg: "#FFEBE9", // official GitHub diff red
	}

	PaletteSolarizedLight = Palette{
		Fg:           "#657B83", // base00
		FgDim:        "#93A1A1", // base1
		Bg:           "#FDF6E3", // base3
		BgAlt:        "#EEE8D5", // base2
		Comment:      "#93A1A1", // base1
		Pink:         "#DC322F", // red
		Cyan:         "#2AA198", // cyan
		Green:        "#859900", // green
		Yellow:       "#B58900", // yellow
		Purple:       "#6C71C4", // violet
		Orange:       "#CB4B16", // orange
		DiffAddFg:    "#859900",
		DiffAddBg:    "#DDE3BA",
		DiffRemoveFg: "#DC322F",
		DiffRemoveBg: "#F3BABA",
	}

	PaletteTokyoNightDay = Palette{
		Fg:           "#3760BF",
		FgDim:        "#8990B3",
		Bg:           "#E1E2E7",
		BgAlt:        "#D0D5E3",
		Comment:      "#848CB5",
		Pink:         "#F52A65",
		Cyan:         "#007197",
		Green:        "#587539",
		Yellow:       "#8C6C3E",
		Purple:       "#7847BD",
		Orange:       "#B15C00",
		DiffAddFg:    "#587539",
		DiffAddBg:    "#C4CEBA",
		DiffRemoveFg: "#F52A65",
		DiffRemoveBg: "#FBBACD",
	}
)

// Theme pairs a name with a palette.
type Theme struct {
	Name    string  `json:"name"`
	Palette Palette `json:"palette"`
}

// Built-in themes — dark, then light, alphabetical within each group.
var (
	ThemeCatppuccinMocha = Theme{"Catppuccin Mocha", PaletteCatppuccinMocha}
	ThemeDracula         = Theme{"Dracula", PaletteDracula}
	ThemeGitHubDark      = Theme{"GitHub Dark", PaletteGitHubDark}
	ThemeMolokai         = Theme{"Molokai", PaletteMolokai}
	ThemeNord            = Theme{"Nord", PaletteNord}
	ThemeSolarizedDark   = Theme{"Solarized Dark", PaletteSolarizedDark}
	ThemeTokyoNight      = Theme{"Tokyo Night", PaletteTokyoNight}

	ThemeCatppuccinLatte = Theme{"Catppuccin Latte", PaletteCatppuccinLatte}
	ThemeGitHubLight     = Theme{"GitHub Light", PaletteGitHubLight}
	ThemeSolarizedLight  = Theme{"Solarized Light", PaletteSolarizedLight}
	ThemeTokyoNightDay   = Theme{"Tokyo Night Day", PaletteTokyoNightDay}
)

// DefaultTheme is the theme used when no config exists.
var DefaultTheme = ThemeMolokai

// Themes is the ordered list of available themes. User themes are appended
// by LoadUserThemes at startup. Dark themes first, then light.
var Themes = []Theme{
	// Dark
	ThemeCatppuccinMocha,
	ThemeDracula,
	ThemeGitHubDark,
	ThemeMolokai,
	ThemeNord,
	ThemeSolarizedDark,
	ThemeTokyoNight,
	// Light
	ThemeCatppuccinLatte,
	ThemeGitHubLight,
	ThemeSolarizedLight,
	ThemeTokyoNightDay,
}

// FindTheme returns the index and palette for a theme name, or -1 if not found.
func FindTheme(name string) (int, Palette, bool) {
	for i, t := range Themes {
		if t.Name == name {
			return i, t.Palette, true
		}
	}
	return -1, Palette{}, false
}

// LoadUserThemes reads JSON theme files from a directory and appends them to
// the Themes list. Files can be either a full Theme object ({"name": "...",
// "palette": {...}}) or a bare Palette object ({...colors...}). When a bare
// palette is used, the theme name is derived from the filename.
func LoadUserThemes(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		// Try parsing as a full Theme first ({"name": "...", "palette": {...}})
		var t Theme
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}

		// If palette fields are empty, the file is a bare Palette object
		if t.Palette.Fg == "" {
			var p Palette
			if err := json.Unmarshal(data, &p); err != nil || p.Fg == "" || p.Bg == "" {
				continue
			}
			t.Palette = p
		}

		if t.Palette.Fg == "" || t.Palette.Bg == "" {
			continue
		}
		if t.Name == "" {
			t.Name = strings.TrimSuffix(e.Name(), ".json")
		}
		Themes = append(Themes, t)
	}
}

// P is the active palette. Use ApplyPalette to change it at runtime.
var P = DefaultTheme.Palette

// ptr returns a pointer to a copy of v (glamour's ansi.StyleConfig needs *string).
func ptr[T any](v T) *T { return &v }
