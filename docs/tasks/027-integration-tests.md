# Task 027: Integration Tests and TUI Test Quality

## Status: DONE

## Depends On: 026 (testify migration, adapter fixtures)

## Problem

1. **No integration tests**: PRD task T13.1 specifies full-flow integration tests using `teatest`. No `teatest` import exists anywhere.
2. **TUI tests are shallow**: Most view tests only verify "doesn't panic" and "returns non-empty string". They don't verify actual content, correct rendering, or end-to-end message flow.

## PRD Reference

Read `CLAUDE.md` — testing requirements, coverage targets
Read `docs/PRD.md` — T13.1 integration test spec

## Files to Create/Modify

- `go.mod` — add `github.com/charmbracelet/x/exp/teatest` dependency
- `internal/tui/integration_test.go` (CREATE) — full flow tests
- `internal/tui/views/*_test.go` — enhance existing tests with content verification
- `internal/tui/app_test.go` — enhance with flow tests (init → repo detect → PR load → open → detail)

## Execution Steps

1. Read `CLAUDE.md`
2. Read docs on `teatest`: https://github.com/charmbracelet/x/tree/main/exp/teatest
3. `go get github.com/charmbracelet/x/exp/teatest`
4. **Integration tests** in `integration_test.go`:
   - Test full flow: Init → WindowSize → RepoDetected → PRsLoaded → verify PR list renders
   - Test navigation flow: PRsLoaded → OpenPR → PRDetailLoaded → verify detail renders
   - Test review flow: OpenPR → StartReview → SubmitReview → ReviewSubmitted → verify toast
   - Test error flow: RepoDetected with error → verify error toast
   - Use mock adapters (not real gh CLI)
5. **Enhance view tests**:
   - PR list: verify View() output contains PR title text, author name, column headers
   - PR detail: verify View() output contains PR number, title in info pane
   - Diff view: verify View() output contains line numbers, +/- markers
   - Help: verify View() output contains key binding text
   - Review: verify View() output contains action labels
6. **teatest tests** (if the library supports BubbleTea v1):
   - Create a test program with mock adapter
   - Send window size, then send key events
   - Assert on final output
   - If teatest requires BubbleTea v2, document this limitation and use manual message-passing tests instead
7. Run `make ci`

## Verification

### Functional
```bash
go test -race -v ./internal/tui/ -run TestIntegration
go test -race -v ./internal/tui/views/
make ci
# Coverage:
go test -coverprofile=coverage.out ./internal/tui/... && go tool cover -func=coverage.out
```

### No Visual (testing-only task)
This task is code-only. No iterm2-driver verification needed.

## Commit

```
test: add integration tests and improve TUI test quality

Added full-flow integration tests: init→load→navigate→review.
Enhanced view tests to verify actual rendered content, not just
non-empty output. Mock adapters used for reproducible testing.
```

## Session Protocol

1. Read `CLAUDE.md`
2. Read this task file
3. Execute
4. Verify (functional only — test pass, coverage)
5. Update `docs/PROGRESS.md` — mark this task done, update coverage numbers
6. `git add` changed files + `git commit`
7. Move to next task or end session
