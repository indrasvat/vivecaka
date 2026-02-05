# Task 025: Persistence (Per-Repo Filter Memory, Unread Indicators)

## Status: TODO

## Depends On: 001 (sort working), 007 (filter panel), 023 (caching infra)

## Problem

Two persistence features from the PRD are missing:
1. **Per-repo filter/sort memory**: Remember last-used filter and sort settings per repo, persisted in XDG data directory
2. **Unread indicators**: `LastViewedAt` field exists in domain but is never read/written/used. No dot badges on PRs with activity since last viewed.

## PRD Reference

Read `docs/PRD.md` sections:
- S1 (Smart Context Awareness) — per-repo filter memory
- S4 (Inbox) — unread indicators
- Look for XDG data dir specs, persistence format

## Files to Create/Modify

- `internal/cache/state.go` (CREATE) — per-repo state persistence (last sort, last filter, last viewed timestamps)
- `internal/cache/state_test.go` (CREATE) — test state save/load
- `internal/tui/app.go` — load/save state on repo switch, track last-viewed per PR
- `internal/tui/views/prlist.go` — apply saved sort/filter on load, show unread dot
- `internal/tui/views/inbox.go` — show unread dot on PRs with activity since last viewed

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Create `internal/cache/state.go`:
   - `RepoState` struct: `LastSort string`, `LastSortAsc bool`, `LastFilter ListOpts`, `LastViewedPRs map[int]time.Time`
   - `SaveRepoState(repo, state)` → JSON file in `XDG_DATA_HOME/vivecaka/state/{owner}_{name}.json`
   - `LoadRepoState(repo) (RepoState, error)`
4. **Per-repo filter/sort memory**:
   - On initial PR load, check for saved state → apply sort/filter
   - On sort/filter change, save state
   - On repo switch, save current state, load new repo's state
5. **Unread indicators**:
   - When user opens a PR detail, save `LastViewedPRs[number] = now`
   - In PR list rendering, if PR's `UpdatedAt > LastViewedPRs[number]`, show unread dot (`●`)
   - In inbox, same logic per-PR across repos
6. Add tests
7. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/cache/
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka`, changes sort to "by author"
2. Quits and relaunches — verify sort is still "by author" (screenshots)
3. Opens a PR, goes back — verify that PR no longer has unread dot (if applicable)
4. Verify state file exists in XDG data dir

## Commit

```
feat: add per-repo filter/sort memory and unread indicators

Sort/filter settings persisted per repo in XDG data dir. Restored
on next launch. PRs with activity since last viewed show unread dot
in PR list and inbox.
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
