// Package theme defines the color palette and styles for the c5s TUI.
package theme

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Palette defines a complete color scheme. All hex values are strings so they
// can be consumed by both lipgloss (via lipgloss.Color) and glamour (via *string).
// DiffPalette defines colors for diff rendering (add/remove backgrounds and inline highlights).
type DiffPalette struct {
	AddFg          string `json:"add_fg"`
	AddBg          string `json:"add_bg"`
	AddInlineBg    string `json:"add_inline_bg"`
	RemoveFg       string `json:"remove_fg"`
	RemoveBg       string `json:"remove_bg"`
	RemoveInlineBg string `json:"remove_inline_bg"`
}

// Palette defines a color scheme. Accent color names are ANSI-inspired slot names —
// themes are free to assign any hex value to any slot.
type Palette struct {
	// Core shades
	Fg      string `json:"fg"`      // primary foreground
	FgAlt   string `json:"fg_alt"`  // alternate/dimmed foreground
	Bg      string `json:"bg"`      // primary background
	BgAlt   string `json:"bg_alt"`  // alternate/highlight background
	Comment string `json:"comment"` // comments, muted text

	// Accent colors (ANSI-inspired)
	Red     string `json:"red"`
	Orange  string `json:"orange"`
	Yellow  string `json:"yellow"`
	Green   string `json:"green"`
	Cyan    string `json:"cyan"`
	Blue    string `json:"blue"`
	Magenta string `json:"magenta"`
	Brown   string `json:"brown"`

	// Diff rendering colors
	Diff DiffPalette `json:"diff"`
}

