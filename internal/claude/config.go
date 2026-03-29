package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds persisted c5s settings.
type Config struct {
	Theme      string `json:"theme"`
	UseThemeBg bool   `json:"use_theme_bg"`
}

// LoadConfig reads the config from C5sConfigDir()/config.json.
// Returns a zero-value Config on any error (missing file, bad JSON).
func LoadConfig() Config {
	data, err := os.ReadFile(filepath.Join(C5sConfigDir(), "config.json"))
	if err != nil {
		return Config{}
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	return cfg
}

// SaveConfig writes the config to C5sConfigDir()/config.json.
func SaveConfig(cfg Config) error {
	dir := C5sConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0o644)
}
