package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

// idleThreshold is how long since the last JSONL write before a session
// is considered idle rather than actively working (used when no hook data exists).
const idleThreshold = 2 * time.Minute

// ApprovalSettleTime is how long to wait before showing a pending tool_use as
// "input" (waiting for approval). Auto-approved tools execute and write their
// result well within this window, so only genuine approval prompts survive it.
const ApprovalSettleTime = 2 * time.Second

// maxScanWorkers limits the number of concurrent goroutines processing
// session files in Scan(), bounding file descriptor usage.
const maxScanWorkers = 10

// tailReadSize is the maximum bytes to read from the end of a JSONL file
// when checking for pending tool use.
const tailReadSize = 512 * 1024

// maxPeekLines is how many lines to read from the start of a JSONL file
// when extracting session metadata (title, model, branch).
const maxPeekLines = 10

// maxSummaryLen is the maximum character length for session summaries
// extracted from user messages. Generous limit — the view truncates to fit.
const maxSummaryLen = 200

// DefaultConfigDir returns the default Claude config directory,
// respecting CLAUDE_CONFIG_DIR if set.
func DefaultConfigDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".claude")
	}
	return filepath.Join(home, ".claude")
}

// pidFile represents the JSON structure in ~/.claude/sessions/{PID}.json.
type pidFile struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Cwd       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"`
}

// Scan discovers live Claude Code sessions by reading PID files,
// checking liveness, and enriching with JSONL metadata.
// When hook events are available, they provide authoritative PID→session mapping.
// Returns sessions and the hook events read during scanning (for reuse by callers).
func Scan(configDir string) ([]Session, map[int]HookEvent, error) {
	sessionsDir := filepath.Join(configDir, "sessions")
	projectsDir := filepath.Join(configDir, "projects")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil, nil, err
	}

	// Read hook events for authoritative PID→session mapping
	hookEvts, _ := ReadHookEvents()

	var (
		mu       sync.Mutex
		sessions []Session
	)

	var g errgroup.Group
	g.SetLimit(maxScanWorkers)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		g.Go(func() error {
			data, err := os.ReadFile(filepath.Join(sessionsDir, entry.Name()))
			if err != nil {
				return nil
			}

			// Trim null bytes — Claude Code's --continue can leave trailing nulls
			data = bytes.TrimRight(data, "\x00")
			// Trim trailing incomplete JSON (e.g., trailing comma after last field)
			data = bytes.TrimRight(data, " \t\n\r")
			if len(data) > 0 && data[len(data)-1] == ',' {
				data[len(data)-1] = '}'
			}

			var pf pidFile
			if err := json.Unmarshal(data, &pf); err != nil || pf.PID == 0 {
				return nil
			}

			if !isProcessAlive(pf.PID) {
				return nil
			}

			cwd := pf.Cwd
			project := filepath.Base(cwd)
			startedAt := time.UnixMilli(pf.StartedAt)

			session := Session{
				ID:        pf.SessionID,
				PID:       pf.PID,
				Project:   project,
				Cwd:       cwd,
				StartedAt: startedAt,
				Status:    StatusWorking,
			}

			var jsonlPath string
			var hookStatus Status

			// If a hook event exists for this PID, use it as the authoritative source
			if ev, ok := hookEvts[pf.PID]; ok {
				if ev.SessionID != "" {
					session.ID = ev.SessionID
				}
				if ev.Cwd != "" {
					session.Cwd = ev.Cwd
					session.Project = filepath.Base(ev.Cwd)
				}
				if ev.TranscriptPath != "" {
					jsonlPath = ev.TranscriptPath
				}
				hookStatus = EventStatus(ev.Event)
			}

			// If no transcript from hook, find it the traditional way
			if jsonlPath == "" {
				projectDir := filepath.Join(projectsDir, encodeCwd(cwd))
				candidate := filepath.Join(projectDir, session.ID+".jsonl")
				if _, statErr := os.Stat(candidate); statErr == nil {
					jsonlPath = candidate
				} else {
					// Fall back to latest JSONL (handles /clear without hooks)
					jsonlPath, _, _ = findLatestJSONL(projectDir)
				}
			}

			var tusState toolUseState
			if jsonlPath != "" {
				info, statErr := os.Stat(jsonlPath)
				if statErr == nil && info.Size() > 0 {
					session.ID = strings.TrimSuffix(filepath.Base(jsonlPath), ".jsonl")
					session.JSONLPath = jsonlPath
					session.LastModified = info.ModTime()
					tusState = readSessionFile(jsonlPath, info.Size(), &session)
				}
			}

			if session.LastModified.IsZero() {
				session.LastModified = startedAt
			}

			// Status priority:
			// 1. Hook-derived status (working/idle/finished) is authoritative
			// 2. JSONL staleness as fallback when no hook data
			// Note: "input" (waiting for approval) is set by the TUI layer
			// based on pending approval files, not JSONL analysis.
			if hookStatus != "" && hookStatus != StatusUnknown {
				session.Status = hookStatus
			} else if session.Status == StatusWorking && tusState == toolUseResolved {
				if time.Since(session.LastModified) > ApprovalSettleTime {
					session.Status = StatusIdle
				}
			}

			mu.Lock()
			sessions = append(sessions, session)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	slices.SortFunc(sessions, func(a, b Session) int {
		if a.Project != b.Project {
			if a.Project < b.Project {
				return -1
			}
			return 1
		}
		return a.PID - b.PID
	})

	return sessions, hookEvts, nil
}

// encodeCwd converts a filesystem path to the directory name format
// used by Claude Code in ~/.claude/projects/.
// Every non-alphanumeric rune is replaced with '-'.
func encodeCwd(path string) string {
	var b strings.Builder
	b.Grow(len(path))
	for _, r := range path {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// isProcessAlive checks whether a process with the given PID is still running.
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 doesn't send anything but checks if the process exists
	return proc.Signal(syscall.Signal(0)) == nil
}

// findLatestJSONL returns the most recently modified .jsonl file in dir.
func findLatestJSONL(dir string) (string, os.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}

	var (
		bestPath string
		bestInfo os.FileInfo
	)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if bestInfo == nil || info.ModTime().After(bestInfo.ModTime()) {
			bestPath = filepath.Join(dir, e.Name())
			bestInfo = info
		}
	}

	if bestInfo == nil {
		return "", nil, os.ErrNotExist
	}
	return bestPath, bestInfo, nil
}

