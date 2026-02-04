# vivecaka (विवेचक)

**A stunningly beautiful, blazing-fast TUI for browsing and reviewing GitHub Pull Requests.**

*vivecaka* (Sanskrit: विवेचक) means "one who examines/analyzes" — a discerning reviewer.

```
 ██▄  ▄██   ████     ██▄  ▄██   ▄████▄    ▄█████▄   ▄█████▄  ██ ▄██▀    ▄█████▄
  ██  ██      ██      ██  ██   ██▄▄▄▄██  ██▀    ▀   ▀ ▄▄▄██  ██▄██      ▀ ▄▄▄██
  ▀█▄▄█▀      ██      ▀█▄▄█▀   ██▀▀▀▀▀▀  ██        ▄██▀▀▀██  ██▀██▄    ▄██▀▀▀██
   ████    ▄▄▄██▄▄▄    ████    ▀██▄▄▄▄█  ▀██▄▄▄▄█  ██▄▄▄███  ██  ▀█▄   ██▄▄▄███
    ▀▀     ▀▀▀▀▀▀▀▀     ▀▀       ▀▀▀▀▀     ▀▀▀▀▀    ▀▀▀▀ ▀▀  ▀▀   ▀▀▀   ▀▀▀▀ ▀▀
```

## Features

- **Plugin architecture** — Everything is a plugin. Swap `gh` CLI for direct API, add GitLab, extend views.
- **Context-aware** — Auto-detects repo, remembers filters per repo, highlights your current branch's PR.
- **Inline review** — Add/view/resolve review comments on diff lines without leaving the terminal.
- **Dual diff viewer** — Built-in syntax-highlighted diff + delegate to delta, difftastic, or any tool.
- **Auto-refresh** — Background polling with visual indicators for new activity.
- **Multi-repo favorites** — Pin repos for quick switching, unified PR inbox across all favorites.
- **Beautiful by default** — Catppuccin, Tokyo Night, Dracula themes with adaptive terminal colors.
- **Keyboard-first** — Full vim-style navigation. No mouse required. No hand gymnastics.

## Requirements

- Go 1.25+
- [GitHub CLI](https://cli.github.com/) (`gh`) installed and authenticated

## Installation

```bash
# From source
go install github.com/indrasvat/vivecaka/cmd/vivecaka@latest

# Or build locally
git clone https://github.com/indrasvat/vivecaka.git
cd vivecaka
make build
./bin/vivecaka
```

## Quick Start

```bash
# Navigate to any GitHub repo and run:
vivecaka

# Or specify a repo:
vivecaka --repo owner/name
```

## Keybindings

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate up/down |
| `Enter` | Open PR detail |
| `d` | View diff |
| `c` | Checkout branch |
| `r` | Submit review |
| `f` | Filter panel |
| `/` | Search |
| `Ctrl+R` | Switch repo |
| `?` | Help |
| `q` | Back / quit |

## Configuration

Config lives at `~/.config/vivecaka/config.toml` (created on first run):

```toml
[general]
theme = "default-dark"
refresh_interval = 30

[diff]
mode = "unified"
external_tool = ""

[repos]
favorites = ["owner/repo1", "owner/repo2"]
```

## Development

```bash
make help     # Show all targets
make ci       # Full quality pipeline
make test     # Tests with race detector
make lint     # golangci-lint
make dev      # Run with auto-reload
```

## License

MIT — see [LICENSE](LICENSE)
