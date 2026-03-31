package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/allir/c5s/internal/config"
)

// HookEvent represents an event written by the c5s hook script.
type HookEvent struct {
	Event          string    `json:"event"`
	SessionID      string    `json:"session_id"`
	Cwd            string    `json:"cwd"`
	TranscriptPath string    `json:"transcript_path"`
	Timestamp      time.Time `json:"timestamp"`
}

// hookScriptPath returns the path to the c5s status hook script.
func hookScriptPath() string {
	return filepath.Join(config.HooksDir(), "status-hook.sh")
}

// approvalHookScriptPath returns the path to the c5s approval hook script.
func approvalHookScriptPath() string {
	return filepath.Join(config.HooksDir(), "approval-hook.sh")
}

// hookScript is the shell script that writes event files.
// It uses grep/sed to parse stdin JSON (no jq dependency).
const hookScript = `#!/bin/sh
# c5s status hook — writes session event data for PID-based discovery
EVENTS_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/c5s/events"
mkdir -p "$EVENTS_DIR"

INPUT=$(cat)

# Extract fields from stdin JSON (no jq dependency)
EVENT=$(printf '%s' "$INPUT" | grep -o '"hook_event_name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"hook_event_name"[[:space:]]*:[[:space:]]*"//;s/"$//')
SESSION_ID=$(printf '%s' "$INPUT" | grep -o '"session_id"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"session_id"[[:space:]]*:[[:space:]]*"//;s/"$//')
CWD=$(printf '%s' "$INPUT" | grep -o '"cwd"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"cwd"[[:space:]]*:[[:space:]]*"//;s/"$//')
TRANSCRIPT=$(printf '%s' "$INPUT" | grep -o '"transcript_path"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"transcript_path"[[:space:]]*:[[:space:]]*"//;s/"$//')

cat > "$EVENTS_DIR/$PPID.json" <<EOF
{"event":"$EVENT","session_id":"$SESSION_ID","cwd":"$CWD","transcript_path":"$TRANSCRIPT","timestamp":"$(date -u +%Y-%m-%dT%H:%M:%SZ)"}
EOF
`

// approvalHookScript is the shell script that handles synchronous PermissionRequest hooks.
// It writes the hook input as a pending request and polls for a decision from the TUI.
// Uses $$ (hook script PID) as the unique request ID since multiple hooks from the same
// Claude Code process ($PPID) can be in-flight simultaneously.
const approvalHookScript = `#!/bin/sh
# c5s approval hook — blocks until the TUI approves/denies or timeout
PENDING_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/c5s/pending"
DECISIONS_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/c5s/decisions"
mkdir -p "$PENDING_DIR" "$DECISIONS_DIR"

# Write full hook input as pending request, keyed by hook PID ($$)
# Include parent PID so the TUI can associate with the session
INPUT=$(cat)
printf '%s' "$INPUT" | sed "s/}$/,\"ppid\":$PPID}/" > "$PENDING_DIR/$$.json"

# Poll for decision (up to 5 minutes, check every 0.5s)
# Also exit if pending file is removed (approval resolved elsewhere)
ELAPSED=0
while [ $ELAPSED -lt 600 ]; do
    if [ -f "$DECISIONS_DIR/$$.json" ]; then
        cat "$DECISIONS_DIR/$$.json"
        rm -f "$DECISIONS_DIR/$$.json" "$PENDING_DIR/$$.json"
        exit 0
    fi
    if [ ! -f "$PENDING_DIR/$$.json" ]; then
        exit 1
    fi
    sleep 0.5
    ELAPSED=$((ELAPSED + 1))
done

# Timeout — clean up, exit 1 to fall through to normal prompt
rm -f "$PENDING_DIR/$$.json"
exit 1
`

// hookEvents lists the events to register and their matchers.
var hookEvents = []struct {
	event   string
	matcher string
}{
	{"SessionStart", ""},
	{"SessionEnd", ""},
	{"Stop", ""},
	{"UserPromptSubmit", ""},
	{"SubagentStart", ""},
	{"PostToolUseFailure", "Bash"},
}

