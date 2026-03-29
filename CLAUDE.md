# CLAUDE.md — c5s

## What is c5s

A terminal dashboard for Claude Code sessions, inspired by k9s. It discovers running Claude Code instances by reading their local session files and displays them in a Bubble Tea TUI.

## Project Structure

```
main.go                          # Entry point — delegates to cmd
cmd/root.go                      # Cobra CLI setup, hook lifecycle, theme loading, launches TUI
internal/
  claude/
    types.go                     # Session struct & Status enum
    config.go                    # Config persistence (LoadConfig, SaveConfig)
    dirs.go                      # XDG directory helpers (C5sConfigDir, C5sStateDir)
    dirs_test.go                 # Unit tests for XDG directory helpers
    discovery.go                 # Scan(), PID liveness, JSONL parsing, tool use detection
    discovery_test.go            # Unit tests for discovery logic
    approval.go                  # Tool approval workflow (read pending, write decisions)
    approval_test.go             # Unit tests for approval lifecycle
    hooks.go                     # Hook install/uninstall, event reading, status mapping
    hooks_test.go                # Unit tests for hook lifecycle and event parsing
    pidfiles.go                  # PID file reading utility with age-based cleanup
    pidfiles_test.go             # Unit tests for PID file reader
    tmux.go                      # Tmux pane discovery and remote key sending
    transcript.go                # JSONL transcript parsing into renderable entries
    transcript_test.go           # Unit tests for transcript parsing
  tui/
    app.go                       # Bubble Tea model, Update/View loop, auto-refresh tick
    header.go                    # Header bar rendering
    statusbar.go                 # Bottom status bar
    keys.go                      # Key bindings
    theme/palette.go             # Palette/Theme types, built-in palettes, user theme loader
    theme/theme.go               # Semantic color aliases and lipgloss styles from palette
    theme/markdown.go            # Glamour markdown style config from palette
    views/sessions.go            # Session table view
    views/detail.go              # Session detail view with transcript and approval UI
    views/highlight.go           # Chroma-based syntax highlighting for diff blocks
    views/settings.go            # Settings view with theme picker
  version/version.go             # Build-time version info via ldflags
```

## Key Concepts

### Session Discovery (`internal/claude/`)

- **PID-based scanning**: `Scan()` reads `~/.claude/sessions/*.json` (PID files), checks liveness with `kill -0`, then enriches from JSONL files under `~/.claude/projects/`. Processing uses bounded concurrency (`maxScanWorkers` goroutines via `errgroup`) to handle large session counts without exhausting file descriptors.
- **Hook-based discovery**: c5s installs Claude Code hooks on startup (`InstallHooks`) and removes them on exit (`UninstallHooks`). The hooks write event files to `~/.local/state/c5s/events/<PID>.json` with authoritative session ID, cwd, and transcript path. This eliminates session ID drift after `/clear`.
- **Hook events registered**: SessionStart, SessionEnd, Stop, UserPromptSubmit, PermissionRequest, SubagentStart, PostToolUseFailure (Bash matcher). All async with 5s timeout.
- **Coexistence**: c5s hooks use `~/.config/c5s/hooks/status-hook.sh`, separate from claude-control's hooks. Both can run simultaneously.
- **XDG directories**: Config at `$XDG_CONFIG_HOME/c5s` (default `~/.config/c5s`), state at `$XDG_STATE_HOME/c5s` (default `~/.local/state/c5s`).
- Path encoding: Claude Code stores project dirs with non-alphanumeric chars replaced by `-`. `encodeCwd()` handles this.
- Error handling is intentionally silent (skip and continue) — a dashboard that auto-refreshes every few seconds should not crash because one session file is temporarily unreadable. Hook installation/uninstallation failures are non-fatal warnings.

### Status Classification

Status is determined by a three-tier priority system:

1. **Hook events** (authoritative): `UserPromptSubmit`/`SubagentStart`/`PostToolUseFailure` → working, `Stop`/`SessionStart` → idle, `SessionEnd` → finished. `PermissionRequest` intentionally excluded (fires for auto-approved tools).
2. **JSONL content analysis**: If the last assistant message has a pending `tool_use` (no `tool_result` after it) and the JSONL is stale for >2s (`approvalSettleTime`), status is `input` (waiting for user approval). If a tool interaction just resolved and the JSONL is stale, status is `idle`.
3. **JSONL mtime heuristic** (fallback): If modified within 2 minutes → working, otherwise → idle.

Status values: `working`, `idle`, `input` (waiting for approval), `finished` (session ended), `unknown` (fallback).

### TUI Layer (`internal/tui/`)

