# Task 020: Diff Side-by-Side Mode

## Status: DONE

## Depends On: 019 (two-pane layout establishes split rendering patterns)

## Problem

The PRD specifies a `t` key to toggle between unified and side-by-side (split) diff view. Only unified mode exists currently.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.3 (Diff Viewer) — `t` toggle unified/split
- Look for side-by-side rendering specs

## Files to Modify

- `internal/tui/views/diffview.go` — add `m.splitMode bool`, `t` key toggle, side-by-side rendering
- `internal/tui/views/diffview_test.go` — test both modes render correctly

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Add `m.splitMode bool` field
4. Add `t` key handler to toggle `splitMode`
5. **Side-by-side rendering**:
   - Split content pane into two equal columns
   - Left column: old file lines (deletions and context)
   - Right column: new file lines (additions and context)
   - Aligned by hunk: matching context lines side by side, deletions on left, additions on right
   - Line numbers on each side
   - Vertical divider between columns
   - Synchronized scrolling (both sides scroll together)
6. **Mode indicator**: show "Unified" or "Split" in status bar or diff header
7. Ensure search and navigation (hunk jump, etc.) work in both modes
8. Add tests for both rendering modes
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestDiffView
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka`, opens a PR with meaningful changes
2. Screenshots in unified mode (default)
3. Presses `t` to toggle to split mode — screenshots showing side-by-side columns
4. Scrolls through split view — screenshots
5. Presses `t` again — screenshots showing back to unified
6. Verify mode indicator in status/header area

## Commit

```
feat(tui): add side-by-side diff mode with t toggle

Press t to toggle between unified and side-by-side diff views.
Split mode shows old/new files in aligned columns with synchronized
scrolling and line numbers on each side.
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
