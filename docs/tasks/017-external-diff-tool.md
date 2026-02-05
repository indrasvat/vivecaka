# Task 017: External Diff Tool

## Status: TODO

## Problem

The PRD specifies `e` key to open an external diff tool (configurable via config). The `ExternalTool` config field exists but there's zero code to launch it. No `e` key binding anywhere.

## PRD Reference

Read `docs/PRD.md` sections:
- F4 (External Diff Tool) — configurable tool, e key binding
- Config section — `external_tool` field
- Look for tool launch specs, arguments format

## Files to Modify

- `internal/tui/views/diffview.go` — add `e` key handler to emit `OpenExternalDiffMsg`
- `internal/tui/views/messages.go` — add `OpenExternalDiffMsg{Repo, Number, FilePath}`
- `internal/tui/app.go` — handle message, suspend TUI, launch tool, resume TUI
- `internal/tui/commands.go` — add `openExternalDiffCmd()` wrapper
- `internal/config/config.go` — verify `ExternalTool` field and default value

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read config for external tool field
4. Add `e` key in diff view → emit `OpenExternalDiffMsg` with current file info
5. In `app.go`, handle the message:
   - Use `tea.ExecProcess` to suspend BubbleTea and launch the external tool
   - Build command: e.g., `delta`, `difft`, or `git difftool` with appropriate args
   - After tool exits, resume TUI
6. If no external tool configured, show toast: "No external diff tool configured. Set [diff] external_tool in config."
7. Common tools to support argument patterns for: `delta`, `difft`, `diff-so-fancy`, `git difftool`
8. Add tests for message emission and config-not-set path
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestDiffView
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Set config with `external_tool = "delta"` (or another installed tool)
2. Launches `bin/vivecaka`, opens PR, opens diff
3. Presses `e` — verify TUI suspends and external tool launches (screenshot before/after)
4. After tool exits — verify TUI resumes
5. Test without tool configured — verify toast error message

## Commit

```
feat(tui): add e key to open external diff tool

Pressing e in diff view launches configured external diff tool via
tea.ExecProcess (suspends TUI). Shows toast error if no tool
configured. Supports delta, difft, and git difftool.
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
