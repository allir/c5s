package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir := Dir()
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".config", "c5s")
		if dir != want {
			t.Errorf("Dir() = %q, want %q", dir, want)
		}
	})

	t.Run("custom XDG", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		dir := Dir()
		if dir != "/custom/config/c5s" {
			t.Errorf("Dir() = %q, want %q", dir, "/custom/config/c5s")
		}
	})
}

func TestStateDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		dir := stateDir()
		if !strings.HasSuffix(dir, filepath.Join(".local", "state", "c5s")) {
			t.Errorf("StateDir() = %q, expected .local/state/c5s suffix", dir)
		}
	})

	t.Run("custom XDG", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		dir := stateDir()
		if dir != "/custom/state/c5s" {
			t.Errorf("StateDir() = %q, want %q", dir, "/custom/state/c5s")
		}
	})
}
