package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// tailTranscriptSize — bytes to read from the tail of a JSONL
// transcript. Must clear past base64 tool results.
const tailTranscriptSize = 2 * 1024 * 1024 // 2MB

// Role represents the type of a transcript entry.
type Role string

const (
	RoleUser       Role = "user"
	RoleAssistant  Role = "assistant"
	RoleToolUse    Role = "tool_use"
	RoleToolResult Role = "tool_result"
	RoleDiff       Role = "diff"
)

// ToolOutcome represents the result status of a tool use.
type ToolOutcome string

const (
	ToolPending ToolOutcome = "pending" // no result yet
	ToolSuccess ToolOutcome = "success" // result came back ok
	ToolError   ToolOutcome = "error"   // denied or failed
)

// TranscriptEntry represents a single renderable entry from a session transcript.
type TranscriptEntry struct {
	Role     Role        // entry type
	Content  string      // rendered text
	ToolID   string      // tool_use ID (for matching tool_use → tool_result)
	Outcome  ToolOutcome // tool use result status (only for tool_use/tool_result entries)
	FilePath string      // source file path (for diff syntax highlighting)
}

// ReadTranscript reads the tail of a session JSONL file and returns
// renderable transcript entries. It reads at most tailTranscriptSize bytes
// from the end of the file. If cwd is non-empty, file paths are made relative.
func ReadTranscript(path, cwd string) ([]TranscriptEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	scanner, err := tailScanner(f, info.Size(), tailTranscriptSize)
	if err != nil {
		return nil, err
	}

	var entries []TranscriptEntry

	for scanner.Scan() {
		var line jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}

		switch line.Type {
		case "user":
			parseUserMessage(line.Message.Content, &entries)
		case "assistant":
			parseAssistantMessage(line.Message.Content, &entries)
		}
	}

	// Match tool_results back to their tool_use entries to set outcomes
	resolveToolOutcomes(entries)

	// Relativize file paths in tool entries
	if cwd != "" {
		cwdPrefix := cwd + "/"
		for i := range entries {
			if entries[i].Role == RoleToolUse || entries[i].Role == RoleDiff {
				entries[i].Content = strings.ReplaceAll(entries[i].Content, cwdPrefix, "")
			}
		}
	}

	return entries, nil
}

// parseUserMessage extracts text and tool_result entries from a user message.
func parseUserMessage(content any, entries *[]TranscriptEntry) {
	switch c := content.(type) {
	case string:
		if text := strings.TrimSpace(c); text != "" {
			*entries = append(*entries, TranscriptEntry{Role: RoleUser, Content: text})
		}
	case []any:
		for _, block := range c {
			m, ok := block.(map[string]any)
			if !ok {
				continue
			}
			switch m["type"] {
			case "text":
				if text, ok := m["text"].(string); ok {
					text = strings.TrimSpace(text)
					if text != "" {
						*entries = append(*entries, TranscriptEntry{Role: RoleUser, Content: text})
					}
				}
			case "tool_result":
				toolUseID, _ := m["tool_use_id"].(string)
				isError, _ := m["is_error"].(bool)
				outcome := ToolSuccess
				if isError {
					outcome = ToolError
				}
				*entries = append(*entries, TranscriptEntry{
					Role:    RoleToolResult,
					Content: summarizeToolResult(m),
					ToolID:  toolUseID,
					Outcome: outcome,
				})
			}
		}
	}
}

// parseAssistantMessage extracts text and tool_use entries from an assistant message.
func parseAssistantMessage(content any, entries *[]TranscriptEntry) {
	blocks, ok := content.([]any)
	if !ok {
		return
	}

	for _, block := range blocks {
		m, ok := block.(map[string]any)
		if !ok {
			continue
		}
		switch m["type"] {
		case "text":
			if text, ok := m["text"].(string); ok {
				text = strings.TrimSpace(text)
				if text != "" {
					*entries = append(*entries, TranscriptEntry{Role: RoleAssistant, Content: text})
				}
			}
		case "tool_use":
			name, _ := m["name"].(string)
			id, _ := m["id"].(string)
			input, _ := m["input"].(map[string]any)
			summary := SummarizeToolInput(name, input)

			// Match Claude Code's display: "Update" for edits, "Write" for creates
			displayName := name
			if name == "Edit" {
				if oldStr, _ := input["old_string"].(string); oldStr != "" {
					displayName = "Update"
				}
			}

			*entries = append(*entries, TranscriptEntry{
				Role:    RoleToolUse,
				Content: fmt.Sprintf("%s(%s)", displayName, summary),
				ToolID:  id,
				Outcome: ToolPending,
			})

			// For Edit calls, add diff preview lines
			if name == "Edit" {
				fp, _ := input["file_path"].(string)
				if diffLines := formatEditDiff(input); len(diffLines) > 0 {
					for _, dl := range diffLines {
						*entries = append(*entries, TranscriptEntry{
							Role:     RoleDiff,
							Content:  dl,
							FilePath: fp,
						})
					}
				}
			}
		}
	}
}

// contextLines is how many unchanged lines to show before/after a diff.
const contextLines = 2

