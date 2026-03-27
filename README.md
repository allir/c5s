# c5s

A k9s-style TUI for discovering and monitoring [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions.

c5s reads Claude Code's local session files (`~/.claude/sessions/`) to find running instances, then displays them in a live-updating terminal dashboard with status, project, branch, model, and session summary.

## Prerequisites

- **Go 1.26+**
- **Claude Code** running locally (creates the session data c5s reads)

## Quick Start

```bash
# Build
make build

# Run
./c5s

# Or build + run in one step
make run
```

## Development

```bash
# Run all checks (format, lint, vet, test)
make check

# Individual targets
make fmt      # Format code
make lint     # Run linters (requires golangci-lint)
make vet      # Run go vet
make test     # Run tests
make test-v   # Run tests with verbose output

# Install required tools
make tools
```

## How It Works

1. **Discovery** — Scans `~/.claude/sessions/*.json` for PID files, checks process liveness via `kill -0`.
2. **Hooks** — Installs Claude Code hooks on startup for real-time status events (prompt submitted, permission requested, session ended).
3. **Enrichment** — Matches each live session to its project JSONL file under `~/.claude/projects/`, extracting title, git branch, and model.
4. **Display** — Renders a Bubble Tea TUI with auto-refresh, showing status indicators: **working**, **idle**, **input** (waiting for approval), **finished**.
5. **Approvals** — When Claude Code requests tool permission, approve or deny directly from the dashboard detail view.
6. **Tmux Input** — If running in tmux, send text input to Claude Code sessions without switching panes.

## Keyboard Shortcuts

### Session List

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate sessions |
| `Enter` | Open session detail |
| `s` | Settings (theme picker) |
| `q` / `Ctrl+C` | Quit |
| `?` | Toggle help |

### Detail View

| Key | Action |
|-----|--------|
| `↑` / `↓` | Scroll transcript (or navigate approval options) |
| `PgUp` / `PgDn` | Page scroll |
| `a` | Approve selected option |
| `x` | Deny approval |
| `Escape` | Back to session list |

## Themes

Built-in themes:

**Dark:** Molokai (default), Catppuccin Mocha, Dracula, GitHub Dark, Nord, Solarized Dark, Tokyo Night

**Light:** Catppuccin Latte, GitHub Light, Solarized Light, Tokyo Night Day

Press `s` to switch themes. Your selection is saved to `~/.config/c5s/config.json`.

### Custom Themes

Drop a JSON file in `~/.config/c5s/themes/` and it will appear in the theme picker. The file can include a `name` field, or the filename (minus `.json`) is used as the theme name.

```json
{
  "name": "My Custom Theme",
  "palette": {
    "fg": "#F8F8F2",
    "fg_dim": "#90908A",
    "bg": "#272822",
    "bg_alt": "#3E3D32",
    "comment": "#75715E",
    "pink": "#F92672",
    "cyan": "#66D9EF",
    "green": "#A6E22E",
    "yellow": "#E6DB74",
    "purple": "#AE81FF",
    "orange": "#FD971F",
    "diff_add_fg": "#A6E22E",
    "diff_add_bg": "#2B3A1A",
    "diff_remove_fg": "#F92672",
    "diff_remove_bg": "#3A1A22"
  }
}
```

## License

[MIT](LICENSE)
