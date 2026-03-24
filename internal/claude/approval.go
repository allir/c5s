package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ApprovalOption represents one choice in the approval prompt.
type ApprovalOption struct {
	Label              string           // display label (e.g., "Yes", "Yes, allow all edits this session")
	Allow              bool             // true=allow, false=deny
	UpdatedPermissions []map[string]any // permission rules to persist (nil for simple allow/deny)
}

// PendingApproval represents a tool approval request from a Claude Code session.
type PendingApproval struct {
	HookPID   int              // hook script PID (unique per request, used for decision files)
	SessionID string           // Claude Code session ID
	ToolName  string           // tool requesting approval
	ToolInput map[string]any   // tool input parameters
	Cwd       string           // working directory
	Timestamp time.Time        // when the request was written
	Options   []ApprovalOption // available approval choices
}

// staleApprovalThreshold is how long before a pending approval file is cleaned up.
const staleApprovalThreshold = 10 * time.Minute

// PendingDir returns the path to the pending approvals directory.
func PendingDir() string {
	return filepath.Join(C5sStateDir(), "pending")
}

// DecisionsDir returns the path to the approval decisions directory.
func DecisionsDir() string {
	return filepath.Join(C5sStateDir(), "decisions")
}

// hookApprovalInput is the structure written by the approval hook script.
// It's the Claude Code hook stdin JSON with an appended "ppid" field.
type hookApprovalInput struct {
	SessionID             string           `json:"session_id"`
	Cwd                   string           `json:"cwd"`
	ToolName              string           `json:"tool_name"`
	ToolInput             any              `json:"tool_input"`
	PermissionSuggestions []map[string]any `json:"permission_suggestions"`
	PPID                  int              `json:"ppid"`
}

// ReadPendingApprovals reads all pending approval files and returns a map of
// session PID → latest pending approval. Files are keyed by hook script PID ($$),
// and the session PID (ppid) is read from the JSON content.
// hookEvents is used to detect approvals that were resolved elsewhere (e.g., in the
// terminal) — if a newer hook event exists for the session, the approval is stale.
// Stale files older than staleApprovalThreshold are cleaned up automatically.
func ReadPendingApprovals(hookEvents map[int]HookEvent) (map[int]PendingApproval, error) {
	files, err := readPIDFiles(PendingDir(), staleApprovalThreshold)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	// Read decision files once upfront instead of stat-per-pending-file
	decisionSet := make(map[string]struct{})
	if decEntries, err := os.ReadDir(DecisionsDir()); err == nil {
		for _, de := range decEntries {
			decisionSet[de.Name()] = struct{}{}
		}
	}

	approvals := make(map[int]PendingApproval)
	for _, f := range files {
		// Skip if we already wrote a decision for this hook
		if _, ok := decisionSet[filepath.Base(f.Path)]; ok {
			continue
		}

		// Clean up if the hook process is no longer alive (timed out or crashed)
		if !isProcessAlive(f.PID) {
			_ = os.Remove(f.Path)
			continue
		}

		var input hookApprovalInput
		if err := json.Unmarshal(f.Data, &input); err != nil {
			continue
		}

		if input.PPID == 0 {
			continue
		}

		// If a hook event for this session is newer than the pending file,
		// the approval was resolved elsewhere (e.g., user approved in terminal).
		// Clean up — the hook script will notice its pending file is gone and exit.
		if ev, ok := hookEvents[input.PPID]; ok {
			if ev.Timestamp.After(f.ModTime) {
				_ = os.Remove(f.Path)
				continue
			}
		}

		toolInput, _ := input.ToolInput.(map[string]any)

		approval := PendingApproval{
			HookPID:   f.PID,
			SessionID: input.SessionID,
			ToolName:  input.ToolName,
			ToolInput: toolInput,
			Cwd:       input.Cwd,
			Timestamp: f.ModTime,
			Options:   buildApprovalOptions(input.ToolName, input.PermissionSuggestions),
		}

		// Keep only the most recent approval per session PID
		if existing, ok := approvals[input.PPID]; ok {
			if approval.Timestamp.After(existing.Timestamp) {
				approvals[input.PPID] = approval
			}
		} else {
			approvals[input.PPID] = approval
		}
	}

	return approvals, nil
}