// Dark palettes.
var (
	PaletteCatppuccinMocha = Palette{
		Fg:      "#CDD6F4", // text
		FgAlt:   "#6C7086", // overlay0
		Bg:      "#1E1E2E", // base
		BgAlt:   "#313244", // surface0
		Comment: "#6C7086", // overlay0
		Red:     "#F38BA8", // red
		Blue:    "#89DCEB", // sky
		Green:   "#A6E3A1", // green
		Cyan:    "#94E2D5", // teal
		Yellow:  "#F9E2AF", // yellow
		Magenta: "#CBA6F7", // mauve
		Orange:  "#FAB387", // peach
		Brown:   "#F2CDCD", // rosewater
		Diff: DiffPalette{
			AddFg:          "#A6E3A1",
			AddBg:          "#323B3F",
			AddInlineBg:    "#394545",
			RemoveFg:       "#F38BA8",
			RemoveBg:       "#3D2E40",
			RemoveInlineBg: "#483346",
		},
	}

	PaletteGitHubDark = Palette{
		Fg:      "#E6EDF3",
		FgAlt:   "#7D8590",
		Bg:      "#0D1117",
		BgAlt:   "#161B22",
		Comment: "#7D8590",
		Red:     "#FF7B72", // red
		Blue:    "#79C0FF", // blue
		Green:   "#7EE787",
		Cyan:    "#39D2C0",
		Yellow:  "#E3B341",
		Magenta: "#D2A8FF",
		Orange:  "#FFA657",
		Brown:   "#A5834B",
		Diff: DiffPalette{
			AddFg:          "#7EE787",
			AddBg:          "#11271D",
			AddInlineBg:    "#31513F",
			RemoveFg:       "#FF7B72",
			RemoveBg:       "#2B171E",
			RemoveInlineBg: "#532F37",
		},
	}

	PaletteMolokai = Palette{
		Fg:      "#F8F8F2",
		FgAlt:   "#90908A",
		Bg:      "#1B1D1E",
		BgAlt:   "#2D2E27",
		Comment: "#7E8E91",
		Red:     "#F92672",
		Blue:    "#66D9EF",
		Green:   "#A6E22E",
		Cyan:    "#A1EFE4",
		Yellow:  "#E6DB74",
		Magenta: "#AE81FF",
		Orange:  "#FD971F",
		Brown:   "#CC6633",
		Diff: DiffPalette{
			AddFg:          "#A6E22E",
			AddBg:          "#2A380C",
			AddInlineBg:    "#39471B",
			RemoveFg:       "#F92672",
			RemoveBg:       "#461824",
			RemoveInlineBg: "#552733",
		},
	}

	PaletteNord = Palette{
		Fg:      "#D8DEE9", // nord4
		FgAlt:   "#4C566A", // nord3
		Bg:      "#2E3440", // nord0
		BgAlt:   "#3B4252", // nord1
		Comment: "#616E88", // brightened nord3
		Red:     "#BF616A", // nord11
		Blue:    "#88C0D0", // nord8
		Green:   "#A3BE8C", // nord14
		Cyan:    "#8FBCBB", // nord7
		Yellow:  "#EBCB8B", // nord13
		Magenta: "#B48EAD", // nord15
		Orange:  "#D08770", // nord12
		Brown:   "#A3685A",
		Diff: DiffPalette{
			AddFg:          "#88C0D0",
			AddBg:          "#2F3A4A",
			AddInlineBg:    "#3E4959",
			RemoveFg:       "#BF616A",
			RemoveBg:       "#4A323D",
			RemoveInlineBg: "#59414C",
		},
	}

	PaletteSolarizedDark = Palette{
		Fg:      "#839496", // base0
		FgAlt:   "#586E75", // base01
		Bg:      "#002B36", // base03
		BgAlt:   "#073642", // base02
		Comment: "#586E75", // base01
		Red:     "#DC322F", // red
		Blue:    "#268BD2", // blue
		Green:   "#859900", // green
		Cyan:    "#2AA198", // solarized cyan
		Yellow:  "#B58900", // yellow
		Magenta: "#6C71C4", // violet
		Orange:  "#CB4B16", // orange
		Brown:   "#D33682",
		Diff: DiffPalette{
			AddFg:          "#14A73A",
			AddBg:          "#003427",
			AddInlineBg:    "#044336",
			RemoveFg:       "#DC322F",
			RemoveBg:       "#1D1D25",
			RemoveInlineBg: "#2C2C34",
		},
	}

	PaletteTokyoNight = Palette{
		Fg:      "#C0CAF5",
		FgAlt:   "#565F89",
		Bg:      "#1A1B26",
		BgAlt:   "#24283B",
		Comment: "#565F89",
		Red:     "#F7768E",
		Blue:    "#7DCFFF",
		Green:   "#9ECE6A",
		Cyan:    "#2AC3DE",
		Yellow:  "#E0AF68",
		Magenta: "#BB9AF7",
		Orange:  "#FF9E64",
		Brown:   "#DB4B4B",
		Diff: DiffPalette{
			AddFg:          "#41A6B5",
			AddBg:          "#1E2C37",
			AddInlineBg:    "#1E2C37",
			RemoveFg:       "#DB4B4B",
			RemoveBg:       "#33212A",
			RemoveInlineBg: "#33212A",
		},
	}
)

