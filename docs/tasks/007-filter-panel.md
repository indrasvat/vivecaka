# Task 007: Filter Panel View

## Status: DONE

## Depends On: 006 (quick filters establish username detection)

## Problem

The PRD specifies a full filter panel overlay opened with `f` from the PR list. This overlay should have checkboxes/toggles for: status (open/closed/merged), author, label, CI state, review state, and draft toggle. No filter panel view exists at all — there's no `filter.go` in views, no `f` key binding.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.6 (Filter Panel) — full filter overlay spec
- Section 7.1 (PR List View) — `f` key binding
- Look for `ListOpts` usage and fields

## Files to Create/Modify

- `internal/tui/views/filter.go` (CREATE) — `FilterModel` with checkbox/toggle fields
- `internal/tui/views/filter_test.go` (CREATE) — test filter state changes and message emission
- `internal/tui/views/prlist.go` — add `f` key to emit `OpenFilterMsg`
- `internal/tui/app.go` — handle filter overlay open/close, apply filter opts to `loadPRsCmd`
- `internal/tui/core/viewstate.go` — add `ViewFilter` state if rendering as overlay
- `internal/domain/repo.go` — verify `ListOpts` has all needed fields (CI, Review, Draft)

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read `internal/domain/repo.go` — check `ListOpts` struct fields
4. Create `internal/tui/views/filter.go`:
   - `FilterModel` struct with: state (open/closed/merged toggles), author text input, label text input, CI filter (any/pass/fail/pending), review filter (any/approved/changes-requested/pending), draft toggle (any/yes/no)
   - Render as a bordered overlay with `huh`-style form controls or manual checkboxes
   - Navigation: j/k between fields, Space/Enter to toggle, Tab between sections
   - `Apply` action (Enter on Apply button) → emit `ApplyFilterMsg{Opts: ListOpts}`
   - `Clear` action → reset all filters
   - Esc → close without applying
5. In `prlist.go`, add `f` key → emit `OpenFilterMsg`
6. In `app.go`:
   - Handle `OpenFilterMsg` → switch to filter view (overlay)
   - Handle `ApplyFilterMsg` → store opts, reload PRs with new opts
   - Handle filter close → return to PR list
7. Add tests
8. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestFilter
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with open PRs
2. Presses `f` — screenshots showing filter panel overlay
3. Navigates filter options with j/k — screenshots
4. Toggles a filter (e.g., "CI: failing") — screenshots
5. Applies filter — screenshots showing filtered PR list
6. Presses `f` again, clears filters — screenshots showing all PRs restored

## Commit

```
feat(tui): add filter panel overlay with f key

New FilterModel view with toggles for status, CI, review state,
draft, author, and label. Applied filters reload the PR list
with updated ListOpts.
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
