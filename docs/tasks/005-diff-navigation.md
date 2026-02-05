# Task 005: Diff Navigation Enhancements

## Status: DONE

## Problem

The PRD specifies several diff navigation features that are missing:
- `[`/`]` to jump between hunks
- `{`/`}` to jump between files
- `gg`/`G` to go to top/bottom of diff
- `za` to collapse/expand file sections

Currently only j/k scroll and Tab/Shift-Tab file switch exist.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.3 (Diff Viewer) — all navigation key bindings
- Look for hunk, file jump, collapse specs

## Files to Modify

- `internal/tui/views/diffview.go` — add key handlers for `[`, `]`, `{`, `}`, `g`, `G`, `z`+`a`
- `internal/tui/core/keymap.go` — add diff-specific bindings if not already present
- `internal/tui/views/diffview_test.go` — test each navigation action

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read `internal/tui/views/diffview.go` fully
4. **Hunk navigation** (`[`/`]`):
   - Identify hunk boundaries (lines starting with `@@`)
   - `]` → scroll to next `@@` line; `[` → scroll to previous
5. **File jump** (`{`/`}`):
   - `}` → switch to next file (like Tab but also scrolls to top of that file)
   - `{` → switch to previous file
6. **Top/bottom** (`gg`/`G`):
   - `G` → scroll to last line of current file
   - `gg` → scroll to first line (needs two-key sequence tracking: if last key was `g` and current is `g`, go to top)
7. **Collapse/expand** (`za`):
   - Track collapsed state per file: `m.collapsed map[int]bool`
   - `za` toggles current file collapsed/expanded
   - When collapsed, show only file header line with "+/- counts"
   - Two-key sequence: `z` then `a`
8. Add tests for each navigation action
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestDiffView
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka`, opens a PR with multi-file diff
2. Screenshots initial diff state
3. Presses `]` to jump to next hunk — screenshots showing scroll position changed
4. Presses `}` to jump to next file — screenshots
5. Presses `G` to go to bottom — screenshots
6. Types `gg` to go to top — screenshots
7. Types `za` to collapse current file — screenshots showing collapsed state
8. Types `za` again to expand — screenshots

## Commit

```
feat(tui): add diff navigation — hunk jump, file jump, gg/G, collapse

Added [/] for hunk navigation, {/} for file jumping, gg/G for
top/bottom, and za for collapse/expand. Two-key sequences tracked
via lastKey state.
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
