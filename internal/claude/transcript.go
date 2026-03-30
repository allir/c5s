package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	udiff "github.com/aymanbagabas/go-udiff"
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

// reLocalCommand matches the <command-name>/foo</command-name> tag in
// local-command-caveat messages that Claude Code emits for slash commands.
var reLocalCommand = regexp.MustCompile(`<command-name>([^<]+)</command-name>`)

// parseUserMessage extracts text and tool_result entries from a user message.
func parseUserMessage(content any, entries *[]TranscriptEntry) {
	switch c := content.(type) {
	case string:
		text := strings.TrimSpace(c)
		if text == "" {
			break
		}
		// Local command messages (e.g. /clear) contain XML markup — extract just the command name.
		if strings.Contains(text, "<local-command-caveat>") {
			if m := reLocalCommand.FindStringSubmatch(text); m != nil {
				*entries = append(*entries, TranscriptEntry{Role: RoleUser, Content: m[1]})
			}
			break
		}
		*entries = append(*entries, TranscriptEntry{Role: RoleUser, Content: text})
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

// contextLines is how many unchanged lines to show around each diff hunk.
const contextLines = 3

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
	var beforeCtx []string
	var afterCtx []diffOp
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

				// Collect file lines after the edit range as extra equal ops
				// so the collapse logic can show trailing context naturally.
				afterStart := startLine - 1 + len(matchLines)
				for i := afterStart; i < min(afterStart+contextLines, len(fileLines)); i++ {
					afterCtx = append(afterCtx, diffOp{kind: diffEqual, text: fileLines[i]})
				}
			}
		}
	}

	// Compute LCS-based diff operations and append file context after
	ops := diffLines(oldLines, newLines)
	ops = append(ops, afterCtx...)

	var lines []string
	lineNum := startLine

	// File context before
	if startLine > 0 {
		ctxLineNum := startLine - len(beforeCtx)
		for _, l := range beforeCtx {
			lines = append(lines, fmt.Sprintf("%3d   %s", ctxLineNum, l))
			ctxLineNum++
		}
	}

	// Diff lines — format: "NNN + code" or "NNN - code" or "NNN   code"
	// Long runs of equal (unchanged) lines are collapsed to `...` with
	// contextLines of context around each changed hunk. Trailing context
	// after the last change is shown without a trailing `...`.

	// Pre-compute: for each op, distance to nearest change (add/delete).
	dist := make([]int, len(ops))
	d := len(ops) // large sentinel
	for i := len(ops) - 1; i >= 0; i-- {
		if ops[i].kind != diffEqual {
			d = 0
		}
		dist[i] = d
		d++
	}
	d = len(ops)
	for i := range ops {
		if ops[i].kind != diffEqual {
			d = 0
		}
		if d < dist[i] {
			dist[i] = d
		}
		d++
	}

	newLineNum := lineNum
	collapsed := false
	for i, op := range ops {
		switch op.kind {
		case diffEqual:
			if dist[i] > contextLines {
				// Check if there's any change ahead — if not, just stop
				hasChangeAhead := false
				for j := i + 1; j < len(ops); j++ {
					if ops[j].kind != diffEqual {
						hasChangeAhead = true
						break
					}
				}
				if !hasChangeAhead {
					// Past the last change's context — done
					goto done
				}
				if !collapsed {
					lines = append(lines, "  ...")
					collapsed = true
				}
				lineNum++
				newLineNum++
				continue
			}
			collapsed = false
			if startLine > 0 {
				lines = append(lines, fmt.Sprintf("%3d   %s", lineNum, op.text))
			} else {
				lines = append(lines, "    "+op.text)
			}
			lineNum++
			newLineNum++
		case diffDelete:
			if startLine > 0 {
				lines = append(lines, fmt.Sprintf("%3d - %s", lineNum, op.text))
			} else {
				lines = append(lines, "  - "+op.text)
			}
			lineNum++
		case diffInsert:
			if startLine > 0 {
				lines = append(lines, fmt.Sprintf("%3d + %s", newLineNum, op.text))
			} else {
				lines = append(lines, "  + "+op.text)
			}
			newLineNum++
		}
	}
done:

	return lines
}

// diffKind represents a diff operation type, aliased from go-udiff.
type diffKind = udiff.OpKind

const (
	diffEqual  = udiff.Equal
	diffDelete = udiff.Delete
	diffInsert = udiff.Insert
)

// diffOp represents a single line in a diff.
type diffOp struct {
	kind diffKind
	text string
}

// diffLines computes a line-level diff using Myers' algorithm (via go-udiff).
// Returns a sequence of equal/delete/insert operations.
func diffLines(old, new []string) []diffOp {
	oldStr := strings.Join(old, "\n") + "\n"
	newStr := strings.Join(new, "\n") + "\n"

	edits := udiff.Lines(oldStr, newStr)
	if len(edits) == 0 {
		// No changes — all lines are equal
		var ops []diffOp
		for _, l := range old {
			ops = append(ops, diffOp{diffEqual, l})
		}
		return ops
	}

	ud, err := udiff.ToUnifiedDiff("", "", oldStr, edits, len(old)+len(new))
	if err != nil || len(ud.Hunks) == 0 {
		// Fallback: delete all old, insert all new
		var ops []diffOp
		for _, l := range old {
			ops = append(ops, diffOp{diffDelete, l})
		}
		for _, l := range new {
			ops = append(ops, diffOp{diffInsert, l})
		}
		return ops
	}

	var ops []diffOp
	for _, h := range ud.Hunks {
		for _, l := range h.Lines {
			ops = append(ops, diffOp{l.Kind, strings.TrimRight(l.Content, "\n")})
		}
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