// InstallHooks writes the hook scripts and registers them in Claude Code settings.
func InstallHooks(settingsPath string) error {
	scriptPath := hookScriptPath()
	approvalPath := approvalHookScriptPath()

	// Create directories
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return fmt.Errorf("create hook dir: %w", err)
	}
	for _, dir := range []string{config.EventsDir(), config.PendingDir(), config.DecisionsDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create state dir: %w", err)
		}
	}

	// Write hook scripts
	if err := os.WriteFile(scriptPath, []byte(hookScript), 0o755); err != nil {
		return fmt.Errorf("write status hook script: %w", err)
	}
	if err := os.WriteFile(approvalPath, []byte(approvalHookScript), 0o755); err != nil {
		return fmt.Errorf("write approval hook script: %w", err)
	}

	// Read existing settings
	settings := make(map[string]any)
	data, err := os.ReadFile(settingsPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read settings: %w", err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse settings: %w", err)
		}
	}

	// Get or create hooks map
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	// Register status hook for each event (async, idempotent)
	for _, he := range hookEvents {
		eventHooks, _ := hooks[he.event].([]any)

		if containsC5sHook(eventHooks, scriptPath) {
			continue
		}

		entry := map[string]any{
			"matcher": he.matcher,
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": scriptPath,
					"async":   true,
					"timeout": 5,
				},
			},
		}
		eventHooks = append(eventHooks, entry)
		hooks[he.event] = eventHooks
	}

	// Register approval hook for PermissionRequest (synchronous, blocks until decision)
	permHooks, _ := hooks["PermissionRequest"].([]any)
	if !containsC5sHook(permHooks, approvalPath) {
		entry := map[string]any{
			"matcher": "",
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": approvalPath,
					"timeout": 300,
				},
			},
		}
		hooks["PermissionRequest"] = append(permHooks, entry)
	}

	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0o644)
}

// UninstallHooks removes c5s hooks from Claude Code settings and cleans up files.
func UninstallHooks(settingsPath string) error {
	scriptPath := hookScriptPath()
	approvalPath := approvalHookScriptPath()

	// Read settings
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read settings: %w", err)
	}

	settings := make(map[string]any)
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse settings: %w", err)
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks != nil {
		for event, entries := range hooks {
			eventHooks, ok := entries.([]any)
			if !ok {
				continue
			}
			// Remove entries matching either c5s script
			filtered := removeC5sHooks(eventHooks, scriptPath)
			filtered = removeC5sHooks(filtered, approvalPath)
			if len(filtered) == 0 {
				delete(hooks, event)
			} else {
				hooks[event] = filtered
			}
		}
		if len(hooks) == 0 {
			delete(settings, "hooks")
		}
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	// Clean up files
	_ = os.Remove(scriptPath)
	_ = os.Remove(approvalPath)
	_ = os.RemoveAll(config.EventsDir())
	_ = os.RemoveAll(config.PendingDir())
	_ = os.RemoveAll(config.DecisionsDir())

	return nil
}

// ReadHookEvents reads all event files and returns a map of PID → HookEvent.
// Stale files (>24h) are cleaned up automatically.
func ReadHookEvents() (map[int]HookEvent, error) {
	files, err := readPIDFiles(config.EventsDir(), 24*time.Hour)
	if err != nil {
		return nil, err
	}

	events := make(map[int]HookEvent)
	for _, f := range files {
		var ev HookEvent
		if err := json.Unmarshal(f.Data, &ev); err != nil {
			continue
		}
		events[f.PID] = ev
	}
	return events, nil
}

// EventStatus maps a hook event name to a session status.
// Returns StatusUnknown for events that don't map to a status (e.g., PermissionRequest).
func EventStatus(event string) Status {
	switch event {
	case "UserPromptSubmit", "SubagentStart", "PostToolUseFailure":
		return StatusWorking
	case "Stop", "SessionStart":
		return StatusIdle
	case "SessionEnd":
		return StatusFinished
	default:
		// PermissionRequest intentionally excluded — it fires for auto-approved
		// tools too. "Waiting" is detected via JSONL content (hasPendingToolUse).
		return StatusUnknown
	}
}

// containsC5sHook checks if any hook entry references the c5s script path.
func containsC5sHook(entries []any, scriptPath string) bool {
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		hooks, _ := m["hooks"].([]any)
		for _, h := range hooks {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, _ := hm["command"].(string); cmd == scriptPath {
				return true
			}
		}
	}
	return false
}

// removeC5sHooks filters out hook entries that reference the c5s script path.
func removeC5sHooks(entries []any, scriptPath string) []any {
	var filtered []any
	for _, entry := range entries {
		hasOurs := containsC5sHook([]any{entry}, scriptPath)
		if !hasOurs {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
