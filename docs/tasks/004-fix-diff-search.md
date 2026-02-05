# Task 004: Fix Diff Search

## Status: TODO

## Problem

The diff viewer accepts search input via `/` key — you can type a query and it's displayed in a search bar — but the search has **zero effect** on the view. No matches are highlighted, there's no navigation between matches with `n`/`N`, and the match count is not shown. The search is completely non-functional.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.3 (Diff Viewer) — `/` search, `n`/`N` next/prev match, highlight matches
- Look for search behavior specs

## Files to Modify

- `internal/tui/views/diffview.go` — implement actual search logic: find matches, highlight them, navigate with n/N, show match count
- `internal/tui/views/diffview_test.go` — test search finds matches, n/N cycles, highlights apply

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read `internal/tui/views/diffview.go` fully — understand current search state
4. Add search result tracking:
   - `m.searchMatches []searchMatch` — list of (fileIndex, lineIndex, colStart, colEnd)
   - `m.currentMatch int` — index into matches
5. When search query changes (each keystroke or on Enter), scan all visible diff lines for matches
6. In line rendering, highlight matched text with a distinct style (e.g., reverse video or bright background)
7. Show match count in search bar: `"/ query [3/17]"`
8. Add `n` key → jump to next match (scroll if needed, switch files if needed)
9. Add `N` key → jump to previous match
10. On Esc, clear search and highlights
11. Add tests verifying match finding, navigation, and that View() output contains highlighted text
12. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestDiffView
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with open PRs
2. Opens a PR, then opens diff view
3. Presses `/` and types a search term that exists in the diff
4. Screenshots showing: search bar with query, match count, highlighted matches
5. Presses `n` to navigate to next match — screenshots showing cursor moved
6. Presses `N` to go back — screenshots
7. Presses Esc to clear — screenshots showing highlights removed

## Commit

```
fix(tui): implement functional search in diff viewer

Search via / was accepting input but never finding or highlighting
matches. Now scans diff lines, highlights matches, shows count
in search bar, and supports n/N navigation between matches.
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
