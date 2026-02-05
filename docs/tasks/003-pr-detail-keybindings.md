# Task 003: PR Detail Key Bindings and CI Summary

## Status: DONE

## Problem

Several key bindings specified in the PRD for the PR detail view are missing:
- `d` key to open diff from anywhere in detail view (currently only Enter on Files pane works)
- `c` key for checkout from detail view (only works from PR list)
- `o` key to open check URL in browser (from Checks pane)
- CI summary line ("3/5 passing, 1 failing") is not rendered

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.2 (PR Detail View) — key bindings: d, c, o
- Section 7.2.4 (Checks pane) — CI summary line, o opens check URL
- Look for check detail linking specs

## Files to Modify

- `internal/tui/views/prdetail.go` — add `d`, `c`, `o` key handling in `handleKey()`; add CI summary rendering in `renderChecksPane()`
- `internal/tui/views/prdetail_test.go` — test new key bindings and CI summary output
- `internal/tui/views/messages.go` — may need `OpenDiffMsg` or similar if not already defined

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read `internal/tui/views/prdetail.go` fully
4. Add to `handleKey()`:
   - `d` → emit `OpenDiffMsg{Number: m.detail.Number}` (opens diff regardless of active pane)
   - `c` → emit `CheckoutPRMsg{Number: m.detail.Number}` (checkout from detail)
   - `o` → if on Checks pane and a check is selected, emit `OpenBrowserMsg{URL: check.URL}`; otherwise emit `OpenBrowserMsg{URL: m.detail.HTMLURL}`
5. Add CI summary line at top of `renderChecksPane()`:
   - Count passing/failing/pending checks
   - Render e.g., "3/5 passing, 1 failing, 1 pending" with color coding
6. Add tests verifying key → message emission and summary rendering
7. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRDetail
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with open PRs that have CI checks
2. Opens a PR with Enter
3. Screenshots the detail view (should show CI summary in Checks pane)
4. Presses `d` — verify navigates to diff view (screenshot)
5. Goes back (Esc), presses `c` — verify checkout toast appears
6. Screenshots each state

## Commit

```
feat(tui): add d/c/o key bindings to PR detail + CI summary line

d opens diff from any pane, c triggers checkout, o opens browser.
Checks pane now shows "X/Y passing" summary at the top.
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
