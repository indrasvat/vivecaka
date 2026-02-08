# Task 028: PR Detail Tabs Layout Redesign

## Status: DONE

## Implementation Notes (Feb 2026)

- Implemented 4-tab layout: Description, Checks, Files, Comments (no Commits tab - not in original PRD)
- Tab bar with counts: `Description  Checks (3)  Files (8)  Comments (1)`
- Active tab styled with Primary background color (orange in Catppuccin Mocha)
- Number keys 1-4 for direct tab access
- Tab/Shift-Tab for cycling tabs
- j/k for scrolling, g/G for top/bottom
- Scroll indicator showing "↓ N more" when content is scrollable
- All existing key bindings preserved (d, c, r, o, za, space, x, X)
- Renamed `ciIcon` to `detailCIIcon` to avoid collision with prlist.go
- Updated tests to use new `tab` field and `Tab*` constants

## Problem

The current PR detail view tries to show all content (Info, Checks, Description, Files, Comments) simultaneously in a fixed layout. This causes:
- Description content gets truncated for long PRs
- Files pane gets cut off when there are many files
- No way to scroll individual sections without truncation
- Poor use of screen real estate

## Solution

Implement a horizontal tab-based layout similar to GitHub's PR view:
- Tab bar with: [Description] [Commits] [Checks] [Files] [Comments]
- Selected tab shows count (e.g., "Files (56)")
- Selected tab has visual highlight (theme accent color with glow/underline)
- Content area is fully scrollable per tab
- Tab switching via Tab/Shift-Tab or number keys 1-5

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.2 (PR Detail View) — view structure
- Section 7.2.4 (Checks pane) — check list rendering
- Section 7.2.5 (Comments pane) — comment threads
- Section 5.2 (Key bindings) — Tab/Shift-Tab navigation

## Layout Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ #1 Build Yamforge YAML Studio design plan          indrasvat → main    Open │
├─────────────────────────────────────────────────────────────────────────────┤
│  [Description]  Commits(11)  Checks(4)  Files(56)  Comments(1)              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  (Full scrollable content for the selected tab)                             │
│                                                                             │
│                                                                    ↓ more   │
└─────────────────────────────────────────────────────────────────────────────┘
 Tab/1-5 switch  j/k scroll  d diff  c checkout  r review  Esc back  ? help
```

## Key Bindings

| Key | Action |
|-----|--------|
| Tab | Next tab |
| Shift-Tab | Previous tab |
| 1 | Description tab |
| 2 | Commits tab |
| 3 | Checks tab |
| 4 | Files tab |
| 5 | Comments tab |
| j/k | Scroll content |
| g/G | Top/bottom of content |
| Enter | Context action (open file, expand check, etc.) |
| d | Open diff view |
| c | Checkout PR |
| r | Start review |
| o | Open in browser |

## Files to Modify

- `internal/tui/views/prdetail.go` — Complete rewrite of layout:
  - Add `activeTab` field (enum: TabDescription, TabCommits, TabChecks, TabFiles, TabComments)
  - Add `scrollY` per-tab or global
  - Rewrite `View()` to render tab bar + content area
  - Add tab switching in `handleKey()`
  - Keep all existing key bindings (d, c, r, o, etc.)
- `internal/tui/core/styles.go` — Add tab styles (active, inactive, counts)
- `internal/tui/views/prdetail_test.go` — Update tests for new tab navigation

## Implementation Steps

1. Define tab enum and add to PRDetailModel
2. Create tab bar render function with theme-matched highlighting
3. Create separate render functions for each tab's content:
   - `renderDescriptionTab()` — PR body markdown, branch info, labels
   - `renderCommitsTab()` — Commit list (use gh pr view --json commits)
   - `renderChecksTab()` — CI status list with pass/fail/pending icons
   - `renderFilesTab()` — File list with +/- counts
   - `renderCommentsTab()` — Comment threads with collapse/expand
4. Implement scrolling per content area
5. Wire tab switching keys
6. Add "more" indicator when content is scrollable
7. Test all key bindings work in new layout

## Tab Bar Styling

Active tab should have:
- Theme's accent/primary color (orangish for Catppuccin Mocha)
- Bold text
- Subtle underline or glow effect
- Brackets or highlight: `[Description]` vs `Description`

Inactive tabs:
- Muted color
- Show counts in parentheses: `Files(56)`

## Execution Steps

1. Read `CLAUDE.md` (especially lipgloss learnings)
2. Read this task file
3. Read relevant PRD sections
4. Backup current prdetail.go
5. Implement tab enum and model changes
6. Implement tab bar rendering
7. Implement each tab content renderer
8. Implement scroll handling
9. Test keyboard navigation thoroughly
10. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRDetail
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with open PRs
2. Opens a PR with Enter
3. Screenshots default tab (Description)
4. Press Tab to switch tabs, screenshot each
5. Press j/k to scroll, verify scrolling works
6. Press 1-5 to jump to specific tabs
7. Verify all existing keys still work (d, c, r, o)

## Commit

```
feat(tui): redesign PR detail with horizontal tabs layout

- Add tab bar: Description, Commits, Checks, Files, Comments
- Tab/Shift-Tab and 1-5 for navigation
- Each tab has full scrollable content area
- Active tab highlighted with theme accent color
- Preserves all existing key bindings
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
