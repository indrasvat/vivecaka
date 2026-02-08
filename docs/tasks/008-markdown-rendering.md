# Task 008: Markdown Rendering with Glamour

## Status: DONE

## Problem

PR bodies and comment text are displayed as raw text. The PRD specifies markdown rendering via Glamour with syntax highlighting. Glamour is not imported anywhere in the codebase despite being a Charm ecosystem library.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.2 (PR Detail View) — PR body rendered as markdown
- Section 7.2.5 (Comments pane) — comment bodies rendered as markdown
- Look for Glamour/markdown rendering specs

## Files to Modify

- `go.mod` — add `github.com/charmbracelet/glamour` dependency
- `internal/tui/views/prdetail.go` — render PR body via Glamour in info pane
- `internal/tui/views/prdetail.go` — render comment bodies via Glamour in comments pane
- `internal/tui/views/prdetail_test.go` — test that rendered output contains formatted markdown elements

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. `go get github.com/charmbracelet/glamour`
4. Create a helper function `renderMarkdown(content string, width int) string`:
   - Use `glamour.NewTermRenderer` with `glamour.WithAutoStyle()` for light/dark detection
   - Set word wrap to `width`
   - Handle errors gracefully — fall back to raw text if rendering fails
5. In `renderInfoPane()`, replace raw `d.Body` with `renderMarkdown(d.Body, paneWidth)`
6. In `renderCommentsPane()`, render each comment body through `renderMarkdown()`
7. Consider caching rendered markdown (PR body doesn't change often)
8. Add tests verifying markdown is processed (e.g., check that `**bold**` produces styled output)
9. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestPRDetail
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with PRs that have markdown in body (headers, code blocks, links, bold)
2. Opens a PR with Enter
3. Screenshots the Info pane — verify markdown formatting is visible (headers, code blocks styled)
4. Tabs to Comments pane — screenshots showing rendered comment markdown
5. Compare against raw text to confirm formatting is applied

## Commit

```
feat(tui): render PR body and comments as markdown via Glamour

PR bodies and comment text were displayed as raw text. Now rendered
through charmbracelet/glamour with auto dark/light detection and
width-aware word wrap. Falls back to raw text on render failure.
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
