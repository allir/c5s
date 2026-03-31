// Package config handles c5s application configuration and XDG directory paths.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFile = "config.json"

// Config holds persisted c5s settings.
type Config struct {
	General GeneralConfig `json:"general"`
	Theme   ThemeConfig   `json:"theme"`
}

// GeneralConfig holds non-theme settings.
type GeneralConfig struct {
	RefreshMS int `json:"refresh_ms,omitempty"` // 0 = use default
}

// BackgroundFillMode controls how the theme background is painted.
type BackgroundFillMode string

const (
	BackgroundFillStandard BackgroundFillMode = "standard" // OSC 11 (works in native terminals)
	BackgroundFillFill     BackgroundFillMode = "fill"     // SGR cell-by-cell (tmux compatible)
)

// ThemeConfig holds theme-related settings.
type ThemeConfig struct {
	Name                string             `json:"name"`
	UseThemeBackground  bool               `json:"use_theme_background"`
	ThemeBackgroundMode BackgroundFillMode `json:"theme_background_mode,omitempty"` // default = standard
}

// Load reads the config from Dir()/config.json.
// Returns a zero-value Config on any error (missing file, bad JSON).
func Load() Config {
	data, err := os.ReadFile(filepath.Join(Dir(), configFile))
	if err != nil {
		return Config{}
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	if cfg.Theme.ThemeBackgroundMode == "" {
		cfg.Theme.ThemeBackgroundMode = BackgroundFillStandard
	}
	return cfg
}

// Save writes the config to Dir()/config.json.
func Save(cfg Config) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, configFile), data, 0o644)
}
