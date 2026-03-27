package views

import (
	"fmt"
	"os"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/claude"
	"github.com/allir/c5s/internal/tui/theme"
)

// DetailModel displays the transcript of a single session.
type DetailModel struct {
	session        claude.Session
	entries        []claude.TranscriptEntry
	lines          []string // cached rendered lines, invalidated on refresh/resize
	scroll         int      // offset from bottom (0 = at bottom)
	width          int
	height         int
	lastMtime      int64                 // last known JSONL mtime (unix nano), for change detection
	approvalCursor int                   // selected option in approval prompt
	mdCache        *glamour.TermRenderer // cached markdown renderer
	mdWidth        int                   // width the renderer was created for
	inputMode      bool                  // true when text input is active
	inputText      string                // current text being typed
	inputCursor    int                   // cursor position in input text
	tmuxPane       string                // cached tmux pane ID for this session
}

// NewDetailModel creates a detail view for the given session.
func NewDetailModel(session claude.Session) DetailModel {
	pane := claude.FindTmuxPane(session.PID)
	m := DetailModel{
		session:   session,
		tmuxPane:  pane,
		inputMode: pane != "", // auto-enable input if tmux is available
	}
	m.loadTranscript()
	return m
}

// CanSendInput returns true if we can send input to this session (via tmux).
func (m *DetailModel) CanSendInput() bool {
	return m.tmuxPane != ""
}

// InputMode returns whether the text input is active.
func (m *DetailModel) InputMode() bool {
	return m.inputMode
}

// EnterInputMode activates the text input.
func (m *DetailModel) EnterInputMode() {
	if m.tmuxPane == "" {
		return
	}
	m.inputMode = true
	m.inputText = ""
	m.inputCursor = 0
}

// ExitInputMode deactivates the text input.
func (m *DetailModel) ExitInputMode() {
	m.inputMode = false
	m.inputText = ""
	m.inputCursor = 0
}

// InputInsert adds a character at the cursor position.
func (m *DetailModel) InputInsert(s string) {
	m.inputText = m.inputText[:m.inputCursor] + s + m.inputText[m.inputCursor:]
	m.inputCursor += len(s)
}

// InputBackspace deletes the character before the cursor.
func (m *DetailModel) InputBackspace() {
	if m.inputCursor > 0 {
		m.inputText = m.inputText[:m.inputCursor-1] + m.inputText[m.inputCursor:]
		m.inputCursor--
	}
}

// InputCursorLeft moves the input cursor left.
func (m *DetailModel) InputCursorLeft() {
	if m.inputCursor > 0 {
		m.inputCursor--
	}
}

// InputCursorRight moves the input cursor right.
func (m *DetailModel) InputCursorRight() {
	if m.inputCursor < len(m.inputText) {
		m.inputCursor++
	}
}

// SendInput sends the current input text to the session via tmux.
func (m *DetailModel) SendInput() error {
	if m.inputText == "" {
		return nil
	}
	err := claude.SendTmuxKeys(m.tmuxPane, m.inputText)
	// Stay in input mode, just clear the text
	m.inputText = ""
	m.inputCursor = 0
	return err
}

// InputLine renders the text input line.
func (m *DetailModel) InputLine(width int) string {
	if !m.inputMode {
		return ""
	}
	prompt := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯ ")

	// Render text with cursor
	before := m.inputText[:m.inputCursor]
	after := m.inputText[m.inputCursor:]
	cursor := lipgloss.NewStyle().Background(theme.ColorText).Foreground(theme.ColorBg).Render(" ")
	if len(after) > 0 {
		cursor = lipgloss.NewStyle().Background(theme.ColorText).Foreground(theme.ColorBg).Render(string(after[0]))
		after = after[1:]
	}

	text := lipgloss.NewStyle().Foreground(theme.ColorText).Render(before) +
		cursor +
		lipgloss.NewStyle().Foreground(theme.ColorText).Render(after)

	return lipgloss.NewStyle().Width(width).PaddingLeft(1).Render(prompt + text)
}

