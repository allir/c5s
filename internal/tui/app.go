// Package tui implements the Bubble Tea terminal user interface for c5s.
package tui

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/allir/c5s/internal/claude"
	"github.com/allir/c5s/internal/tui/theme"
	"github.com/allir/c5s/internal/tui/views"
)

// nbsp is used by fillBg for padding instead of regular spaces. The Bubble Tea
// renderer treats regular spaces as "clearable" and uses erase operations for
// them. tmux renders erased cells differently from written cells when a
// background color is set, causing visible shade mismatches. Non-breaking
// spaces are not clearable, forcing the renderer to write each cell.
const nbsp = "\u00a0"

// DisplayConfig holds display settings passed from the config layer to the TUI.
type DisplayConfig struct {
	ActiveTheme string
	UseThemeBg  bool
	FillBg      bool
}

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
	display         DisplayConfig
	refreshInterval time.Duration
	err             error
}

// NewModel creates a new root model.
func NewModel(configDir string, refreshInterval time.Duration, cfg DisplayConfig) Model {
	return Model{
		sessions:        views.NewSessionsModel(),
		keys:            DefaultKeyMap(),
		configDir:       configDir,
		display:         cfg,
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

	if m.display.UseThemeBg && m.display.FillBg && m.width > 0 && m.height > 0 {
		content = applyFillBg(content, theme.ColorBg, m.width, m.height)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if m.display.UseThemeBg && !m.display.FillBg {
		v.BackgroundColor = theme.ColorBg
	}
	return v
}

func (m Model) saveConfig() {
	go func() {
		_ = claude.SaveConfig(claude.Config{
			Theme:      m.display.ActiveTheme,
			UseThemeBg: m.display.UseThemeBg,
			FillBg:     m.display.FillBg,
		})
	}()
}

// applyFillBg paints an explicit SGR background on every cell in the viewport.
// This avoids OSC 11 (View.BackgroundColor) which tmux renders differently
// from SGR backgrounds — written cells and erased cells show different shades.
func applyFillBg(content string, bg color.Color, width, height int) string {
	r, g, b, _ := bg.RGBA()
	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r>>8, g>>8, b>>8)
	nbspPad := strings.Repeat(nbsp, width)
	emptyLine := bgSeq + nbspPad

	// Single-pass replacer: spaces→NBSP and re-inject bg after every SGR reset.
	// Regular spaces are "clearable" by the renderer which triggers erase
	// operations that tmux renders at a different shade. NBSP is visually
	// identical but non-clearable.
	replacer := strings.NewReplacer(
		" ", nbsp,
		"\x1b[0m", "\x1b[0m"+bgSeq,
		"\x1b[m", "\x1b[m"+bgSeq,
	)

	lines := strings.Split(content, "\n")

	var buf strings.Builder
	buf.Grow(width * 3 * height) // estimate: content + ANSI overhead

	for i, line := range lines {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(bgSeq)
		_, _ = replacer.WriteString(&buf, line)
		if pad := width - lipgloss.Width(line); pad > 0 {
			buf.WriteString(bgSeq)
			buf.WriteString(nbspPad[:pad*len(nbsp)])
		}
	}

	for i := len(lines); i < height; i++ {
		buf.WriteByte('\n')
		buf.WriteString(emptyLine)
	}

	return buf.String()
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

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// Diff debug view
	if m.view == viewDiffDebug && m.diffDebug != nil {
		switch {
		case matches(key, m.keys.Back), key == "q":
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
			if m.settings.IsOnBgToggle() {
				m.display.UseThemeBg = !m.display.UseThemeBg
				m.settings.UseThemeBg = m.display.UseThemeBg
				m.settings.ClampCursor()
				m.saveConfig()
			} else if m.settings.IsOnFillBgToggle() {
				m.display.FillBg = !m.display.FillBg
				m.settings.FillBg = m.display.FillBg
				m.saveConfig()
			} else if name, palette := m.settings.SelectedTheme(); name != "" {
				theme.ApplyPalette(palette)
				m.display.ActiveTheme = name
				m.settings.SetActive(m.settings.Cursor())
				if m.detail != nil {
					m.detail.InvalidateCache()
				}
				m.saveConfig()
			}
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
		settings := views.NewSettingsModel(m.display.ActiveTheme, m.display.UseThemeBg, m.display.FillBg)
		m.settings = &settings
		m.view = viewSettings
	case key == "d" && debugEnabled:
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