// formatEditDiff computes a line-by-line diff between old_string and new_string
// with line numbers and context. Reads the file to determine the starting line
// number and surrounding context. Uses LCS-based diff for accurate results.
func formatEditDiff(input map[string]any) []string {
	oldStr, _ := input["old_string"].(string)
	newStr, _ := input["new_string"].(string)
	filePath, _ := input["file_path"].(string)
	if oldStr == "" && newStr == "" {
		return nil
	}

	oldLines := strings.Split(strings.TrimRight(oldStr, "\n"), "\n")
	newLines := strings.Split(strings.TrimRight(newStr, "\n"), "\n")

	// Find start line and file context
	startLine := 0
	var beforeCtx, afterCtx []string
	if filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			fileContent := string(data)
			matchLines := oldLines
			idx := strings.Index(fileContent, oldStr)
			if idx == -1 {
				idx = strings.Index(fileContent, newStr)
				matchLines = newLines
			}
			if idx != -1 {
				startLine = strings.Count(fileContent[:idx], "\n") + 1
				fileLines := strings.Split(fileContent, "\n")

				ctxStart := max(startLine-1-contextLines, 0)
				for i := ctxStart; i < startLine-1; i++ {
					beforeCtx = append(beforeCtx, fileLines[i])
				}

				afterStart := startLine - 1 + len(matchLines)
				for i := afterStart; i < min(afterStart+contextLines, len(fileLines)); i++ {
					afterCtx = append(afterCtx, fileLines[i])
				}
			}
		}
	}

	// Compute LCS-based diff operations
	ops := diffLines(oldLines, newLines)

	var lines []string
	maxLines := 20
	lineNum := startLine

	// File context before
	if startLine > 0 {
		ctxLineNum := startLine - len(beforeCtx)
		for _, l := range beforeCtx {
			lines = append(lines, fmt.Sprintf("  %3d  %s", ctxLineNum, l))
			ctxLineNum++
		}
	}

	// Diff lines
	newLineNum := lineNum
	for _, op := range ops {
		if len(lines) >= maxLines {
			lines = append(lines, "  ...")
			break
		}
		switch op.kind {
		case diffEqual:
			if startLine > 0 {
				lines = append(lines, fmt.Sprintf("  %3d  %s", lineNum, op.text))
			} else {
				lines = append(lines, "  "+op.text)
			}
			lineNum++
			newLineNum++
		case diffDelete:
			if startLine > 0 {
				lines = append(lines, fmt.Sprintf("- %3d  %s", lineNum, op.text))
			} else {
				lines = append(lines, "- "+op.text)
			}
			lineNum++
		case diffInsert:
			if startLine > 0 {
				lines = append(lines, fmt.Sprintf("+ %3d  %s", newLineNum, op.text))
			} else {
				lines = append(lines, "+ "+op.text)
			}
			newLineNum++
		}
	}

	// File context after
	if startLine > 0 {
		for _, l := range afterCtx {
			if len(lines) >= maxLines {
				break
			}
			lines = append(lines, fmt.Sprintf("  %3d  %s", newLineNum, l))
			newLineNum++
		}
	}

	return lines
}

// diffKind represents a diff operation type.
type diffKind int

const (
	diffEqual diffKind = iota
	diffDelete
	diffInsert
)

// diffOp represents a single line in a diff.
type diffOp struct {
	kind diffKind
	text string
}

// maxDiffInputLines caps the LCS diff input to prevent quadratic memory usage.
const maxDiffInputLines = 100

// diffLines computes a line-level diff using the LCS (longest common subsequence)
// algorithm. Returns a sequence of equal/delete/insert operations.
// Falls back to simple remove-all/add-all if input exceeds maxDiffInputLines.
func diffLines(old, new []string) []diffOp {
	// Fall back to simple diff if input is too large for LCS
	if len(old) > maxDiffInputLines || len(new) > maxDiffInputLines {
		var ops []diffOp
		for _, l := range old {
			ops = append(ops, diffOp{diffDelete, l})
		}
		for _, l := range new {
			ops = append(ops, diffOp{diffInsert, l})
		}
		return ops
	}

	// Build LCS table
	m, n := len(old), len(new)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if old[i-1] == new[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to produce diff ops
	var ops []diffOp
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && old[i-1] == new[j-1] {
			ops = append(ops, diffOp{diffEqual, old[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, diffOp{diffInsert, new[j-1]})
			j--
		} else {
			ops = append(ops, diffOp{diffDelete, old[i-1]})
			i--
		}
	}

	// Reverse (backtracking produces ops in reverse order)
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}

	return ops
}

// resolveToolOutcomes matches tool_result entries back to their tool_use
// entries by ToolID, updating the tool_use outcome from pending to success/error.
func resolveToolOutcomes(entries []TranscriptEntry) {
	// Build a map of tool_use_id → result outcome
	outcomes := make(map[string]ToolOutcome)
	for _, e := range entries {
		if e.Role == RoleToolResult && e.ToolID != "" {
			outcomes[e.ToolID] = e.Outcome
		}
	}

	// Update tool_use entries with their outcomes
	for i := range entries {
		if entries[i].Role == RoleToolUse && entries[i].ToolID != "" {
			if outcome, ok := outcomes[entries[i].ToolID]; ok {
				entries[i].Outcome = outcome
			}
		}
	}
}

// summarizeToolResult returns a one-line summary of a tool_result block.
func summarizeToolResult(m map[string]any) string {
	isError, _ := m["is_error"].(bool)
	if isError {
		return "✗ error"
	}

	// Try to extract text content from the result
	if content, ok := m["content"].([]any); ok {
		for _, block := range content {
			if bm, ok := block.(map[string]any); ok {
				if bm["type"] == "text" {
					if text, ok := bm["text"].(string); ok {
						text = strings.TrimSpace(text)
						// Collapse to first line
						if idx := strings.IndexByte(text, '\n'); idx != -1 {
							text = text[:idx] + "…"
						}
						return Truncate(text, 100)
					}
				}
			}
		}
	}

	// Simple string content
	if content, ok := m["content"].(string); ok {
		content = strings.TrimSpace(content)
		if idx := strings.IndexByte(content, '\n'); idx != -1 {
			content = content[:idx] + "…"
		}
		return Truncate(content, 100)
	}

	return "✓ ok"
}
