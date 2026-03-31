package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/allir/c5s/internal/config"
)

func TestInstallUninstallHooks(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Override XDG dirs for test isolation
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	// Install into fresh settings
	if err := InstallHooks(settingsPath); err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}

	// Verify settings were written
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("no hooks in settings")
	}

	// Check all expected events are registered
	for _, he := range hookEvents {
		eventHooks, ok := hooks[he.event].([]any)
		if !ok || len(eventHooks) == 0 {
			t.Errorf("event %q not registered", he.event)
		}
	}

	// Verify hook scripts exist and are executable
	scriptPath := hookScriptPath()
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("status hook script not found: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Error("status hook script not executable")
	}

	approvalPath := approvalHookScriptPath()
	info, err = os.Stat(approvalPath)
	if err != nil {
		t.Fatalf("approval hook script not found: %v", err)
	}
	if info.Mode()&0o100 == 0 {
		t.Error("approval hook script not executable")
	}

	// Verify state directories exist
	for _, dir := range []string{config.EventsDir(), config.PendingDir(), config.DecisionsDir()} {
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("state dir not found: %v (%s)", err, dir)
		}
	}

	// Install again — should be idempotent
	if err := InstallHooks(settingsPath); err != nil {
		t.Fatalf("InstallHooks (idempotent): %v", err)
	}

	data2, _ := os.ReadFile(settingsPath)
	var settings2 map[string]any
	_ = json.Unmarshal(data2, &settings2)
	hooks2, _ := settings2["hooks"].(map[string]any)

	for _, he := range hookEvents {
		eventHooks, _ := hooks2[he.event].([]any)
		if len(eventHooks) != 1 {
			t.Errorf("event %q has %d entries after idempotent install, want 1", he.event, len(eventHooks))
		}
	}

	// PermissionRequest should have exactly 1 entry (approval hook only, no status hook)
	permHooks, _ := hooks2["PermissionRequest"].([]any)
	if len(permHooks) != 1 {
		t.Errorf("PermissionRequest has %d entries after idempotent install, want 1", len(permHooks))
	}

	// Uninstall
	if err := UninstallHooks(settingsPath); err != nil {
		t.Fatalf("UninstallHooks: %v", err)
	}

	data3, _ := os.ReadFile(settingsPath)
	var settings3 map[string]any
	_ = json.Unmarshal(data3, &settings3)

	if _, ok := settings3["hooks"]; ok {
		t.Error("hooks key still present after uninstall")
	}

	// Scripts should be removed
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Error("status hook script not cleaned up")
	}
	if _, err := os.Stat(approvalPath); !os.IsNotExist(err) {
		t.Error("approval hook script not cleaned up")
	}

	// State dirs should be removed
	for _, dir := range []string{config.EventsDir(), config.PendingDir(), config.DecisionsDir()} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("state dir not cleaned up: %s", dir)
		}
	}
}

func TestInstallPreservesExistingSettings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))

	// Write existing settings with other hooks
	existing := map[string]any{
		"permissions": map[string]any{"allow": []any{"Read"}},
		"hooks": map[string]any{
			"UserPromptSubmit": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": "/other/tool/hook.sh",
						},
					},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	_ = os.WriteFile(settingsPath, data, 0o644)

	if err := InstallHooks(settingsPath); err != nil {
		t.Fatalf("InstallHooks: %v", err)
	}

	// Verify existing settings preserved
	data, _ = os.ReadFile(settingsPath)
	var settings map[string]any
	_ = json.Unmarshal(data, &settings)

	if _, ok := settings["permissions"]; !ok {
		t.Error("permissions key lost after install")
	}

	hooks := settings["hooks"].(map[string]any)
	promptHooks := hooks["UserPromptSubmit"].([]any)

	// Should have both: the existing hook and our new one
	if len(promptHooks) != 2 {
		t.Errorf("UserPromptSubmit has %d entries, want 2", len(promptHooks))
	}

	// Uninstall should only remove ours
	if err := UninstallHooks(settingsPath); err != nil {
		t.Fatalf("UninstallHooks: %v", err)
	}

	data, _ = os.ReadFile(settingsPath)
	_ = json.Unmarshal(data, &settings)

	hooks = settings["hooks"].(map[string]any)
	promptHooks = hooks["UserPromptSubmit"].([]any)
	if len(promptHooks) != 1 {
		t.Errorf("UserPromptSubmit has %d entries after uninstall, want 1", len(promptHooks))
	}
}

func TestReadHookEvents(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	evDir := filepath.Join(tmpDir, "c5s", "events")
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a valid event file
	ev := HookEvent{
		Event:          "UserPromptSubmit",
		SessionID:      "abc123",
		Cwd:            "/home/user/project",
		TranscriptPath: "/home/user/.claude/projects/project/abc123.jsonl",
		Timestamp:      time.Now(),
	}
	data, _ := json.Marshal(ev)
	_ = os.WriteFile(filepath.Join(evDir, "12345.json"), data, 0o644)

	// Write a stale event file (fake old mtime)
	_ = os.WriteFile(filepath.Join(evDir, "99999.json"), data, 0o644)
	staleTime := time.Now().Add(-25 * time.Hour)
	_ = os.Chtimes(filepath.Join(evDir, "99999.json"), staleTime, staleTime)

	// Write a non-JSON file (should be ignored)
	_ = os.WriteFile(filepath.Join(evDir, "readme.txt"), []byte("nope"), 0o644)

	events, err := ReadHookEvents()
	if err != nil {
		t.Fatalf("ReadHookEvents: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	got, ok := events[12345]
	if !ok {
		t.Fatal("event for PID 12345 not found")
	}
	if got.SessionID != "abc123" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "abc123")
	}
	if got.Event != "UserPromptSubmit" {
		t.Errorf("Event = %q, want %q", got.Event, "UserPromptSubmit")
	}

	// Stale file should be cleaned up
	if _, err := os.Stat(filepath.Join(evDir, "99999.json")); !os.IsNotExist(err) {
		t.Error("stale event file not cleaned up")
	}
}

func TestEventStatus(t *testing.T) {
	tests := []struct {
		event string
		want  Status
	}{
		{"UserPromptSubmit", StatusWorking},
		{"SubagentStart", StatusWorking},
		{"PostToolUseFailure", StatusWorking},
		{"Stop", StatusIdle},
		{"SessionStart", StatusIdle},
		{"SessionEnd", StatusFinished},
		{"PermissionRequest", StatusUnknown},
		{"SomethingElse", StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			got := EventStatus(tt.event)
			if got != tt.want {
				t.Errorf("EventStatus(%q) = %q, want %q", tt.event, got, tt.want)
			}
		})
	}
}
