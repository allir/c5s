// Package tui implements the Bubble Tea terminal user interface for c5s.
package tui

import (
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/claude"
	"github.com/allir/c5s/internal/tui/theme"
	"github.com/allir/c5s/internal/tui/views"
)

// DefaultRefreshInterval is how often the session list auto-refreshes.
const DefaultRefreshInterval = 1500 * time.Millisecond

// chromeHeight is the number of lines used by header (2), separator, and status bar.
const chromeHeight = 4

// viewState tracks which view is currently active.
type viewState int

const (
	viewSessions viewState = iota
	viewDetail
	viewSettings
	viewDiffDebug
)

// Messages

type sessionsFetchedMsg struct {
	sessions []claude.Session
	err      error
}

type approvalWrittenMsg struct {
	err error
}

type tickMsg struct{}

// Model is the root Bubble Tea model for the c5s application.
type Model struct {
	width           int
	height          int
	view            viewState
	sessions        views.SessionsModel
	detail          *views.DetailModel
	settings        *views.SettingsModel
	diffDebug       *views.DiffDebugModel
	keys            KeyMap
	configDir       string
	activeTheme     string // current theme name (for config persistence)
	refreshInterval time.Duration
	err             error
}

// NewModel creates a new root model.
func NewModel(configDir string, refreshInterval time.Duration, activeTheme string) Model {
	return Model{
		sessions:        views.NewSessionsModel(),
		keys:            DefaultKeyMap(),
		configDir:       configDir,
		activeTheme:     activeTheme,
		refreshInterval: refreshInterval,
	}
}

// Init returns the initial command to run on startup.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchSessions(), m.tickCmd())
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.fetchSessions(), m.tickCmd())

	case sessionsFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.sessions.SetSessions(msg.sessions)
		// Update detail view session if open
		if m.detail != nil {
			for _, s := range msg.sessions {
				if s.PID == m.detail.Session().PID {
					m.detail.UpdateSession(s)
					m.detail.Refresh()
					break
				}
			}
		}
		return m, nil

	case approvalWrittenMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		return m, m.fetchSessions()

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		if m.view == viewDetail && m.detail != nil {
			if msg.Mouse().Button == tea.MouseWheelUp {
				m.detail.ScrollUp()
			} else if msg.Mouse().Button == tea.MouseWheelDown {
				m.detail.ScrollDown()
			}
		}
	}

	return m, nil
}

