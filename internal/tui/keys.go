package tui

import "slices"

// KeyMap defines the global key bindings for the application.
type KeyMap struct {
	Quit    []string
	Up      []string
	Down    []string
	PageUp  []string
	PageDn  []string
	Select  []string
	Back    []string
	Help    []string
	Approve []string
	Deny    []string
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:    []string{"q", "ctrl+c"},
		Up:      []string{"up", "k"},
		Down:    []string{"down", "j"},
		PageUp:  []string{"pgup"},
		PageDn:  []string{"pgdown"},
		Select:  []string{"enter"},
		Back:    []string{"escape", "esc"},
		Help:    []string{"?"},
		Approve: []string{"a"},
		Deny:    []string{"x"},
	}
}

// matches checks if a key string matches any binding in the list.
func matches(key string, bindings []string) bool {
	return slices.Contains(bindings, key)
}
