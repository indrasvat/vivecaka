# Task 019: Diff Two-Pane Layout (File Tree + Content)

## Status: DONE

## Problem

The PRD specifies a two-pane split layout for the diff viewer: a vertical file tree on the left and diff content on the right. Currently, files are shown as a horizontal tab bar at the top with no file tree.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.3 (Diff Viewer) — two-pane layout spec, file tree
- Look for pane sizing, resize, collapse specs

## Files to Modify

- `internal/tui/views/diffview.go` — add file tree pane on left, restructure rendering to two-pane layout
- `internal/tui/views/diffview_test.go` — test two-pane rendering, pane focus switching

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read current `diffview.go` layout rendering
4. **File tree pane**:
   - Left pane (configurable width, ~25-30 chars)
   - Lists all changed files with: status icon (+/-/~), filename, +/- line counts
   - Current file highlighted
   - Tree-style if files share directories (optional: flat list first, tree later)
   - Navigate with j/k when tree pane is focused
   - Enter on file → select it, show in content pane
5. **Pane focus**:
   - Tab switches focus between file tree and content pane
   - Active pane has distinct border color
   - When file tree focused: j/k navigate files, Enter selects
   - When content focused: j/k scroll diff, all other diff keys work
6. **Layout**:
   - Use lipgloss `JoinHorizontal` with a vertical divider
   - File tree width: min 20, max 40, default 25% of terminal width
   - Content takes remaining width
7. Add tests
8. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestDiffView
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with PRs touching multiple files
2. Opens PR, opens diff
3. Screenshots showing two-pane layout: file tree on left, diff content on right
4. Presses Tab to switch to file tree pane — screenshots showing focus change (border color)
5. Navigates files with j/k in tree — screenshots
6. Selects different file with Enter — screenshots showing content updates
7. Presses Tab back to content — screenshots

## Commit

```
feat(tui): add two-pane diff layout with file tree

Diff viewer now shows file tree on left with status icons and line
counts. Tab switches focus between panes. File selection in tree
updates content pane. Active pane has distinct border color.
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