// toolUseState describes the state of tool use at the tail of a JSONL file.
type toolUseState int

const (
	toolUseNone     toolUseState = iota // no recent tool interaction
	toolUsePending                      // assistant has tool_use, no tool_result yet
	toolUseResolved                     // tool_use was followed by tool_result (approved or rejected)
)

// contentHasToolUse checks if an Anthropic content field contains a tool_use block.
func contentHasToolUse(content any) bool {
	return contentHasBlockType(content, "tool_use")
}

// contentHasToolResult checks if an Anthropic content field contains a tool_result block.
func contentHasToolResult(content any) bool {
	return contentHasBlockType(content, "tool_result")
}

// contentHasBlockType checks if an Anthropic content field contains a block of the given type.
func contentHasBlockType(content any, blockType string) bool {
	blocks, ok := content.([]any)
	if !ok {
		return false
	}
	for _, block := range blocks {
		if m, ok := block.(map[string]any); ok {
			if m["type"] == blockType {
				return true
			}
		}
	}
	return false
}

// newJSONLScanner creates a bufio.Scanner with a buffer large enough for
// JSONL lines that may contain base64 artifacts or large tool results.
// Max line size is 4MB to handle screenshot/file content tool results.
func newJSONLScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 256*1024), 4*1024*1024)
	return s
}

// tailScanner seeks to the last maxBytes of a file (given its size) and returns
// a scanner ready to iterate JSONL lines. The first partial line after seeking
// is discarded. If the file is smaller than maxBytes, it rewinds to the start.
func tailScanner(f *os.File, fileSize, maxBytes int64) (*bufio.Scanner, error) {
	offset := max(fileSize-maxBytes, 0)
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := newJSONLScanner(f)

	// Discard first partial line if we seeked mid-file
	if offset > 0 {
		scanner.Scan()
	}

	return scanner, nil
}