// SetSize updates the available dimensions and invalidates the line cache.
func (m *DetailModel) SetSize(width, height int) {
	if m.width != width {
		m.lines = nil // invalidate cache on width change
	}
	m.width = width
	m.height = height
}

// Refresh reloads the transcript if the file has changed.
func (m *DetailModel) Refresh() {
	m.loadTranscript()
}

// ScrollUp scrolls the view up.
func (m *DetailModel) ScrollUp() {
	m.scroll = min(m.scroll+3, m.maxScroll())
}

// ScrollDown scrolls the view toward the bottom.
func (m *DetailModel) ScrollDown() {
	m.scroll = max(m.scroll-3, 0)
}

// PageUp scrolls the view up by a full page.
func (m *DetailModel) PageUp() {
	m.scroll = min(m.scroll+max(m.height-2, 1), m.maxScroll())
}

// PageDown scrolls the view down by a full page.
func (m *DetailModel) PageDown() {
	m.scroll = max(m.scroll-max(m.height-2, 1), 0)
}

func (m *DetailModel) maxScroll() int {
	return max(len(m.getLines())-m.height, 0)
}

// Session returns the current session.
func (m *DetailModel) Session() claude.Session {
	return m.session
}

// ApprovalBlock returns a multi-line approval prompt for this session, or empty string.
// Styled to match Claude Code's permission request display with selectable options.
func (m *DetailModel) ApprovalBlock(width int) string {
	if m.session.PendingApproval == nil {
		return ""
	}
	a := m.session.PendingApproval
	summary := claude.SummarizeToolInput(a.ToolName, a.ToolInput)

	// Tool name header (e.g., "Bash command", "Edit file")
	toolHeader := a.ToolName
	switch a.ToolName {
	case "Bash":
		toolHeader = "Bash command"
	case "Edit":
		toolHeader = "Edit file"
	case "Write":
		toolHeader = "Write file"
	}
	toolLabel := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorWarning).Render(toolHeader)

	// Command/file details
	detail := lipgloss.NewStyle().Foreground(theme.ColorDimText).PaddingLeft(2).Render(summary)

	// Description if available (e.g., "Echo with subshell to trigger approval")
	var descLine string
	if desc, ok := a.ToolInput["description"].(string); ok && desc != "" {
		descLine = lipgloss.NewStyle().Foreground(theme.ColorMuted).PaddingLeft(2).Render(desc)
	}

	// Context about why approval is needed
	var contextLine string
	if a.ToolName == "Bash" {
		if cmd, ok := a.ToolInput["command"].(string); ok {
			if strings.Contains(cmd, "$(") || strings.Contains(cmd, "`") {
				contextLine = lipgloss.NewStyle().Foreground(theme.ColorText).Render(
					"Command contains $() command substitution",
				)
			}
		}
	}

	prompt := lipgloss.NewStyle().Foreground(theme.ColorText).Render("Do you want to proceed?")

	var optLines []string
	for i, opt := range a.Options {
		var line string
		if i == m.approvalCursor {
			cursor := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯")
			label := lipgloss.NewStyle().Foreground(theme.ColorText).Bold(true).Render(
				fmt.Sprintf(" %d. %s", i+1, opt.Label),
			)
			line = cursor + label
		} else {
			label := lipgloss.NewStyle().Foreground(theme.ColorDimText).Render(
				fmt.Sprintf("  %d. %s", i+1, opt.Label),
			)
			line = label
		}
		optLines = append(optLines, line)
	}

	// Build block
	lines := []string{toolLabel, "", detail}
	if descLine != "" {
		lines = append(lines, descLine)
	}
	if contextLine != "" {
		lines = append(lines, "", contextLine)
	}
	lines = append(lines, "", prompt)
	lines = append(lines, optLines...)

	return lipgloss.NewStyle().Width(width).PaddingLeft(1).Render(strings.Join(lines, "\n"))
}

// ApprovalCursorUp moves the approval selection up.
func (m *DetailModel) ApprovalCursorUp() {
	if m.approvalCursor > 0 {
		m.approvalCursor--
	}
}

