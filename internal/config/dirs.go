package config

import (
	"os"
	"path/filepath"
)

const (
	appName = "c5s"

	dirThemes    = "themes"
	dirHooks     = "hooks"
	dirEvents    = "events"
	dirPending   = "pending"
	dirDecisions = "decisions"

	xdgConfigDir = ".config"
	xdgStateDir  = ".local/state"
)

// Dir returns the c5s config directory, respecting XDG_CONFIG_HOME.
func Dir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), appName)
	}
	return filepath.Join(home, xdgConfigDir, appName)
}

// ThemesDir returns the directory for user theme files.
func ThemesDir() string { return filepath.Join(Dir(), dirThemes) }

// HooksDir returns the directory for c5s hook scripts.
func HooksDir() string { return filepath.Join(Dir(), dirHooks) }

// stateDir returns the c5s state directory, respecting XDG_STATE_HOME.
func stateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, appName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), appName)
	}
	return filepath.Join(home, xdgStateDir, appName)
}

// EventsDir returns the directory for hook event files.
func EventsDir() string { return filepath.Join(stateDir(), dirEvents) }

// PendingDir returns the directory for pending approval files.
func PendingDir() string { return filepath.Join(stateDir(), dirPending) }

// DecisionsDir returns the directory for approval decision files.
func DecisionsDir() string { return filepath.Join(stateDir(), dirDecisions) }