// jsonlLine represents the minimal fields we care about from a JSONL entry.
type jsonlLine struct {
	Type        string `json:"type"`
	CustomTitle string `json:"customTitle"`
	GitBranch   string `json:"gitBranch"`
	Message     struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
		Model   string `json:"model"`
	} `json:"message"`
}

// readSessionFile opens a JSONL file once, peeks the head for metadata
// (title, summary, model, branch) and reads the tail for tool use state.
// Sets session status based on JSONL mtime heuristic.
func readSessionFile(path string, fileSize int64, session *Session) toolUseState {
	f, err := os.Open(path)
	if err != nil {
		return toolUseNone
	}
	defer func() { _ = f.Close() }()

	// — Head: extract metadata from first few lines —
	scanner := newJSONLScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount > maxPeekLines {
			break
		}

		var entry jsonlLine
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if entry.GitBranch != "" && session.GitBranch == "" {
			session.GitBranch = entry.GitBranch
		}

		switch entry.Type {
		case "custom-title":
			if entry.CustomTitle != "" {
				session.Summary = entry.CustomTitle
			}
		case "user":
			if session.Summary == "" {
				session.Summary = extractTextContent(entry.Message.Content)
			}
			if entry.Message.Model != "" && session.Model == "" {
				session.Model = entry.Message.Model
			}
		case "assistant":
			if entry.Message.Model != "" && session.Model == "" {
				session.Model = entry.Message.Model
			}
		}

		if session.GitBranch != "" && session.Summary != "" && session.Model != "" {
			break
		}
	}

	// Status based on JSONL age — all live sessions are working or idle
	if time.Since(session.LastModified) < idleThreshold {
		session.Status = StatusWorking
	} else {
		session.Status = StatusIdle
	}

	// — Tail: check tool use state —
	tScanner, err := tailScanner(f, fileSize, int64(tailReadSize))
	if err != nil {
		return toolUseNone
	}

	var lastType string
	var lastHasToolUse bool
	var sawToolUse bool
	var sawToolResult bool

	for tScanner.Scan() {
		var entry jsonlLine
		if err := json.Unmarshal(tScanner.Bytes(), &entry); err != nil {
			continue
		}
		switch entry.Type {
		case "assistant":
			lastType = "assistant"
			lastHasToolUse = contentHasToolUse(entry.Message.Content)
			if lastHasToolUse {
				sawToolUse = true
				sawToolResult = false
			}
		case "user":
			lastType = "user"
			if sawToolUse && contentHasToolResult(entry.Message.Content) {
				sawToolResult = true
			}
			lastHasToolUse = false
		}
	}

	if lastType == "assistant" && lastHasToolUse {
		return toolUsePending
	}
	if sawToolUse && sawToolResult {
		return toolUseResolved
	}
	return toolUseNone
}

// extractTextContent pulls text from an Anthropic-style content field,
// which can be a string or an array of content blocks.
func extractTextContent(content any) string {
	switch c := content.(type) {
	case string:
		return Truncate(c, maxSummaryLen)
	case []any:
		for _, block := range c {
			if m, ok := block.(map[string]any); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						return Truncate(text, maxSummaryLen)
					}
				}
			}
		}
	}
	return ""
}

func Truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	for strings.HasPrefix(s, "<") {
		if end := strings.Index(s, ">"); end != -1 {
			tagName := s[1:end]
			if strings.Contains(tagName, "/") || strings.Contains(tagName, " ") {
				break
			}
			closeTag := "</" + tagName + ">"
			if idx := strings.Index(s, closeTag); idx != -1 {
				s = strings.TrimSpace(s[idx+len(closeTag):])
				continue
			}
			s = strings.TrimSpace(s[end+1:])
		} else {
			break
		}
	}

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
