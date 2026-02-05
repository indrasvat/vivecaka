# Task 002: Clipboard and Browser Integration

## Status: DONE

## Problem

The `y` (copy URL) and `o` (open in browser) keys in the PR list emit `CopyURLMsg` and `OpenBrowserMsg`, but `app.go` never handles these messages — they are silently dropped. No clipboard library is imported. No `exec.Command("open", ...)` exists. These same actions are also missing from the PR detail view and diff view.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.1 (PR List View) — `y` copy URL, `o` open in browser
- Section 7.2 (PR Detail View) — `o` open check URL, `o` open PR in browser
- Section 7.3 (Diff Viewer) — any browser-open mentions

## Files to Modify

- `internal/tui/app.go` — handle `CopyURLMsg` and `OpenBrowserMsg`, call platform helpers
- `internal/tui/platform.go` (CREATE) — cross-platform `openBrowser(url)` and `copyToClipboard(text)` using `exec.Command`
- `internal/tui/platform_test.go` (CREATE) — test helper functions
- `internal/tui/views/prdetail.go` — add `o` key binding to open PR URL or check URL in browser
- `internal/tui/views/prlist.go` — verify existing `y`/`o` emit correct messages with URLs

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Create `internal/tui/platform.go` with:
   - `openBrowser(url string) error` — uses `open` on macOS, `xdg-open` on Linux, `start` on Windows
   - `copyToClipboard(text string) error` — uses `pbcopy` on macOS, `xclip`/`xsel` on Linux, `clip` on Windows
4. In `app.go`, add cases for `CopyURLMsg` and `OpenBrowserMsg`:
   - `CopyURLMsg` → call `copyToClipboard(msg.URL)`, show success/error toast
   - `OpenBrowserMsg` → call `openBrowser(msg.URL)`, show error toast on failure
5. In `prdetail.go`, bind `o` key to emit `OpenBrowserMsg` with the PR's HTML URL
6. Verify `prlist.go` already sets URLs correctly in emitted messages
7. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/...
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with open PRs
2. Waits for PR list to load
3. Presses `y` on a PR — verify toast shows "URL copied" (screenshot)
4. Presses `o` on a PR — verify browser opens (screenshot after, check for toast)
5. Presses Enter to open PR detail
6. Presses `o` in detail view — verify browser opens
7. Screenshots each state

## Commit

```
feat(tui): wire clipboard copy (y) and browser open (o) keys

Both keys were emitting messages that app.go silently dropped. Added
platform helpers for cross-platform clipboard/browser and message
handlers with toast feedback.
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
