# Task 001: Fix PR List Sort

## Status: TODO

## Problem

The `s` key in the PR list cycles the sort field name visually (`cycleSort()` in `prlist.go`), but `applyFilter()` never actually sorts the PR slice. The data stays in whatever order GitHub returned it. The PRD also specifies a sort direction indicator (up/down triangle) on the active column, which is missing.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.1 (PR List View) — sort cycling spec
- Section 7.6 (Filter Panel) — sort fields: updated, created, number, title, author
- Look for any mention of sort direction indicator

## Files to Modify

- `internal/tui/views/prlist.go` — `applyFilter()` must actually sort; `cycleSort()` must toggle direction; `renderRow()` must show sort indicator on column header
- `internal/tui/views/prlist_test.go` — add tests that verify sort order changes

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read `internal/tui/views/prlist.go` fully
4. In `applyFilter()`, after filtering, add `sort.Slice` call based on `m.sortField` and a new `m.sortAsc` bool field
5. In `cycleSort()`, first press cycles field, second press on same field toggles direction
6. In the column header rendering, add a triangle indicator (e.g., `▲`/`▼`) next to the active sort column
7. Add/update tests in `prlist_test.go` to verify:
   - Sort by each field produces correct order
   - Direction toggle works
   - Sort indicator appears in header
8. Run `make ci` — must pass

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRList
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with multiple open PRs
2. Screenshots the default PR list
3. Presses `s` to cycle sort — screenshots after each press
4. Verifies sort indicator (triangle) appears in column header
5. Verifies PR order actually changes between sorts
6. Presses `s` again on same field to toggle direction
7. Screenshots showing reversed order

## Commit

```
fix(tui): wire PR list sort to actually reorder data

Sort field cycling (s key) was visual-only — the PR list stayed in API
order. Now applyFilter() sorts by the active field with direction
toggle. Column header shows ▲/▼ indicator on active sort.
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
