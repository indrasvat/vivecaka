# Task 013: Repo Switcher Wiring

## Status: TODO

## Problem

Ctrl+R opens the repo switcher overlay, but the repo list is **never populated**. `SetRepos()` is never called. The switcher renders an empty list. Favorites from config are never read or passed to the switcher. Last-used repo is not persisted.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.7 (Repo Switching) — favorites, fuzzy search, current repo highlight
- S1 (Smart Context Awareness) — repo persistence
- Config section — `[repos] favorites` field

## Files to Modify

- `internal/tui/app.go` — on init, read favorites from config and call `repoSwitcher.SetRepos()`; on repo switch, persist last-used
- `internal/config/config.go` — verify `Favorites []string` field exists and is parsed correctly
- `internal/tui/views/reposwitcher.go` — verify `SetRepos()` works correctly, verify current repo highlighting
- `internal/adapter/ghcli/repodetect.go` — may need `ListUserRepos(ctx) ([]RepoRef, error)` to auto-discover repos if favorites empty

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read config loading to understand how favorites get into `cfg.Repos.Favorites`
4. In `app.go` `New()` or `Init()`:
   - Parse favorites strings into `[]domain.RepoRef` (split on `/`)
   - If the detected repo isn't in favorites, prepend it
   - Call `a.repoSwitcher.SetRepos(repos)`
   - Set current repo highlight: `a.repoSwitcher.SetCurrentRepo(a.repo)`
5. In `handleSwitchRepo()`, after switching:
   - Update `a.repoSwitcher.SetCurrentRepo(a.repo)`
6. Optional: if favorites is empty, call `gh repo list --json nameWithOwner -L 20` to auto-discover user repos
7. Test with a config file that has favorites set
8. Add tests
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/... -run TestRepoSwitch
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Set up a config file with 3-4 favorite repos in `~/.config/vivecaka/config.toml`
2. Launches `bin/vivecaka`
3. Presses Ctrl+R — screenshots showing repo switcher with favorites listed
4. Types a search query to fuzzy-filter — screenshots
5. Selects a different repo — screenshots showing PR list reloads for new repo
6. Verifies header shows new repo name

## Commit

```
feat(tui): wire repo switcher to load favorites from config

Ctrl+R now shows actual favorite repos from config.toml. Current
repo is highlighted. Selecting a repo switches context and reloads
PRs. Auto-discovers repos if favorites list is empty.
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
