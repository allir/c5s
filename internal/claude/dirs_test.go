package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestC5sConfigDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		dir := C5sConfigDir()
		home, _ := os.UserHomeDir()
		want := filepath.Join(home, ".config", "c5s")
		if dir != want {
			t.Errorf("C5sConfigDir() = %q, want %q", dir, want)
		}
	})

	t.Run("custom XDG", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		dir := C5sConfigDir()
		if dir != "/custom/config/c5s" {
			t.Errorf("C5sConfigDir() = %q, want %q", dir, "/custom/config/c5s")
		}
	})
}

func TestC5sStateDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		dir := C5sStateDir()
		if !strings.HasSuffix(dir, filepath.Join(".local", "state", "c5s")) {
			t.Errorf("C5sStateDir() = %q, expected .local/state/c5s suffix", dir)
		}
	})

	t.Run("custom XDG", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		dir := C5sStateDir()
		if dir != "/custom/state/c5s" {
			t.Errorf("C5sStateDir() = %q, want %q", dir, "/custom/state/c5s")
		}
	})
}