// View renders the entire application.
func (m Model) View() tea.View {
	var content string

	if m.width == 0 {
		content = "Starting c5s..."
	} else if m.view == viewDiffDebug && m.diffDebug != nil {
		m.diffDebug.SetSize(m.width, m.height)
		content = m.diffDebug.View()
	} else if m.view == viewSettings && m.settings != nil {
		content = m.renderSettingsView()
	} else if m.view == viewDetail && m.detail != nil {
		content = m.renderDetailView()
	} else {
		content = m.renderSessionsView()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) renderSessionsView() string {
	header := headerView(m.sessions.SessionCount(), m.width, []keyHint{
		{"q", "quit"},
		{"a", "approve"},
		{"x", "deny"},
		{"enter", "details"},
		{"s", "settings"},
		{"?", "help"},
	})
	approvalLine := m.sessions.ApprovalLine(m.width)

	extra := 0
	if approvalLine != "" {
		extra = 1
	}
	contentHeight := max(m.height-chromeHeight-extra, 1)
	m.sessions.SetSize(m.width, contentHeight)

	body := m.sessions.View()
	separator := theme.SeparatorLine(m.width)

	parts := []string{header, separator, body}
	if approvalLine != "" {
		parts = append(parts, approvalLine)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderDetailView() string {
	header := m.detail.HeaderInfo()
	header = lipgloss.NewStyle().Width(m.width).PaddingLeft(1).Render(header)
	statusBar := detailStatusBar(m.width)
	separator := theme.SeparatorLine(m.width)
	approvalBlock := m.detail.ApprovalBlock(m.width)

	// Hide input when approval is showing
	var inputLine string
	if approvalBlock == "" {
		inputLine = m.detail.InputLine(m.width)
	}

	// Detail chrome: header(2) + sep + body + sep + statusbar = chromeHeight+1
	extra := 1 // separator above status bar
	if approvalBlock != "" {
		extra += 1 + strings.Count(approvalBlock, "\n") + 1 // separator + block lines
	}
	if inputLine != "" {
		extra += 2 // separator + input line
	}
	contentHeight := max(m.height-chromeHeight-extra, 1)
	m.detail.SetSize(m.width, contentHeight)
	body := m.detail.View()

	parts := []string{header, separator, body}
	if approvalBlock != "" {
		parts = append(parts, theme.SeparatorLine(m.width), approvalBlock)
	}
	if inputLine != "" {
		parts = append(parts, theme.SeparatorLine(m.width), inputLine)
	}
	parts = append(parts, theme.SeparatorLine(m.width), statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderSettingsView() string {
	header := headerView(m.sessions.SessionCount(), m.width, nil)
	separator := theme.SeparatorLine(m.width)
	statusBar := settingsStatusBar(m.width)

	contentHeight := max(m.height-chromeHeight, 1)
	m.settings.SetSize(m.width, contentHeight)
	body := m.settings.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, separator, body, statusBar)
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Diff debug view
	if m.view == viewDiffDebug && m.diffDebug != nil {
		switch {
		case matches(key, m.keys.Back), matches(key, m.keys.Quit):
			m.view = viewSessions
			m.diffDebug = nil
		case matches(key, m.keys.Up):
			m.diffDebug.ScrollUp()
		case matches(key, m.keys.Down):
			m.diffDebug.ScrollDown()
		}
		return m, nil
	}

	// Settings view
	if m.view == viewSettings && m.settings != nil {
		switch {
		case matches(key, m.keys.Back):
			m.view = viewSessions
			m.settings = nil
		case matches(key, m.keys.Quit):
			return m, tea.Quit
		case matches(key, m.keys.Up):
			m.settings.MoveUp()
		case matches(key, m.keys.Down):
			m.settings.MoveDown()
		case matches(key, m.keys.Select):
			name, palette := m.settings.SelectedTheme()
			theme.ApplyPalette(palette)
			m.activeTheme = name
			m.settings.SetActive(m.settings.Cursor())
			if m.detail != nil {
				m.detail.InvalidateCache()
			}
			// Save config asynchronously
			go func() { _ = claude.SaveConfig(claude.Config{Theme: name}) }()
			m.view = viewSessions
			m.settings = nil
		}
		return m, nil
	}

	// Detail view with input mode: input box captures most keys
	if m.view == viewDetail && m.detail != nil && m.detail.InputMode() {
		hasApproval := m.detail.Session().PendingApproval != nil

		switch {
		case matches(key, m.keys.Back):
			m.view = viewSessions
			m.detail = nil
		case matches(key, m.keys.Quit):
			return m, tea.Quit
		case matches(key, m.keys.Select):
			// Enter: confirm approval option if pending, otherwise send input
			if hasApproval {
				return m, m.writeSelectedApproval()
			}
			if err := m.detail.SendInput(); err != nil {
				m.err = err
			}
		case matches(key, m.keys.Up):
			if hasApproval {
				m.detail.ApprovalCursorUp()
			} else {
				m.detail.ScrollUp()
			}
		case matches(key, m.keys.Down):
			if hasApproval {
				m.detail.ApprovalCursorDown()
			} else {
				m.detail.ScrollDown()
			}
		case key == "backspace":
			m.detail.InputBackspace()
		case key == "left":
			m.detail.InputCursorLeft()
		case key == "right":
			m.detail.InputCursorRight()
		case matches(key, m.keys.PageUp):
			m.detail.PageUp()
		case matches(key, m.keys.PageDn):
			m.detail.PageDown()
		default:
			if k := tea.KeyPressMsg(msg); k.Text != "" {
				m.detail.InputInsert(k.Text)
			}
		}
		return m, nil
	}

	// Detail view without input (no tmux)
	if m.view == viewDetail && m.detail != nil {
		hasApproval := m.detail.Session().PendingApproval != nil

		switch {
		case matches(key, m.keys.Back):
			m.view = viewSessions
			m.detail = nil
		case matches(key, m.keys.Quit):
			return m, tea.Quit
		case matches(key, m.keys.Up):
			if hasApproval {
				m.detail.ApprovalCursorUp()
			} else {
				m.detail.ScrollUp()
			}
		case matches(key, m.keys.Down):
			if hasApproval {
				m.detail.ApprovalCursorDown()
			} else {
				m.detail.ScrollDown()
			}
		case matches(key, m.keys.PageUp):
			m.detail.PageUp()
		case matches(key, m.keys.PageDn):
			m.detail.PageDown()
		case matches(key, m.keys.Select):
			if hasApproval {
				return m, m.writeSelectedApproval()
			}
		case matches(key, m.keys.Approve):
			return m, m.writeApproval(claude.ApprovalOption{Label: "Yes", Allow: true})
		case matches(key, m.keys.Deny):
			return m, m.writeApproval(claude.ApprovalOption{Label: "No", Allow: false})
		}
		return m, nil
	}

	// Sessions view keys
	switch {
	case matches(key, m.keys.Quit):
		return m, tea.Quit
	case matches(key, m.keys.Up):
		m.sessions.MoveUp()
	case matches(key, m.keys.Down):
		m.sessions.MoveDown()
	case matches(key, m.keys.Select):
		if sel := m.sessions.SelectedSession(); sel != nil && sel.JSONLPath != "" {
			detail := views.NewDetailModel(*sel)
			m.detail = &detail
			m.view = viewDetail
		}
	case matches(key, m.keys.Help):
		// Placeholder — will show help overlay
	case matches(key, m.keys.Settings):
		settings := views.NewSettingsModel(m.activeTheme)
		m.settings = &settings
		m.view = viewSettings
	case key == "d":
		dd := views.NewDiffDebugModel()
		m.diffDebug = &dd
		m.view = viewDiffDebug
	case matches(key, m.keys.Approve):
		return m, m.writeApproval(claude.ApprovalOption{Label: "Yes", Allow: true})
	case matches(key, m.keys.Deny):
		return m, m.writeApproval(claude.ApprovalOption{Label: "No", Allow: false})
	}

	return m, nil
}

func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) fetchSessions() tea.Cmd {
	configDir := m.configDir
	return func() tea.Msg {
		sessions, hookEvents, err := claude.Scan(configDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return sessionsFetchedMsg{err: err}
			}
			sessions = nil
		}

		approvals, _ := claude.ReadPendingApprovals(hookEvents)

		if len(approvals) > 0 {
			for i := range sessions {
				if a, ok := approvals[sessions[i].PID]; ok {
					if sessions[i].LastModified.After(a.Timestamp.Add(claude.ApprovalSettleTime)) {
						continue
					}
					sessions[i].PendingApproval = &a
					sessions[i].Status = claude.StatusInput
				}
			}
		}

		return sessionsFetchedMsg{sessions: sessions}
	}
}

func (m Model) activeSession() *claude.Session {
	if m.view == viewDetail && m.detail != nil {
		s := m.detail.Session()
		return &s
	}
	return m.sessions.SelectedSession()
}

func (m Model) writeApproval(option claude.ApprovalOption) tea.Cmd {
	sel := m.activeSession()
	if sel == nil || sel.PendingApproval == nil {
		return nil
	}
	hookPID := sel.PendingApproval.HookPID
	return func() tea.Msg {
		return approvalWrittenMsg{err: claude.WriteApprovalDecision(hookPID, option)}
	}
}

func (m Model) writeSelectedApproval() tea.Cmd {
	if m.detail == nil {
		return nil
	}
	opt := m.detail.SelectedApprovalOption()
	if opt == nil {
		return nil
	}
	return m.writeApproval(*opt)
}
