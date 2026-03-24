# CLAUDE.md — c5s

## What is c5s

A terminal dashboard for Claude Code sessions, inspired by k9s. It discovers running Claude Code instances by reading their local session files and displays them in a Bubble Tea TUI.

## Project Structure

```
main.go                          # Entry point — delegates to cmd
cmd/root.go                      # Cobra CLI setup, hook lifecycle, launches the TUI
internal/
  claude/
    types.go                     # Session struct & Status enum
    dirs.go                      # XDG directory helpers (C5sConfigDir, C5sStateDir)
    discovery.go                 # Scan(), PID liveness, JSONL parsing, tool use detection
    hooks.go                     # Hook install/uninstall, event reading, status mapping
    discovery_test.go            # Unit tests for discovery logic
    dirs_test.go                 # Unit tests for XDG directory helpers
    hooks_test.go                # Unit tests for hook lifecycle and event parsing
  tui/
    app.go                       # Bubble Tea model, Update/View loop, auto-refresh tick
    header.go                    # Header bar rendering
    statusbar.go                 # Bottom status bar
    keys.go                      # Key bindings
    theme/theme.go               # Color palette and styles
    views/sessions.go            # Session table view
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
- Key bindings defined in `keys.go`, theme in `theme/theme.go`.

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
- Direct dependencies: Bubble Tea, Lip Gloss, Cobra, x/sync. Keep it minimal.
- Tests live next to the code they test (`_test.go` suffix).
- Version info injected at build time via ldflags (see Makefile).
