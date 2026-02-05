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
4. **Mock check** — use `npm exec agent-browser --` to open `docs/mocks.html` and the relevant `screenshots/mock_*.png`, confirm TUI matches the mock
5. Update `docs/PROGRESS.md` — mark the task as DONE with notes
6. Update the task file `docs/tasks/NNN-*.md` — set `Status: DONE`
7. Commit the changes (atomic, conventional commit)
8. Immediately pick up the next TODO task — do not wait for user input

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

## BubbleTea/LipGloss Learnings

Critical patterns learned from yukti project for terminal UI rendering:

### Terminal Background Color (CRITICAL)
```go
// Set BEFORE starting BubbleTea program in main.go
output := termenv.NewOutput(os.Stdout)
output.SetBackgroundColor(output.Color("#1E1E2E"))  // Catppuccin Mocha base
// ... run program ...
output.Reset()  // Reset BEFORE os.Exit
```
- `lipgloss.Background()` only works on styled characters
- Empty cells use terminal's default background
- `termenv.SetBackgroundColor()` via OSC 11 sets ALL cells

### Screen Clearing on View Transitions
- When transitioning from full-screen views (like banner), use `tea.ClearScreen`
- Return it as a command from Update(): `return a, tea.ClearScreen`
- Otherwise previous content bleeds through unwritten areas

### Full-Width Padding Lines (CRITICAL)
```go
// Padding lines MUST be full-width spaces, NOT empty strings
func ensureExactHeight(content string, height, width int) string {
    lines := strings.Split(content, "\n")
    if len(lines) > height {
        lines = lines[:height]
    }
    emptyLine := strings.Repeat(" ", width)  // FULL WIDTH!
    for len(lines) < height {
        lines = append(lines, emptyLine)
    }
    return strings.Join(lines, "\n")
}
```
- Empty strings `""` break `ansi.Cut()` for modal overlays
- Newlines alone don't overwrite previous content
- Each line must have actual space characters

### ANSI Resets Between Styled Elements
```go
result.WriteString("\033[0m")  // Reset after styled element
result.WriteString(styledContent)
result.WriteString("\033[0m")  // Reset before padding
result.WriteString(padding)
```
- Add explicit resets to prevent style bleeding
- Especially important when compositing modals over backgrounds

### APIs to AVOID
| Don't Use | Why | Use Instead |
|-----------|-----|-------------|
| `lipgloss.Height(n)` | Sets MINIMUM, not exact | `MaxHeight + manual padding` |
| `lipgloss.Background()` | Only works on styled chars | `termenv.SetBackgroundColor()` |
| Empty string padding `""` | Breaks `ansi.Cut()` overlays | `strings.Repeat(" ", width)` |
| `switch msg := msg.(type)` | Causes shadowing | Use `typedMsg` + reassign to `msg` |

### View Height Management
```go
// In app.go handleWindowSize:
contentHeight := max(1, a.height - 2)  // Subtract header + status bar
a.prList.SetSize(a.width, contentHeight)
```
- Always subtract chrome (header/footer) from content height
- Views should render to their allocated height, not full terminal

### Glamour Markdown Rendering (CRITICAL)
```go
// DON'T use WithAutoStyle() - it does slow terminal detection (~5 seconds!)
renderer, _ := glamour.NewTermRenderer(
    glamour.WithAutoStyle(),  // SLOW - avoid this
    glamour.WithWordWrap(80),
)

// DO use WithStandardStyle() with explicit style name
renderer, _ := glamour.NewTermRenderer(
    glamour.WithStandardStyle("dracula"),  // FAST - use this
    glamour.WithWordWrap(100),
)
```
- `WithAutoStyle()` queries the terminal for background color which takes ~5 seconds
- `WithStandardStyle("dark")` or `WithStandardStyle("dracula")` is instant
- Cache the renderer globally - creating it is fast but reusing is better
- Available styles: "dark", "light", "dracula", "tokyo-night", "pink", "ascii"