// Light palettes.
var (
	PaletteCatppuccinLatte = Palette{
		Fg:      "#4C4F69", // text
		FgAlt:   "#9CA0B0", // overlay0
		Bg:      "#EFF1F5", // base
		BgAlt:   "#E6E9EF", // mantle
		Comment: "#9CA0B0", // overlay0
		Red:     "#D20F39", // red
		Blue:    "#04A5E5", // sky
		Green:   "#40A02B", // green
		Cyan:    "#179299", // teal
		Yellow:  "#DF8E1D", // yellow
		Magenta: "#8839EF", // mauve
		Orange:  "#FE640B", // peach
		Brown:   "#DD7878", // flamingo
		Diff: DiffPalette{
			AddFg:          "#40A02B",
			AddBg:          "#D4E4D6",
			AddInlineBg:    "#CCE0CC",
			RemoveFg:       "#D20F39",
			RemoveBg:       "#EACFD8",
			RemoveInlineBg: "#E9C3CF",
		},
	}

	PaletteGitHubLight = Palette{
		Fg:      "#1F2328",
		FgAlt:   "#656D76",
		Bg:      "#FFFFFF",
		BgAlt:   "#F6F8FA",
		Comment: "#656D76",
		Red:     "#CF222E", // red
		Blue:    "#0969DA", // blue
		Green:   "#116329",
		Cyan:    "#1B7C83",
		Yellow:  "#9A6700",
		Magenta: "#8250DF",
		Orange:  "#BC4C00",
		Brown:   "#953800",
		Diff: DiffPalette{
			AddFg:          "#116329",
			AddBg:          "#F4FFF6",
			AddInlineBg:    "#DEFAE5",
			RemoveFg:       "#CF222E",
			RemoveBg:       "#FFF4F5",
			RemoveInlineBg: "#FCC7CD",
		},
	}

	PaletteOneLight = Palette{
		Fg:      "#383A42", // mono-1
		FgAlt:   "#696C77", // mono-2
		Bg:      "#FAFAFA",
		BgAlt:   "#F2F2F2",
		Comment: "#A0A1A7", // mono-3
		Red:     "#A626A4", // hue-3 (keywords)
		Blue:    "#4078F2", // hue-2 (functions)
		Green:   "#50A14F", // hue-4 (strings)
		Cyan:    "#0184BC",
		Yellow:  "#C18401", // hue-6-2 (types)
		Magenta: "#986801", // hue-6 (numbers)
		Orange:  "#E45649", // hue-5 (tags/variables)
		Brown:   "#CA1243",
		Diff: DiffPalette{
			AddFg:          "#50A14F",
			AddBg:          "#D4ECCE",
			AddInlineBg:    "#BBE0B4",
			RemoveFg:       "#E45649",
			RemoveBg:       "#F8D7D0",
			RemoveInlineBg: "#F2BBB5",
		},
	}

	PaletteSolarizedLight = Palette{
		Fg:      "#657B83", // base00
		FgAlt:   "#93A1A1", // base1
		Bg:      "#FDF6E3", // base3
		BgAlt:   "#EEE8D5", // base2
		Comment: "#93A1A1", // base1
		Red:     "#DC322F", // red
		Blue:    "#268BD2", // blue
		Green:   "#859900", // green
		Cyan:    "#2AA198", // solarized cyan
		Yellow:  "#B58900", // yellow
		Magenta: "#6C71C4", // violet
		Orange:  "#CB4B16", // orange
		Brown:   "#D33682",
		Diff: DiffPalette{
			AddFg:          "#14A73A",
			AddBg:          "#DDF5D0",
			AddInlineBg:    "#CEE6C1",
			RemoveFg:       "#DC322F",
			RemoveBg:       "#FFDDCE",
			RemoveInlineBg: "#F6CEBF",
		},
	}

	PaletteNordLight = Palette{
		Fg:      "#2E3440", // nord0
		FgAlt:   "#4C566A", // nord3
		Bg:      "#ECEFF4", // nord6
		BgAlt:   "#E5E9F0", // nord5
		Comment: "#4C566A", // nord3
		Red:     "#BF616A", // nord11
		Blue:    "#88C0D0", // nord8
		Green:   "#A3BE8C", // nord14
		Cyan:    "#8FBCBB", // nord7
		Yellow:  "#EBCB8B", // nord13
		Magenta: "#B48EAD", // nord15
		Orange:  "#D08770", // nord12
		Brown:   "#A3685A",
		Diff: DiffPalette{
			AddFg:          "#81A1C1",
			AddBg:          "#DBE3EC",
			AddInlineBg:    "#D6DFE9",
			RemoveFg:       "#BF616A",
			RemoveBg:       "#E5D9DF",
			RemoveInlineBg: "#E3D2D8",
		},
	}

	PaletteTokyoNightLight = Palette{
		Fg:      "#3760BF",
		FgAlt:   "#8990B3",
		Bg:      "#E1E2E7",
		BgAlt:   "#D0D5E3",
		Comment: "#848CB5",
		Red:     "#F52A65",
		Blue:    "#2E7DE9",
		Green:   "#587539",
		Cyan:    "#007197",
		Yellow:  "#8C6C3E",
		Magenta: "#7847BD",
		Orange:  "#B15C00",
		Brown:   "#8C4351",
		Diff: DiffPalette{
			AddFg:          "#2D9C91",
			AddBg:          "#DDECF0",
			AddInlineBg:    "#CEDDE1",
			RemoveFg:       "#E86868",
			RemoveBg:       "#F5EDF2",
			RemoveInlineBg: "#E6DEE3",
		},
	}
)

