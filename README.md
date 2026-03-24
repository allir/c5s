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
2. **Enrichment** — Matches each live session to its project JSONL file under `~/.claude/projects/`, extracting title, git branch, and model from the first few lines.
3. **Display** — Renders a Bubble Tea TUI with auto-refresh, sortable columns, and status indicators (working/idle).

## License

[MIT](LICENSE)
