# vivecaka (विवेचक)

**A plugin-based TUI for browsing and reviewing GitHub Pull Requests — entirely from the terminal.**

*vivecaka* (Sanskrit: विवेचक) means "one who examines" — a discerning reviewer.

```
 ██▄  ▄██   ████     ██▄  ▄██   ▄████▄    ▄█████▄   ▄█████▄  ██ ▄██▀    ▄█████▄
  ██  ██      ██      ██  ██   ██▄▄▄▄██  ██▀    ▀   ▀ ▄▄▄██  ██▄██      ▀ ▄▄▄██
  ▀█▄▄█▀      ██      ▀█▄▄█▀   ██▀▀▀▀▀▀  ██        ▄██▀▀▀██  ██▀██▄    ▄██▀▀▀██
   ████    ▄▄▄██▄▄▄    ████    ▀██▄▄▄▄█  ▀██▄▄▄▄█  ██▄▄▄███  ██  ▀█▄   ██▄▄▄███
    ▀▀     ▀▀▀▀▀▀▀▀     ▀▀       ▀▀▀▀▀     ▀▀▀▀▀    ▀▀▀▀ ▀▀  ▀▀   ▀▀▀   ▀▀▀▀ ▀▀
```

## Features

- **Full PR workflow** — Browse, review, comment, and checkout PRs without leaving the terminal
- **Rich diff viewer** — Unified and side-by-side modes, syntax highlighting, inline comments, file tree navigation
- **Smart checkout** — Context-aware checkout with worktree support and known-repo auto-learning
- **Multi-repo inbox** — Pin favorite repos, unified PR inbox with priority sorting across all of them
- **Plugin architecture** — Clean interface-based design. Swap `gh` CLI for direct API, add GitLab, extend views
- **5 built-in themes** — Catppuccin Mocha (default), Tokyo Night, Dracula, and more
- **Keyboard-first** — Vim-style navigation throughout. Customizable keybindings via config

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

## Usage

```bash
# In any GitHub repo:
vivecaka

# Or specify a repo:
vivecaka --repo owner/name

# Debug mode:
vivecaka --debug
```

## Keybindings

### PR List

| Key | Action |
|-----|--------|
| `j`/`k` | Navigate up/down |
| `Enter` | Open PR detail |
| `/` | Search |
| `s` | Cycle sort (updated/created/title/comments) |
| `m` | My PRs |
| `n` | Needs my review |
| `f` | Filter panel |
| `v` | Visual selection mode |
| `y` | Copy PR URL |
| `o` | Open in browser |
| `I` | Unified inbox |
| `R` | Switch repo |
| `T` | Cycle theme |
| `?` | Help |
| `q` | Quit |

### PR Detail

| Key | Action |
|-----|--------|
| `1`–`4` | Switch tabs (Description, Checks, Files, Comments) |
| `d` | View diff |
| `c` | Checkout branch |
| `r` | Submit review |
| `o` | Open in browser |

### Diff Viewer

| Key | Action |
|-----|--------|
| `Tab` | Toggle file tree / diff pane focus |
| `t` | Toggle unified / side-by-side |
| `[`/`]` | Previous / next hunk |
| `{`/`}` | Previous / next file |
| `c` | Add inline comment |
| `e` | Open in external diff tool |
| `/` | Search in diff |
| `za` | Collapse/expand file |
| `gg`/`G` | Top / bottom |

## Configuration

Config at `~/.config/vivecaka/config.toml` (auto-created on first run):

```toml
[general]
theme = "default-dark"    # default-dark, catppuccin, tokyonight, dracula, solarized
refresh_interval = 30
page_size = 50

[diff]
mode = "unified"          # unified or side-by-side
external_tool = ""        # e.g. "delta", "difftastic"

[repos]
favorites = ["owner/repo1", "owner/repo2"]

[keybindings]
quit = "q"
search = "/"
# Override any binding
```

## Architecture

Clean Architecture with strict dependency rules:

```
tui → usecase → domain ← adapter
         ↑
       plugin (bridges domain + tui)
```

| Layer | Responsibility |
|-------|----------------|
| `domain` | Pure Go entities and interfaces — no framework deps |
| `usecase` | Business logic with `errgroup` concurrency |
| `adapter` | `gh` CLI integration via `go-gh` |
| `plugin` | Interface-based plugin system (compile-time) |
| `tui` | BubbleTea views and components |
| `config` | TOML config with XDG-compliant paths |
| `cache` | JSON file cache for instant startup + per-repo state |

## Development

```bash
make build    # Build binary → bin/vivecaka
make test     # Tests with -race
make lint     # golangci-lint
make ci       # Full pipeline: fmt → vet → lint → test → build
```

## License

MIT — see [LICENSE](LICENSE)
