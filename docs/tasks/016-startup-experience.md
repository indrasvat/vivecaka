# Task 016: Startup Experience (Tutorial, Banner, Branch Detection)

## Status: DONE

## Problem

Multiple startup features are broken or missing:
1. **Tutorial**: Code exists in `tutorial.go` with `IsFirstLaunch()` and `Show()`, but nobody calls it during Init(). Dead code.
2. **Startup banner**: `show_banner` config exists but no banner rendering code.
3. **Current branch detection**: Branch is only set after checkout, not detected on startup. No `git rev-parse --abbrev-ref HEAD` on init.
4. **Status bar branch**: Current branch not shown in status bar.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.8 (Help System) — first-launch tutorial, 5-step walkthrough
- S1 (Smart Context Awareness) — current branch highlight, branch detection
- Look for startup banner specs
- Status bar — branch display

## Files to Modify

- `internal/tui/app.go` — call tutorial on first launch, detect branch on init, show banner
- `internal/tui/commands.go` — add `detectBranchCmd()` using `git rev-parse --abbrev-ref HEAD`
- `internal/tui/views/messages.go` — add `BranchDetectedMsg{Branch string; Err error}`
- `internal/tui/components/statusbar.go` — add branch display
- `internal/tui/views/tutorial.go` — verify it works when actually called

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read `internal/tui/views/tutorial.go` — understand `IsFirstLaunch()` and `Show()` API
4. **Branch detection**:
   - Add `detectBranchCmd()` to `commands.go` — runs `git rev-parse --abbrev-ref HEAD`
   - Fire it in `Init()` alongside repo detection
   - Handle `BranchDetectedMsg` in `app.go` — set `a.currentBranch`, pass to PR list via `SetCurrentBranch()`
5. **Status bar branch**:
   - In `statusbar.go`, add branch display (e.g., `"⎇ main"` or `"⎇ feature-x"`)
   - Call `SetBranch(branch)` from app after detection
6. **Tutorial**:
   - In `Init()` or after first `WindowSizeMsg`, check `tutorial.IsFirstLaunch()`
   - If true, switch to tutorial view before showing PR list
   - After tutorial completes, mark as shown (XDG flag file) and proceed to PR list
7. **Startup banner** (if PRD specifies):
   - Show brief ASCII art banner for ~200ms before TUI renders
   - Respect `show_banner` config toggle
8. Add tests
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/ -run TestStartup
go test -race -v ./internal/tui/ -run TestTutorial
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Delete the XDG flag file to simulate first launch
2. Launches `bin/vivecaka` — screenshots showing tutorial appears
3. Steps through tutorial — screenshots of each step
4. After tutorial completes, screenshots showing PR list with branch in status bar
5. Relaunch — verify tutorial does NOT appear again
6. Screenshots showing branch name in status bar

## Commit

```
feat(tui): wire startup experience — tutorial, branch detect, status bar

First-launch tutorial now actually shows. Current git branch detected
on startup and displayed in status bar. PR list highlights current
branch PR. Tutorial marks XDG flag file after completion.
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
