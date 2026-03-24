package claude

import (
	"os"
	"path/filepath"
)

// C5sConfigDir returns the c5s config directory, respecting XDG_CONFIG_HOME.
func C5sConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "c5s")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "c5s")
	}
	return filepath.Join(home, ".config", "c5s")
}

// C5sStateDir returns the c5s state directory, respecting XDG_STATE_HOME.
func C5sStateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "c5s")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "c5s")
	}
	return filepath.Join(home, ".local", "state", "c5s")
}