- Built on Bubble Tea v2 + Lip Gloss v2.
- Auto-refreshes session list on a tick.
- Key bindings defined in `keys.go`.
- Three views: session list, detail/transcript, and settings (theme picker).

### Tool Approval System

- When Claude Code requests tool approval, the `PermissionRequest` hook writes a pending file to `~/.local/state/c5s/pending/<hookPID>.json` containing session ID, tool name, tool input, and available permission suggestions.
- `ReadPendingApprovals()` reads pending files, filters out stale/decided entries, matches to sessions by parent PID, and builds selectable options via `buildApprovalOptions()`.
- The detail view renders an approval prompt with numbered options. Users navigate with up/down and approve (`a`) or deny (`x`).
- `WriteApprovalDecision()` writes the decision to `~/.local/state/c5s/decisions/<hookPID>.json`. The hook script polls for this file and exits with the result.
- Stale approvals (>10 min) are automatically cleaned up during reads.

### Detail View & Transcript (`internal/tui/views/detail.go`, `internal/claude/transcript.go`)

- Pressing Enter on a session opens the detail view, which reads the JSONL transcript tail (last 2MB) via `ReadTranscript()`.
- Transcript entries are parsed into typed entries: user prompts, assistant text, tool_use calls (with inline diffs for Edit), and tool_result summaries.
- Tool outcomes (pending/success/error) are resolved by matching `tool_use` → `tool_result` entries by ID.
- Markdown content is rendered via glamour with a palette-driven style config (`theme/markdown.go`).
- Edit diffs show syntax-highlighted code with green/red diff backgrounds (`views/highlight.go`).
- The view auto-refreshes on tick, reloading only when the JSONL mtime changes.

### Theme System (`internal/tui/theme/`)

- **Palette**: raw hex color values for a complete color scheme. Fields have JSON tags for user theme files.
- **Theme**: pairs a name with a palette. Built-in themes are package-level vars (`ThemeMolokai`, `ThemeNord`, etc.).
- **`ApplyPalette()`**: rebuilds all derived `Color*` vars and `Style*` vars at runtime for instant theme switching.
- **Built-in themes** (6 dark, 6 light): Molokai (default), Catppuccin Mocha, GitHub Dark, Nord, Solarized Dark, Tokyo Night, Catppuccin Latte, GitHub Light, Nord Light, One Light, Solarized Light, Tokyo Night Light.
- **User themes**: JSON files in `~/.config/c5s/themes/` are loaded at startup via `LoadUserThemes()`. Supports both full `{"name": "...", "palette": {...}}` and bare palette formats.
- **Settings view** (`views/settings.go`): press `s` to open, shows themes grouped by dark/light (auto-detected from bg luminance), with color swatches.
- **Config persistence** (`claude/config.go`): theme selection saved to `~/.config/c5s/config.json`.
- **Diff bg colors**: computed as dark shades of the diff fg colors, with red boosted ~40% for perceptual luminance compensation (see `tmp/diff-color-tuning.md`).

### Syntax Highlighting (`internal/tui/views/highlight.go`)

- Uses Chroma (transitive dep via glamour) to tokenize code in diff blocks by file extension.
- Token colors are mapped to the active palette, matching the glamour Chroma config in `markdown.go`.
- **Inline diff highlighting**: Adjacent delete/insert line pairs get character-level sub-diffs via `go-udiff`. Changed characters render with a brighter background (`DiffAddInlineBg`/`DiffRemoveInlineBg`) while unchanged portions keep the normal diff background.
- Each token is rendered with its syntax fg color AND the line's diff background, avoiding ANSI reset issues.
- Consecutive diff entries are batched for block-level tokenization (better context for the lexer).

### Tmux Integration (`internal/claude/tmux.go`)

- If a session's PID maps to a tmux pane (discovered via `FindTmuxPane`), the detail view enables text input mode.
- Users can type messages and send them to the Claude Code session via `SendTmuxKeys`, enabling remote interaction without switching terminals.
- Input mode auto-enables when a tmux pane is detected for the session.

## Development

```bash
# The one command you need
make check    # fmt + lint + vet + test

# Build and run
make build && ./c5s

# Install tooling (golangci-lint)
make tools
```

## Conventions

- Go standard project layout with `internal/`.
- No CGO. Pure Go.
- Direct dependencies: Bubble Tea, Lip Gloss, Glamour, Cobra, Chroma (via glamour), go-udiff, x/sync. Keep it minimal.
- Tests live next to the code they test (`_test.go` suffix).
- Version info injected at build time via ldflags (see Makefile).
