# Task 021: Visual Selection Mode

## Status: DONE

## Problem

The PRD specifies `v` key to enter visual selection mode in the PR list for multi-select and batch operations (e.g., batch checkout, batch review). This is entirely missing.

## PRD Reference

Read `docs/PRD.md` sections:
- S5 (Progressive Disclosure / Visual Selection Mode)
- Look for multi-select specs, batch operation specs

## Files to Modify

- `internal/tui/views/prlist.go` — add `v` key to enter selection mode, Space to toggle selection, visual indicators
- `internal/tui/views/messages.go` — add batch operation messages
- `internal/tui/app.go` — handle batch operations
- `internal/tui/views/prlist_test.go` — test selection mode

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Add `m.selectionMode bool`, `m.selected map[int]bool` to PR list model
4. `v` key → toggle selection mode on/off
5. In selection mode:
   - Space → toggle current PR selected/unselected
   - Visual indicator (e.g., checkbox `[x]` or highlight) on selected PRs
   - Status bar shows "N selected" count
   - `a` → select all visible
   - Esc → exit selection mode, clear selections
6. Batch operations on selected PRs:
   - Enter → batch open (show list of selected PRs)
   - `y` → copy all selected PR URLs
   - `o` → open all selected in browser
7. Add tests
8. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRList
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` with multiple PRs
2. Presses `v` to enter selection mode — screenshots showing mode indicator
3. Moves cursor with j/k, presses Space to select 2-3 PRs — screenshots showing selection markers
4. Verifies status bar shows "3 selected"
5. Presses Esc — screenshots showing selection cleared

## Commit

```
feat(tui): add visual selection mode for batch operations

v key enters selection mode in PR list. Space toggles selection,
a selects all. Status bar shows selection count. Batch operations:
y copies URLs, o opens in browser.
```

## Session Protocol

1. Read `CLAUDE.md`
2. Read this task file
3. Read relevant PRD section
4. Execute
5. Verify (functional + visual)
6. Update `docs/PROGRESS.md` — mark this task done
7. `git add` changed files + `git commit`
8. Move to next task or end session
