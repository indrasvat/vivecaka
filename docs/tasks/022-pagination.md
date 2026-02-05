# Task 022: Pagination and Infinite Scroll

## Status: DONE

## Problem

All PRs are loaded in one shot with no pagination. Repos with many PRs (100+) will be slow and may hit API limits. The PRD specifies paginated loading with infinite scroll.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.1 (PR List View) — pagination, infinite scroll
- Look for page size, loading indicator specs

## Files to Modify

- `internal/domain/repo.go` — add `Page`, `PerPage` to `ListOpts` if not present
- `internal/adapter/ghcli/reader.go` — add `--limit` and pagination params to `gh pr list`
- `internal/tui/views/prlist.go` — detect when cursor nears bottom, request next page
- `internal/tui/app.go` — handle `LoadMorePRsMsg`, append to existing list
- `internal/tui/views/messages.go` — add `LoadMorePRsMsg`, `MorePRsLoadedMsg`

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Add pagination fields to `ListOpts`: `Page int`, `PerPage int` (default 30)
4. In `ghcli/reader.go`, pass `--limit` to gh CLI and add cursor/page tracking
5. In `prlist.go`:
   - Track `m.hasMore bool` and `m.loading bool`
   - When cursor is within 5 items of the bottom and `hasMore && !loading`, emit `LoadMorePRsMsg`
   - Show "Loading more..." indicator at bottom of list
6. In `app.go`:
   - Handle `LoadMorePRsMsg` → trigger `loadPRsCmd` with next page
   - Handle `MorePRsLoadedMsg` → append to existing list (don't replace)
7. Handle edge case: no more pages (empty result) → set `hasMore = false`
8. Add tests
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRList
go test -race -v ./internal/adapter/ghcli/ -run TestReader
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with many open PRs (or use a large public repo)
2. Screenshots initial load (first page)
3. Scrolls to bottom with repeated j presses
4. Screenshots showing "Loading more..." indicator
5. Screenshots after more PRs load — verify list grew
6. Scrolls to actual bottom — verify no more loading when all PRs shown

## Commit

```
feat(tui): add paginated PR loading with infinite scroll

PRs load in pages (default 30). When cursor nears bottom,
automatically fetches next page and appends. "Loading more..."
indicator shown during fetch. Stops when no more pages.
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
