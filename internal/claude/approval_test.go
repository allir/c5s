package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestReadPendingApprovals(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	pendingDir := filepath.Join(tmpDir, "c5s", "pending")
	decisionsDir := filepath.Join(tmpDir, "c5s", "decisions")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(decisionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Use current process PID as the hook PID so liveness check passes.
	hookPID := os.Getpid()
	sessionPID := 12345

	input := hookApprovalInput{
		SessionID: "sess-123",
		Cwd:       "/home/user/project",
		ToolName:  "Bash",
		ToolInput: map[string]any{"command": "npm test"},
		PPID:      sessionPID,
	}
	data, _ := json.Marshal(input)
	writePendingFile(t, pendingDir, hookPID, data)

	// Write a stale approval (should be cleaned up)
	writePendingFile(t, pendingDir, 99999, data)
	staleTime := time.Now().Add(-11 * time.Minute)
	_ = os.Chtimes(filepath.Join(pendingDir, "99999.json"), staleTime, staleTime)

	// Write a pending approval for a dead hook process (should be cleaned up)
	writePendingFile(t, pendingDir, 99998, data)

	// Write a pending approval that already has a decision (should be skipped)
	// Use current PID + 1 — it may or may not be alive, but the decision check runs first
	writePendingFile(t, pendingDir, hookPID+1, data)
	_ = os.WriteFile(filepath.Join(decisionsDir, strconv.Itoa(hookPID+1)+".json"), []byte("{}"), 0o644)

	// Write a non-JSON file (should be ignored)
	_ = os.WriteFile(filepath.Join(pendingDir, "notes.txt"), []byte("nope"), 0o644)

	approvals, err := ReadPendingApprovals(nil)
	if err != nil {
		t.Fatalf("ReadPendingApprovals: %v", err)
	}

	// Should have exactly one approval, keyed by session PID (ppid)
	if len(approvals) != 1 {
		t.Fatalf("got %d approvals, want 1", len(approvals))
	}

	got, ok := approvals[sessionPID]
	if !ok {
		t.Fatalf("approval for session PID %d not found", sessionPID)
	}
	if got.HookPID != hookPID {
		t.Errorf("HookPID = %d, want %d", got.HookPID, hookPID)
	}
	if got.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want %q", got.ToolName, "Bash")
	}
	if got.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-123")
	}
	if cmd, _ := got.ToolInput["command"].(string); cmd != "npm test" {
		t.Errorf("ToolInput.command = %q, want %q", cmd, "npm test")
	}

	// Stale file should be cleaned up
	if _, err := os.Stat(filepath.Join(pendingDir, "99999.json")); !os.IsNotExist(err) {
		t.Error("stale approval file not cleaned up")
	}

	// Dead hook process file should be cleaned up
	if _, err := os.Stat(filepath.Join(pendingDir, "99998.json")); !os.IsNotExist(err) {
		t.Error("dead process approval file not cleaned up")
	}
}

func TestReadPendingApprovals_NoDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	approvals, err := ReadPendingApprovals(nil)
	if err != nil {
		t.Fatalf("ReadPendingApprovals: %v", err)
	}
	if approvals != nil {
		t.Errorf("expected nil, got %v", approvals)
	}
}

func TestWriteApprovalDecision(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	tests := []struct {
		name    string
		option  ApprovalOption
		wantBeh string
	}{
		{"allow", ApprovalOption{Label: "Yes", Allow: true}, "allow"},
		{"deny", ApprovalOption{Label: "No", Allow: false}, "deny"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := WriteApprovalDecision(42, tt.option); err != nil {
				t.Fatalf("WriteApprovalDecision: %v", err)
			}

			path := filepath.Join(tmpDir, "c5s", "decisions", "42.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read decision file: %v", err)
			}

			var result map[string]any
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("parse decision: %v", err)
			}

			hso, _ := result["hookSpecificOutput"].(map[string]any)
			if hso == nil {
				t.Fatal("missing hookSpecificOutput")
			}
			if hso["hookEventName"] != "PermissionRequest" {
				t.Errorf("hookEventName = %v, want PermissionRequest", hso["hookEventName"])
			}
			decision, _ := hso["decision"].(map[string]any)
			if decision["behavior"] != tt.wantBeh {
				t.Errorf("behavior = %v, want %v", decision["behavior"], tt.wantBeh)
			}

			_ = os.Remove(path)
		})
	}
}

func TestSummarizeToolInput(t *testing.T) {
	tests := []struct {
		tool  string
		input map[string]any
		want  string
	}{
		{"Bash", map[string]any{"command": "npm test"}, "npm test"},
		{"Edit", map[string]any{"file_path": "/foo/bar.go"}, "/foo/bar.go"},
		{"Read", map[string]any{"file_path": "/foo/bar.go"}, "/foo/bar.go"},
		{"Write", map[string]any{"file_path": "/foo/bar.go"}, "/foo/bar.go"},
		{"Glob", map[string]any{"pattern": "**/*.ts"}, "**/*.ts"},
		{"Grep", map[string]any{"pattern": "TODO"}, "TODO"},
		{"Agent", map[string]any{"description": "find all tests"}, "find all tests"},
		{"UnknownTool", map[string]any{}, "UnknownTool"},
		{"Bash", map[string]any{}, "Bash"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			got := SummarizeToolInput(tt.tool, tt.input)
			if got != tt.want {
				t.Errorf("SummarizeToolInput(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func writePendingFile(t *testing.T, dir string, pid int, data []byte) {
	t.Helper()
	path := filepath.Join(dir, strconv.Itoa(pid)+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write pending file: %v", err)
	}
}
