# Task 011: Review Form with Huh Library

## Status: TODO

## Problem

The review form is a bare-bones manual implementation with single-line append-only text editing. The PRD specifies using the `huh` library for the review form with multi-line textarea, action selector, and confirmation. `huh` is listed as a key dependency but never imported.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.5 (Review Forms) — huh library usage, multi-line body
- Look for form field specs, submit confirmation

## Files to Modify

- `internal/tui/views/review.go` — rewrite to use `huh` form components
- `internal/tui/views/review_test.go` — update tests for new form behavior
- May need to check `go.mod` for huh version

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Read current `internal/tui/views/review.go` — understand existing message types and integration points
4. Verify `huh` is in go.mod: `github.com/charmbracelet/huh`
5. Rewrite `ReviewModel` to use `huh.Form`:
   - `huh.NewSelect` for action (Comment/Approve/Request Changes)
   - `huh.NewText` (multi-line textarea) for review body
   - `huh.NewConfirm` for submit confirmation
6. Wire the form lifecycle:
   - `Init()` → create and start the form
   - `Update()` → pass messages to the huh form
   - `View()` → render the form
   - On form completion → emit `SubmitReviewMsg` with action + body
7. Preserve existing message types so `app.go` integration doesn't break
8. Handle cancel (Esc) → emit `CloseReviewMsg`
9. Add tests
10. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestReview
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka`, opens a PR, presses `r` for review
2. Screenshots the review form — verify multi-line textarea, action selector visible
3. Cycles through actions (Tab/arrow) — screenshots
4. Types multi-line review body — screenshots showing text wrapping
5. Submits — screenshots showing confirmation step
6. Verifies toast appears on completion

## Commit

```
feat(tui): rewrite review form using huh library

Replaced bare-bones single-line editor with huh form components:
Select for action type, multi-line Text for body, Confirm for
submit. Proper cursor, multi-line editing, and form navigation.
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
