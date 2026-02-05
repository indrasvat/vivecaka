# Task 006: Quick Filters (m/n keys)

## Status: DONE

## Problem

The PRD specifies first-class quick filters in the PR list:
- `m` — "My PRs" (filter to PRs authored by current user)
- `n` — "Needs My Review" (filter to PRs where review is requested from current user)

These are entirely missing. The current user's GitHub username also needs to be detected or configured.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.1 (PR List View) — quick filter specs
- S1 (Smart Context Awareness) — "My PRs" / "Needs My Review" quick filters
- Look for username detection specs

## Files to Modify

- `internal/adapter/ghcli/repodetect.go` or new `internal/adapter/ghcli/user.go` — add `DetectUser(ctx) (string, error)` using `gh api user --jq .login`
- `internal/tui/views/prlist.go` — add `m` and `n` key handlers, add `m.username string`, add filter functions
- `internal/tui/views/messages.go` — add `UserDetectedMsg{Username string; Err error}` if needed
- `internal/tui/app.go` — detect username on init, pass to views
- `internal/tui/views/prlist_test.go` — test quick filters

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Add username detection: `gh api user --jq .login`
4. Pass username to PR list view on init or via a `SetUsername(string)` method
5. Add `m` key handler: toggle filter to only show PRs where `pr.Author == username`
6. Add `n` key handler: toggle filter to only show PRs where review state is pending and user is not author
7. When a quick filter is active, show indicator in header or status bar (e.g., "My PRs" or "Needs Review")
8. Pressing the same key again clears the quick filter
9. Add tests
10. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRList
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with PRs from multiple authors
2. Screenshots default PR list (all PRs)
3. Presses `m` — screenshots showing only "My PRs" with filter indicator
4. Presses `m` again — screenshots showing filter cleared, all PRs back
5. Presses `n` — screenshots showing "Needs Review" filter
6. Verifies filter indicator text appears on screen

## Commit

```
feat(tui): add quick filters — m for My PRs, n for Needs Review

Detects GitHub username via gh api, filters PR list by author (m)
or pending review state (n). Toggle on/off with repeat press.
Filter indicator shown in status area.
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
