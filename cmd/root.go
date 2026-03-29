// Package cmd implements the CLI commands for c5s.
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/allir/c5s/internal/claude"
	"github.com/allir/c5s/internal/tui"
	"github.com/allir/c5s/internal/tui/theme"
	"github.com/allir/c5s/internal/version"
)

var refreshInterval time.Duration

var rootCmd = &cobra.Command{
	Use:   "c5s",
	Short: "A k9s-style TUI for managing Claude Code sessions",
	Long:  "c5s is a terminal user interface for discovering, monitoring, and managing multiple Claude Code sessions.",
	RunE:  runTUI,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("c5s %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
	},
}

func init() {
	rootCmd.Flags().DurationVar(&refreshInterval, "refresh", tui.DefaultRefreshInterval, "auto-refresh interval")
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Load user themes from config directory, then apply saved preference
	theme.LoadUserThemes(filepath.Join(claude.C5sConfigDir(), "themes"))
	cfg := claude.LoadConfig()
	activeTheme := theme.DefaultTheme.Name
	if _, p, ok := theme.FindTheme(cfg.Theme); ok {
		theme.ApplyPalette(p)
		activeTheme = cfg.Theme
	}

	configDir := claude.DefaultConfigDir()
	settingsPath := filepath.Join(configDir, "settings.json")

	// Install hooks for authoritative session discovery
	if err := claude.InstallHooks(settingsPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to install hooks: %v\n", err)
	}

	// Ensure cleanup runs exactly once on any exit path
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			_ = claude.UninstallHooks(settingsPath)
		})
	}
	defer cleanup()

	// Signal handler catches SIGINT/SIGTERM for cleanup even on abnormal exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cleanup()
	}()

	m := tui.NewModel(configDir, refreshInterval, activeTheme, cfg.UseThemeBg)
	p := tea.NewProgram(m)

	_, err := p.Run()
	return err
}
