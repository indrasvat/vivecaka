# Task 018: Inline Comments in Diff View

## Status: TODO

## Depends On: 010 (comments enhancement)

## Problem

The PRD specifies inline comments directly in the diff view:
- `c` on a diff line to add a comment
- View existing inline comments anchored to their lines
- Reply to threads with `r`
- Resolve threads with `x`
- Multi-line comment editor with Ctrl+S to submit

The AddComment and ResolveThread use cases exist but are never called from the diff view. No inline comment display exists.

## PRD Reference

Read `docs/PRD.md` sections:
- S3 (Inline Comments / Suggestion Blocks) — full spec
- Section 7.3 (Diff Viewer) — inline comment key bindings
- Look for comment anchoring, line mapping specs

## Files to Modify

- `internal/tui/views/diffview.go` — add inline comment display between diff lines, add `c`/`r`/`x` key handlers, add comment editor overlay
- `internal/tui/views/messages.go` — add `AddInlineCommentMsg`, `InlineCommentAddedMsg`
- `internal/tui/app.go` — handle inline comment messages, call use cases
- `internal/tui/commands.go` — add `addInlineCommentCmd()` wrapper
- `internal/tui/views/diffview_test.go` — test inline comment rendering and key handling

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections (especially S3)
3. Read `internal/usecase/comment.go` — understand AddComment interface
4. Read `internal/domain/interfaces.go` — `InlineCommentInput` fields (path, line, side, commitSHA)
5. **Display existing comments**:
   - When diff loads, also fetch comments for the PR
   - Map each comment to its file path + line number
   - Render inline comment blocks between diff lines (bordered, indented, with author + timestamp)
   - Use distinct style (e.g., muted background, different border)
6. **Add comment** (`c`):
   - Opens a comment editor below the current diff line
   - Multi-line text input
   - Ctrl+S submits, Esc cancels
   - On submit: emit `AddInlineCommentMsg` with path, line, side, body
7. **Reply** (`r`):
   - If cursor is on a line with existing comment thread, open reply editor
   - Submit adds to the thread
8. **Resolve** (`x`):
   - If cursor is on a comment thread, resolve it
9. In `app.go`: handle messages, call use cases, show toast on success/error
10. Add tests
11. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestDiffView
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with PRs that have inline comments
2. Opens PR, opens diff
3. Screenshots showing inline comments rendered between diff lines
4. Presses `c` on a line — screenshots showing comment editor appears
5. Types comment text — screenshots
6. (If testing against real repo) submits and verifies toast

## Commit

```
feat(tui): add inline comments in diff view

Existing inline comments rendered between diff lines. c key opens
comment editor, r replies to threads, x resolves. Uses AddComment
and ResolveThread use cases. Multi-line editor with Ctrl+S submit.
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