// WriteApprovalDecision writes a decision file that the hook script polls for.
// The hookPID identifies which specific hook invocation to respond to.
// The option determines the behavior (allow/deny) and any permission updates.
func WriteApprovalDecision(hookPID int, option ApprovalOption) error {
	dir := DecisionsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	behavior := "deny"
	if option.Allow {
		behavior = "allow"
	}

	dec := map[string]any{
		"behavior": behavior,
	}
	if option.Allow && len(option.UpdatedPermissions) > 0 {
		dec["updatedPermissions"] = option.UpdatedPermissions
	}

	decision := map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName": "PermissionRequest",
			"decision":      dec,
		},
	}

	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, strconv.Itoa(hookPID)+".json"), data, 0o644)
}

// buildApprovalOptions creates the list of approval choices from the tool name
// and permission suggestions. Always includes "Yes" (allow once) and "No" (deny).
// For tools that commonly need blanket approval (Edit, Write), adds a session-scoped
// "allow all" option matching Claude Code's behavior.
func buildApprovalOptions(toolName string, suggestions []map[string]any) []ApprovalOption {
	options := []ApprovalOption{
		{Label: "Yes", Allow: true},
	}

	// Add "allow all [tool] this session" for tools that benefit from it
	switch toolName {
	case "Edit", "Write":
		// Use setMode to acceptEdits — covers all edit/write operations
		options = append(options, ApprovalOption{
			Label: "Yes, allow all edits this session",
			Allow: true,
			UpdatedPermissions: []map[string]any{{
				"type":        "setMode",
				"mode":        "acceptEdits",
				"destination": "session",
			}},
		})
	case "Bash", "Glob", "Grep", "Read":
		// Use addRules for specific tool
		options = append(options, ApprovalOption{
			Label: "Yes, allow all " + toolName + " this session",
			Allow: true,
			UpdatedPermissions: []map[string]any{{
				"type": "addRules",
				"rules": []any{map[string]any{
					"toolName": toolName,
				}},
				"behavior":    "allow",
				"destination": "session",
			}},
		})
	}

	// Add specific suggestions from Claude Code (e.g., allow specific patterns)
	for _, s := range suggestions {
		dest, _ := s["destination"].(string)
		behavior, _ := s["behavior"].(string)
		if behavior != "allow" {
			continue
		}

		label := "Yes, always"
		switch dest {
		case "session":
			label = "Yes, for this session"
		case "localSettings":
			label = "Yes, always (local)"
		case "projectSettings":
			label = "Yes, always (project)"
		case "userSettings":
			label = "Yes, always (global)"
		}

		// Describe the rule
		if rules, ok := s["rules"].([]any); ok && len(rules) > 0 {
			if rule, ok := rules[0].(map[string]any); ok {
				if rc, ok := rule["ruleContent"].(string); ok && rc != "" {
					label += " (" + Truncate(rc, 40) + ")"
				}
			}
		}

		// Avoid duplicate labels
		duplicate := false
		for _, o := range options {
			if o.Label == label {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		options = append(options, ApprovalOption{
			Label:              label,
			Allow:              true,
			UpdatedPermissions: []map[string]any{s},
		})
	}

	options = append(options, ApprovalOption{Label: "No", Allow: false})
	return options
}

// SummarizeToolInput returns a human-readable one-liner for a tool use.
func SummarizeToolInput(toolName string, toolInput map[string]any) string {
	switch toolName {
	case "Bash":
		if cmd, ok := toolInput["command"].(string); ok {
			return Truncate(cmd, 80)
		}
	case "Edit", "Read", "Write":
		if fp, ok := toolInput["file_path"].(string); ok {
			return fp
		}
	case "Glob", "Grep":
		if p, ok := toolInput["pattern"].(string); ok {
			return p
		}
	case "Agent":
		if d, ok := toolInput["description"].(string); ok {
			return d
		}
	}
	return toolName
}
