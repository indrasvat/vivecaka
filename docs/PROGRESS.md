# vivecaka — Progress Tracker

> Updated after each task completion. See `docs/PRD.md` for full task definitions.

## Current Phase

**Phase 13: Integration & Polish — COMPLETE**

## Task Status

### Phase 0: Project Bootstrap

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T0.1 | Initialize Go module & project structure | ✅ Done | All packages compile |
| T0.2 | Create Makefile | ✅ Done | 17 targets |
| T0.3 | Create CLAUDE.md | ✅ Done | |
| T0.4 | Create PROGRESS.md | ✅ Done | This file |
| T0.5 | Setup lefthook | ✅ Done | |
| T0.6 | Setup golangci-lint | ✅ Done | v2 format |
| T0.7 | Setup GoReleaser | ✅ Done | v2 format, clean check |
| T0.8 | LICENSE + README | ✅ Done | |

### Phase 1: Domain Layer

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T1.1 | Define domain entities | ✅ Done | 100% coverage |
| T1.2 | Define domain interfaces | ✅ Done | PRReader, PRReviewer, PRWriter |
| T1.3 | Define domain errors | ✅ Done | errors.Is/As tests pass |

### Phase 2: Plugin Infrastructure

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T2.1 | Plugin interfaces | ✅ Done | Plugin, ViewPlugin, KeyPlugin |
| T2.2 | Plugin registry | ✅ Done | 95.2% coverage, race-safe |
| T2.3 | Hook system | ✅ Done | On/Emit, error chain, race-safe |

### Phase 3: Configuration

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T3.1 | Config struct and TOML loading | ✅ Done | 94% coverage, validation |
| T3.2 | XDG path helpers | ✅ Done | Env var override tests |
| T3.3 | Default config generation | ✅ Done | Auto-create on first run |

### Phase 4: GH CLI Adapter

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T4.1 | GH CLI plugin wrapper | ✅ Done | Auth detection, go-gh v2 |
| T4.2 | PRReader (GH CLI) | ✅ Done | ListPRs, GetPR, GetDiff, GetChecks, GetComments |
| T4.3 | PRReviewer (GH CLI) | ✅ Done | SubmitReview, AddComment, ResolveThread |
| T4.4 | PRWriter (GH CLI) | ✅ Done | Checkout, Merge, UpdateLabels |
| T4.5 | Diff parser | ✅ Done | Unified diff, renames, line numbers |

### Phase 5: Use Cases

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T5.1 | ListPRs use case | ✅ Done | 100% coverage |
| T5.2 | GetPRDetail use case | ✅ Done | errgroup parallel fetch |
| T5.3 | ReviewPR use case | ✅ Done | Validation + delegation |
| T5.4 | CheckoutPR use case | ✅ Done | |
| T5.5 | Comment use cases | ✅ Done | AddComment + ResolveThread |

### Phase 6: TUI Foundation

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T6.1 | Theme system | ✅ Done | 5 themes, Catppuccin Mocha default |
| T6.2 | Keymap system | ✅ Done | Global + per-view, help.KeyMap |
| T6.3 | Root app model | ✅ Done | View routing, 95% coverage |
| T6.4 | Header component | ✅ Done | Repo + count + filter + refresh |
| T6.5 | Status bar | ✅ Done | Context-aware key hints |
| T6.6 | Toast notifications | ✅ Done | Stacked, auto-dismiss, 4 levels |

### Phase 7: PR List View

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T7.1 | PR list table | ✅ Done | Columns: #, Title, Author, CI, Review, Age |
| T7.2 | Search/filter | ✅ Done | Fuzzy search by title/author |
| T7.3 | Sort cycling | ✅ Done | updated/created/number/title/author |
| T7.4 | Branch highlight | ✅ Done | Current branch indicator (◉) |
| T7.5 | Draft dimming | ✅ Done | [DRAFT] prefix, muted style |
| T7.6 | Clipboard/browser | ✅ Done | y=copy URL, o=open browser |

### Phase 8: PR Detail View

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T8.1 | Detail model | ✅ Done | 4-pane tabbed layout |
| T8.2 | Info pane | ✅ Done | Author, branch, state, labels, body |
| T8.3 | Files pane | ✅ Done | File list with +/- counts |
| T8.4 | Checks pane | ✅ Done | CI status with duration |
| T8.5 | Comments pane | ✅ Done | Threaded, resolved badges |

### Phase 9: Diff Viewer

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T9.1 | Diff model | ✅ Done | File tabs, scroll, search |
| T9.2 | Line rendering | ✅ Done | Line numbers, add/delete coloring |
| T9.3 | File navigation | ✅ Done | Tab/Shift-Tab between files |
| T9.4 | Search in diff | ✅ Done | / to search, Esc to clear |

### Phase 10: Review Forms

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T10.1 | Review model | ✅ Done | Action cycle, body edit, submit |
| T10.2 | Action selector | ✅ Done | Comment/Approve/RequestChanges |
| T10.3 | Body editor | ✅ Done | Inline text editing with cursor |

### Phase 11: Repo Switching

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T11.1 | Repo switcher overlay | ✅ Done | Fuzzy search, favorites, current highlight |
| T11.2 | Unified PR Inbox (S4) | ✅ Done | 4 tabs, priority sort, multi-repo |
| T11.3 | Repo auto-detection | ✅ Done | SSH + HTTPS GitHub URL parsing |

### Phase 12: Help System

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T12.1 | Help overlay | ✅ Done | Context-aware per-view bindings |
| T12.2 | First-launch tutorial | ✅ Done | 5-step walkthrough, XDG flag |
| T12.3 | Status bar hints | ✅ Done | Per-view, truncation-safe |

### Phase 13: Integration & Polish

| Task | Description | Status | Notes |
|------|-------------|--------|-------|
| T13.1 | App tests + coverage | ✅ Done | 95% tui, 90.8% views |
| T13.2 | Progress tracking | ✅ Done | This file |

## Coverage Summary

| Package | Coverage |
|---------|----------|
| domain | 100% |
| usecase | 100% |
| plugin | 95.2% |
| tui | 95.0% |
| config | 94.0% |
| views | 90.8% |
| ghcli | 25.3% (parser only; reader/reviewer/writer need live gh) |

## Blocked

_None_

## Notes

- Go 1.25.7 on darwin/arm64
- BubbleTea v1.3.10, LipGloss v1.1.1, Bubbles v0.21.1
- go-gh v2.13.0
- All 14 phases (0-13) complete
- 0 lint issues, all tests pass with -race