// Appearance indicates whether a theme is dark or light.
type Appearance string

const (
	Dark  Appearance = "dark"
	Light Appearance = "light"
)

// Theme pairs a name with a palette and its appearance.
type Theme struct {
	Name       string     `json:"name"`
	Appearance Appearance `json:"appearance"`
	Palette    Palette    `json:"palette"`
}

// Built-in themes — dark, then light, alphabetical within each group.
var (
	ThemeCatppuccinMocha = Theme{"Catppuccin Mocha", Dark, PaletteCatppuccinMocha}
	ThemeGitHubDark      = Theme{"GitHub Dark", Dark, PaletteGitHubDark}
	ThemeMolokai         = Theme{"Molokai", Dark, PaletteMolokai}
	ThemeNord            = Theme{"Nord", Dark, PaletteNord}
	ThemeSolarizedDark   = Theme{"Solarized Dark", Dark, PaletteSolarizedDark}
	ThemeTokyoNight      = Theme{"Tokyo Night", Dark, PaletteTokyoNight}

	ThemeCatppuccinLatte = Theme{"Catppuccin Latte", Light, PaletteCatppuccinLatte}
	ThemeGitHubLight     = Theme{"GitHub Light", Light, PaletteGitHubLight}
	ThemeNordLight       = Theme{"Nord Light", Light, PaletteNordLight}
	ThemeOneLight        = Theme{"One Light", Light, PaletteOneLight}
	ThemeSolarizedLight  = Theme{"Solarized Light", Light, PaletteSolarizedLight}
	ThemeTokyoNightLight = Theme{"Tokyo Night Light", Light, PaletteTokyoNightLight}
)

// DefaultTheme is the theme used when no config exists.
var DefaultTheme = ThemeMolokai

// Themes is the ordered list of available themes. User themes are appended
// by LoadUserThemes at startup. Dark themes first, then light.
var Themes = []Theme{
	// Dark
	ThemeCatppuccinMocha,
	ThemeGitHubDark,
	ThemeMolokai,
	ThemeNord,
	ThemeSolarizedDark,
	ThemeTokyoNight,
	// Light
	ThemeCatppuccinLatte,
	ThemeGitHubLight,
	ThemeNordLight,
	ThemeOneLight,
	ThemeSolarizedLight,
	ThemeTokyoNightLight,
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
		if t.Appearance == "" {
			t.Appearance = Dark // default to dark if not specified
		}
		Themes = append(Themes, t)
	}

	// Re-sort: dark themes first, then light, alphabetical within each group.
	slices.SortStableFunc(Themes, func(a, b Theme) int {
		if a.Appearance != b.Appearance {
			if a.Appearance == Light {
				return 1 // light after dark
			}
			return -1
		}
		return cmp.Compare(a.Name, b.Name)
	})
}

// P is the active palette. Use ApplyPalette to change it at runtime.
var P = DefaultTheme.Palette

// ptr returns a pointer to a copy of v (glamour's ansi.StyleConfig needs *string).
func ptr[T any](v T) *T { return &v }
