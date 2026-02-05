# Task 015: Auto-Refresh with Polling

## Status: TODO

## Problem

The PRD specifies background polling every 30s (configurable) with visual indicators, toast notifications for new activity, pause/resume with `p`, and a countdown timer in the header. None of this exists. `refresh_interval` is in config but completely unused. Only manual refresh via `R` key works.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.1 (PR List View) — auto-refresh, pause, countdown
- F10 (Auto-Refresh) — full spec
- Header component — countdown timer
- Look for notification/badge specs on new activity

## Files to Modify

- `internal/tui/app.go` — add tick-based polling loop, `p` key to pause/resume, countdown state
- `internal/tui/components/header.go` — add countdown display (e.g., "↻ 25s")
- `internal/tui/views/prlist.go` — handle refresh: diff new PRs vs old, detect new/updated PRs
- `internal/config/config.go` — verify `refresh_interval` default and validation

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read config to find `refresh_interval` field and its default value
4. In `app.go`:
   - Add `a.refreshPaused bool`, `a.refreshCountdown int` fields
   - After initial PR load, start a `tea.Tick` command that fires every second
   - On each tick: decrement countdown, update header display
   - When countdown reaches 0: trigger `loadPRsCmd()` again, reset countdown
   - `p` key toggles `a.refreshPaused` — when paused, stop decrementing
   - `R` key (manual refresh) resets countdown and triggers immediate reload
   - When `PRsLoadedMsg` arrives during auto-refresh: compare with previous PR list, show toast for new PRs (e.g., "2 new PRs")
5. In `header.go`:
   - Add `SetRefreshCountdown(seconds int, paused bool)` method
   - Render countdown: `"↻ 25s"` or `"⏸ paused"` when paused
6. Add tests for tick/countdown logic
7. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/ -run TestRefresh
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with open PRs
2. Screenshots showing countdown in header (e.g., "↻ 28s")
3. Waits a few seconds — screenshots showing countdown decreasing
4. Presses `p` — screenshots showing "paused" indicator
5. Presses `p` again — screenshots showing countdown resumed
6. Presses `R` — screenshots showing immediate refresh (countdown resets)

## Commit

```
feat(tui): add auto-refresh polling with countdown and pause

PR list now auto-refreshes at configurable interval (default 30s).
Countdown displayed in header. p key pauses/resumes. R resets and
refreshes immediately. Toast shown when new PRs detected.
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
