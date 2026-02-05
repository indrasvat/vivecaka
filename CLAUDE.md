# CLAUDE.md — Agent Conventions for vivecaka

## Continuous Task Loop

While Phase 14 tasks remain incomplete, keep executing tasks back-to-back. Auto-compactions will occur naturally — that's fine. After each compaction or at session start, re-orient:

### Re-orient (after compaction or session start)
1. Run `git log --oneline -5` — what just changed
2. Read `docs/PROGRESS.md` — find the next `TODO` task in Phase 14
3. Read the task file at `docs/tasks/NNN-task-name.md` — full context
4. Read this file (`CLAUDE.md`) for conventions

### Per-task loop
1. Read the relevant `docs/PRD.md` sections listed in the task file
2. Execute the task
3. **Verify** — run `make ci`, then run iterm2-driver visual QA (every task, no exceptions)
4. Update `docs/PROGRESS.md` — mark the task as DONE with notes
5. Update the task file `docs/tasks/NNN-*.md` — set `Status: DONE`
6. Commit the changes (atomic, conventional commit)
7. Immediately pick up the next TODO task — do not wait for user input

Do NOT skip verification. Do NOT mark tasks done without functional + visual confirmation.
The source of truth is always on disk: `PROGRESS.md`, task files, and git history.

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
| Shared TUI types | `internal/tui/core/*.go` (theme, styles, keymap, viewstate) |
| Task files | `docs/tasks/NNN-task-name.md` |
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
