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

**Dark:** Molokai (default), Catppuccin Mocha, GitHub Dark, Nord, Solarized Dark, Tokyo Night

**Light:** Catppuccin Latte, GitHub Light, Nord Light, One Light, Solarized Light, Tokyo Night Light

Press `s` to switch themes. Your selection is saved to `~/.config/c5s/config.json`.

### Custom Themes

Place `.json` files in `~/.config/c5s/themes/` — they appear in the theme picker on next launch. The `name` field sets the display name; if omitted, the filename (minus `.json`) is used. At minimum, `fg` and `bg` must be set. Omitted fields default to empty (no color).

```json
{
  "name": "My Custom Theme",
  "appearance": "dark",
  "palette": {
    "fg": "#F8F8F2",
    "fg_alt": "#90908A",
    "bg": "#272822",
    "bg_alt": "#3E3D32",
    "comment": "#75715E",
    "red": "#F92672",
    "orange": "#FD971F",
    "yellow": "#E6DB74",
    "green": "#A6E22E",
    "cyan": "#A1EFE4",
    "blue": "#66D9EF",
    "magenta": "#AE81FF",
    "brown": "#CC6633",
    "diff": {
      "add_fg": "#A6E22E",
      "add_bg": "#233009",
      "remove_fg": "#F92672",
      "remove_bg": "#420A1E"
    }
  }
}
```

Accent color slots are ANSI-inspired: `red`, `orange`, `yellow`, `green`, `cyan`, `blue`, `magenta`, `brown`. These are slot names — themes assign any hex value they want. The `diff` object controls inline diff highlighting in the transcript view.

## License

[MIT](LICENSE)
