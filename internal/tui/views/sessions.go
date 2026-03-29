// Package views contains the TUI view components for c5s.
package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/claude"
	"github.com/allir/c5s/internal/tui/theme"
)

// SessionsModel displays a table of discovered Claude Code sessions.
type SessionsModel struct {
	sessions []claude.Session
	cursor   int
	width    int
	height   int
}

// NewSessionsModel creates a new sessions view.
func NewSessionsModel() SessionsModel {
	return SessionsModel{}
}

// SetSessions updates the session list.
func (m *SessionsModel) SetSessions(sessions []claude.Session) {
	m.sessions = sessions
	if m.cursor >= len(sessions) {
		m.cursor = max(0, len(sessions)-1)
	}
}

// SetSize updates the available dimensions.
func (m *SessionsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SessionCount returns the number of sessions.
func (m *SessionsModel) SessionCount() int {
	return len(m.sessions)
}

// MoveUp moves the cursor up one row.
func (m *SessionsModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// MoveDown moves the cursor down one row.
func (m *SessionsModel) MoveDown() {
	if m.cursor < len(m.sessions)-1 {
		m.cursor++
	}
}

// SelectedSession returns the currently selected session, if any.
func (m *SessionsModel) SelectedSession() *claude.Session {
	if m.cursor < len(m.sessions) {
		return &m.sessions[m.cursor]
	}
	return nil
}

// column defines a table column.
type column struct {
	title string
	width int
}

// View renders the sessions table.
func (m *SessionsModel) View() string {
	if len(m.sessions) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(theme.ColorFgAlt).
			Padding(2, 0).
			Width(m.width).
			Align(lipgloss.Center).
			Render("No sessions found. Start a Claude Code session and press r to refresh.")
		return empty
	}

	cols := m.columns()

	// Header row
	headerCells := make([]string, len(cols))
	for i, c := range cols {
		headerCells[i] = theme.StyleTableCell.Width(c.width).Render(
			theme.StyleTableHeader.Render(c.title),
		)
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerCells...)

	// Separator
	separator := theme.SeparatorLine(m.width)

	// Data rows
	visibleRows := max(m.height-2, 1)

	scrollOffset := 0
	if m.cursor >= visibleRows {
		scrollOffset = m.cursor - visibleRows + 1
	}

	rows := make([]string, 0, visibleRows)
	for i := scrollOffset; i < len(m.sessions) && i < scrollOffset+visibleRows; i++ {
		s := m.sessions[i]
		selected := i == m.cursor
		rowData := m.rowData(i, s, cols, selected)

		cells := make([]string, len(cols))
		for ci, c := range cols {
			style := theme.StyleTableCell.Width(c.width)
			if selected {
				style = style.Background(theme.ColorBgAlt)
			}
			content := rowData[ci]

			if selected {
				content = theme.StyleTableRowSelected.Render(content)
			} else {
				content = theme.StyleTableRow.Render(content)
			}

			cells[ci] = style.Render(content)
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
		if selected {
			row = lipgloss.NewStyle().Width(m.width).Background(theme.ColorBgAlt).Render(row)
		}
		rows = append(rows, row)
	}

	return header + "\n" + separator + "\n" + strings.Join(rows, "\n")
}

func (m *SessionsModel) columns() []column {
	fixed := 4 + 8 + 12 + 10 // #, pid, status, activity
	remaining := max(m.width-fixed, 30)

	// Scale flexible columns proportionally
	projectW := remaining * 20 / 100
	branchW := remaining * 12 / 100
	summaryW := remaining - projectW - branchW

	return []column{
		{"#", 4},
		{"PID", 8},
		{"PROJECT", max(projectW, 10)},
		{"BRANCH", max(branchW, 8)},
		{"STATUS", 12},
		{"SUMMARY", max(summaryW, 10)},
		{"ACTIVITY", 10},
	}
}

func columnWidth(cols []column, title string) int {
	for _, c := range cols {
		if c.title == title {
			return c.width
		}
	}
	return 0
}

// ApprovalLine returns a rendered line showing tool approval details for the selected session,
// or empty string if no approval is pending.
func (m *SessionsModel) ApprovalLine(width int) string {
	sel := m.SelectedSession()
	if sel == nil || sel.PendingApproval == nil {
		return ""
	}

	a := sel.PendingApproval
	summary := claude.SummarizeToolInput(a.ToolName, a.ToolInput)
	label := lipgloss.NewStyle().Foreground(theme.ColorWarning).Bold(true).Render("⚠ " + a.ToolName + ":")
	detail := lipgloss.NewStyle().Foreground(theme.ColorText).Render(" " + summary)
	hint := lipgloss.NewStyle().Foreground(theme.ColorFgAlt).Render("  (a:approve  x:deny)")

	line := label + detail + hint
	return lipgloss.NewStyle().Width(width).PaddingLeft(1).Render(line)
}

func (m *SessionsModel) rowData(idx int, s claude.Session, cols []column, selected bool) []string {
	pid := fmt.Sprintf("%d", s.PID)

	branch := claude.Truncate(s.GitBranch, max(columnWidth(cols, "BRANCH")-2, 4))
	project := claude.Truncate(s.Project, max(columnWidth(cols, "PROJECT")-2, 4))
	summary := claude.Truncate(s.Summary, max(columnWidth(cols, "SUMMARY")-2, 0))

	var status string
	if selected {
		status = theme.StatusIndicator(s.Status, theme.ColorBgAlt)
	} else {
		status = theme.StatusIndicator(s.Status)
	}

	return []string{
		fmt.Sprintf("%d", idx+1),
		pid,
		project,
		branch,
		status,
		summary,
		relativeTime(s.LastModified),
	}
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
