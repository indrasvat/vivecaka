# Task 012: Confirmation Dialogs

## Status: DONE

## Problem

Destructive actions happen immediately with no confirmation:
- Checkout (`c`) immediately switches branch with no prompt
- Review submit sends immediately with no "Are you sure?"

The PRD specifies confirmation dialogs before these actions.

## PRD Reference

Read `docs/PRD.md` sections:
- Section 7.1 (PR List View) — checkout confirmation with branch name
- Section 7.5 (Review Forms) — submit confirmation
- Look for any other confirmation specs

## Files to Modify

- `internal/tui/views/confirm.go` (CREATE) — reusable confirmation dialog model
- `internal/tui/views/confirm_test.go` (CREATE) — test confirm/cancel flows
- `internal/tui/app.go` — intercept checkout/review messages with confirmation step
- `internal/tui/core/viewstate.go` — may need `ViewConfirm` state

## Execution Steps

1. Read `CLAUDE.md`
2. Read relevant PRD sections
3. Create `internal/tui/views/confirm.go`:
   - `ConfirmModel` with: title, message, confirmLabel, cancelLabel, onConfirm tea.Msg
   - Render as centered bordered box with Y/N or Enter/Esc
   - On confirm → emit the original action message (e.g., the actual checkout msg)
   - On cancel → emit close message, return to previous view
4. In `app.go`, when `CheckoutPRMsg` arrives:
   - Instead of immediately calling `checkoutPRCmd`, show confirmation: "Check out branch `feature-x` for PR #42?"
   - On confirm → proceed with checkout
5. Review submit confirmation is handled by Task 011 (huh form has confirm step), but add fallback here if needed
6. Add tests for confirm/cancel flows
7. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/views/ -run TestConfirm
make ci
```

### Visual (iterm2-driver)
Create and run an iterm2-driver script that:
1. Launches `bin/vivecaka` in a repo with open PRs
2. Presses `c` to checkout — screenshots showing confirmation dialog with branch name
3. Presses Esc to cancel — screenshots showing return to PR list
4. Presses `c` again, then Enter to confirm — screenshots showing checkout proceeds with toast

## Commit

```
feat(tui): add confirmation dialogs before checkout and review submit

Checkout now shows "Check out branch X?" prompt before proceeding.
Reusable ConfirmModel can be applied to any destructive action.
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
