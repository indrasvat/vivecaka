# Task 014: Inbox Wiring (Multi-Repo PR Aggregation)

## Status: TODO

## Depends On: 006 (username detection), 013 (repo switcher / favorites loading)

## Problem

The Inbox view UI is fully built (filtering, sorting, multi-repo column rendering) but has no data pipeline:
- No `I` key binding to open inbox
- No use case to fetch PRs from multiple repos
- `OpenInboxPRMsg` not handled in app.go
- `PrioritySort` function exists but is never called (dead code)
- "Assigned" tab uses author as proxy (broken)

## PRD Reference

Read `docs/PRD.md` sections:
- S4 (Unified PR Inbox) — full inbox spec
- Look for: tabs, priority sorting, unread indicators, multi-repo aggregation

## Files to Create/Modify

- `internal/usecase/inbox.go` (CREATE) — `GetInboxPRs` use case that fetches PRs from multiple repos via errgroup
- `internal/usecase/inbox_test.go` (CREATE) — test parallel fetching, error handling
- `internal/tui/commands.go` — add `loadInboxCmd()` wrapper
- `internal/tui/views/messages.go` — add `InboxLoadedMsg{PRs []InboxPR; Err error}`
- `internal/tui/app.go` — add `I` key binding, handle inbox loading/opening, handle `OpenInboxPRMsg`
- `internal/tui/views/inbox.go` — fix "Assigned" tab filter to use actual assignees or review requested state
- `internal/tui/core/keymap.go` — add `I` to global keys

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read `internal/tui/views/inbox.go` fully — understand existing data model
4. Create `internal/usecase/inbox.go`:
   ```go
   type GetInboxPRs struct { reader domain.PRReader }
   func (uc *GetInboxPRs) Execute(ctx, repos []domain.RepoRef, opts) ([]views.InboxPR, error)
   ```
   - Use errgroup to fetch PRs from each repo in parallel
   - Convert each `domain.PR` + `domain.RepoRef` into `views.InboxPR`
   - Return aggregated list
   - Handle partial failures (some repos may fail) — return what succeeded + aggregate errors
5. Add `loadInboxCmd()` to `commands.go`
6. In `app.go`:
   - Bind `I` key → switch to `ViewInbox`, trigger `loadInboxCmd()`
   - Handle `InboxLoadedMsg` → call `a.inbox.SetPRs(msg.PRs)`, set username for filters
   - Handle `OpenInboxPRMsg` → set `a.repo` to the inbox PR's repo, trigger `loadPRDetailCmd`, switch to detail view
7. Fix "Assigned" tab in `inbox.go`: use review state or a proper assignees field
8. Wire `PrioritySort` call after inbox data loads
9. Add tests
10. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/usecase/ -run TestInbox
go test -race -v ./internal/tui/ -run TestAppInbox
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Set up config with 2-3 favorite repos that have open PRs
2. Launches `bin/vivecaka`
3. Presses `I` — screenshots showing inbox loading
4. Screenshots showing inbox with PRs from multiple repos, repo column visible
5. Tabs between filter tabs (All, Assigned, Review Requested, My PRs) — screenshots
6. Selects a PR from a different repo — screenshots showing detail view loads for that repo
7. Presses Esc to return to inbox

## Commit

```
feat(tui): wire inbox with multi-repo PR aggregation

Added GetInboxPRs use case that fetches PRs from all favorite repos
in parallel. I key opens inbox, OpenInboxPRMsg switches repo context.
PrioritySort applied on load. Fixed Assigned tab filter.
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
