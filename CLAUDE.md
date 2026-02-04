# CLAUDE.md — Agent Conventions for vivecaka

## Project Overview

vivecaka (विवेचक) is a plugin-based GitHub PR TUI built with Go + BubbleTea.
See `docs/PRD.md` for the full specification and `docs/PROGRESS.md` for live progress.

## Build & Test

```bash
make build     # Build binary → bin/vivecaka
make test      # Run tests with -race
make lint      # golangci-lint
make ci        # Full pipeline: fmt → vet → lint → test → build
make run       # Quick run with go run
```

## Architecture Rules

- **Dependency rule**: `tui → usecase → domain ← adapter`, `plugin` bridges domain + tui
- **domain**: Pure Go. No framework imports. No BubbleTea types.
- **usecase**: Depends only on domain interfaces. Use `errgroup` for concurrency, NOT `tea.Batch`.
- **plugin**: Lives in `internal/plugin/`. MAY import BubbleTea types (bridges domain + tui).
- **adapter**: Implements domain interfaces. Only `internal/adapter/ghcli/` for MVP.
- **tui**: Consumes use cases. Never imports adapters directly.
- **config**: Injected everywhere via functional options.

## Commit Conventions

- Use conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`
- Scope optional: `feat(domain):`, `fix(tui):`
- Keep commits atomic — one logical change per commit
- Run `make ci` before pushing (lefthook enforces this)

## File Placement

| What | Where |
|------|-------|
| Domain entities | `internal/domain/*.go` |
| Domain interfaces | `internal/domain/interfaces.go` |
| Use cases | `internal/usecase/*.go` |
| GH CLI adapter | `internal/adapter/ghcli/*.go` |
| Plugin infra | `internal/plugin/*.go` |
| TUI root model | `internal/tui/app.go` |
| View models | `internal/tui/views/*.go` |
| UI components | `internal/tui/components/*.go` |
| Theme/keymap | `internal/tui/theme.go`, `internal/tui/keymap.go` |
| Config | `internal/config/*.go` |
| Test fixtures | `internal/adapter/ghcli/testdata/*.json` |

## Testing Requirements

- All domain entities: 100% coverage
- Use cases: 90%+ with mocked adapters
- Adapters: 80%+ with fixture data in `testdata/`
- TUI models: test `Update()` and `View()` methods
- Run with `-race` flag always
- Use `testify` for assertions

## Progress Tracking

After completing each task:
1. Update `docs/PROGRESS.md` with task status
2. Note any deviations from the PRD
3. Record any issues discovered

## Key Dependencies

| Package | Import Path | Purpose |
|---------|-------------|---------|
| BubbleTea | `github.com/charmbracelet/bubbletea` | TUI framework |
| LipGloss | `github.com/charmbracelet/lipgloss` | Styling |
| Bubbles | `github.com/charmbracelet/bubbles` | UI components |
| Huh | `github.com/charmbracelet/huh` | Forms |
| go-toml | `github.com/pelletier/go-toml/v2` | Config |
| testify | `github.com/stretchr/testify` | Testing |

## Visual QA

After each phase, use the `iterm2-driver` skill to:
1. Launch `bin/vivecaka` in iTerm2
2. Screenshot each view state
3. Compare against `docs/mocks.html`
4. Verify: borders, colors, truncation, keyboard nav