// ApprovalCursorDown moves the approval selection down.
func (m *DetailModel) ApprovalCursorDown() {
	if m.session.PendingApproval != nil && m.approvalCursor < len(m.session.PendingApproval.Options)-1 {
		m.approvalCursor++
	}
}

// SelectedApprovalOption returns the currently selected approval option, or nil.
func (m *DetailModel) SelectedApprovalOption() *claude.ApprovalOption {
	if m.session.PendingApproval == nil {
		return nil
	}
	opts := m.session.PendingApproval.Options
	if m.approvalCursor >= len(opts) {
		return nil
	}
	return &opts[m.approvalCursor]
}

// UpdateSession updates the session metadata (e.g., status changes on refresh).
func (m *DetailModel) UpdateSession(session claude.Session) {
	// Reset approval cursor if the approval identity changed
	oldHookPID := 0
	if m.session.PendingApproval != nil {
		oldHookPID = m.session.PendingApproval.HookPID
	}
	newHookPID := 0
	if session.PendingApproval != nil {
		newHookPID = session.PendingApproval.HookPID
	}
	if oldHookPID != newHookPID {
		m.approvalCursor = 0
	}

	m.session = session
}

// View renders the detail view.
func (m *DetailModel) View() string {
	lines := m.getLines()

	if len(lines) == 0 {
		return lipgloss.NewStyle().
			Foreground(theme.ColorDimText).
			Padding(2, 0).
			Width(m.width).
			Align(lipgloss.Center).
			Render("No transcript data.")
	}

	// Show a window of lines from the bottom, offset by scroll
	end := max(len(lines)-m.scroll, 0)
	start := max(end-m.height, 0)

	return strings.Join(lines[start:end], "\n")
}

// getLines returns the cached rendered lines, rebuilding if needed.
func (m *DetailModel) getLines() []string {
	if m.lines == nil {
		m.lines = m.renderLines()
	}
	return m.lines
}

// HeaderInfo returns a multi-line session header, styled like Claude Code's banner.
func (m *DetailModel) HeaderInfo() string {
	s := m.session

	// Line 1: project name + branch
	project := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorText).Render(s.Project)
	line1 := project
	if s.GitBranch != "" {
		branch := lipgloss.NewStyle().Foreground(theme.ColorDimText).Render(" (" + s.GitBranch + ")")
		line1 += branch
	}

	// Line 2: model · status · PID
	var meta []string
	if s.Model != "" {
		meta = append(meta, s.Model)
	}
	meta = append(meta, string(s.Status))
	meta = append(meta, fmt.Sprintf("PID %d", s.PID))

	line2 := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(strings.Join(meta, " · "))

	return line1 + "\n" + line2
}

func (m *DetailModel) loadTranscript() {
	path := m.session.JSONLPath
	if path == "" {
		return
	}

	// Stat the file directly for fresh mtime (don't rely on Scan's cached value)
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	mtime := info.ModTime().UnixNano()

	// Skip reload if the file hasn't changed and we have data
	if mtime == m.lastMtime && len(m.entries) > 0 {
		return
	}

	entries, err := claude.ReadTranscript(path, m.session.Cwd)
	if err != nil || len(entries) == 0 {
		return // keep existing entries on error or empty result
	}
	atBottom := m.scroll == 0
	m.entries = entries
	m.lines = nil // invalidate rendered line cache
	m.lastMtime = mtime
	if atBottom {
		m.scroll = 0
	}
}

