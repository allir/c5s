package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadTranscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	lines := []string{
		`{"type":"user","message":{"content":"hello world"}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Sure, let me help."},{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"echo hi"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu_1","content":"hi\n"}]}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	entries, err := ReadTranscript(path, "")
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(entries))
	}

	// Entry 0: user text
	if entries[0].Role != RoleUser {
		t.Errorf("entry 0 role = %q, want %q", entries[0].Role, RoleUser)
	}
	if entries[0].Content != "hello world" {
		t.Errorf("entry 0 content = %q, want %q", entries[0].Content, "hello world")
	}

	// Entry 1: assistant text
	if entries[1].Role != RoleAssistant {
		t.Errorf("entry 1 role = %q, want %q", entries[1].Role, RoleAssistant)
	}
	if entries[1].Content != "Sure, let me help." {
		t.Errorf("entry 1 content = %q, want %q", entries[1].Content, "Sure, let me help.")
	}

	// Entry 2: tool_use
	if entries[2].Role != RoleToolUse {
		t.Errorf("entry 2 role = %q, want %q", entries[2].Role, RoleToolUse)
	}
	if !strings.Contains(entries[2].Content, "Bash") {
		t.Errorf("entry 2 content = %q, want it to contain %q", entries[2].Content, "Bash")
	}
	if entries[2].ToolID != "tu_1" {
		t.Errorf("entry 2 ToolID = %q, want %q", entries[2].ToolID, "tu_1")
	}
	if entries[2].Outcome != ToolSuccess {
		t.Errorf("entry 2 Outcome = %q, want %q (resolved from result)", entries[2].Outcome, ToolSuccess)
	}

	// Entry 3: tool_result
	if entries[3].Role != RoleToolResult {
		t.Errorf("entry 3 role = %q, want %q", entries[3].Role, RoleToolResult)
	}
	if entries[3].ToolID != "tu_1" {
		t.Errorf("entry 3 ToolID = %q, want %q", entries[3].ToolID, "tu_1")
	}
	if entries[3].Outcome != ToolSuccess {
		t.Errorf("entry 3 Outcome = %q, want %q", entries[3].Outcome, ToolSuccess)
	}
}

func TestReadTranscript_CwdRelativization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.jsonl")

	lines := []string{
		`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_2","name":"Read","input":{"file_path":"/home/user/project/main.go"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tu_2","content":"package main"}]}}`,
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	entries, err := ReadTranscript(path, "/home/user/project")
	if err != nil {
		t.Fatalf("ReadTranscript: %v", err)
	}

	// The tool_use entry should have the cwd prefix stripped
	var toolUse *TranscriptEntry
	for i := range entries {
		if entries[i].Role == RoleToolUse {
			toolUse = &entries[i]
			break
		}
	}
	if toolUse == nil {
		t.Fatal("no tool_use entry found")
	}
	if strings.Contains(toolUse.Content, "/home/user/project/") {
		t.Errorf("tool_use content still contains cwd prefix: %q", toolUse.Content)
	}
	if !strings.Contains(toolUse.Content, "main.go") {
		t.Errorf("tool_use content missing file name: %q", toolUse.Content)
	}
}

func TestParseUserMessage(t *testing.T) {
	tests := []struct {
		name        string
		content     any
		wantCount   int
		wantRole    Role
		wantContent string
		wantOutcome ToolOutcome
	}{
		{
			name:      "string content",
			content:   "hello",
			wantCount: 1,
			wantRole:  RoleUser,
		},
		{
			name: "array with text",
			content: []any{
				map[string]any{"type": "text", "text": "hi"},
			},
			wantCount: 1,
			wantRole:  RoleUser,
		},
		{
			name: "array with tool_result success",
			content: []any{
				map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "ok"},
			},
			wantCount:   1,
			wantRole:    RoleToolResult,
			wantOutcome: ToolSuccess,
		},
		{
			name: "array with tool_result error",
			content: []any{
				map[string]any{"type": "tool_result", "tool_use_id": "t1", "is_error": true, "content": "fail"},
			},
			wantCount:   1,
			wantRole:    RoleToolResult,
			wantOutcome: ToolError,
		},
		{
			name:      "empty string",
			content:   "",
			wantCount: 0,
		},
		{
			name:        "local command /clear",
			content:     "<local-command-caveat>Caveat: blah</local-command-caveat>\n<command-name>/clear</command-name>\n            <command-message>clear</command-message>\n            <command-args></command-args>",
			wantCount:   1,
			wantRole:    RoleUser,
			wantContent: "/clear",
		},
		{
			name:      "local command without command-name",
			content:   "<local-command-caveat>Caveat: blah</local-command-caveat>",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entries []TranscriptEntry
			parseUserMessage(tt.content, &entries)

			if len(entries) != tt.wantCount {
				t.Fatalf("got %d entries, want %d", len(entries), tt.wantCount)
			}
			if tt.wantCount == 0 {
				return
			}
			if entries[0].Role != tt.wantRole {
				t.Errorf("role = %q, want %q", entries[0].Role, tt.wantRole)
			}
			if tt.wantContent != "" && entries[0].Content != tt.wantContent {
				t.Errorf("content = %q, want %q", entries[0].Content, tt.wantContent)
			}
			if tt.wantOutcome != "" && entries[0].Outcome != tt.wantOutcome {
				t.Errorf("outcome = %q, want %q", entries[0].Outcome, tt.wantOutcome)
			}
		})
	}
}

func TestDiffLines(t *testing.T) {
	tests := []struct {
		name    string
		old     []string
		new     []string
		wantOps []diffOp
	}{
		{
			name: "equal",
			old:  []string{"a", "b"},
			new:  []string{"a", "b"},
			wantOps: []diffOp{
				{diffEqual, "a"},
				{diffEqual, "b"},
			},
		},
		{
			name: "delete",
			old:  []string{"a", "b"},
			new:  []string{"a"},
			wantOps: []diffOp{
				{diffEqual, "a"},
				{diffDelete, "b"},
			},
		},
		{
			name: "insert",
			old:  []string{"a"},
			new:  []string{"a", "b"},
			wantOps: []diffOp{
				{diffEqual, "a"},
				{diffInsert, "b"},
			},
		},
		{
			name: "replace",
			old:  []string{"a"},
			new:  []string{"b"},
			wantOps: []diffOp{
				{diffDelete, "a"},
				{diffInsert, "b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := diffLines(tt.old, tt.new)
			if len(ops) != len(tt.wantOps) {
				t.Fatalf("got %d ops, want %d: %+v", len(ops), len(tt.wantOps), ops)
			}
			for i, op := range ops {
				if op.kind != tt.wantOps[i].kind || op.text != tt.wantOps[i].text {
					t.Errorf("op[%d] = {%d, %q}, want {%d, %q}", i, op.kind, op.text, tt.wantOps[i].kind, tt.wantOps[i].text)
				}
			}
		})
	}

	// Large input fallback
	t.Run("large input fallback", func(t *testing.T) {
		old := make([]string, 101)
		new := make([]string, 101)
		for i := range old {
			old[i] = "old"
			new[i] = "new"
		}
		ops := diffLines(old, new)
		wantLen := 202 // 101 deletes + 101 inserts
		if len(ops) != wantLen {
			t.Fatalf("got %d ops, want %d", len(ops), wantLen)
		}
		// First 101 should be deletes
		for i := range 101 {
			if ops[i].kind != diffDelete {
				t.Errorf("op[%d].kind = %d, want diffDelete (%d)", i, ops[i].kind, diffDelete)
				break
			}
		}
		// Next 101 should be inserts
		for i := 101; i < 202; i++ {
			if ops[i].kind != diffInsert {
				t.Errorf("op[%d].kind = %d, want diffInsert (%d)", i, ops[i].kind, diffInsert)
				break
			}
		}
	})
}

func TestResolveToolOutcomes(t *testing.T) {
	entries := []TranscriptEntry{
		{Role: RoleToolUse, ToolID: "t1", Outcome: ToolPending},
		{Role: RoleToolResult, ToolID: "t1", Outcome: ToolSuccess},
	}

	resolveToolOutcomes(entries)

	if entries[0].Outcome != ToolSuccess {
		t.Errorf("tool_use outcome = %q, want %q", entries[0].Outcome, ToolSuccess)
	}
}

func TestSummarizeToolResult(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "error result",
			input: `{"is_error": true}`,
			want:  "\u2717 error",
		},
		{
			name:  "array content with text",
			input: `{"content": [{"type":"text","text":"hello world"}]}`,
			want:  "hello world",
		},
		{
			name:  "string content",
			input: `{"content": "simple text"}`,
			want:  "simple text",
		},
		{
			name:  "no content",
			input: `{}`,
			want:  "\u2713 ok",
		},
		{
			name:  "multi-line string content",
			input: `{"content": "line1\nline2"}`,
			want:  "line1\u2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m map[string]any
			if err := json.Unmarshal([]byte(tt.input), &m); err != nil {
				t.Fatalf("failed to parse input JSON: %v", err)
			}
			got := summarizeToolResult(m)
			if got != tt.want {
				t.Errorf("summarizeToolResult() = %q, want %q", got, tt.want)
			}
		})
	}
}
