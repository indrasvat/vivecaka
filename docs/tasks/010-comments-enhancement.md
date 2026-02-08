# Task 010: Comments Pane Enhancement

## Status: DONE

## Depends On: 008 (markdown rendering for comment bodies)

## Problem

The comments pane in PR detail is read-only. The PRD specifies:
- Collapse/expand comment threads
- Reply to a thread
- Resolve/unresolve threads
- Navigate between threads

Currently it just displays threads with `[resolved]` badges — no interaction.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.2.5 (Comments pane) — thread collapse, reply, resolve
- Look for comment interaction key bindings

## Files to Modify

- `internal/tui/views/prdetail.go` — enhance comments pane rendering with collapse state, add key handlers for reply/resolve
- `internal/tui/views/messages.go` — add `ReplyToThreadMsg`, `ResolveThreadMsg`, `UnresolveThreadMsg` if not present
- `internal/tui/app.go` — handle reply/resolve messages, call use cases
- `internal/tui/commands.go` — add `addCommentCmd()` and `resolveThreadCmd()` wrappers if not present
- `internal/tui/views/prdetail_test.go` — test collapse/expand, message emission

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read comments pane rendering in `prdetail.go`
4. Add per-thread state tracking:
   - `m.commentCollapsed map[int]bool` — collapsed state per thread index
   - `m.commentCursor int` — selected thread/comment
5. Add key handlers when on Comments pane:
   - `j`/`k` — navigate between threads
   - `za` or Space — collapse/expand current thread
   - `r` — reply to current thread (open a text input, emit AddCommentMsg on submit)
   - `x` — resolve current thread (emit ResolveThreadMsg)
   - `X` — unresolve current thread
6. Collapsed threads show: `▶ threadAuthor: firstLine... (N replies) [resolved]`
7. Expanded threads show full comment chain with markdown rendering
8. In `app.go`, handle resolve/reply messages → call use cases → show toast on success
9. Add tests
10. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRDetail
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with PRs that have comment threads
2. Opens a PR, tabs to Comments pane
3. Screenshots showing comment threads rendered
4. Presses Space/za to collapse a thread — screenshots
5. Navigates between threads with j/k — screenshots showing cursor movement
6. (If possible with real repo) presses `r` to start reply — screenshots showing reply editor

## Commit

```
feat(tui): add comment thread collapse, reply, and resolve

Comments pane now supports collapse/expand (Space), reply (r),
and resolve/unresolve (x/X) with j/k thread navigation. Collapsed
threads show summary line with reply count.
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