func (m *DetailModel) renderLines() []string {
	if len(m.entries) == 0 {
		return nil
	}

	maxContentWidth := max(m.width-4, 20) // 2 padding + 2 margin

	var lines []string
	var lastRole claude.Role

	for _, e := range m.entries {
		// Blank line between role changes and before tool interactions
		switch {
		case e.Role == claude.RoleToolUse:
			lines = append(lines, "")
		case lastRole != "" && e.Role != lastRole && e.Role != claude.RoleToolResult:
			lines = append(lines, "")
		}

		switch e.Role {
		case claude.RoleUser:
			// User prompt: ❯ styled like Claude Code's input prompt
			prompt := lipgloss.NewStyle().Foreground(theme.ColorSecondary).Bold(true).Render("❯")
			wrapped := wrapText(e.Content, maxContentWidth-2)
			if len(wrapped) > 0 {
				lines = append(lines, prompt+" "+lipgloss.NewStyle().Foreground(theme.ColorText).Bold(true).Render(wrapped[0]))
				for _, l := range wrapped[1:] {
					lines = append(lines, "  "+lipgloss.NewStyle().Foreground(theme.ColorText).Render(l))
				}
			}

		case claude.RoleAssistant:
			// Assistant message: white ● bullet, markdown rendered
			bullet := lipgloss.NewStyle().Foreground(theme.ColorText).Render("●")
			rendered := strings.Trim(m.renderMarkdown(e.Content), "\n")
			mdLines := strings.Split(rendered, "\n")
			if len(mdLines) > 0 {
				lines = append(lines, bullet+" "+mdLines[0])
				for _, l := range mdLines[1:] {
					lines = append(lines, "  "+l)
				}
			}

		case claude.RoleToolUse:
			// Tool use: ● colored by outcome — pending=yellow, success=green, error=red
			bulletStyle := lipgloss.NewStyle()
			switch e.Outcome {
			case claude.ToolSuccess:
				bulletStyle = bulletStyle.Foreground(theme.ColorSuccess)
			case claude.ToolError:
				bulletStyle = bulletStyle.Foreground(theme.ColorDanger)
			default:
				bulletStyle = bulletStyle.Foreground(theme.ColorWarning)
			}
			bullet := bulletStyle.Render("●")
			tool := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorText).Render(e.Content)
			lines = append(lines, bullet+" "+tool)

		case claude.RoleDiff:
			// Diff lines: red bg for -, green bg for +, dim for context
			if len(e.Content) >= 2 {
				switch e.Content[0] {
				case '-':
					styled := lipgloss.NewStyle().
						Foreground(lipgloss.Color("#E06060")).
						Background(lipgloss.Color("#2A1215")).
						Render("  " + e.Content)
					lines = append(lines, styled)
				case '+':
					styled := lipgloss.NewStyle().
						Foreground(lipgloss.Color("#60C060")).
						Background(lipgloss.Color("#122A15")).
						Render("  " + e.Content)
					lines = append(lines, styled)
				default:
					styled := lipgloss.NewStyle().
						Foreground(theme.ColorMuted).
						Render("  " + e.Content)
					lines = append(lines, styled)
				}
			}

		case claude.RoleToolResult:
			// Tool result: indented with └─ connector
			connector := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("  └ ")
			result := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(e.Content)
			lines = append(lines, connector+result)
		}

		lastRole = e.Role
	}

	return lines
}

// wrapText wraps a string to the given width, preserving existing newlines.
// mdRenderer returns a cached glamour renderer for the current width.
func (m *DetailModel) mdRenderer() *glamour.TermRenderer {
	if m.mdCache == nil || m.mdWidth != m.width {
		r, err := glamour.NewTermRenderer(
			glamour.WithStyles(theme.MonokaiStyleConfig),
			glamour.WithWordWrap(max(m.width-4, 20)),
		)
		if err == nil {
			m.mdCache = r
			m.mdWidth = m.width
		}
	}
	return m.mdCache
}

// renderMarkdown renders markdown content for the terminal.
// Falls back to plain text if rendering fails.
func (m *DetailModel) renderMarkdown(content string) string {
	r := m.mdRenderer()
	if r == nil {
		return content
	}
	rendered, err := r.Render(content)
	if err != nil {
		return content
	}
	return rendered
}

func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}

	var result []string
	for _, paragraph := range strings.Split(s, "\n") {
		if len(paragraph) <= width {
			result = append(result, paragraph)
			continue
		}
		// Simple word wrap
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}
		line := words[0]
		for _, w := range words[1:] {
			if len(line)+1+len(w) > width {
				result = append(result, line)
				line = w
			} else {
				line += " " + w
			}
		}
		result = append(result, line)
	}
	return result
}
