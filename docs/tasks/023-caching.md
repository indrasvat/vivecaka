# Task 023: Caching and Instant Startup

## Status: TODO

## Problem

The PRD specifies caching the last-fetched PR list as JSON in the XDG cache directory for instant startup. The app should show cached data immediately, then fetch fresh data in the background and update. `CacheDir()` and `cache_ttl` config exist but no caching code.

## PRD Reference

Read `docs/PRD.md` sections:
- S6 (Instant Startup / Caching) — cache spec
- Config section — `cache_ttl` field
- Look for optimistic UI, stale-while-revalidate specs

## Files to Create/Modify

- `internal/cache/cache.go` (CREATE) — JSON file cache: Save, Load, IsStale
- `internal/cache/cache_test.go` (CREATE) — test save/load/staleness
- `internal/tui/app.go` — on init: load cache → show immediately → fetch fresh → update
- `internal/tui/commands.go` — add `loadCachedPRsCmd()`, save cache after fresh load
- `internal/config/config.go` — verify `cache_ttl` field

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Create `internal/cache/cache.go`:
   - `CachePath(repo RepoRef) string` — returns `XDG_CACHE_HOME/vivecaka/repos/{owner}_{name}.json`
   - `Save(repo, prs []domain.PR) error` — marshal to JSON, write atomically
   - `Load(repo) ([]domain.PR, time.Time, error)` — read JSON, return data + last-modified time
   - `IsStale(repo, ttl time.Duration) bool` — check if cache is older than TTL
4. In `app.go` Init():
   - Try loading cached PRs first → if found, immediately populate PR list
   - Then trigger fresh fetch in background
   - When fresh data arrives, update PR list and save to cache
5. Visual indicator: show "cached" or "stale" badge in header when showing cached data, remove after fresh load
6. Respect `cache_ttl` config — if cache is within TTL, don't fetch (unless manual refresh)
7. Add tests
8. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/cache/
go test -race -v ./internal/tui/ -run TestCache
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` first time — PR list loads from API
2. Quit and relaunch immediately — screenshots showing PRs appear instantly (from cache)
3. Verify "cached" indicator shows briefly, then disappears after fresh fetch
4. Check cache file exists in XDG cache dir

## Commit

```
feat: add PR list caching for instant startup

Cached PR data loaded from XDG cache dir on startup for instant
display. Fresh data fetched in background and swapped in. Cache
respects TTL from config. Stale indicator shown while cached.
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
